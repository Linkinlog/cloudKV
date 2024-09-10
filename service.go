package main

import (
	"context"
	"log/slog"
	"os"
	"runtime"

	"gitlab.com/linkinlog/cloudKV/env"
	ff "gitlab.com/linkinlog/cloudKV/featureflags"
	"gitlab.com/linkinlog/cloudKV/frontend"
	"gitlab.com/linkinlog/cloudKV/logger"
	"gitlab.com/linkinlog/cloudKV/store"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func NewService(f frontend.Frontend, l logger.Logger, sl *slog.Logger) *Service {
	return &Service{
		frontend: f,
		logger:   l,
		slogger:  sl,
	}
}

type Service struct {
	frontend frontend.Frontend
	logger   logger.Logger
	slogger  *slog.Logger

	cancel context.CancelFunc
}

func (s *Service) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	s.logger.Run()

	exporter, err := prometheus.New()
	if err != nil {
		panic(err)
	}

	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	defer func() { _ = provider.Shutdown(ctx) }()

	meter := provider.Meter(env.ServiceName())

	if err := buildRuntimeObservers(meter); err != nil {
		panic(err)
	}

	telemetry := ff.New(ff.UseTelemetry, nil).Enabled()

	if telemetry {
		if err, shutdown := setupTelemetry(); err != nil {
			panic(err)
		} else if shutdown != nil {
			defer func() { _ = shutdown(ctx) }()
		}
	}

	keyVal := store.New(telemetry)
	if err := s.replay(keyVal); err != nil {
		panic(err)
	}

	frontendErrors := s.frontend.Start(keyVal)

	for {
		select {
		case err := <-frontendErrors:
			if err != nil {
				s.slogger.Error("s.frontend", "error", err)
			}
		case err := <-s.logger.Err():
			if err != nil {
				s.slogger.Error("s.logger", "error", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) Stop() {
	if s.cancel != nil {
		s.cancel()
	}

	s.logger.Close()

	if err := s.frontend.Close(context.Background()); err != nil {
		s.slogger.Error("s.frontend.Close()", "error", err.Error())
	}
}

func (s *Service) SwitchFrontend(f frontend.Frontend) {
	if f != nil {
		s.frontend = f
	}
}

func (s *Service) SwitchLogger(l logger.Logger) {
	if l != nil {
		s.logger = l
	}
}

func (s *Service) replay(kv *store.KeyValueStore) error {
	events, errs := s.logger.ReadEvents()

	var (
		ok  bool = true
		e   store.Event
		err error
	)

	for ok && err == nil {
		select {
		case err, ok = <-errs:
			if err != nil {
				return err
			}
		case e, ok = <-events:
			switch e.EventType {
			case store.EventPut:
				if err := kv.Put(e.Key, e.Value); err != nil {
					return err
				}
			case store.EventDelete:
				if err := kv.Delete(e.Key); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func setupTelemetry() (error, func(context.Context) error) {
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(env.ServiceName()),
		),
	)
	if err != nil {
		return err, nil
	}

	endpoint := env.JaegerEndpoint()

	jaeger, err := otlptracegrpc.New(
		context.Background(),
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return err, nil
	}

	tp := trace.NewTracerProvider(
		trace.WithResource(res),
		trace.WithBatcher(jaeger),
	)

	otel.SetTracerProvider(tp)

	return nil, tp.Shutdown
}

var attributes = []attribute.KeyValue{
	attribute.Key("application").String(env.ServiceName()),
	attribute.Key("container_id").String(os.Getenv("HOSTNAME")),
}

func buildRuntimeObservers(meter metric.Meter) error {
	var err error
	var m runtime.MemStats

	_, err = meter.Int64ObservableUpDownCounter("cloudkv_memory_usage_bytes",
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			runtime.ReadMemStats(&m)
			o.Observe(int64(m.Sys), metric.WithAttributes(attributes...))
			return nil
		}),
		metric.WithDescription("Amount of memory used."),
		metric.WithUnit("By"),
	)
	if err != nil {
		return err
	}

	_, err = meter.Int64ObservableGauge(
		"cloudkv_num_goroutines",
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			o.Observe(int64(runtime.NumGoroutine()), metric.WithAttributes(attributes...))
			return nil
		}),
		metric.WithDescription("Number of running goroutines."),
		metric.WithUnit("{goroutine}"),
	)
	if err != nil {
		return err
	}

	return nil
}

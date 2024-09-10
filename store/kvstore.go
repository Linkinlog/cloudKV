package store

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"gitlab.com/linkinlog/cloudKV/env"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var ErrNoSuchKey = errors.New("no such key")

type KeyValueStore struct {
	lock      *sync.Mutex
	m         map[string]string
	telemetry bool
}

func New(telemetry bool) *KeyValueStore {
	m := make(map[string]string)
	lock := &sync.Mutex{}

	return &KeyValueStore{lock: lock, m: m, telemetry: telemetry}
}

func (k *KeyValueStore) Put(key, value string) error {
	k.lock.Lock()
	defer k.lock.Unlock()

	var sp trace.Span
	if k.telemetry {
		tr := otel.GetTracerProvider().Tracer(env.ServiceName())

		_, sp = tr.Start(context.Background(),
			fmt.Sprintf("Put(%s, %s)", key, value),
			trace.WithAttributes(attribute.String("key", key)),
			trace.WithAttributes(attribute.String("value", value)),
		)
		defer sp.End()
	}

	k.m[key] = value

	if k.telemetry && sp != nil {
		sp.SetAttributes(attribute.Bool("success", true))
	}
	return nil
}

func (k *KeyValueStore) Delete(key string) error {
	k.lock.Lock()
	defer k.lock.Unlock()

	var sp trace.Span
	if k.telemetry {
		tr := otel.GetTracerProvider().Tracer(env.ServiceName())

		_, sp = tr.Start(context.Background(),
			fmt.Sprintf("Delete(%s)", key),
			trace.WithAttributes(attribute.String("key", key)),
		)
		defer sp.End()
	}

	delete(k.m, key)

	if k.telemetry && sp != nil {
		sp.SetAttributes(attribute.Bool("success", true))
	}

	return nil
}

func (k *KeyValueStore) Get(key string) (string, error) {
	k.lock.Lock()
	defer k.lock.Unlock()

	var sp trace.Span
	if k.telemetry {
		tr := otel.GetTracerProvider().Tracer(env.ServiceName())

		_, sp = tr.Start(context.Background(),
			fmt.Sprintf("Get(%s)", key),
			trace.WithAttributes(attribute.String("key", key)),
		)

		defer sp.End()
	}

	value, ok := k.m[key]
	if !ok {
		return "", ErrNoSuchKey
	}

	if k.telemetry && sp != nil {
		sp.SetAttributes(attribute.Bool("success", ok))
	}

	return value, nil
}

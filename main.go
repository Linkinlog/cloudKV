package main

import (
	"log/slog"
	"os"

	"gitlab.com/linkinlog/cloudKV/env"
	"gitlab.com/linkinlog/cloudKV/frontend"
	"gitlab.com/linkinlog/cloudKV/logger"
	"gitlab.com/linkinlog/cloudKV/store"
)

func main() {
	opts := slog.HandlerOptions{AddSource: true, Level: slog.LevelInfo}
	slogger := slog.New(slog.NewTextHandler(os.Stdout, &opts))

	loggerType := logger.ToLoggerType(env.Logger())
	logger, err := logger.New(loggerType)
	if err != nil {
		panic(err)
	}
    defer logger.Close()

	keyVal := store.New()

	if err := replay(logger, keyVal); err != nil {
		panic(err)
	}

	logger.Run()

	frontendType := frontend.ToFrontendType(env.Frontend())

	frontend := frontend.New(logger, frontendType)
	frontendErrors := frontend.Start(keyVal)

	slogger.Info("listening", "frontend", frontendType.String(), "logger", loggerType.String())

	for {
		select {
		case err := <-frontendErrors:
			if err != nil {
				slogger.Error("frontend", "error", err)
			}
		case err := <-logger.Err():
			if err != nil {
				slogger.Error("logger", "error", err)
			}
		}
	}
}

func replay(l logger.Logger, kv *store.KeyValueStore) error {
	events, errs := l.ReadEvents()

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

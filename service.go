package main

import (
	"context"
	"log/slog"

	"gitlab.com/linkinlog/cloudKV/frontend"
	"gitlab.com/linkinlog/cloudKV/logger"
	"gitlab.com/linkinlog/cloudKV/store"
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

	keyVal := store.New()
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
	if err := s.frontend.Close(context.Background()); err != nil {
		s.slogger.Error("s.frontend.Close()", "error", err.Error())
	}
	s.logger.Close()
}

func (s *Service) Switch(l logger.Logger, f frontend.Frontend) {
	s.Stop()

	if l != nil {
		s.logger = l
	}
	if f != nil {
		s.frontend = f
	}

	go s.Start()
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

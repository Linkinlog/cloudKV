package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"

	"github.com/fsnotify/fsnotify"
	"gitlab.com/linkinlog/cloudKV/frontend"
	"gitlab.com/linkinlog/cloudKV/logger"
)

type ConfigFile struct {
	Logger   string `json:"logger"`
	Frontend string `json:"frontend"`
}

var defaultConfig = ConfigFile{
	Logger:   "File",
	Frontend: "REST",
}

func watchFile(configPath string, s *Service, sl *slog.Logger) (<-chan error, context.CancelFunc) {
	errs := make(chan error, 1)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		errs <- err
		return errs, nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if !event.Has(fsnotify.Rename) || !event.Has(fsnotify.Remove) {
					conf, err := GetConfig(configPath)
					if err != nil {
						errs <- err
						return
					}

					s.Stop()

					lt := logger.ToLoggerType(conf.Logger)
					l, err := logger.New(lt)
					if err != nil {
						errs <- err
						return
					}
					s.SwitchLogger(l)

					ft := frontend.ToFrontendType(conf.Frontend)
					f := frontend.New(l, ft)
					s.SwitchFrontend(f)

					sl.Info("config change detected, reloading", "logger", lt.String(), "frontend", ft.String())
					go s.Start()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				errs <- err
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	err = watcher.Add(configPath)
	if err != nil {
		errs <- err
		return errs, cancel
	}

	sl.Info("config watcher", "watching", configPath)

	return errs, cancel
}

func GetConfig(configPath string) (*ConfigFile, error) {
	file, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	conf := &ConfigFile{}
	if err := json.Unmarshal(file, conf); err != nil {
		return nil, err
	}

	return conf, nil
}

func GetOrMakeConfig(configPath string) (*ConfigFile, error) {
	existingConf, err := GetConfig(configPath)
	if existingConf != nil && err == nil {
		return existingConf, nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	f, err := os.Create(configPath)
	if err != nil {
		return nil, err
	}

	contents, err := json.Marshal(defaultConfig)
	if err != nil {
		return nil, err
	}

	if _, err := f.Write(contents); err != nil {
		return nil, err
	}

	return &defaultConfig, nil
}

package main

import (
	"log/slog"
	"os"
	"path/filepath"

	"gitlab.com/linkinlog/cloudKV/env"
	"gitlab.com/linkinlog/cloudKV/frontend"
	"gitlab.com/linkinlog/cloudKV/logger"
)

const (
	configFileName string = "cloudKV.json"
	configFileRoot string = "cloudKV"
)

var configPath string

func init() {
	dirName := env.ConfigPath()

	if _, err := os.Stat(dirName); err != nil {
		if err := os.MkdirAll(dirName, os.ModeDir); err != nil {
			panic(err)
		}
	}

	configRootPath := filepath.Join(dirName, configFileRoot)

	if _, err := os.Stat(configRootPath); err != nil {
		if err := os.MkdirAll(configRootPath, os.ModeDir); err != nil {
			panic(err)
		}
	}

	configPath = filepath.Join(configRootPath, configFileName)
}

func main() {
	opts := slog.HandlerOptions{AddSource: true, Level: slog.LevelInfo}
	slogger := slog.New(slog.NewJSONHandler(os.Stdout, &opts))

	conf, err := GetOrMakeConfig(configPath)
	if err != nil {
		panic(err)
	}

	loggerType := logger.ToLoggerType(conf.Logger)
	logger, err := logger.New(loggerType)
	if err != nil {
		panic(err)
	}
	frontendType := frontend.ToFrontendType(conf.Frontend)
	frontend := frontend.New(logger, frontendType)

	slogger.Info("listening", "s.frontend", frontendType.String(), "s.logger", loggerType.String())
	s := NewService(frontend, logger, slogger)
	go s.Start()

	errChan, cancel := watchFile(configPath, s, slogger)
	defer cancel()

	if err := <-errChan; err != nil {
		panic(err)
	}
}

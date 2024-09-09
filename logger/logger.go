package logger

import (
	"fmt"

	"gitlab.com/linkinlog/cloudKV/env"
	"gitlab.com/linkinlog/cloudKV/store"
)

type Logger interface {
	LogPut(key, value string) error
	LogDelete(key string) error

	Close() error

	Err() <-chan error

	ReadEvents() (<-chan store.Event, <-chan error)

	Run()
}

func New(l LoggerType) (Logger, error) {
	switch l {
	case File:
		return NewFileTransactionLogger(env.ConfigPath() + "/data")
	case PSQL:
		params := PostgresDBParams{
			dbName:   env.DBName(),
			host:     env.DBHost(),
			user:     env.DBUser(),
			password: env.DBPass(),
		}

		return NewPostgresTransactionLogger(params)
	}
	return nil, fmt.Errorf("invalid loggerType %v", l)
}

func ToLoggerType(s string) LoggerType {
	switch s {
	case "File":
		return File
	case "PSQL":
		return PSQL
	}
	return 0
}

type LoggerType int

const (
	_ LoggerType = iota
	File
	PSQL
)

func (l LoggerType) String() string {
	return []string{"File", "PSQL"}[l-1]
}

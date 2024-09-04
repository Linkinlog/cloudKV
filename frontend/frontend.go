package frontend

import (
	"gitlab.com/linkinlog/cloudKV/frontend/grpc"
	"gitlab.com/linkinlog/cloudKV/logger"
	"gitlab.com/linkinlog/cloudKV/store"
)

type Frontend interface {
	Start(*store.KeyValueStore) <-chan error
}

func New(l logger.Logger, f FrontendType) Frontend {
	switch f {
	case GRPC:
		return grpc.NewGRPCServer(l)
	case REST:
		return NewRESTServer(l)
	}

	return nil
}

func ToFrontendType(s string) FrontendType {
	switch s {
	case "GRPC":
		return GRPC
	case "REST":
		return REST
	}
	return 0
}

type FrontendType int

const (
	_ FrontendType = iota
	GRPC
	REST
)

func (f FrontendType) String() string {
    return []string{"GRPC", "REST"}[f-1]
}

package grpc

import (
	context "context"
	"fmt"
	"net"

	"gitlab.com/linkinlog/cloudKV/env"
	"gitlab.com/linkinlog/cloudKV/logger"
	"gitlab.com/linkinlog/cloudKV/store"
	"google.golang.org/grpc"
)

func NewGRPCServer(l logger.Logger) *GRPCServer {
	return &GRPCServer{
		l: l,
	}
}

type GRPCServer struct {
	UnimplementedKeyValueServer
	kv *store.KeyValueStore

	l logger.Logger

	err chan error
}

func (s *GRPCServer) Get(ctx context.Context, gr *GetRequest) (*GetResponse, error) {
	val, err := s.kv.Get(gr.Key)
	if err != nil {
		return nil, err
	}
	return &GetResponse{Value: val}, nil
}

func (s *GRPCServer) Put(ctx context.Context, pr *PutRequest) (*PutResponse, error) {
	err := s.kv.Put(pr.Key, pr.Value)
	if err != nil {
		return nil, err
	}

	if err := s.l.LogPut(pr.Key, pr.Value); err != nil {
		return nil, err
	}

    return &PutResponse{Key: pr.Key, Value: pr.Value}, nil
}

func (s *GRPCServer) Delete(ctx context.Context, dr *DeleteRequest) (*DeleteResponse, error) {
	if err := s.kv.Delete(dr.Key); err != nil {
		return nil, err
	}

	if err := s.l.LogDelete(dr.Key); err != nil {
		return nil, err
	}

    return &DeleteResponse{Key: dr.Key}, nil
}

func (s *GRPCServer) Start(kv *store.KeyValueStore) <-chan error {
	s.kv = kv
	s.err = make(chan error)

	gs := grpc.NewServer()

	RegisterKeyValueServer(gs, s)

	go func() {
		lis, err := net.Listen("tcp", env.FrontendPort())
		if err != nil {
			s.err <- fmt.Errorf("can't hear shit! %w", err)
		}

		if err := gs.Serve(lis); err != nil {
			s.err <- fmt.Errorf("failed to serve game! %w", err)
		}
	}()

	return s.err
}

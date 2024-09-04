package frontend

import (
	"fmt"
	"net/http"

	"gitlab.com/linkinlog/cloudKV/env"
	"gitlab.com/linkinlog/cloudKV/logger"
	"gitlab.com/linkinlog/cloudKV/store"
)

func NewRESTServer(l logger.Logger) *RESTServer {
	return &RESTServer{l: l}
}

type RESTServer struct {
	l logger.Logger
}

func (s *RESTServer) Start(kv *store.KeyValueStore) <-chan error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{key}", get(kv))
	mux.HandleFunc("PUT /{key}", s.put(kv))
	mux.HandleFunc("DELETE /{key}", s.del(kv))

	errs := make(chan error)

	go func() {
		if err := http.ListenAndServe(env.FrontendPort(), mux); err != nil {
			errs <- fmt.Errorf("can't hear shit! %w", err)
		}
	}()

	return errs
}

func get(kv *store.KeyValueStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")

		if key == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("invalid key"))
			return
		}

		val, err := kv.Get(key)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("unable to get key"))
			return
		}

		_, _ = w.Write([]byte(val))
	}
}

func (s *RESTServer) put(kv *store.KeyValueStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")

		if key == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("invalid key"))
			return
		}

		val := r.FormValue("value")
		if val == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("invalid value"))
			return
		}

		if err := kv.Put(key, val); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("unable to set key"))
			return
		}

		if err := s.l.LogPut(key, val); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func (s *RESTServer) del(kv *store.KeyValueStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")

		if key == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("invalid key"))
			return
		}

		if err := kv.Delete(key); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("unable to delete key"))
			return
		}

		if err := s.l.LogDelete(key); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

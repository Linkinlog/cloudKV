package frontend

import (
	"context"
	"fmt"
	"net/http"

	"gitlab.com/linkinlog/cloudKV/env"
	"gitlab.com/linkinlog/cloudKV/featureflags"
	"gitlab.com/linkinlog/cloudKV/logger"
	"gitlab.com/linkinlog/cloudKV/store"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func NewRESTServer(l logger.Logger) *RESTServer {
	return &RESTServer{l: l}
}

type RESTServer struct {
	l logger.Logger
    s *http.Server
}

func (s *RESTServer) Start(kv *store.KeyValueStore) <-chan error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{key}", tracingMiddleware(get(kv)))
	mux.HandleFunc("PUT /{key}", tracingMiddleware(s.put(kv)))
	mux.HandleFunc("DELETE /{key}", tracingMiddleware(s.del(kv)))

	errs := make(chan error)

    server := &http.Server{
        Addr: env.FrontendPort(),
        Handler: mux,
    }
    s.s = server

	go func() {
		if err := server.ListenAndServe(); err != nil {
			errs <- fmt.Errorf("can't hear shit! %w", err)
		}
	}()

	return errs
}

func (s *RESTServer) Close(ctx context.Context) error {
    if s.s == nil {
        return nil
    }
    return s.s.Shutdown(ctx)
}

func tracingMiddleware(next http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if featureflags.Enabled("tracing", r) {
			handler := otelhttp.NewHandler(next, "root")

			handler.ServeHTTP(w, r)
		} else {
			next.ServeHTTP(w, r)
		}
	})
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

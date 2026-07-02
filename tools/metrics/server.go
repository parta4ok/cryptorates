package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"cryptorates/tools/port"
)

var (
	ErrInvalidParam = errors.New("invalid param")
)

var (
	_ port.Port = (*Server)(nil)
)

type Server struct {
	router *http.Server
}

func NewServer(port string, timeout time.Duration) (*Server, error) {
	if port == "" {
		return nil, errors.Wrap(ErrInvalidParam, "port not set")
	}

	if timeout == 0 {
		return nil, errors.Wrap(ErrInvalidParam, "timeout not set")
	}

	return &Server{
		router: &http.Server{
			ReadTimeout:  timeout,
			WriteTimeout: timeout,
			Addr:         fmt.Sprintf(":%s", port),
		},
	}, nil
}

func (s *Server) Start() error {
	s.registerRoutes()

	go func() {
		if err := s.router.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server stopped", "err", err.Error())
		}
	}()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	slog.Info("server will be stopping")

	if err := s.router.Shutdown(ctx); err != nil {
		slog.Error("server shutdown", "error", err)
		return err //nolint:wrapcheck //ok
	}

	return nil
}

func (s *Server) registerRoutes() {
	router := chi.NewRouter()

	router.Get("/metrics", promhttp.Handler().ServeHTTP)
	router.HandleFunc("/debug/pprof/", pprof.Index)
	router.HandleFunc("/debug/pprof/{profile}", pprof.Index)
	router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	router.HandleFunc("/debug/pprof/trace", pprof.Trace)

	s.router.Handler = router
}

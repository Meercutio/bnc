package httpx

import (
	"context"
	"errors"
	"net/http"
	"time"
)

type Server struct {
	http *http.Server
}

func New(addr string, handler http.Handler) *Server {
	return &Server{
		http: &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	err := s.http.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

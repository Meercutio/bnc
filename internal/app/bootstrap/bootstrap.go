package bootstrap

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/meercutio/bnc/internal/platform/health"
	"github.com/meercutio/bnc/internal/platform/httpx"
)

type HTTPBundle struct {
	Server *httpx.Server
	Probe  *health.Probe
}

func NewHTTP(service string, _ *slog.Logger, addr string) *HTTPBundle {
	probe := health.New()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", probe.Liveness)
	mux.HandleFunc("/readyz", probe.Readiness)
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(service))
	})

	srv := httpx.New(addr, mux)
	return &HTTPBundle{Server: srv, Probe: probe}
}

func WithSignals(ctx context.Context, logger *slog.Logger) (context.Context, func()) {
	ctx, cancel := context.WithCancel(ctx)

	ch := make(chan os.Signal, 2)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-ch
		logger.Info("shutdown signal", "signal", sig.String())
		cancel()
	}()

	return ctx, func() {
		signal.Stop(ch)
		close(ch)
	}
}

func ShutdownHTTP(logger *slog.Logger, bundle *HTTPBundle) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := bundle.Server.Shutdown(ctx); err != nil {
		logger.Error("http shutdown error", "err", err)
	}
}

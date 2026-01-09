package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/meercutio/bnc/internal/app/bootstrap"
	"github.com/meercutio/bnc/internal/platform/log"
)

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	service := "gateway"
	logger := log.New(service)

	httpAddr := getenv("HTTP_ADDR", ":8080")

	ctx, stopSignals := bootstrap.WithSignals(context.Background(), logger)
	defer stopSignals()

	httpBundle := bootstrap.NewHTTP(service, logger, httpAddr)

	// Iteration 0: no real handlers yet (future: REST + WS + gRPC clients)
	httpBundle.Probe.MarkReadyAfter(ctx, 300*time.Millisecond)

	go func() {
		logger.Info("http start", "addr", httpAddr)
		if err := httpBundle.Server.Start(); err != nil && err != http.ErrServerClosed {
			logger.Error("http error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("stopping")
	bootstrap.ShutdownHTTP(logger, httpBundle)
	logger.Info("stopped")
}

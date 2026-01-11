package main

import (
	"context"
	"os"
	"time"

	"github.com/yourname/bulls-cows/internal/app/bootstrap"
	"github.com/yourname/bulls-cows/internal/platform/grpcx"
	"github.com/yourname/bulls-cows/internal/platform/log"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	service := "game"
	logger := log.New(service)

	httpAddr := getenv("HTTP_ADDR", ":8003")
	grpcAddr := getenv("GRPC_ADDR", ":50053")

	ctx, stopSignals := bootstrap.WithSignals(context.Background(), logger)
	defer stopSignals()

	httpBundle := bootstrap.NewHTTP(service, logger, httpAddr, nil)

	gs, err := grpcx.New(grpcAddr)
	if err != nil {
		logger.Error("grpc listen error", "err", err)
		os.Exit(1)
	}

	hs := health.NewServer()
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(gs.GRPC(), hs)

	// Iteration 0: mark ready shortly after start
	httpBundle.Probe.MarkReadyAfter(ctx, 500*time.Millisecond)

	go func() {
		logger.Info("http start", "addr", httpAddr)
		if err := httpBundle.Server.Start(); err != nil {
			logger.Error("http error", "err", err)
			os.Exit(1)
		}
	}()
	go func() {
		logger.Info("grpc start", "addr", grpcAddr)
		if err := gs.Serve(); err != nil {
			logger.Error("grpc error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("stopping")

	gs.Stop()
	bootstrap.ShutdownHTTP(logger, httpBundle)
	logger.Info("stopped")
}

package main

import (
	"context"
	"os"

	mmv1 "github.com/yourname/bulls-cows/gen/go/api/proto/matchmaking/v1"
	"github.com/yourname/bulls-cows/internal/app/bootstrap"
	mmgrpc "github.com/yourname/bulls-cows/internal/matchmaking/grpc"
	mmsvc "github.com/yourname/bulls-cows/internal/matchmaking/service"
	"github.com/yourname/bulls-cows/internal/platform/grpcx"
	"github.com/yourname/bulls-cows/internal/platform/log"
	"github.com/yourname/bulls-cows/internal/platform/redisx"
	"github.com/yourname/bulls-cows/internal/realtime"
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
	service := "matchmaking"
	logger := log.New(service)

	httpAddr := getenv("HTTP_ADDR", ":8002")
	grpcAddr := getenv("GRPC_ADDR", ":50052")
	redisAddr := getenv("REDIS_ADDR", "redis:6379")

	ctx, stop := bootstrap.WithSignals(context.Background(), logger)
	defer stop()

	httpBundle := bootstrap.NewHTTP(service, logger, httpAddr, nil)

	// Redis
	rdb := redisx.New(redisAddr)
	defer rdb.Close()
	if err := redisx.Ping(ctx, rdb); err != nil {
		logger.Error("redis ping error", "err", err)
		os.Exit(1)
	}

	pub := realtime.NewPublisher(rdb)
	svc := mmsvc.New(rdb, pub)
	handler := mmgrpc.New(svc)

	// gRPC
	gs, err := grpcx.New(grpcAddr)
	if err != nil {
		logger.Error("grpc listen error", "err", err)
		os.Exit(1)
	}

	mmv1.RegisterMatchmakingServiceServer(gs.GRPC(), handler)

	hs := health.NewServer()
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(gs.GRPC(), hs)

	httpBundle.Probe.SetReady(true)

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

package main

import (
	"context"
	"os"

	authv1 "github.com/meercutio/bnc/gen/go/api/proto/auth/v1"
	"github.com/meercutio/bnc/internal/app/bootstrap"
	"github.com/meercutio/bnc/internal/auth/config"
	authgrpc "github.com/meercutio/bnc/internal/auth/grpc"
	"github.com/meercutio/bnc/internal/auth/jwt"
	"github.com/meercutio/bnc/internal/auth/store"
	"github.com/meercutio/bnc/internal/platform/db"
	"github.com/meercutio/bnc/internal/platform/grpcx"
	"github.com/meercutio/bnc/internal/platform/log"
	"github.com/pressly/goose/v3"
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
	service := "auth"
	logger := log.New(service)

	httpAddr := getenv("HTTP_ADDR", ":8001")
	grpcAddr := getenv("GRPC_ADDR", ":50051")

	cfg := config.Load()
	if cfg.PostgresDSN == "" {
		logger.Error("POSTGRES_DSN is required")
		os.Exit(1)
	}

	ctx, stopSignals := bootstrap.WithSignals(context.Background(), logger)
	defer stopSignals()

	httpBundle := bootstrap.NewHTTP(service, logger, httpAddr, nil)

	// 1) DB connect + ping
	pg, err := db.Open(cfg.PostgresDSN)
	if err != nil {
		logger.Error("db open error", "err", err)
		os.Exit(1)
	}
	defer pg.Close()

	if err := db.Ping(ctx, pg); err != nil {
		logger.Error("db ping error", "err", err)
		os.Exit(1)
	}

	// 2) Migrations (goose)
	goose.SetDialect("postgres")
	if err := goose.Up(pg, cfg.MigrationsDir); err != nil {
		logger.Error("migrations failed", "err", err)
		os.Exit(1)
	}

	// 3) Create auth components
	st := store.New(pg)
	jm := jwt.New(cfg.JWTSecret, cfg.AccessTTL)
	handler := authgrpc.New(st, jm, cfg.RefreshTTL)

	// 4) gRPC server
	gs, err := grpcx.New(grpcAddr)
	if err != nil {
		logger.Error("grpc listen error", "err", err)
		os.Exit(1)
	}

	authv1.RegisterAuthServiceServer(gs.GRPC(), handler)

	hs := health.NewServer()
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(gs.GRPC(), hs)

	// ready only after DB+migrations ok
	httpBundle.Probe.SetReady(true)

	// start servers
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

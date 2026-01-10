package main

import (
	"context"
	"net/http"
	"os"
	"time"

	authv1 "github.com/meercutio/bnc/gen/go/api/proto/auth/v1"
	"github.com/meercutio/bnc/internal/app/bootstrap"
	"github.com/meercutio/bnc/internal/gateway/authhttp"
	"github.com/meercutio/bnc/internal/gateway/httpapi"
	"github.com/meercutio/bnc/internal/gateway/middleware"
	"github.com/meercutio/bnc/internal/platform/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	authAddr := getenv("AUTH_GRPC_ADDR", "auth:50051")

	ctx, stopSignals := bootstrap.WithSignals(context.Background(), logger)
	defer stopSignals()

	// gRPC client (auth)
	conn, err := dialWithRetry(authAddr, 20*time.Second)
	if err != nil {
		logger.Error("cannot dial auth grpc", "err", err)
		os.Exit(1)
	}
	defer conn.Close()
	authClient := authv1.NewAuthServiceClient(conn)

	authH := authhttp.New(authClient)

	httpBundle := bootstrap.NewHTTP(service, logger, httpAddr, func(mux *http.ServeMux) {
		// public auth endpoints
		mux.HandleFunc("/auth/register", authH.Register)
		mux.HandleFunc("/auth/login", authH.Login)
		mux.HandleFunc("/auth/refresh", authH.Refresh)

		// protected demo endpoint
		mux.Handle("/me", middleware.RequireAuth(authClient, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uid, _ := r.Context().Value(middleware.UserIDKey).(string)
			httpapi.WriteJSON(w, http.StatusOK, map[string]string{"user_id": uid})
		})))
	})

	httpBundle.Probe.SetReady(true)

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

func dialWithRetry(addr string, timeout time.Duration) (*grpc.ClientConn, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err == nil {
			return conn, nil
		}
		lastErr = err
		time.Sleep(300 * time.Millisecond)
	}
	return nil, lastErr
}

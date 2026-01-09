package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	authv1 "github.com/meercutio/bnc/gen/go/api/proto/auth/v1"
	"github.com/meercutio/bnc/internal/app/bootstrap"
	"github.com/meercutio/bnc/internal/platform/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ctxKey string

const userIDKey ctxKey = "user_id"

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func main() {
	service := "gateway"
	logger := log.New(service)

	httpAddr := getenv("HTTP_ADDR", ":8080")
	authAddr := getenv("AUTH_GRPC_ADDR", "auth:50051")

	ctx, stopSignals := bootstrap.WithSignals(context.Background(), logger)
	defer stopSignals()

	// gRPC client to auth (with simple retry)
	var conn *grpc.ClientConn
	var err error
	deadline := time.Now().Add(20 * time.Second)
	for {
		conn, err = grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			logger.Error("cannot dial auth grpc", "err", err)
			os.Exit(1)
		}
		time.Sleep(300 * time.Millisecond)
	}
	defer conn.Close()
	authClient := authv1.NewAuthServiceClient(conn)

	httpBundle := bootstrap.NewHTTP(service, logger, httpAddr, func(mux *http.ServeMux) {
		// public auth routes
		mux.HandleFunc("/auth/register", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			var in struct {
				Email    string `json:"email"`
				Password string `json:"password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				writeJSON(w, 400, map[string]string{"error": "bad json"})
				return
			}
			resp, err := authClient.Register(r.Context(), &authv1.RegisterRequest{Email: in.Email, Password: in.Password})
			if err != nil {
				writeJSON(w, 401, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, resp)
		})

		mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			var in struct {
				Email    string `json:"email"`
				Password string `json:"password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				writeJSON(w, 400, map[string]string{"error": "bad json"})
				return
			}
			resp, err := authClient.Login(r.Context(), &authv1.LoginRequest{Email: in.Email, Password: in.Password})
			if err != nil {
				writeJSON(w, 401, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, resp)
		})

		mux.HandleFunc("/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			var in struct {
				RefreshToken string `json:"refresh_token"`
			}
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				writeJSON(w, 400, map[string]string{"error": "bad json"})
				return
			}
			resp, err := authClient.Refresh(r.Context(), &authv1.RefreshRequest{RefreshToken: in.RefreshToken})
			if err != nil {
				writeJSON(w, 401, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, 200, resp)
		})

		// protected demo endpoint
		mux.Handle("/me", authMiddleware(authClient, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uid, _ := r.Context().Value(userIDKey).(string)
			writeJSON(w, 200, map[string]string{"user_id": uid})
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

func authMiddleware(authClient authv1.AuthServiceClient, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if h == "" || !strings.HasPrefix(h, "Bearer ") {
			writeJSON(w, 401, map[string]string{"error": "missing bearer token"})
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
		v, err := authClient.Validate(r.Context(), &authv1.ValidateRequest{AccessToken: token})
		if err != nil || v.GetUserId() == "" {
			writeJSON(w, 401, map[string]string{"error": "invalid token"})
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, v.GetUserId())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

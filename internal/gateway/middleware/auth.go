package middleware

import (
	"context"
	"net/http"
	"strings"

	authv1 "github.com/yourname/bulls-cows/gen/go/api/proto/auth/v1"
	"github.com/yourname/bulls-cows/internal/gateway/grpcerr"
	"github.com/yourname/bulls-cows/internal/gateway/httpapi"
)

type ctxKey string

const UserIDKey ctxKey = "user_id"

func RequireAuth(auth authv1.AuthServiceClient, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if h == "" || !strings.HasPrefix(h, "Bearer ") {
			httpapi.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))

		v, err := auth.Validate(r.Context(), &authv1.ValidateRequest{AccessToken: token})
		if err != nil {
			grpcerr.Write(w, err) // будет 401/403 и т.п. в зависимости от gRPC
			return
		}
		if v.GetUserId() == "" {
			httpapi.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, v.GetUserId())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

package grpc

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	authv1 "github.com/yourname/bulls-cows/gen/go/api/proto/auth/v1"
	"github.com/yourname/bulls-cows/internal/auth/jwt"
	"github.com/yourname/bulls-cows/internal/auth/store"
	"github.com/yourname/bulls-cows/internal/auth/tokens"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	authv1.UnimplementedAuthServiceServer
	store      *store.Store
	jwt        *jwt.Manager
	refreshTTL time.Duration
}

func New(store *store.Store, jwt *jwt.Manager, refreshTTL time.Duration) *Handler {
	return &Handler{store: store, jwt: jwt, refreshTTL: refreshTTL}
}

func (h *Handler) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.AuthResponse, error) {
	email := strings.TrimSpace(req.GetEmail())
	pass := req.GetPassword()
	if email == "" || len(pass) < 6 {
		return nil, status.Error(codes.InvalidArgument, "invalid email or password")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(pass), 12)
	if err != nil {
		return nil, status.Error(codes.Internal, "hash failed")
	}

	userID, err := h.store.CreateUser(ctx, email, string(hash))
	if errors.Is(err, store.ErrEmailExists) {
		return nil, status.Error(codes.AlreadyExists, "email already exists")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "db error")
	}

	return h.issueTokens(ctx, userID)
}

func (h *Handler) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.AuthResponse, error) {
	email := strings.TrimSpace(req.GetEmail())
	pass := req.GetPassword()
	if email == "" || pass == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid credentials")
	}

	u, err := h.store.GetUserByEmail(ctx, email)
	if errors.Is(err, store.ErrNotFound) {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "db error")
	}

	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(pass)) != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	return h.issueTokens(ctx, u.ID)
}

func (h *Handler) Refresh(ctx context.Context, req *authv1.RefreshRequest) (*authv1.AuthResponse, error) {
	raw := strings.TrimSpace(req.GetRefreshToken())
	if raw == "" {
		return nil, status.Error(codes.InvalidArgument, "missing refresh token")
	}
	hash := tokens.HashRefreshToken(raw)

	rt, err := h.store.GetRefreshByHash(ctx, hash)
	if errors.Is(err, store.ErrNotFound) {
		return nil, status.Error(codes.Unauthenticated, "invalid refresh token")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "db error")
	}

	if rt.RevokedAt.Valid || time.Now().After(rt.ExpiresAt) {
		return nil, status.Error(codes.Unauthenticated, "refresh token expired or revoked")
	}

	// rotation: revoke old token and issue new pair
	_ = h.store.RevokeRefresh(ctx, rt.ID)
	return h.issueTokens(ctx, rt.UserID)
}

func (h *Handler) Validate(_ context.Context, req *authv1.ValidateRequest) (*authv1.ValidateResponse, error) {
	token := strings.TrimSpace(req.GetAccessToken())
	if token == "" {
		return nil, status.Error(codes.Unauthenticated, "missing token")
	}
	userID, err := h.jwt.ValidateAccessToken(token)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}
	return &authv1.ValidateResponse{UserId: userID}, nil
}

func (h *Handler) issueTokens(ctx context.Context, userID uuid.UUID) (*authv1.AuthResponse, error) {
	access, err := h.jwt.NewAccessToken(userID.String())
	if err != nil {
		return nil, status.Error(codes.Internal, "token error")
	}

	rawRefresh, refreshHash, err := tokens.NewRefreshToken()
	if err != nil {
		return nil, status.Error(codes.Internal, "refresh error")
	}

	rtID := uuid.New()
	expires := time.Now().Add(h.refreshTTL)
	if err := h.store.CreateRefreshToken(ctx, rtID, userID, refreshHash, expires); err != nil {
		return nil, status.Error(codes.Internal, "db error")
	}

	return &authv1.AuthResponse{
		UserId:       userID.String(),
		AccessToken:  access,
		RefreshToken: rawRefresh,
	}, nil
}

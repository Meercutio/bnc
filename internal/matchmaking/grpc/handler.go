package grpc

import (
	"context"
	"strings"

	mmv1 "github.com/yourname/bulls-cows/gen/go/api/proto/matchmaking/v1"
	"github.com/yourname/bulls-cows/internal/matchmaking/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	mmv1.UnimplementedMatchmakingServiceServer
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) StartSearch(ctx context.Context, req *mmv1.StartSearchRequest) (*mmv1.StartSearchResponse, error) {
	uid := strings.TrimSpace(req.GetUserId())
	if uid == "" {
		return nil, status.Error(codes.InvalidArgument, "missing user_id")
	}

	res, err := h.svc.StartSearch(ctx, uid)
	if err != nil {
		return nil, status.Error(codes.Internal, "matchmaking error")
	}

	return &mmv1.StartSearchResponse{
		Status:     res.Status,
		MatchId:    res.MatchID,
		OpponentId: res.OpponentID,
	}, nil
}

func (h *Handler) CancelSearch(ctx context.Context, req *mmv1.CancelSearchRequest) (*mmv1.CancelSearchResponse, error) {
	uid := strings.TrimSpace(req.GetUserId())
	if uid == "" {
		return nil, status.Error(codes.InvalidArgument, "missing user_id")
	}

	res, err := h.svc.CancelSearch(ctx, uid)
	if err != nil {
		return nil, status.Error(codes.Internal, "matchmaking error")
	}

	return &mmv1.CancelSearchResponse{
		Status:  res.Status,
		MatchId: res.MatchID,
	}, nil
}

func (h *Handler) Status(ctx context.Context, req *mmv1.StatusRequest) (*mmv1.StatusResponse, error) {
	uid := strings.TrimSpace(req.GetUserId())
	if uid == "" {
		return nil, status.Error(codes.InvalidArgument, "missing user_id")
	}

	res, err := h.svc.Status(ctx, uid)
	if err != nil {
		return nil, status.Error(codes.Internal, "matchmaking error")
	}

	return &mmv1.StatusResponse{
		Status:     res.Status,
		MatchId:    res.MatchID,
		OpponentId: res.OpponentID,
	}, nil
}

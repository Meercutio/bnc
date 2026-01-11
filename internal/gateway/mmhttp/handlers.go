package mmhttp

import (
	"net/http"

	mmv1 "github.com/yourname/bulls-cows/gen/go/api/proto/matchmaking/v1"
	"github.com/yourname/bulls-cows/internal/gateway/grpcerr"
	"github.com/yourname/bulls-cows/internal/gateway/httpapi"
	"github.com/yourname/bulls-cows/internal/gateway/middleware"
)

type Handler struct {
	mm mmv1.MatchmakingServiceClient
}

func New(mm mmv1.MatchmakingServiceClient) *Handler {
	return &Handler{mm: mm}
}

func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	uid, _ := r.Context().Value(middleware.UserIDKey).(string)

	resp, err := h.mm.StartSearch(r.Context(), &mmv1.StartSearchRequest{UserId: uid})
	if err != nil {
		grpcerr.Write(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	uid, _ := r.Context().Value(middleware.UserIDKey).(string)

	resp, err := h.mm.CancelSearch(r.Context(), &mmv1.CancelSearchRequest{UserId: uid})
	if err != nil {
		grpcerr.Write(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	uid, _ := r.Context().Value(middleware.UserIDKey).(string)

	resp, err := h.mm.Status(r.Context(), &mmv1.StatusRequest{UserId: uid})
	if err != nil {
		grpcerr.Write(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, resp)
}

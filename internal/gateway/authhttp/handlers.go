package authhttp

import (
	"net/http"

	authv1 "github.com/yourname/bulls-cows/gen/go/api/proto/auth/v1"
	"github.com/yourname/bulls-cows/internal/gateway/grpcerr"
	"github.com/yourname/bulls-cows/internal/gateway/httpapi"
)

type Handler struct {
	auth authv1.AuthServiceClient
}

func New(auth authv1.AuthServiceClient) *Handler {
	return &Handler{auth: auth}
}

type registerReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var in registerReq
	if err := httpapi.DecodeJSON(r, &in); err != nil {
		httpapi.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "bad json"})
		return
	}

	resp, err := h.auth.Register(r.Context(), &authv1.RegisterRequest{
		Email:    in.Email,
		Password: in.Password,
	})
	if err != nil {
		grpcerr.Write(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, resp)
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var in loginReq
	if err := httpapi.DecodeJSON(r, &in); err != nil {
		httpapi.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "bad json"})
		return
	}

	resp, err := h.auth.Login(r.Context(), &authv1.LoginRequest{
		Email:    in.Email,
		Password: in.Password,
	})
	if err != nil {
		grpcerr.Write(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, resp)
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var in refreshReq
	if err := httpapi.DecodeJSON(r, &in); err != nil {
		httpapi.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "bad json"})
		return
	}

	resp, err := h.auth.Refresh(r.Context(), &authv1.RefreshRequest{
		RefreshToken: in.RefreshToken,
	})
	if err != nil {
		grpcerr.Write(w, err)
		return
	}
	httpapi.WriteJSON(w, http.StatusOK, resp)
}

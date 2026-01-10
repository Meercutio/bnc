package grpcerr

import (
	"encoding/json"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ErrorBody struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"` // grpc code string for debugging
}

func Write(w http.ResponseWriter, err error) {
	s, ok := status.FromError(err)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, ErrorBody{
			Error: "internal error",
		})
		return
	}

	httpCode := toHTTPStatus(s.Code())
	writeJSON(w, httpCode, ErrorBody{
		Error: s.Message(),
		Code:  s.Code().String(),
	})
}

func toHTTPStatus(c codes.Code) int {
	switch c {
	case codes.OK:
		return http.StatusOK
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.FailedPrecondition:
		return http.StatusPreconditionFailed
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.Canceled:
		// не стандартный, но часто используют как "client closed request"
		return 499
	default:
		return http.StatusInternalServerError
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

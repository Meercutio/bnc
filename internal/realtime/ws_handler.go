package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	authv1 "github.com/yourname/bulls-cows/gen/go/api/proto/auth/v1"
)

type WSHandler struct {
	Auth     authv1.AuthServiceClient
	Hub      *Hub
	Registry *Registry
}

func NewWSHandler(auth authv1.AuthServiceClient, hub *Hub, reg *Registry) *WSHandler {
	return &WSHandler{Auth: auth, Hub: hub, Registry: reg}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// На проде обязательно проверяй Origin. Для pet-проекта пока разрешим.
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1) auth: Bearer token
	userID, ok := h.authenticate(r)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// 2) upgrade to WS
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	// 3) register connection
	c := newConn(userID, ws)
	h.Hub.Add(c)

	if h.Registry != nil {
		_ = h.Registry.Add(r.Context(), userID, c.ID)
	}

	// 4) configure WS keepalive
	ws.SetReadLimit(64 * 1024)
	_ = ws.SetReadDeadline(time.Now().Add(60 * time.Second))
	ws.SetPongHandler(func(string) error {
		c.LastPong = time.Now()
		_ = ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// 5) notify connected
	c.Send <- MustEnvelope("connected", map[string]any{
		"user_id": userID,
		"conn_id": c.ID,
	})

	// 6) start pumps
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	go h.writePump(ctx, c)
	h.readPump(ctx, c)

	// 7) cleanup
	h.Hub.Remove(c)
	if h.Registry != nil {
		_ = h.Registry.Remove(context.Background(), userID, c.ID)
	}
	c.Close()
}

func (h *WSHandler) authenticate(r *http.Request) (string, bool) {
	hdr := r.Header.Get("Authorization")
	if hdr == "" || !strings.HasPrefix(hdr, "Bearer ") {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(hdr, "Bearer "))
	v, err := h.Auth.Validate(r.Context(), &authv1.ValidateRequest{AccessToken: token})
	if err != nil || v.GetUserId() == "" {
		return "", false
	}
	return v.GetUserId(), true
}

func (h *WSHandler) readPump(ctx context.Context, c *Conn) {
	defer func() {
		_ = c.WS.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_, data, err := c.WS.ReadMessage()
		if err != nil {
			return
		}

		var env Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			c.Send <- MustEnvelope("error", map[string]any{"message": "bad envelope"})
			continue
		}

		switch env.Type {
		case "ping":
			c.Send <- MustEnvelope("pong", map[string]any{"ts": time.Now().Unix()})

		case "subscribe_match":
			var p struct {
				MatchID string `json:"match_id"`
			}
			if err := json.Unmarshal(env.Payload, &p); err != nil || p.MatchID == "" {
				c.Send <- MustEnvelope("error", map[string]any{"message": "bad subscribe_match payload"})
				continue
			}
			h.Hub.SubscribeMatch(c, p.MatchID)
			c.Send <- MustEnvelope("subscribed", map[string]any{"match_id": p.MatchID})

		default:
			c.Send <- MustEnvelope("error", map[string]any{"message": "unknown type"})
		}
	}
}

func (h *WSHandler) writePump(ctx context.Context, c *Conn) {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			_ = c.WS.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"))
			return

		case msg, ok := <-c.Send:
			if !ok {
				return
			}
			_ = c.WS.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.WS.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			// Ping frame to keep connection alive (browser responds with pong)
			_ = c.WS.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.WS.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
				return
			}
		}
	}
}

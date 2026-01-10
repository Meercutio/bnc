package realtime

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var ErrSendQueueFull = errors.New("send queue full")

type Conn struct {
	ID       string
	UserID   string
	WS       *websocket.Conn
	Send     chan []byte
	LastPong time.Time

	mu        sync.Mutex
	matchSubs map[string]struct{} // match_id subscriptions (под игру)
	closed    bool
	closeOnce sync.Once
}

func newConn(userID string, ws *websocket.Conn) *Conn {
	return &Conn{
		ID:        uuid.NewString(),
		UserID:    userID,
		WS:        ws,
		Send:      make(chan []byte, 64),
		LastPong:  time.Now(),
		matchSubs: make(map[string]struct{}),
	}
}

func (c *Conn) Close() {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.closed = true
		c.mu.Unlock()
		_ = c.WS.Close()
		close(c.Send)
	})
}

type Hub struct {
	mu sync.RWMutex
	// userID -> connID -> *Conn
	byUser map[string]map[string]*Conn
	// matchID -> connID -> *Conn (только подписанные)
	byMatch map[string]map[string]*Conn
}

func NewHub() *Hub {
	return &Hub{
		byUser:  make(map[string]map[string]*Conn),
		byMatch: make(map[string]map[string]*Conn),
	}
}

func (h *Hub) Add(c *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	m := h.byUser[c.UserID]
	if m == nil {
		m = make(map[string]*Conn)
		h.byUser[c.UserID] = m
	}
	m[c.ID] = c
}

func (h *Hub) Remove(c *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if m := h.byUser[c.UserID]; m != nil {
		delete(m, c.ID)
		if len(m) == 0 {
			delete(h.byUser, c.UserID)
		}
	}

	// убрать из match подписок
	for matchID := range c.matchSubs {
		if mm := h.byMatch[matchID]; mm != nil {
			delete(mm, c.ID)
			if len(mm) == 0 {
				delete(h.byMatch, matchID)
			}
		}
	}
}

func (h *Hub) SubscribeMatch(c *Conn, matchID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	c.mu.Lock()
	c.matchSubs[matchID] = struct{}{}
	c.mu.Unlock()

	m := h.byMatch[matchID]
	if m == nil {
		m = make(map[string]*Conn)
		h.byMatch[matchID] = m
	}
	m[c.ID] = c
}

func (h *Hub) SendToUser(userID string, msg []byte) {
	h.mu.RLock()
	conns := h.byUser[userID]
	h.mu.RUnlock()

	for _, c := range conns {
		select {
		case c.Send <- msg:
		default:
			// если клиент не читает — не блокируем весь сервер
			// в writePump закроем по таймауту/ошибке
		}
	}
}

func (h *Hub) SendToMatch(matchID string, msg []byte) {
	h.mu.RLock()
	conns := h.byMatch[matchID]
	h.mu.RUnlock()

	for _, c := range conns {
		select {
		case c.Send <- msg:
		default:
		}
	}
}

package health

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"
)

type Probe struct {
	ready atomic.Bool
}

func New() *Probe {
	p := &Probe{}
	p.ready.Store(false)
	return p
}

func (p *Probe) SetReady(v bool) { p.ready.Store(v) }

func (p *Probe) Liveness(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (p *Probe) Readiness(w http.ResponseWriter, _ *http.Request) {
	if !p.ready.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("not ready"))
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready"))
}

func (p *Probe) MarkReadyAfter(ctx context.Context, d time.Duration) {
	go func() {
		select {
		case <-time.After(d):
			p.SetReady(true)
		case <-ctx.Done():
		}
	}()
}

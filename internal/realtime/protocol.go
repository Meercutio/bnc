package realtime

import "encoding/json"

type Envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

func MustEnvelope(t string, payload any) []byte {
	var raw json.RawMessage
	if payload != nil {
		b, _ := json.Marshal(payload)
		raw = b
	}
	env := Envelope{Type: t, Payload: raw}
	b, _ := json.Marshal(env)
	return b
}

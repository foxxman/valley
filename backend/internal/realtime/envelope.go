package realtime

import "encoding/json"

// Envelope is the frame every message on the bus uses.
// Payload is interpreted by the handler registered for Type.
type Envelope struct {
	Type    string          `json:"type"`
	Room    string          `json:"room,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

func NewEnvelope(typ, room string, payload any) (Envelope, error) {
	env := Envelope{Type: typ, Room: room}
	if payload == nil {
		return env, nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, err
	}
	env.Payload = raw
	return env, nil
}

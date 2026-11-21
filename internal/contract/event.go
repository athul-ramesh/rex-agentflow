package contract

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const Version = 1

type EventType string

const (
	EventSessionBind  EventType = "session.bind"
	EventWorkerStart  EventType = "worker.start"
	EventWorkerPatch  EventType = "worker.patch"
	EventWorkerStop   EventType = "worker.stop"
	EventToolReturn   EventType = "tool.return"
	EventTurnSettle   EventType = "turn.settle"
	EventSessionEnd   EventType = "session.end"
)

var validTypes = map[EventType]bool{
	EventSessionBind: true,
	EventWorkerStart: true,
	EventWorkerPatch: true,
	EventWorkerStop:  true,
	EventToolReturn:  true,
	EventTurnSettle:  true,
	EventSessionEnd:  true,
}

type WorkerRef struct {
	ID         string `json:"id,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Title      string `json:"title,omitempty"`
	Msg        string `json:"msg,omitempty"`
	State      string `json:"state,omitempty"`
	ParentID   string `json:"parent_id,omitempty"`
}

type Event struct {
	V       int       `json:"v"`
	Type    EventType `json:"type"`
	Harness string    `json:"harness,omitempty"`

	RunID     string `json:"run_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Title     string `json:"title,omitempty"`

	Worker WorkerRef `json:"worker,omitempty"`
	Status string    `json:"status,omitempty"`
	Msg    string    `json:"msg,omitempty"`
}

func Parse(data []byte) (Event, error) {
	var ev Event
	if err := json.Unmarshal(data, &ev); err != nil {
		return Event{}, err
	}
	return ev, Validate(ev)
}

func Validate(ev Event) error {
	if ev.V == 0 {
		ev.V = Version
	}
	if ev.V != Version {
		return fmt.Errorf("unsupported event version %d", ev.V)
	}
	if ev.Type == "" {
		return errors.New("missing event type")
	}
	if !validTypes[ev.Type] {
		return fmt.Errorf("unknown event type %q", ev.Type)
	}
	return nil
}

func (ev Event) SessionKey() string {
	return strings.TrimSpace(ev.SessionID)
}

func SanitizeID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.', r == '/':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}
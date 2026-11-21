package contract

import (
	"encoding/json"
	"testing"
)

func TestParseValidWorkerStart(t *testing.T) {
	raw := []byte(`{"v":1,"type":"worker.start","session_id":"s1","worker":{"id":"abc","kind":"explore"}}`)
	ev, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Type != EventWorkerStart || ev.SessionKey() != "s1" {
		t.Fatalf("%+v", ev)
	}
}

func TestValidateRejectsUnknownType(t *testing.T) {
	raw := []byte(`{"v":1,"type":"nope"}`)
	if _, err := Parse(raw); err == nil {
		t.Fatal("expected error")
	}
}

func TestSanitizeKeepsLongID(t *testing.T) {
	long := "a" + string(make([]byte, 0))
	for i := 0; i < 80; i++ {
		long += "x"
	}
	got := SanitizeID(long)
	if len(got) < 80 {
		t.Fatalf("truncated id len=%d", len(got))
	}
}

func TestAllEventTypesRoundTrip(t *testing.T) {
	types := []EventType{
		EventSessionBind, EventWorkerStart, EventWorkerPatch,
		EventWorkerStop, EventToolReturn, EventTurnSettle, EventSessionEnd,
	}
	for _, typ := range types {
		b, _ := json.Marshal(Event{V: Version, Type: typ})
		if _, err := Parse(b); err != nil {
			t.Fatalf("type %s: %v", typ, err)
		}
	}
}
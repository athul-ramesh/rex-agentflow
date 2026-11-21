package identity

import "testing"

func TestPayloadIDConversationWins(t *testing.T) {
	if got := PayloadID("conv-1", "sess-1"); got != "conv-1" {
		t.Fatalf("got %q", got)
	}
}

func TestPayloadIDSessionFallback(t *testing.T) {
	if got := PayloadID("", "sess-1"); got != "sess-1" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveEventPayloadBeatsEnv(t *testing.T) {
	r := Resolve(Input{
		Event:          true,
		EnvGraph:       "pane-block",
		ConversationID: "conv-a",
	})
	if r.ID != "conv-a" || r.Source != SourcePayload || r.Pinned {
		t.Fatalf("got %+v", r)
	}
}

func TestResolveEventFlagPins(t *testing.T) {
	r := Resolve(Input{Event: true, Flag: "demo", ConversationID: "conv-a"})
	if r.ID != "demo" || !r.Pinned {
		t.Fatalf("got %+v", r)
	}
}

func TestResolveCLIQueueBeatsEnv(t *testing.T) {
	r := Resolve(Input{EnvGraph: "sid-env", Queued: "sid-queue"})
	if r.ID != "sid-queue" || r.Source != SourceQueue {
		t.Fatalf("got %+v", r)
	}
}

func TestResolveCLITwoSessionsDistinct(t *testing.T) {
	a := Resolve(Input{Event: true, SessionID: "session-alpha"})
	b := Resolve(Input{Event: true, SessionID: "session-beta"})
	if a.ID == b.ID {
		t.Fatal("expected distinct ids")
	}
}

func TestResolveCLIAmbiguous(t *testing.T) {
	r := Resolve(Input{Adopted: []string{"a", "b"}})
	if !r.Ambiguous || len(r.IDs) != 2 {
		t.Fatalf("got %+v", r)
	}
}

func TestResolveFallback(t *testing.T) {
	r := Resolve(Input{RexBlock: "blk"})
	if !r.Fallback() {
		t.Fatal("rex block should be fallback")
	}
	r = Resolve(Input{})
	if r.ID != "default" || !r.Fallback() {
		t.Fatalf("got %+v", r)
	}
}
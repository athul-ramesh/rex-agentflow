package cursor

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/athulramesh/rex-agentflow/internal/graph"
)

func TestSubagentStartStop(t *testing.T) {
	start := []byte(`{"hook_event_name":"subagentStart","subagent_id":"abc","subagent_type":"explore","task":"map APIs"}`)
	g := graph.New(Harness)
	if err := ApplyHook(&g, start); err != nil {
		t.Fatal(err)
	}
	_, w := g.FindWorker("abc")
	if w == nil || w.State != graph.PhaseWorking || w.Kind != "explore" {
		t.Fatalf("after start: %+v", w)
	}
	if w.Title != "map APIs" {
		t.Fatalf("short task should stay title, got %q", w.Title)
	}

	stop := []byte(`{"hook_event_name":"subagentStop","subagent_id":"abc","subagent_type":"explore","status":"completed","summary":"done mapping"}`)
	if err := ApplyHook(&g, stop); err != nil {
		t.Fatal(err)
	}
	_, w = g.FindWorker("abc")
	if w == nil || w.State != graph.PhaseFinished {
		t.Fatalf("after stop: %+v", w)
	}
	if w.Msg == "" {
		t.Fatal("expected summary msg")
	}
}

func TestSubagentStartLongTaskUsesKindTitle(t *testing.T) {
	task := "Independently inspect this repository and explain its purpose, integrations, and what the user-facing commands do"
	start, err := json.Marshal(map[string]string{
		"hook_event_name": "subagentStart",
		"subagent_id":     "long",
		"subagent_type":   "explore",
		"task":            task,
	})
	if err != nil {
		t.Fatal(err)
	}
	g := graph.New(Harness)
	if err := ApplyHook(&g, start); err != nil {
		t.Fatal(err)
	}
	_, w := g.FindWorker("long")
	if w == nil || w.Title != "explore" {
		t.Fatalf("want kind as title, got %+v", w)
	}
	if w.Msg == "" || !strings.HasPrefix(w.Msg, "Independently") {
		t.Fatalf("want task in msg, got %q", w.Msg)
	}
}

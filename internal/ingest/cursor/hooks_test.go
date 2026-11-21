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

func TestTwoSameKindWorkersReverseOrderStop(t *testing.T) {
	g := graph.New(Harness)
	start1 := []byte(`{"hook_event_name":"subagentStart","subagent_id":"w1","subagent_type":"explore","task":"first"}`)
	start2 := []byte(`{"hook_event_name":"subagentStart","subagent_id":"w2","subagent_type":"explore","task":"second"}`)
	stop2 := []byte(`{"hook_event_name":"subagentStop","subagent_id":"w2","subagent_type":"explore","status":"completed"}`)
	stop1 := []byte(`{"hook_event_name":"subagentStop","subagent_id":"w1","subagent_type":"explore","status":"completed"}`)
	for _, raw := range [][]byte{start1, start2, stop2, stop1} {
		if err := ApplyHook(&g, raw); err != nil {
			t.Fatal(err)
		}
	}
	_, a := g.FindWorker("w1")
	_, b := g.FindWorker("w2")
	if a.State != graph.PhaseFinished || b.State != graph.PhaseFinished {
		t.Fatalf("w1=%s w2=%s", a.State, b.State)
	}
}

func TestToolCallIDThenSubagentIDSameRow(t *testing.T) {
	g := graph.New(Harness)
	pre := []byte(`{"hook_event_name":"preToolUse","tool_name":"Task","tool_call_id":"call-9","subagent_type":"explore"}`)
	start := []byte(`{"hook_event_name":"subagentStart","subagent_id":"sub-9","subagent_type":"explore","tool_call_id":"call-9","task":"work"}`)
	stop := []byte(`{"hook_event_name":"subagentStop","subagent_id":"sub-9","subagent_type":"explore","status":"completed"}`)
	for _, raw := range [][]byte{pre, start, stop} {
		if err := ApplyHook(&g, raw); err != nil {
			t.Fatal(err)
		}
	}
	_, byCall := g.FindWorker("call-9")
	_, bySub := g.FindWorker("sub-9")
	if byCall == nil && bySub == nil {
		t.Fatalf("workers %+v", g.Workers)
	}
	finished := bySub
	if finished == nil {
		finished = byCall
	}
	if finished.State != graph.PhaseFinished {
		t.Fatalf("state %s workers %+v", finished.State, g.Workers)
	}
	if g.Correlation["call-9"] != "sub-9" && g.Correlation["sub-9"] != "call-9" {
		t.Fatalf("correlation %+v", g.Correlation)
	}
}

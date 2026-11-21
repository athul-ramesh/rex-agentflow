package cursor

import (
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
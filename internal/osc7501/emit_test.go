package osc7501

import (
	"bytes"
	"strings"
	"testing"

	"github.com/athulramesh/rex-agentflow/internal/graph"
)

func TestEmitWorkingWithID(t *testing.T) {
	g := graph.New("cursor-hooks")
	g.UpsertWorker(graph.Worker{
		ID:    "abc",
		Title: "map APIs",
		State: graph.PhaseWorking,
		Kind:  "explore",
	})
	var buf bytes.Buffer
	if err := WriteGraph(&buf, g); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "state=working") {
		t.Fatalf("missing working: %q", out)
	}
	if !strings.Contains(out, "id=w/abc") {
		t.Fatalf("missing id: %q", out)
	}
}

func TestPhaseMapping(t *testing.T) {
	if OSCState(graph.PhaseFinished) != "done" {
		t.Fatal("finished -> done")
	}
	if OSCState(graph.PhaseQueued) != "idle" {
		t.Fatal("queued -> idle")
	}
}
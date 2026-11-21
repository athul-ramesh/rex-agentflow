package cursor

import (
	"bytes"
	"strings"
	"testing"

	"github.com/athulramesh/rex-agentflow/internal/graph"
)

func TestStreamRunIDAndResult(t *testing.T) {
	ndjson := strings.Join([]string{
		`{"type":"system","subtype":"init","session_id":"sess-99","run_id":"run-99"}`,
		`{"type":"tool_call","name":"Grep","status":"running"}`,
		`{"type":"result","text":"all done"}`,
	}, "\n")

	g := graph.New(StreamHarness)
	var n int
	err := ProcessStream(&g, strings.NewReader(ndjson), func() error {
		n++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if g.RunID != "sess-99" && g.RunID != "run-99" {
		t.Fatalf("run_id %q", g.RunID)
	}
	if n == 0 {
		t.Fatal("expected onChange calls")
	}
	_, w := g.FindWorker(graph.MainWorkerID)
	if w == nil || w.State != graph.PhaseFinished {
		t.Fatalf("main worker %+v", w)
	}
}

func TestStreamCursorCLIToolCallShape(t *testing.T) {
	ndjson := strings.Join([]string{
		`{"type":"system","subtype":"init","session_id":"sess-cli"}`,
		`{"type":"tool_call","subtype":"started","call_id":"call-1","tool_call":{"readToolCall":{"args":{"path":"go.mod"}}}}`,
		`{"type":"tool_call","subtype":"completed","call_id":"call-1","tool_call":{"readToolCall":{"args":{"path":"go.mod"}}}}`,
		`{"type":"result","subtype":"success","result":"done"}`,
	}, "\n")

	g := graph.New(StreamHarness)
	if err := ProcessStream(&g, strings.NewReader(ndjson), nil); err != nil {
		t.Fatal(err)
	}
	_, w := g.FindWorker("call-1")
	if w == nil {
		t.Fatalf("workers=%+v", g.Workers)
	}
	if w.Title != "read" || w.State != graph.PhaseFinished {
		t.Fatalf("worker %+v", w)
	}
	_, main := g.FindWorker(graph.MainWorkerID)
	if main == nil || main.State != graph.PhaseFinished {
		t.Fatalf("main %+v", main)
	}
}

func TestStreamIgnoresGarbage(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("not json\n")
	buf.WriteString(`{"type":"unknown"}` + "\n")
	g := graph.New(StreamHarness)
	err := ProcessStream(&g, &buf, nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = g
}
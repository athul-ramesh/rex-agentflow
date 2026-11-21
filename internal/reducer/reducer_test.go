package reducer

import (
	"testing"

	"github.com/athulramesh/rex-agentflow/internal/contract"
	"github.com/athulramesh/rex-agentflow/internal/graph"
)

func TestLateStartIgnored(t *testing.T) {
	g := graph.New("test")
	g.UpsertWorker(graph.Worker{ID: "a", State: graph.PhaseFinished})
	Apply(&g, contract.Event{
		V:    contract.Version,
		Type: contract.EventWorkerStart,
		Worker: contract.WorkerRef{ID: "a", Kind: "task"},
	})
	_, w := g.FindWorker("a")
	if w.State != graph.PhaseFinished {
		t.Fatalf("state %s", w.State)
	}
	if len(g.Diagnostics) != 1 || g.Diagnostics[0].Code != "late_start" {
		t.Fatalf("diags %+v", g.Diagnostics)
	}
}

func TestStopAmbiguousRecordsDiagnostic(t *testing.T) {
	g := graph.New("test")
	g.UpsertWorker(graph.Worker{ID: "a", Kind: "explore", State: graph.PhaseWorking})
	g.UpsertWorker(graph.Worker{ID: "b", Kind: "explore", State: graph.PhaseWorking})
	Apply(&g, contract.Event{
		V:    contract.Version,
		Type: contract.EventWorkerStop,
		Worker: contract.WorkerRef{Kind: "explore"},
		Status: "completed",
	})
	_, wa := g.FindWorker("a")
	_, wb := g.FindWorker("b")
	if wa.State != graph.PhaseWorking || wb.State != graph.PhaseWorking {
		t.Fatal("should not finish either")
	}
	found := false
	for _, d := range g.Diagnostics {
		if d.Code == "worker_stop_ambiguous" {
			found = true
		}
	}
	if !found {
		t.Fatalf("diagnostics %+v", g.Diagnostics)
	}
}

func TestStopSingleKindMatch(t *testing.T) {
	g := graph.New("test")
	g.UpsertWorker(graph.Worker{ID: "only", Kind: "explore", State: graph.PhaseWorking})
	Apply(&g, contract.Event{
		V:    contract.Version,
		Type: contract.EventWorkerStop,
		Worker: contract.WorkerRef{Kind: "explore"},
		Status: "completed",
	})
	_, w := g.FindWorker("only")
	if w.State != graph.PhaseFinished {
		t.Fatalf("got %s", w.State)
	}
}

func TestAliasCorrelation(t *testing.T) {
	g := graph.New("test")
	Apply(&g, contract.Event{
		V: contract.Version, Type: contract.EventWorkerStart,
		Worker: contract.WorkerRef{ToolCallID: "call-1", Kind: "task", Title: "Task"},
	})
	_, w := g.FindWorker("call-1")
	if w == nil {
		t.Fatal("expected call-1 worker")
	}
	Apply(&g, contract.Event{
		V: contract.Version, Type: contract.EventWorkerStart,
		Worker: contract.WorkerRef{ID: "sub-1", ToolCallID: "call-1", Kind: "explore"},
	})
	if g.Correlation["call-1"] != "sub-1" {
		t.Fatalf("correlation %+v", g.Correlation)
	}
	Apply(&g, contract.Event{
		V: contract.Version, Type: contract.EventWorkerStop,
		Worker: contract.WorkerRef{ID: "sub-1"},
		Status: "completed",
	})
	_, w1 := g.FindWorker("call-1")
	_, w2 := g.FindWorker("sub-1")
	if w1 == nil && w2 == nil {
		t.Fatal("missing worker")
	}
	target := w2
	if target == nil {
		target = w1
	}
	if target.State != graph.PhaseFinished {
		t.Fatalf("state %s", target.State)
	}
}

func TestStaleSessionSkipped(t *testing.T) {
	g := graph.New("test")
	g.RunID = "run-a"
	Apply(&g, contract.Event{
		V: contract.Version, Type: contract.EventSessionBind,
		SessionID: "run-b",
	})
	if g.RunID != "run-a" {
		t.Fatalf("mutated run_id to %q", g.RunID)
	}
	if len(g.Diagnostics) != 1 || g.Diagnostics[0].Code != "stale_session" {
		t.Fatalf("diags %+v", g.Diagnostics)
	}
}

func TestTwoSameKindReverseCompletion(t *testing.T) {
	g := graph.New("test")
	Apply(&g, contract.Event{V: contract.Version, Type: contract.EventWorkerStart,
		Worker: contract.WorkerRef{ID: "w1", Kind: "explore", Title: "first"}})
	Apply(&g, contract.Event{V: contract.Version, Type: contract.EventWorkerStart,
		Worker: contract.WorkerRef{ID: "w2", Kind: "explore", Title: "second"}})
	Apply(&g, contract.Event{V: contract.Version, Type: contract.EventWorkerStop,
		Worker: contract.WorkerRef{ID: "w2"}, Status: "completed"})
	Apply(&g, contract.Event{V: contract.Version, Type: contract.EventWorkerStop,
		Worker: contract.WorkerRef{ID: "w1"}, Status: "completed"})
	_, a := g.FindWorker("w1")
	_, b := g.FindWorker("w2")
	if a.State != graph.PhaseFinished || b.State != graph.PhaseFinished {
		t.Fatalf("w1=%s w2=%s", a.State, b.State)
	}
}
package osc7501

import (
	"testing"

	"github.com/athulramesh/rex-agentflow/internal/graph"
)

func TestRootStatusEmptyIdle(t *testing.T) {
	if RootStatus(graph.New("x")) != "idle" {
		t.Fatal("empty graph should be idle")
	}
}

func TestRootStatusAllFinishedDone(t *testing.T) {
	g := graph.New("x")
	g.UpsertWorker(graph.Worker{ID: "a", State: graph.PhaseFinished})
	g.UpsertWorker(graph.Worker{ID: graph.MainWorkerID, State: graph.PhaseFinished})
	if RootStatus(g) != "done" {
		t.Fatalf("want done")
	}
}

func TestRootStatusPriority(t *testing.T) {
	g := graph.New("x")
	g.UpsertWorker(graph.Worker{ID: "a", State: graph.PhaseFinished})
	g.UpsertWorker(graph.Worker{ID: "b", State: graph.PhaseWorking})
	if RootStatus(g) != "working" {
		t.Fatal("working wins over finished")
	}
	g.Workers[1].State = graph.PhaseBlocked
	if RootStatus(g) != "blocked" {
		t.Fatal("blocked over finished")
	}
	g.UpsertWorker(graph.Worker{ID: "c", State: graph.PhaseError})
	if RootStatus(g) != "error" {
		t.Fatal("error wins")
	}
}

func TestRootRecordAllFinishedNotWorking(t *testing.T) {
	g := graph.New("x")
	g.UpsertWorker(graph.Worker{ID: "a", State: graph.PhaseFinished})
	rec := rootRecord(g)
	if rec.State == "working" {
		t.Fatalf("root %+v", rec)
	}
	if rec.State != "done" {
		t.Fatalf("state %q", rec.State)
	}
}
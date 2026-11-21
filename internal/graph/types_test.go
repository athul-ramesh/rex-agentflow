package graph

import (
	"testing"
	"time"
)

func TestUpsertWorker(t *testing.T) {
	g := New("test")
	g.UpsertWorker(Worker{ID: "a", Title: "one", State: PhaseWorking})
	g.UpsertWorker(Worker{ID: "a", State: PhaseFinished, Msg: "ok"})
	_, w := g.FindWorker("a")
	if w == nil || w.State != PhaseFinished || w.Msg != "ok" {
		t.Fatalf("worker %+v", w)
	}
}

func TestValidateDuplicateID(t *testing.T) {
	g := WorkGraph{Version: Version, Workers: []Worker{{ID: "x"}, {ID: "x"}}, UpdatedAt: time.Now()}
	if Validate(&g) == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestSanitizeOSCID(t *testing.T) {
	if got := WorkerOSCPath("abc/def"); got != "w/abc/def" {
		t.Fatalf("got %q", got)
	}
}
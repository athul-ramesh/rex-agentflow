package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/athulramesh/rex-agentflow/internal/contract"
	"github.com/athulramesh/rex-agentflow/internal/graph"
	"github.com/athulramesh/rex-agentflow/internal/identity"
	"github.com/athulramesh/rex-agentflow/internal/reducer"
)

func TestTwoSessionsDistinctFileKeys(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)
	a := identity.PayloadID("conv-a", "")
	b := identity.PayloadID("conv-b", "")
	if FileKey(a) == FileKey(b) {
		t.Fatal("file keys must differ")
	}
	_ = WithGraph(a, "test", func(g *graph.WorkGraph) error {
		g.Title = "alpha"
		g.SessionID = a
		return nil
	})
	_ = WithGraph(b, "test", func(g *graph.WorkGraph) error {
		g.Title = "beta"
		g.SessionID = b
		return nil
	})
	ga, err := Load(a)
	if err != nil {
		t.Fatal(err)
	}
	gb, err := Load(b)
	if err != nil {
		t.Fatal(err)
	}
	if ga.Title == gb.Title {
		t.Fatalf("graphs merged: %+v %+v", ga, gb)
	}
	if _, err := os.Stat(GraphPath(a)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(GraphPath(b)); err != nil {
		t.Fatal(err)
	}
	if GraphPath(a) == GraphPath(b) {
		t.Fatal("paths must differ")
	}
}

func TestShellQueueLatestWithinTTL(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)
	cwd := filepath.Join(dir, "proj")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := EnqueueShell(cwd, "sess-from-hook"); err != nil {
		t.Fatal(err)
	}
	got, ok := LatestShell(cwd)
	if !ok || got != "sess-from-hook" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
}

func TestWithGraphStaleSessionIsolated(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)
	id := "sess-one"
	_ = WithGraph(id, "test", func(g *graph.WorkGraph) error {
		g.SessionID = id
		return nil
	})
	ev := contract.Event{
		V: contract.Version, Type: contract.EventWorkerStart,
		SessionID: "other-session",
		Worker:    contract.WorkerRef{ID: "w", Kind: "task"},
	}
	err := WithGraph(id, "test", func(g *graph.WorkGraph) error {
		diags := reducer.Apply(g, ev)
		if len(diags) != 1 || diags[0].Code != "stale_session" {
			t.Fatalf("diags %+v", diags)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
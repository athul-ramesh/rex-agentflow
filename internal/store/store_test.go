package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/athulramesh/rex-agentflow/internal/graph"
)

func TestRoundtrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)

	g := graph.New("test")
	g.RunID = "run-1"
	g.UpsertWorker(graph.Worker{ID: "w1", Title: "task", State: graph.PhaseWorking})

	if err := Save("demo", &g); err != nil {
		t.Fatal(err)
	}
	got, err := Load("demo")
	if err != nil {
		t.Fatal(err)
	}
	if got.RunID != "run-1" {
		t.Fatalf("run_id %q", got.RunID)
	}
	if len(got.Workers) != 1 || got.Workers[0].ID != "w1" {
		t.Fatalf("workers %+v", got.Workers)
	}

	data, _ := os.ReadFile(GraphPath("demo"))
	if !filepath.IsAbs(GraphPath("demo")) {
		t.Fatal("path not absolute")
	}
	if len(data) == 0 {
		t.Fatal("empty file")
	}
}
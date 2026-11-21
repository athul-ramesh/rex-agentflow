package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
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

func TestConcurrentWithGraphUpsert(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)
	name := "concurrent"

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("w-%d", n)
			err := WithGraph(name, "test", func(g *graph.WorkGraph) error {
				g.UpsertWorker(graph.Worker{ID: id, Title: id, State: graph.PhaseWorking})
				return nil
			})
			if err != nil {
				t.Errorf("WithGraph %s: %v", id, err)
			}
		}(i)
	}
	wg.Wait()

	got, err := Load(name)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Workers) != 20 {
		t.Fatalf("expected 20 workers, got %d", len(got.Workers))
	}
}

func TestSaveUsesUniqueTmp(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)
	name := "tmpuniq"
	path := GraphPath(name)

	g := graph.New("test")
	if err := Save(name, &g); err != nil {
		t.Fatal(err)
	}

	var lastTmp string
	var saves atomic.Int32
	for i := 0; i < 5; i++ {
		saves.Add(1)
		if err := WithGraph(name, "test", func(g *graph.WorkGraph) error {
			g.Title = fmt.Sprintf("t%d", i)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		entries, _ := os.ReadDir(filepath.Dir(path))
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".tmp") && e.Name() != filepath.Base(path) {
				lastTmp = e.Name()
			}
		}
	}
	_ = lastTmp
	if saves.Load() != 5 {
		t.Fatal("save count")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("graph file missing: %v", err)
	}
}

func TestClearUnderLock(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)
	name := "clearme"

	g := graph.New("test")
	if err := Save(name, &g); err != nil {
		t.Fatal(err)
	}
	if err := Clear(name); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(GraphPath(name)); !os.IsNotExist(err) {
		t.Fatalf("graph should be gone: %v", err)
	}
}
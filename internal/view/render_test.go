package view

import (
	"testing"

	"github.com/athulramesh/rex-agentflow/internal/graph"
)

func TestRenderPlanAndWorkers(t *testing.T) {
	g := graph.New("test")
	g.Title = "demo run"
	g.Plan = []graph.PlanItem{
		{ID: "1", Text: "scaffold", Status: graph.PhaseFinished},
		{ID: "2", Text: "ingest", Status: graph.PhaseWorking},
	}
	g.UpsertWorker(graph.Worker{ID: "abc", Title: "explore", State: graph.PhaseWorking, Kind: "explore"})
	img := Render(g, 400, 300)
	if img.Bounds().Dx() != 400 {
		t.Fatalf("width %d", img.Bounds().Dx())
	}
	if img.At(20, 30) == nil {
		t.Fatal("unexpected nil color")
	}
}
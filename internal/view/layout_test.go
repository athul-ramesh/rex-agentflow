package view

import (
	"strings"
	"testing"

	"github.com/athulramesh/rex-agentflow/internal/graph"
)

func TestBuildLayoutFooter(t *testing.T) {
	g := graph.New("cursor-hooks")
	g.Workers = []graph.Worker{
		{ID: "1", Title: "main", State: graph.PhaseFinished},
		{ID: "2", Title: "scout", State: graph.PhaseWorking},
		{ID: "3", Title: "other", State: graph.PhaseError},
	}
	rows := buildLayout(g)
	var footer string
	for _, r := range rows {
		if r.Kind == rowFooter {
			footer = r.Text
		}
	}
	want := "3 workers · 1 working · 1 error"
	if footer != want {
		t.Fatalf("footer %q want %q", footer, want)
	}
}

func TestBuildLayoutWorkerSub(t *testing.T) {
	g := graph.New("cursor-hooks")
	g.Workers = []graph.Worker{
		{ID: "1", Title: "t", State: graph.PhaseError, Kind: "scout", Msg: "boom"},
	}
	rows := buildLayout(g)
	for _, r := range rows {
		if r.Kind == rowWorker && r.Sub != "scout · boom" {
			t.Fatalf("sub %q", r.Sub)
		}
	}
}

func TestFitEllipsis(t *testing.T) {
	fs, err := getFonts()
	if err != nil {
		t.Fatal(err)
	}
	tp := &textPainter{fonts: fs, scale: 1}
	long := strings.Repeat("word ", 40)
	shown, _, truncated := tp.fit(long, false, fontSize, 200)
	if !truncated {
		t.Fatal("expected truncation")
	}
	if !strings.HasSuffix(shown, ell) {
		t.Fatalf("expected ellipsis suffix, got %q", shown)
	}
}
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

func TestWorkerLabelPrefersKindForLongPrompt(t *testing.T) {
	w := graph.Worker{
		ID:    "1",
		Title: "Independently inspect this repository and explain its purpose, integrations, and commands in detail",
		Kind:  "explore",
		State: graph.PhaseWorking,
	}
	if got := workerLabel(w); got != "explore" {
		t.Fatalf("label %q", got)
	}
	detail := workerDetail(w, "explore")
	if !strings.HasPrefix(detail, "Independently") {
		t.Fatalf("detail %q", detail)
	}
	if strings.Contains(detail, "detail") && !strings.HasSuffix(detail, "…") {
		// long prompt should be soft-truncated
		if utf8Count(detail) > detailRunes+1 {
			t.Fatalf("detail too long: %q", detail)
		}
	}
}

func TestWorkerLabelKeepsShortTitle(t *testing.T) {
	w := graph.Worker{ID: "1", Title: "map APIs", Kind: "explore", Msg: "running"}
	if got := workerLabel(w); got != "map APIs" {
		t.Fatalf("label %q", got)
	}
	if got := workerDetail(w, "map APIs"); got != "running" {
		t.Fatalf("detail %q", got)
	}
}

func TestWorkerDetailCombinesPromptAndMsg(t *testing.T) {
	w := graph.Worker{
		ID:    "1",
		Title: "t",
		Kind:  "scout",
		Msg:   "boom",
		State: graph.PhaseError,
	}
	label := workerLabel(w)
	if label != "t" {
		t.Fatalf("label %q", label)
	}
	if got := workerDetail(w, label); got != "boom" {
		t.Fatalf("detail %q", got)
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
	body := strings.TrimSuffix(shown, ell)
	if strings.HasSuffix(body, "wor") {
		t.Fatalf("truncated mid-word: %q", shown)
	}
}

func TestSoftTruncateWordBoundary(t *testing.T) {
	s := softTruncate("one two three four five six seven eight nine ten eleven", 20)
	if !strings.HasSuffix(s, "…") {
		t.Fatalf("expected ellipsis: %q", s)
	}
	if strings.Contains(s, "eleve…") || strings.HasSuffix(strings.TrimSuffix(s, "…"), "eleve") {
		t.Fatalf("mid-word cut: %q", s)
	}
}

func utf8Count(s string) int {
	return len([]rune(s))
}

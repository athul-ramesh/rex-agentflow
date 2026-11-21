package claude

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/athulramesh/rex-agentflow/internal/contract"
	"github.com/athulramesh/rex-agentflow/internal/graph"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "..", "testdata", "claude", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestTwoParallelSubagentsByAgentID(t *testing.T) {
	g := graph.New(Harness)
	g.SessionID = "sess-parallel"
	for _, name := range []string{
		"subagent_start_explore.json",
		"subagent_start_general.json",
		"subagent_stop_explore.json",
		"subagent_stop_general.json",
	} {
		if err := ApplyHook(&g, loadFixture(t, name)); err != nil {
			t.Fatal(err)
		}
	}
	_, explore := g.FindWorker("agent-explore-1")
	_, general := g.FindWorker("agent-gp-1")
	if explore == nil || general == nil {
		t.Fatalf("workers: %+v", g.Workers)
	}
	if explore.State != graph.PhaseFinished || general.State != graph.PhaseFinished {
		t.Fatalf("explore=%s general=%s", explore.State, general.State)
	}
	if explore.Kind != "Explore" || general.Kind != "general-purpose" {
		t.Fatalf("kinds explore=%q general=%q", explore.Kind, general.Kind)
	}
}

func TestSecondPromptSameSessionID(t *testing.T) {
	g := graph.New(Harness)
	first := loadFixture(t, "user_prompt_first.json")
	second := loadFixture(t, "user_prompt_second.json")
	if err := ApplyHook(&g, first); err != nil {
		t.Fatal(err)
	}
	if g.SessionID != "sess-repeat" {
		t.Fatalf("after first prompt session_id=%q", g.SessionID)
	}
	if g.Title != "Add claude hook adapter" {
		t.Fatalf("title after first: %q", g.Title)
	}
	if err := ApplyHook(&g, second); err != nil {
		t.Fatal(err)
	}
	if g.SessionID != "sess-repeat" {
		t.Fatalf("after second prompt session_id=%q", g.SessionID)
	}
	if g.Title != "Wire main.go next" {
		t.Fatalf("title after second: %q", g.Title)
	}
}

func TestSessionEnd(t *testing.T) {
	g := graph.New(Harness)
	g.SessionID = "sess-shutdown"
	if err := ApplyHook(&g, loadFixture(t, "session_end.json")); err != nil {
		t.Fatal(err)
	}
	main := g.EnsureMainWorker()
	if main.State != graph.PhaseFinished {
		t.Fatalf("main state %s", main.State)
	}
	if g.Title != "other" {
		t.Fatalf("title %q", g.Title)
	}
}

func TestTranslateHookMappings(t *testing.T) {
	cases := []struct {
		fixture  string
		wantType contract.EventType
	}{
		{"subagent_start_explore.json", contract.EventWorkerStart},
		{"subagent_stop_explore.json", contract.EventWorkerStop},
		{"user_prompt_first.json", contract.EventSessionBind},
		{"session_end.json", contract.EventSessionEnd},
	}
	for _, tc := range cases {
		evs, err := TranslateHook(loadFixture(t, tc.fixture))
		if err != nil {
			t.Fatal(err)
		}
		if len(evs) != 1 || evs[0].Type != tc.wantType {
			t.Fatalf("%s: got %+v want %s", tc.fixture, evs, tc.wantType)
		}
		if evs[0].Harness != Harness {
			t.Fatalf("%s harness %q", tc.fixture, evs[0].Harness)
		}
	}
}

func TestStopTurnSettleNotSessionEnd(t *testing.T) {
	raw := []byte(`{"hook_event_name":"Stop","session_id":"s1","last_assistant_message":"done for now"}`)
	evs, err := TranslateHook(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 1 || evs[0].Type != contract.EventTurnSettle {
		t.Fatalf("got %+v", evs)
	}
	g := graph.New(Harness)
	g.SessionID = "s1"
	if err := ApplyHook(&g, raw); err != nil {
		t.Fatal(err)
	}
	if g.EnsureMainWorker().State == graph.PhaseFinished {
		t.Fatal("Stop should not end session")
	}
}
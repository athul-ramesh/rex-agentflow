package devin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/athulramesh/rex-agentflow/internal/contract"
	"github.com/athulramesh/rex-agentflow/internal/graph"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "..", "testdata", "devin", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestSessionStartBinds(t *testing.T) {
	g := graph.New(Harness)
	if err := ApplyHook(&g, loadFixture(t, "session_start.json")); err != nil {
		t.Fatal(err)
	}
	if g.SessionID != "devin-sess-1" {
		t.Fatalf("session_id=%q", g.SessionID)
	}
	if g.Title != "startup" {
		t.Fatalf("title=%q", g.Title)
	}
}

func TestUserPromptMainWorking(t *testing.T) {
	g := graph.New(Harness)
	if err := ApplyHook(&g, loadFixture(t, "session_start.json")); err != nil {
		t.Fatal(err)
	}
	if err := ApplyHook(&g, loadFixture(t, "user_prompt.json")); err != nil {
		t.Fatal(err)
	}
	if g.SessionID != "devin-sess-1" {
		t.Fatalf("session_id=%q", g.SessionID)
	}
	if g.Title != "Ship devin hooks" {
		t.Fatalf("title=%q", g.Title)
	}
	main := g.EnsureMainWorker()
	if main.State != graph.PhaseWorking {
		t.Fatalf("main state %s", main.State)
	}
}

func TestSecondPromptSameSession(t *testing.T) {
	g := graph.New(Harness)
	for _, name := range []string{"session_start.json", "user_prompt.json", "user_prompt_second.json"} {
		if err := ApplyHook(&g, loadFixture(t, name)); err != nil {
			t.Fatal(err)
		}
	}
	if g.SessionID != "devin-sess-1" {
		t.Fatalf("session_id=%q", g.SessionID)
	}
	if g.Title != "Add tests" {
		t.Fatalf("title=%q", g.Title)
	}
}

func TestStopSessionEnd(t *testing.T) {
	g := graph.New(Harness)
	g.SessionID = "devin-sess-1"
	if err := ApplyHook(&g, loadFixture(t, "stop.json")); err != nil {
		t.Fatal(err)
	}
	main := g.EnsureMainWorker()
	if main.State != graph.PhaseFinished {
		t.Fatalf("main state %s", main.State)
	}
}

func TestSessionEnd(t *testing.T) {
	g := graph.New(Harness)
	g.SessionID = "devin-sess-1"
	if err := ApplyHook(&g, loadFixture(t, "session_end.json")); err != nil {
		t.Fatal(err)
	}
	main := g.EnsureMainWorker()
	if main.State != graph.PhaseFinished {
		t.Fatalf("main state %s", main.State)
	}
	if g.Title != "logout" {
		t.Fatalf("title %q", g.Title)
	}
}

func TestPermissionRequestNoOp(t *testing.T) {
	evs, err := TranslateHook(loadFixture(t, "permission_request.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 0 {
		t.Fatalf("want noop, got %+v", evs)
	}
	g := graph.New(Harness)
	g.SessionID = "devin-sess-1"
	before := len(g.Workers)
	if err := ApplyHook(&g, loadFixture(t, "permission_request.json")); err != nil {
		t.Fatal(err)
	}
	if len(g.Workers) != before {
		t.Fatalf("workers changed: %+v", g.Workers)
	}
}

func TestPreToolUseRunSubagentNoOp(t *testing.T) {
	raw := []byte(`{"hook_event_name":"PreToolUse","session_id":"s1","prompt_id":"p1","tool_name":"run_subagent"}`)
	evs, err := TranslateHook(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 0 {
		t.Fatalf("want noop without subagent identity, got %+v", evs)
	}
}

func TestTranslateHookMappings(t *testing.T) {
	cases := []struct {
		fixture  string
		wantMin  int
		wantType contract.EventType
	}{
		{"session_start.json", 1, contract.EventSessionBind},
		{"user_prompt.json", 2, contract.EventSessionBind},
		{"stop.json", 1, contract.EventSessionEnd},
		{"session_end.json", 1, contract.EventSessionEnd},
		{"permission_request.json", 0, ""},
	}
	for _, tc := range cases {
		evs, err := TranslateHook(loadFixture(t, tc.fixture))
		if err != nil {
			t.Fatal(err)
		}
		if len(evs) < tc.wantMin {
			t.Fatalf("%s: got %d events", tc.fixture, len(evs))
		}
		if tc.wantMin == 0 {
			if len(evs) != 0 {
				t.Fatalf("%s: want noop", tc.fixture)
			}
			continue
		}
		if evs[0].Type != tc.wantType {
			t.Fatalf("%s: got %+v want %s", tc.fixture, evs, tc.wantType)
		}
		if evs[0].Harness != Harness {
			t.Fatalf("%s harness %q", tc.fixture, evs[0].Harness)
		}
	}
}
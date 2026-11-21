package cursor

import "testing"

func TestPeekConversationFirst(t *testing.T) {
	raw := []byte(`{"conversation_id":"c1","session_id":"s1","cwd":"/tmp","command":"ls"}`)
	c, s, cwd, cmd := Peek(raw)
	if c != "c1" || s != "s1" || cwd != "/tmp" || cmd != "ls" {
		t.Fatalf("got %q %q %q %q", c, s, cwd, cmd)
	}
}

func TestIsAgentflowShell(t *testing.T) {
	if !IsAgentflowShell("agentflow show") {
		t.Fatal("expected match")
	}
	if IsAgentflowShell("agentflow --graph x show") {
		t.Fatal("flag pin should skip enqueue")
	}
}
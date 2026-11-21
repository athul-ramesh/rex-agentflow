// Claude hook commands reserve stdout for hook decision JSON; do not emit OSC from ApplyHook.
package claude

import (
	"strings"

	"github.com/athulramesh/rex-agentflow/internal/graph"
	"github.com/athulramesh/rex-agentflow/internal/reducer"
)

const Harness = "claude-hooks"

type hookPayload struct {
	HookEventName string `json:"hook_event_name"`

	SessionID string `json:"session_id"`
	AgentID   string `json:"agent_id"`
	AgentType string `json:"agent_type"`

	Prompt       string `json:"prompt"`
	SessionTitle string `json:"session_title"`

	LastAssistantMessage string `json:"last_assistant_message"`
	Reason               string `json:"reason"`
	Source               string `json:"source"`
}

func ApplyHook(g *graph.WorkGraph, raw []byte) error {
	evs, err := TranslateHook(raw)
	if err != nil {
		return err
	}
	if len(evs) == 0 {
		return nil
	}
	reducer.ApplyAll(g, evs)
	return nil
}

func inferEvent(p hookPayload) string {
	name := strings.TrimSpace(p.HookEventName)
	if name != "" {
		return name
	}
	if p.AgentID != "" && p.AgentType != "" && p.LastAssistantMessage == "" {
		return "SubagentStart"
	}
	if p.AgentID != "" && p.LastAssistantMessage != "" {
		return "SubagentStop"
	}
	if p.Prompt != "" {
		return "UserPromptSubmit"
	}
	if p.Reason != "" && p.Prompt == "" && p.AgentID == "" {
		return "SessionEnd"
	}
	return ""
}
// Devin hook commands reserve stdout for hook decision JSON; do not emit OSC from ApplyHook.
package devin

import (
	"strings"

	"github.com/athulramesh/rex-agentflow/internal/graph"
	"github.com/athulramesh/rex-agentflow/internal/reducer"
)

const Harness = "devin-hooks"

type hookPayload struct {
	HookEventName string `json:"hook_event_name"`

	SessionID string `json:"session_id"`
	PromptID  string `json:"prompt_id"`

	Prompt string `json:"prompt"`
	Source string `json:"source"`
	Reason string `json:"reason"`

	StopHookActive bool `json:"stop_hook_active"`

	ToolName string `json:"tool_name"`
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
	if p.SessionID != "" && p.Prompt == "" && p.Reason == "" && p.Source != "" {
		return "SessionStart"
	}
	if p.Prompt != "" {
		return "UserPromptSubmit"
	}
	if p.Reason != "" && p.Prompt == "" {
		return "SessionEnd"
	}
	return ""
}
// Package devin maps Devin CLI hook JSON to the shared contract.Event stream.
//
// Devin may also load hooks from .claude/ when read_config_from.claude is enabled
// (the default). This adapter is Devin-specific (session_id, prompt_id, PascalCase
// hook_event_name values). Use .devin/hooks.v1.json with agentflow devin-hook so
// agentflow does not compete with Claude-oriented hook wiring.
//
// Devin hook commands must reserve stdout for hook decision JSON (approve/block,
// additionalContext, updatedInput). ApplyHook never writes OSC or other graph
// output to stdout. Keep AGENTFLOW_OSC unset when agentflow runs inside Devin hooks;
// push Rex status with agentflow emit-osc in a separate step.
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

// ApplyHook updates g from one Devin CLI hook JSON object. Unknown events are no-ops.
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
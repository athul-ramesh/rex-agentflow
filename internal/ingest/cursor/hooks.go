package cursor

import (
	"strings"

	"github.com/athulramesh/rex-agentflow/internal/graph"
	"github.com/athulramesh/rex-agentflow/internal/reducer"
)

type hookPayload struct {
	HookEventName string `json:"hook_event_name"`

	SubagentID   string `json:"subagent_id"`
	SubagentType string `json:"subagent_type"`
	Task         string `json:"task"`
	Summary      string `json:"summary"`
	Status       string `json:"status"`
	Description  string `json:"description"`

	ToolName string `json:"tool_name"`
	ToolCall string `json:"tool_call_id"`

	ConversationID string `json:"conversation_id"`
	SessionID      string `json:"session_id"`
	Model          string `json:"model"`
}

const Harness = "cursor-hooks"

// ApplyHook updates g from one Cursor hook JSON object. Unknown events are no-ops.
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
	if p.SubagentID != "" && p.SubagentType != "" && p.Summary == "" && p.Status == "" {
		return "subagentStart"
	}
	if p.SubagentType != "" && (p.Status != "" || p.Summary != "") {
		return "subagentStop"
	}
	if p.ToolName != "" {
		return "preToolUse"
	}
	return ""
}

func isTaskTool(name string) bool {
	return strings.EqualFold(name, "Task") || strings.EqualFold(name, "task")
}

// subagentTitleMsg keeps the worker title short: kind (or a short task) up front,
// with long Task prompts demoted to msg.
func subagentTitleMsg(kind, task string) (title, msg string) {
	kind = strings.TrimSpace(kind)
	task = strings.TrimSpace(task)
	if task == "" {
		return kind, ""
	}
	if !isLongTask(task) {
		return task, ""
	}
	if kind != "" {
		return kind, graph.TruncateOneLine(task, 100)
	}
	return graph.TruncateOneLine(task, 40), ""
}

func isLongTask(s string) bool {
	if len(s) > 48 {
		return true
	}
	return strings.Count(s, " ") >= 8
}
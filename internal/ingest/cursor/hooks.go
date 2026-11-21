package cursor

import (
	"encoding/json"
	"strings"

	"github.com/athulramesh/rex-agentflow/internal/graph"
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
	var p hookPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil
	}
	g.Harness = Harness

	name := p.HookEventName
	if name == "" {
		name = inferEvent(p)
	}

	switch name {
	case "subagentStart":
		title := p.Task
		if title == "" {
			title = p.SubagentType
		}
		id := p.SubagentID
		if id == "" {
			id = p.ToolCall
		}
		if id == "" {
			id = "subagent-" + p.SubagentType
		}
		g.UpsertWorker(graph.Worker{
			ID:    id,
			Title: title,
			Kind:  p.SubagentType,
			State: graph.PhaseWorking,
		})
		root := g.EnsureMainWorker()
		root.State = graph.PhaseWorking

	case "subagentStop":
		id := p.SubagentID
		w := findStopTarget(g, id, p.SubagentType, p.Task)
		if w == nil {
			return nil
		}
		msg := graph.TruncateOneLine(p.Summary, 120)
		if msg == "" {
			msg = graph.TruncateOneLine(p.Description, 120)
		}
		w.State = graph.StatusToPhase(p.Status)
		if msg != "" {
			w.Msg = msg
		}
		g.Touch()

	case "preToolUse":
		if !isTaskTool(p.ToolName) {
			return nil
		}
		id := p.ToolCall
		if id == "" {
			id = "task-" + p.SubagentType
		}
		_, existing := g.FindWorker(id)
		if existing != nil {
			return nil
		}
		g.UpsertWorker(graph.Worker{
			ID:    id,
			Title: "Task",
			Kind:  "task",
			State: graph.PhaseWorking,
		})

	case "postToolUse", "postToolUseFailure":
		if !isTaskTool(p.ToolName) {
			return nil
		}
		id := p.ToolCall
		if id == "" {
			w := g.FindWorkingByKind("task", "")
			if w != nil {
				id = w.ID
			}
		}
		if id == "" {
			return nil
		}
		state := graph.PhaseFinished
		if name == "postToolUseFailure" {
			state = graph.PhaseError
		}
		g.SetWorkerState(id, state, "")

	case "stop":
		if p.Status != "" {
			g.Title = graph.TruncateOneLine(p.Status, 80)
		}
		if p.Summary != "" {
			g.Title = graph.TruncateOneLine(p.Summary, 80)
		}
		main := g.EnsureMainWorker()
		main.State = graph.PhaseFinished
		g.Touch()
	}
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

func findStopTarget(g *graph.WorkGraph, id, kind, task string) *graph.Worker {
	if id != "" {
		_, w := g.FindWorker(id)
		if w != nil {
			return w
		}
	}
	if task != "" {
		if w := g.FindWorkingByKind("", task); w != nil {
			return w
		}
	}
	return g.FindWorkingByKind(kind, "")
}
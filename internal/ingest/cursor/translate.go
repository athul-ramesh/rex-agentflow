package cursor

import (
	"encoding/json"
	"strings"

	"github.com/athulramesh/rex-agentflow/internal/contract"
)

func TranslateHook(raw []byte) ([]contract.Event, error) {
	var p hookPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, nil
	}
	name := p.HookEventName
	if name == "" {
		name = inferEvent(p)
	}
	if name == "" {
		return nil, nil
	}

	base := contract.Event{V: contract.Version, Harness: Harness}
	if p.ConversationID != "" {
		base.SessionID = p.ConversationID
	} else if p.SessionID != "" {
		base.SessionID = p.SessionID
	}

	switch name {
	case "subagentStart":
		title, msg := subagentTitleMsg(p.SubagentType, p.Task)
		id := p.SubagentID
		if id == "" {
			id = p.ToolCall
		}
		if id == "" {
			id = "subagent-" + p.SubagentType
		}
		return []contract.Event{{
			V: contract.Version, Type: contract.EventWorkerStart, Harness: Harness,
			SessionID: base.SessionID,
			Worker: contract.WorkerRef{
				ID: id, ToolCallID: p.ToolCall, Kind: p.SubagentType,
				Title: title, Msg: msg, State: "working",
			},
		}}, nil

	case "subagentStop":
		id := p.SubagentID
		msg := p.Summary
		if msg == "" {
			msg = p.Description
		}
		return []contract.Event{{
			V: contract.Version, Type: contract.EventWorkerStop, Harness: Harness,
			SessionID: base.SessionID,
			Status:    p.Status,
			Msg:       msg,
			Worker: contract.WorkerRef{
				ID: id, ToolCallID: p.ToolCall, Kind: p.SubagentType, Title: p.Task, Msg: msg,
			},
		}}, nil

	case "preToolUse":
		if !isTaskTool(p.ToolName) {
			return nil, nil
		}
		id := p.ToolCall
		if id == "" {
			id = "task-" + p.SubagentType
		}
		return []contract.Event{{
			V: contract.Version, Type: contract.EventWorkerStart, Harness: Harness,
			SessionID: base.SessionID,
			Worker: contract.WorkerRef{
				ID: id, ToolCallID: p.ToolCall, Kind: "task", Title: "Task", State: "working",
			},
		}}, nil

	case "postToolUse", "postToolUseFailure":
		if !isTaskTool(p.ToolName) {
			return nil, nil
		}
		status := "completed"
		if name == "postToolUseFailure" {
			status = "error"
		}
		if p.ToolCall == "" {
			return []contract.Event{{
				V: contract.Version, Type: contract.EventWorkerStop, Harness: Harness,
				SessionID: base.SessionID,
				Status:    status,
				Worker:    contract.WorkerRef{Kind: "task"},
			}}, nil
		}
		return []contract.Event{{
			V: contract.Version, Type: contract.EventToolReturn, Harness: Harness,
			SessionID: base.SessionID,
			Status:    status,
			Worker:    contract.WorkerRef{ID: p.ToolCall, ToolCallID: p.ToolCall, Kind: "task"},
		}}, nil

	case "stop":
		evs := []contract.Event{}
		if p.Status != "" || p.Summary != "" {
			title := p.Summary
			if title == "" {
				title = p.Status
			}
			evs = append(evs, contract.Event{
				V: contract.Version, Type: contract.EventTurnSettle, Harness: Harness,
				SessionID: base.SessionID, Title: title,
			})
		}
		evs = append(evs, contract.Event{
			V: contract.Version, Type: contract.EventSessionEnd, Harness: Harness,
			SessionID: base.SessionID, Title: p.Summary, Msg: p.Status,
		})
		return evs, nil
	}
	return nil, nil
}

func TranslateStreamLine(line []byte) ([]contract.Event, bool) {
	var ev streamLine
	if err := json.Unmarshal(line, &ev); err != nil {
		return nil, false
	}
	if ev.Type == "" {
		var generic map[string]json.RawMessage
		if json.Unmarshal(line, &generic) != nil {
			return nil, false
		}
		if raw, ok := generic["session_id"]; ok {
			var sid string
			_ = json.Unmarshal(raw, &sid)
			if sid != "" {
				return []contract.Event{{
					V: contract.Version, Type: contract.EventSessionBind, Harness: StreamHarness,
					SessionID: sid,
				}}, true
			}
		}
		return nil, false
	}

	switch ev.Type {
	case "system":
		if ev.Subtype == "init" || ev.RunID != "" || ev.SessionID != "" {
			sid := ev.SessionID
			if sid == "" {
				sid = ev.RunID
			}
			if sid == "" {
				return nil, false
			}
			return []contract.Event{{
				V: contract.Version, Type: contract.EventSessionBind, Harness: StreamHarness,
				SessionID: sid, RunID: ev.RunID,
			}}, true
		}
	case "init":
		if ev.SessionID != "" {
			return []contract.Event{{
				V: contract.Version, Type: contract.EventSessionBind, Harness: StreamHarness,
				SessionID: ev.SessionID,
			}}, true
		}

	case "tool_call":
		name := ev.Name
		if name == "" {
			name = toolNameFromPayload(ev.ToolCall)
		}
		if name == "" {
			name = "tool"
		}
		phase := strings.ToLower(ev.Status)
		if phase == "" {
			phase = strings.ToLower(ev.Subtype)
		}
		id := contract.SanitizeID(ev.CallID)
		if id == "" {
			id = "tool-" + name
		}
		switch phase {
		case "completed", "complete", "success":
			return []contract.Event{{
				V: contract.Version, Type: contract.EventToolReturn, Harness: StreamHarness,
				SessionID: ev.SessionID, RunID: ev.RunID,
				Status:    "completed",
				Worker: contract.WorkerRef{
					ID: id, ToolCallID: ev.CallID, Kind: "tool", Title: name,
					Msg: name + " done",
				},
			}}, true
		case "error", "failed":
			return []contract.Event{{
				V: contract.Version, Type: contract.EventToolReturn, Harness: StreamHarness,
				SessionID: ev.SessionID, RunID: ev.RunID,
				Status:    "error",
				Worker: contract.WorkerRef{
					ID: id, ToolCallID: ev.CallID, Kind: "tool", Title: name,
					Msg: name + " failed",
				},
			}}, true
		default:
			return []contract.Event{{
				V: contract.Version, Type: contract.EventWorkerStart, Harness: StreamHarness,
				SessionID: ev.SessionID, RunID: ev.RunID,
				Worker: contract.WorkerRef{
					ID: id, ToolCallID: ev.CallID, Kind: "tool", Title: name,
					Msg: name, State: "working",
				},
			}}, true
		}

	case "result":
		msg := ev.Text
		if msg == "" {
			msg = ev.Result
		}
		return []contract.Event{{
			V: contract.Version, Type: contract.EventSessionEnd, Harness: StreamHarness,
			SessionID: ev.SessionID, RunID: ev.RunID, Msg: msg,
		}}, true
	}
	return nil, false
}
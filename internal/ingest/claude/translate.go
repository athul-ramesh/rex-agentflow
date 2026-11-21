package claude

import (
	"encoding/json"
	"strings"

	"github.com/athulramesh/rex-agentflow/internal/contract"
	"github.com/athulramesh/rex-agentflow/internal/graph"
)

// TranslateHook maps one Claude Code hook JSON object to contract events.
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

	switch hookKey(name) {
	case "sessionstart":
		title := strings.TrimSpace(p.SessionTitle)
		if title == "" && p.Source != "" {
			title = p.Source
		}
		return []contract.Event{{
			V: contract.Version, Type: contract.EventSessionBind, Harness: Harness,
			SessionID: p.SessionID, Title: title,
		}}, nil

	case "userpromptsubmit":
		title := strings.TrimSpace(p.SessionTitle)
		if title == "" {
			title = graph.TruncateOneLine(strings.TrimSpace(p.Prompt), 80)
		}
		return []contract.Event{{
			V: contract.Version, Type: contract.EventSessionBind, Harness: Harness,
			SessionID: p.SessionID, Title: title,
		}}, nil

	case "subagentstart":
		id := p.AgentID
		if id == "" {
			id = "subagent-" + p.AgentType
		}
		kind := strings.TrimSpace(p.AgentType)
		title := kind
		if title == "" {
			title = "subagent"
		}
		return []contract.Event{{
			V: contract.Version, Type: contract.EventWorkerStart, Harness: Harness,
			SessionID: p.SessionID,
			Worker: contract.WorkerRef{
				ID: id, Kind: kind, Title: title, State: "working",
			},
		}}, nil

	case "subagentstop":
		id := p.AgentID
		status := "completed"
		msg := strings.TrimSpace(p.LastAssistantMessage)
		kind := strings.TrimSpace(p.AgentType)
		return []contract.Event{{
			V: contract.Version, Type: contract.EventWorkerStop, Harness: Harness,
			SessionID: p.SessionID,
			Status:    status,
			Msg:       msg,
			Worker: contract.WorkerRef{
				ID: id, Kind: kind, Title: kind, Msg: msg,
			},
		}}, nil

	case "stop":
		title := graph.TruncateOneLine(strings.TrimSpace(p.LastAssistantMessage), 80)
		if title == "" {
			title = graph.TruncateOneLine(strings.TrimSpace(p.SessionTitle), 80)
		}
		evs := []contract.Event{}
		if title != "" {
			evs = append(evs, contract.Event{
				V: contract.Version, Type: contract.EventTurnSettle, Harness: Harness,
				SessionID: p.SessionID, Title: title,
			})
		}
		return evs, nil

	case "sessionend":
		title := strings.TrimSpace(p.Reason)
		if title == "" {
			title = "session ended"
		}
		return []contract.Event{{
			V: contract.Version, Type: contract.EventSessionEnd, Harness: Harness,
			SessionID: p.SessionID, Title: title, Msg: p.Reason,
		}}, nil
	}
	return nil, nil
}

func hookKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
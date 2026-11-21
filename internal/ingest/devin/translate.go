package devin

import (
	"encoding/json"
	"strings"

	"github.com/athulramesh/rex-agentflow/internal/contract"
	"github.com/athulramesh/rex-agentflow/internal/graph"
)

// TranslateHook maps one Devin CLI hook JSON object to contract events.
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
		title := strings.TrimSpace(p.Source)
		if title == "" {
			title = "session"
		}
		return []contract.Event{{
			V: contract.Version, Type: contract.EventSessionBind, Harness: Harness,
			SessionID: p.SessionID, Title: title,
		}}, nil

	case "userpromptsubmit":
		title := graph.TruncateOneLine(strings.TrimSpace(p.Prompt), 80)
		evs := []contract.Event{{
			V: contract.Version, Type: contract.EventSessionBind, Harness: Harness,
			SessionID: p.SessionID, RunID: p.PromptID, Title: title,
		}}
		if title != "" {
			evs = append(evs, contract.Event{
				V: contract.Version, Type: contract.EventWorkerStart, Harness: Harness,
				SessionID: p.SessionID, RunID: p.PromptID,
				Worker: contract.WorkerRef{
					ID: graph.MainWorkerID, Kind: "main", Title: title,
					State: "working", ToolCallID: p.PromptID,
				},
			})
		}
		return evs, nil

	case "stop", "sessionend":
		title := strings.TrimSpace(p.Reason)
		if title == "" && hookKey(name) == "stop" {
			title = "stopped"
		}
		if title == "" {
			title = "session ended"
		}
		return []contract.Event{{
			V: contract.Version, Type: contract.EventSessionEnd, Harness: Harness,
			SessionID: p.SessionID, RunID: p.PromptID, Title: title, Msg: p.Reason,
		}}, nil

	// Permission and tool hooks are diagnostic/audit only for agentflow; they do
	// not carry stable subagent identity in Devin's documented stdin fields.
	case "permissionrequest", "pretooluse", "posttooluse", "postcompaction":
		return nil, nil
	}
	return nil, nil
}

func hookKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
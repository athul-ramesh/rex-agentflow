package cursor

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/athulramesh/rex-agentflow/internal/graph"
)

const StreamHarness = "cursor-stream"

type streamLine struct {
	Type      string `json:"type"`
	Subtype   string `json:"subtype"`
	SessionID string `json:"session_id"`
	RunID     string `json:"run_id"`

	CallID   string          `json:"call_id"`
	Name     string          `json:"name"`
	Status   string          `json:"status"`
	Text     string          `json:"text"`
	ToolCall json.RawMessage `json:"tool_call"`
	Result   string          `json:"result"`

	// legacy init line
	Model string `json:"model"`
}

// ProcessStream reads NDJSON until EOF, mutating g and calling onChange after each applied line.
func ProcessStream(g *graph.WorkGraph, r io.Reader, onChange func() error) error {
	if g.Harness == "" {
		g.Harness = StreamHarness
	}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		if applyStreamLine(g, line) {
			if onChange != nil {
				if err := onChange(); err != nil {
					return err
				}
			}
		}
	}
	return sc.Err()
}

func applyStreamLine(g *graph.WorkGraph, line []byte) bool {
	var ev streamLine
	if err := json.Unmarshal(line, &ev); err != nil {
		return false
	}
	if ev.Type == "" {
		// try hook-shaped init
		var generic map[string]json.RawMessage
		if json.Unmarshal(line, &generic) != nil {
			return false
		}
		if raw, ok := generic["session_id"]; ok {
			var sid string
			_ = json.Unmarshal(raw, &sid)
			if sid != "" {
				g.RunID = sid
				g.Touch()
				return true
			}
		}
		return false
	}

	switch ev.Type {
	case "system":
		if ev.Subtype == "init" || ev.RunID != "" {
			if ev.RunID != "" {
				g.RunID = ev.RunID
			}
			if ev.SessionID != "" {
				g.RunID = ev.SessionID
			}
			g.Harness = StreamHarness
			g.Touch()
			return true
		}
	case "init":
		if ev.SessionID != "" {
			g.RunID = ev.SessionID
		}
		g.Harness = StreamHarness
		g.Touch()
		return true

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
		main := g.EnsureMainWorker()
		main.State = graph.PhaseWorking
		main.Msg = name
		id := sanitizeWorkerID(ev.CallID)
		if id == "" {
			id = "tool-" + name
		}
		w := graph.Worker{ID: id, Title: name, Kind: "tool", State: graph.PhaseWorking, Msg: name}
		switch phase {
		case "running", "started", "in_progress", "":
			w.State = graph.PhaseWorking
		case "completed", "complete", "success":
			w.State = graph.PhaseFinished
			w.Msg = name + " done"
		case "error", "failed":
			w.State = graph.PhaseError
			w.Msg = name + " failed"
			main.State = graph.PhaseError
			main.Msg = w.Msg
		}
		g.UpsertWorker(w)
		g.Touch()
		return true

	case "result":
		main := g.EnsureMainWorker()
		main.State = graph.PhaseFinished
		msg := ev.Text
		if msg == "" {
			msg = ev.Result
		}
		if msg != "" {
			main.Msg = graph.TruncateOneLine(msg, 120)
		}
		g.Touch()
		return true
	}
	return false
}

func toolNameFromPayload(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) != nil {
		return ""
	}
	for key := range obj {
		if strings.HasSuffix(key, "ToolCall") {
			name := strings.TrimSuffix(key, "ToolCall")
			if name == "" {
				continue
			}
			return strings.ToLower(name[:1]) + name[1:]
		}
	}
	return ""
}

func sanitizeWorkerID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	id = strings.ReplaceAll(id, "\n", "-")
	id = strings.ReplaceAll(id, "/", "-")
	if len(id) > 48 {
		id = id[:48]
	}
	return id
}
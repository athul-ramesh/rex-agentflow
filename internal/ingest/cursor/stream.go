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

	CallID string `json:"call_id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Text   string `json:"text"`

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
		main := g.EnsureMainWorker()
		name := ev.Name
		if name == "" {
			name = "tool"
		}
		switch strings.ToLower(ev.Status) {
		case "running", "started", "in_progress", "":
			main.State = graph.PhaseWorking
			main.Msg = name
			main.Title = "main"
			g.Touch()
		case "completed", "complete", "success":
			main.Msg = name + " done"
			if main.State == graph.PhaseWorking {
				main.State = graph.PhaseWorking
			}
			g.Touch()
		case "error", "failed":
			main.State = graph.PhaseError
			main.Msg = name + " failed"
			g.Touch()
		}
		return true

	case "result":
		main := g.EnsureMainWorker()
		main.State = graph.PhaseFinished
		if ev.Text != "" {
			main.Msg = graph.TruncateOneLine(ev.Text, 120)
		}
		g.Touch()
		return true
	}
	return false
}
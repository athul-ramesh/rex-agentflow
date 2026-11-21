package cursor

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/athulramesh/rex-agentflow/internal/graph"
	"github.com/athulramesh/rex-agentflow/internal/reducer"
	"github.com/athulramesh/rex-agentflow/internal/store"
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

	Model string `json:"model"`
}

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
		evs, ok := TranslateStreamLine(line)
		if !ok || len(evs) == 0 {
			continue
		}
		reducer.ApplyAll(g, evs)
		if onChange != nil {
			if err := onChange(); err != nil {
				return err
			}
		}
	}
	return sc.Err()
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

func ProcessStreamLocked(name string, r io.Reader, onChange func() error) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		evs, ok := TranslateStreamLine(line)
		if !ok || len(evs) == 0 {
			continue
		}
		if err := store.WithGraph(name, StreamHarness, func(g *graph.WorkGraph) error {
			reducer.ApplyAll(g, evs)
			return nil
		}); err != nil {
			return err
		}
		if onChange != nil {
			if err := onChange(); err != nil {
				return err
			}
		}
	}
	return sc.Err()
}
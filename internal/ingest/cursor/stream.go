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

// ProcessStreamLocked routes each NDJSON line to a graph file keyed by session id.
// pinned is the --graph flag (empty if unset). onReady runs once with the latched raw id for --show.
func ProcessStreamLocked(pinned, fallback string, r io.Reader, onReady func(graphKey string) error) error {
	pinned = strings.TrimSpace(pinned)
	fallback = strings.TrimSpace(fallback)
	if fallback == "" {
		fallback = "default"
	}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var held [][]byte
	latched := pinned
	readyCalled := false

	flushLine := func(line []byte, target string) error {
		target = strings.TrimSpace(target)
		if target == "" {
			target = "default"
		}
		evs, ok := TranslateStreamLine(line)
		if !ok || len(evs) == 0 {
			return nil
		}
		return store.WithGraph(target, StreamHarness, func(g *graph.WorkGraph) error {
			reducer.ApplyAll(g, evs)
			return nil
		})
	}

	callReady := func(id string) error {
		if readyCalled || onReady == nil {
			return nil
		}
		readyCalled = true
		return onReady(id)
	}

	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		evs, ok := TranslateStreamLine(line)
		if !ok || len(evs) == 0 {
			continue
		}
		sid := strings.TrimSpace(evs[0].SessionID)
		if latched == "" {
			if sid != "" {
				latched = sid
				for _, h := range held {
					if err := flushLine(h, latched); err != nil {
						return err
					}
				}
				held = nil
				if err := callReady(latched); err != nil {
					return err
				}
				if err := flushLine(line, latched); err != nil {
					return err
				}
				continue
			}
			held = append(held, append([]byte(nil), line...))
			continue
		}
		target := latched
		if sid != "" && sid != latched {
			target = sid
		}
		if err := flushLine(line, target); err != nil {
			return err
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	if latched == "" {
		latched = fallback
		for _, h := range held {
			if err := flushLine(h, latched); err != nil {
				return err
			}
		}
		if err := callReady(latched); err != nil {
			return err
		}
		return nil
	}
	for _, h := range held {
		if err := flushLine(h, latched); err != nil {
			return err
		}
	}
	return nil
}

func SessionIDFromStreamLine(line []byte) string {
	evs, ok := TranslateStreamLine(line)
	if !ok || len(evs) == 0 {
		return ""
	}
	return strings.TrimSpace(evs[0].SessionID)
}
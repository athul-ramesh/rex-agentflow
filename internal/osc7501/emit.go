package osc7501

import (
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"github.com/athulramesh/rex-agentflow/internal/graph"
)

// OSCState maps graph phases to Rex program status states.
func OSCState(p graph.Phase) string {
	switch p {
	case graph.PhaseWorking:
		return "working"
	case graph.PhaseBlocked:
		return "blocked"
	case graph.PhaseFinished:
		return "done"
	case graph.PhaseError:
		return "error"
	case graph.PhaseAborted, graph.PhaseQueued:
		return "idle"
	default:
		return "idle"
	}
}

func b64(s string) string {
	if s == "" {
		return ""
	}
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// Record is one OSC 7501 emission.
type Record struct {
	State    string
	ID       string
	Title    string
	Msg      string
	Progress *int
	Kind     string
	App      string
}

func (r Record) Sequence() string {
	if r.State == "" {
		return ""
	}
	// queued → omit unless explicitly idle
	if r.State == "idle" && r.ID != "" {
		// still emit idle updates for workers clearing
	}
	var parts []string
	parts = append(parts, "state="+r.State)
	if r.ID != "" {
		parts = append(parts, "id="+r.ID)
	}
	if r.Title != "" {
		parts = append(parts, "title="+b64(r.Title))
	}
	if r.Msg != "" {
		parts = append(parts, "msg="+b64(r.Msg))
	}
	if r.Progress != nil {
		parts = append(parts, fmt.Sprintf("progress=%d", *r.Progress))
	}
	if r.Kind != "" {
		parts = append(parts, "kind="+r.Kind)
	}
	if r.App != "" {
		parts = append(parts, "app="+r.App)
	}
	return fmt.Sprintf("\x1b]7501;%s\x1b\\", strings.Join(parts, ":"))
}

func workerRecord(w graph.Worker) Record {
	st := OSCState(w.State)
	if w.State == graph.PhaseQueued && st == "idle" {
		return Record{} // omit queued workers
	}
	return Record{
		State:    st,
		ID:       graph.WorkerOSCPath(w.ID),
		Title:    w.Title,
		Msg:      w.Msg,
		Progress: w.Progress,
		Kind:     w.Kind,
	}
}

func rootRecord(g graph.WorkGraph) Record {
	title := g.Title
	if title == "" {
		title = "agent"
	}
	msg := ""
	for _, w := range g.Workers {
		if w.State == graph.PhaseWorking && w.Msg != "" {
			msg = w.Msg
			break
		}
	}
	st := RootStatus(g)
	if st == "done" {
		st = "done"
	}
	return Record{
		State: st,
		Title: title,
		Msg:   msg,
	}
}

// GraphRecords returns root plus each worker with a non-empty OSC state.
func GraphRecords(g graph.WorkGraph) []Record {
	var out []Record
	if rec := rootRecord(g); rec.State != "" && rec.State != "idle" {
		out = append(out, rec)
	} else if g.Title != "" || len(g.Workers) > 0 {
		out = append(out, rec)
	}
	for _, w := range g.Workers {
		if w.ID == graph.MainWorkerID {
			continue
		}
		if rec := workerRecord(w); rec.State != "" {
			out = append(out, rec)
		}
	}
	return out
}

// WriteGraph prints OSC sequences for the graph to w.
func WriteGraph(w io.Writer, g graph.WorkGraph) error {
	for _, rec := range GraphRecords(g) {
		seq := rec.Sequence()
		if seq == "" {
			continue
		}
		if _, err := io.WriteString(w, seq); err != nil {
			return err
		}
	}
	return nil
}

// ContainsWorking reports whether any sequence would include state=working with id=.
func ContainsWorking(g graph.WorkGraph) (working bool, hasID bool) {
	for _, rec := range GraphRecords(g) {
		if rec.State == "working" {
			working = true
			if rec.ID != "" {
				hasID = true
			}
		}
	}
	return working, hasID
}
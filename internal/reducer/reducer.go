package reducer

import (
	"strings"
	"time"

	"github.com/athulramesh/rex-agentflow/internal/contract"
	"github.com/athulramesh/rex-agentflow/internal/graph"
)

func Apply(g *graph.WorkGraph, ev contract.Event) []graph.Diagnostic {
	var out []graph.Diagnostic
	if g == nil {
		return out
	}
	graph.Normalize(g)

	if ev.Harness != "" && g.Harness == "" {
		g.Harness = ev.Harness
	}

	sid := ev.SessionKey()
	if sid != "" {
		if g.SessionID == "" {
			g.SessionID = sid
		} else if g.SessionID != sid {
			d := diag("stale_session", "event session "+sid+" does not match graph session "+g.SessionID)
			appendDiag(g, d)
			return []graph.Diagnostic{d}
		}
	}
	if ev.RunID != "" {
		g.RunID = ev.RunID
	}

	recordAliases(g, ev)

	switch ev.Type {
	case contract.EventSessionBind:
		if ev.Title != "" {
			g.Title = graph.TruncateOneLine(ev.Title, 80)
		}
		g.Touch()

	case contract.EventWorkerStart:
		id := resolveWorkerID(g, ev.Worker)
		if id == "" {
			d := diag("worker_start_no_id", "worker.start with no resolvable id")
			appendDiag(g, d)
			return []graph.Diagnostic{d}
		}
		if _, w := g.FindWorker(id); w != nil && isTerminal(w.State) {
			d := diag("late_start", "worker.start for terminal worker "+id)
			appendDiag(g, d)
			return []graph.Diagnostic{d}
		}
		state := graph.PhaseWorking
		if ev.Worker.State != "" {
			state = graph.StatusToPhase(ev.Worker.State)
		}
		w := graph.Worker{
			ID:       id,
			Title:    ev.Worker.Title,
			Kind:     ev.Worker.Kind,
			Msg:      ev.Worker.Msg,
			State:    state,
			ParentID: ev.Worker.ParentID,
		}
		g.UpsertWorker(w)
		root := g.EnsureMainWorker()
		root.State = graph.PhaseWorking
		if w.Kind == "tool" && w.Title != "" {
			root.Msg = w.Title
		}

	case contract.EventWorkerPatch:
		id := resolveWorkerID(g, ev.Worker)
		if id == "" {
			d := diag("worker_patch_no_id", "worker.patch with no resolvable id")
			appendDiag(g, d)
			return []graph.Diagnostic{d}
		}
		patch := graph.Worker{ID: id}
		if ev.Worker.Title != "" {
			patch.Title = ev.Worker.Title
		}
		if ev.Worker.Msg != "" {
			patch.Msg = ev.Worker.Msg
		}
		if ev.Worker.Kind != "" {
			patch.Kind = ev.Worker.Kind
		}
		if ev.Worker.State != "" {
			patch.State = graph.StatusToPhase(ev.Worker.State)
		}
		g.UpsertWorker(patch)

	case contract.EventWorkerStop:
		id := resolveWorkerID(g, ev.Worker)
		if id == "" {
			var d graph.Diagnostic
			id, d = resolveStopByKind(g, ev.Worker.Kind, ev.Worker.Title)
			if d.Code != "" {
				appendDiag(g, d)
				return []graph.Diagnostic{d}
			}
		}
		if id == "" {
			d := diag("worker_stop_unresolved", "worker.stop could not resolve target")
			appendDiag(g, d)
			return []graph.Diagnostic{d}
		}
		state := graph.PhaseFinished
		if ev.Status != "" {
			state = graph.StatusToPhase(ev.Status)
		} else if ev.Worker.State != "" {
			state = graph.StatusToPhase(ev.Worker.State)
		}
		msg := ev.Msg
		if msg == "" {
			msg = ev.Worker.Msg
		}
		g.SetWorkerState(id, state, graph.TruncateOneLine(msg, 120))

	case contract.EventToolReturn:
		id := resolveWorkerID(g, ev.Worker)
		if id == "" {
			d := diag("tool_return_unresolved", "tool.return with no resolvable id")
			appendDiag(g, d)
			return []graph.Diagnostic{d}
		}
		state := graph.PhaseFinished
		if ev.Status != "" {
			state = graph.StatusToPhase(ev.Status)
		}
		if strings.EqualFold(ev.Status, "failed") || strings.EqualFold(ev.Status, "error") {
			state = graph.PhaseError
		}
		msg := graph.TruncateOneLine(ev.Msg, 120)
		if msg == "" {
			msg = graph.TruncateOneLine(ev.Worker.Msg, 120)
		}
		g.SetWorkerState(id, state, msg)
		if state == graph.PhaseError {
			main := g.EnsureMainWorker()
			main.State = graph.PhaseError
			if msg != "" {
				main.Msg = msg
			}
		}

	case contract.EventTurnSettle:
		title := ev.Title
		if title == "" {
			title = ev.Msg
		}
		if title != "" {
			g.Title = graph.TruncateOneLine(title, 80)
		}
		g.Touch()

	case contract.EventSessionEnd:
		if ev.Title != "" {
			g.Title = graph.TruncateOneLine(ev.Title, 80)
		} else if ev.Msg != "" {
			g.Title = graph.TruncateOneLine(ev.Msg, 80)
		}
		main := g.EnsureMainWorker()
		main.State = graph.PhaseFinished
		if ev.Msg != "" {
			main.Msg = graph.TruncateOneLine(ev.Msg, 120)
		}
		g.Touch()
	}
	return out
}

func ApplyAll(g *graph.WorkGraph, evs []contract.Event) []graph.Diagnostic {
	var all []graph.Diagnostic
	for _, ev := range evs {
		all = append(all, Apply(g, ev)...)
	}
	return all
}

func isTerminal(p graph.Phase) bool {
	switch p {
	case graph.PhaseFinished, graph.PhaseError, graph.PhaseAborted:
		return true
	default:
		return false
	}
}

func recordAliases(g *graph.WorkGraph, ev contract.Event) {
	id := contract.SanitizeID(ev.Worker.ID)
	tool := contract.SanitizeID(ev.Worker.ToolCallID)
	if id == "" || tool == "" || id == tool {
		return
	}
	if g.Correlation == nil {
		g.Correlation = make(map[string]string)
	}
	g.Correlation[id] = tool
	g.Correlation[tool] = id
}

func resolveWorkerID(g *graph.WorkGraph, ref contract.WorkerRef) string {
	id := contract.SanitizeID(ref.ID)
	if id == "" {
		id = contract.SanitizeID(ref.ToolCallID)
	}
	if id == "" {
		return ""
	}
	if g.Correlation != nil {
		if alt, ok := g.Correlation[id]; ok && alt != "" {
			if _, w := g.FindWorker(alt); w != nil {
				return alt
			}
		}
	}
	if _, w := g.FindWorker(id); w != nil {
		return id
	}
	if g.Correlation != nil {
		if alt, ok := g.Correlation[id]; ok {
			return alt
		}
	}
	return id
}

func resolveStopByKind(g *graph.WorkGraph, kind, title string) (string, graph.Diagnostic) {
	var matches []*graph.Worker
	for i := range g.Workers {
		w := &g.Workers[i]
		if w.State != graph.PhaseWorking {
			continue
		}
		if kind != "" && w.Kind == kind {
			matches = append(matches, w)
		}
		if title != "" && w.Title == title {
			return w.ID, graph.Diagnostic{}
		}
	}
	if len(matches) == 1 {
		return matches[0].ID, graph.Diagnostic{}
	}
	if len(matches) == 0 {
		return "", diag("worker_stop_no_match", "no working worker for kind "+kind)
	}
	return "", diag("worker_stop_ambiguous", "multiple working workers for kind "+kind)
}

func diag(code, msg string) graph.Diagnostic {
	return graph.Diagnostic{Code: code, Msg: msg, At: time.Now().UTC()}
}

func appendDiag(g *graph.WorkGraph, d graph.Diagnostic) {
	g.Diagnostics = append(g.Diagnostics, d)
	g.Touch()
}
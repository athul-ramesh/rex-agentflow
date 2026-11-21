package graph

import (
	"fmt"
	"strings"
	"time"
)

func (g *WorkGraph) Touch() {
	g.UpdatedAt = time.Now().UTC()
}

func (g *WorkGraph) FindWorker(id string) (int, *Worker) {
	for i := range g.Workers {
		if g.Workers[i].ID == id {
			return i, &g.Workers[i]
		}
	}
	return -1, nil
}

func (g *WorkGraph) UpsertWorker(w Worker) {
	_, existing := g.FindWorker(w.ID)
	if existing != nil {
		if w.Title != "" {
			existing.Title = w.Title
		}
		if w.Msg != "" {
			existing.Msg = w.Msg
		}
		if w.State != "" {
			existing.State = w.State
		}
		if w.ParentID != "" {
			existing.ParentID = w.ParentID
		}
		if w.Kind != "" {
			existing.Kind = w.Kind
		}
		if w.Progress != nil {
			existing.Progress = w.Progress
		}
		g.Touch()
		return
	}
	g.Workers = append(g.Workers, w)
	g.Touch()
}

func (g *WorkGraph) SetWorkerState(id string, state Phase, msg string) {
	_, w := g.FindWorker(id)
	if w == nil {
		return
	}
	w.State = state
	if msg != "" {
		w.Msg = msg
	}
	g.Touch()
}

func (g *WorkGraph) EnsureMainWorker() *Worker {
	_, w := g.FindWorker(MainWorkerID)
	if w != nil {
		return w
	}
	g.UpsertWorker(Worker{ID: MainWorkerID, Title: "main", State: PhaseWorking})
	_, w = g.FindWorker(MainWorkerID)
	return w
}

// FindWorkingByKind returns the last working worker matching kind (or task title).
func (g *WorkGraph) FindWorkingByKind(kind, task string) *Worker {
	var cand *Worker
	for i := range g.Workers {
		w := &g.Workers[i]
		if w.State != PhaseWorking {
			continue
		}
		if kind != "" && w.Kind == kind {
			cand = w
		}
		if task != "" && w.Title == task {
			return w
		}
	}
	return cand
}

func TruncateOneLine(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func StatusToPhase(status string) Phase {
	switch strings.ToLower(status) {
	case "completed", "complete", "success", "succeeded", "finished", "done":
		return PhaseFinished
	case "error", "failed", "failure":
		return PhaseError
	case "aborted", "cancelled", "canceled":
		return PhaseAborted
	case "blocked":
		return PhaseBlocked
	case "working", "running", "in_progress":
		return PhaseWorking
	default:
		return PhaseFinished
	}
}

func SanitizeOSCID(id string) string {
	id = strings.TrimPrefix(id, "w/")
	var b strings.Builder
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '/':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		default:
			b.WriteRune('_')
		}
	}
	out := b.String()
	if out == "" {
		return "unknown"
	}
	return out
}

func WorkerOSCPath(id string) string {
	if id == "" || id == MainWorkerID {
		return ""
	}
	return fmt.Sprintf("w/%s", SanitizeOSCID(id))
}
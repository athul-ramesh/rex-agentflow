package view

import (
	"fmt"
	"strings"

	"github.com/athulramesh/rex-agentflow/internal/graph"
)

type rowKind int

const (
	rowTitle rowKind = iota
	rowHairline
	rowSection
	rowWorker
	rowPlan
	rowFooter
	rowEmpty
)

type layoutRow struct {
	Kind     rowKind
	Text     string
	Sub      string
	Phase    graph.Phase
	Depth    int
	Active   bool
	Progress *int
}

func buildLayout(g graph.WorkGraph) []layoutRow {
	var rows []layoutRow
	rows = append(rows, layoutRow{Kind: rowTitle, Text: titleOrDefault(g)})

	if len(g.Plan) > 0 {
		rows = append(rows, layoutRow{Kind: rowSection, Text: "Plan"})
		for _, item := range g.Plan {
			rows = append(rows, layoutRow{Kind: rowPlan, Text: item.Text, Phase: item.Status})
		}
	}

	rows = append(rows, layoutRow{Kind: rowSection, Text: "Workers"})

	roots := workersByParent(g.Workers, "")
	if len(g.Workers) == 0 {
		rows = append(rows, layoutRow{Kind: rowEmpty, Text: "(empty)"})
	} else {
		appendWorkers(&rows, g.Workers, roots, 0)
	}

	if n := len(g.Workers); n > 0 {
		rows = append(rows, layoutRow{Kind: rowFooter, Text: footerText(g)})
	}
	return rows
}

func appendWorkers(rows *[]layoutRow, all []graph.Worker, level []graph.Worker, depth int) {
	for _, w := range level {
		sub := workerSub(w)
		*rows = append(*rows, layoutRow{
			Kind:     rowWorker,
			Text:     w.Title,
			Sub:      sub,
			Phase:    w.State,
			Depth:    depth,
			Active:   w.State == graph.PhaseWorking,
			Progress: w.Progress,
		})
		children := workersByParent(all, w.ID)
		appendWorkers(rows, all, children, depth+1)
	}
}

func workerSub(w graph.Worker) string {
	if w.Kind == "" && w.Msg == "" {
		return ""
	}
	if w.Msg != "" {
		if w.Kind != "" {
			return w.Kind + " · " + w.Msg
		}
		return w.Msg
	}
	return w.Kind
}

func footerText(g graph.WorkGraph) string {
	n := len(g.Workers)
	var parts []string
	parts = append(parts, fmt.Sprintf("%d worker", n))
	if n != 1 {
		parts[0] += "s"
	}
	counts := map[graph.Phase]int{}
	for _, w := range g.Workers {
		counts[w.State]++
	}
	for _, ph := range []graph.Phase{graph.PhaseWorking, graph.PhaseError, graph.PhaseBlocked, graph.PhaseQueued} {
		c := counts[ph]
		if c == 0 {
			continue
		}
		label := string(ph)
		if c != 1 {
			if ph == graph.PhaseError {
				label = "errors"
			} else {
				label += "s"
			}
		}
		parts = append(parts, fmt.Sprintf("%d %s", c, label))
	}
	return strings.Join(parts, " · ")
}

func workersByParent(workers []graph.Worker, parent string) []graph.Worker {
	var out []graph.Worker
	for _, w := range workers {
		if w.ParentID == parent {
			out = append(out, w)
		}
	}
	if parent == "" && len(out) == 0 {
		return workers
	}
	return out
}

func titleOrDefault(g graph.WorkGraph) string {
	if g.Title != "" {
		return g.Title
	}
	if g.RunID != "" {
		return "run " + truncateRunID(g.RunID, 24)
	}
	return "agentflow"
}

func truncateRunID(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
package osc7501

import "github.com/athulramesh/rex-agentflow/internal/graph"

// RootStatus picks aggregate agent status from worker phases.
// Priority: error > blocked > working > finished > idle/queued/aborted.
// When non-main workers exist, only they drive working/blocked/error so a
// synthetic main left at "working" cannot keep the root active after every
// real worker has finished.
func RootStatus(g graph.WorkGraph) string {
	if len(g.Workers) == 0 {
		return "idle"
	}
	hasError := false
	hasBlocked := false
	hasWorking := false
	hasFinished := false
	nonMain := 0
	for _, w := range g.Workers {
		if w.ID == graph.MainWorkerID {
			continue
		}
		nonMain++
		switch w.State {
		case graph.PhaseError:
			hasError = true
		case graph.PhaseBlocked:
			hasBlocked = true
		case graph.PhaseWorking:
			hasWorking = true
		case graph.PhaseFinished:
			hasFinished = true
		}
	}
	if nonMain == 0 {
		if _, main := g.FindWorker(graph.MainWorkerID); main != nil {
			switch main.State {
			case graph.PhaseError:
				hasError = true
			case graph.PhaseBlocked:
				hasBlocked = true
			case graph.PhaseWorking:
				hasWorking = true
			case graph.PhaseFinished:
				hasFinished = true
			}
		}
	}
	if hasError {
		return "error"
	}
	if hasBlocked {
		return "blocked"
	}
	if hasWorking {
		return "working"
	}
	if hasFinished || nonMain > 0 {
		return "done"
	}
	return "idle"
}
package osc7501

import "github.com/athulramesh/rex-agentflow/internal/graph"

type phaseFlags struct {
	error    bool
	blocked  bool
	working  bool
	finished bool
}

func accumulatePhase(f *phaseFlags, state graph.Phase) {
	switch state {
	case graph.PhaseError:
		f.error = true
	case graph.PhaseBlocked:
		f.blocked = true
	case graph.PhaseWorking:
		f.working = true
	case graph.PhaseFinished:
		f.finished = true
	}
}

func RootStatus(g graph.WorkGraph) string {
	if len(g.Workers) == 0 {
		return "idle"
	}
	var f phaseFlags
	nonMain := 0
	for _, w := range g.Workers {
		if w.ID == graph.MainWorkerID {
			continue
		}
		nonMain++
		accumulatePhase(&f, w.State)
	}
	if nonMain == 0 {
		if _, main := g.FindWorker(graph.MainWorkerID); main != nil {
			accumulatePhase(&f, main.State)
		}
	}
	if f.error {
		return "error"
	}
	if f.blocked {
		return "blocked"
	}
	if f.working {
		return "working"
	}
	if f.finished || nonMain > 0 {
		return "done"
	}
	return "idle"
}

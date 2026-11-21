package graph

import "time"

type Phase string

const (
	PhaseQueued   Phase = "queued"
	PhaseWorking  Phase = "working"
	PhaseBlocked  Phase = "blocked"
	PhaseFinished Phase = "finished"
	PhaseError    Phase = "error"
	PhaseAborted  Phase = "aborted"
)

const Version = 1

const MainWorkerID = "main"
const RootWorkerID = ""

type PlanItem struct {
	ID     string `json:"id"`
	Text   string `json:"text"`
	Status Phase  `json:"status"`
}

type Worker struct {
	ID       string `json:"id"`
	ParentID string `json:"parent_id,omitempty"`
	Title    string `json:"title"`
	Msg      string `json:"msg,omitempty"`
	State    Phase  `json:"state"`
	Progress *int   `json:"progress,omitempty"`
	Kind     string `json:"kind,omitempty"`
}

type WorkGraph struct {
	Version   int        `json:"version"`
	RunID     string     `json:"run_id"`
	Harness   string     `json:"harness"`
	Title     string     `json:"title,omitempty"`
	Plan      []PlanItem `json:"plan,omitempty"`
	Workers   []Worker   `json:"workers"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func New(harness string) WorkGraph {
	return WorkGraph{
		Version:   Version,
		Harness:   harness,
		Workers:   nil,
		UpdatedAt: time.Now().UTC(),
	}
}
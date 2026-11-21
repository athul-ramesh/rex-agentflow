package graph

import (
	"errors"
	"fmt"
)

func Validate(g *WorkGraph) error {
	if g == nil {
		return errors.New("nil graph")
	}
	if g.Version != Version {
		return fmt.Errorf("unsupported version %d", g.Version)
	}
	seen := make(map[string]bool, len(g.Workers))
	for _, w := range g.Workers {
		if w.ID == "" {
			return errors.New("worker missing id")
		}
		if seen[w.ID] {
			return fmt.Errorf("duplicate worker id %q", w.ID)
		}
		seen[w.ID] = true
		if w.ParentID != "" && !seen[w.ParentID] && w.ParentID != MainWorkerID {
			// parent may appear later; second pass
		}
	}
	for _, w := range g.Workers {
		if w.ParentID == "" {
			continue
		}
		if _, ok := seen[w.ParentID]; !ok && w.ParentID != MainWorkerID {
			return fmt.Errorf("worker %q unknown parent %q", w.ID, w.ParentID)
		}
	}
	return nil
}

func Normalize(g *WorkGraph) {
	if g.Version == 0 {
		g.Version = Version
	}
	if g.Workers == nil {
		g.Workers = []Worker{}
	}
	if g.Diagnostics == nil {
		g.Diagnostics = []Diagnostic{}
	}
}
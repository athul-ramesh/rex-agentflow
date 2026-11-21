package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/athulramesh/rex-agentflow/internal/graph"
)

func StateDir() string {
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return filepath.Join(dir, "agentflow")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("/tmp", "agentflow")
	}
	return filepath.Join(home, ".local", "state", "agentflow")
}

func GraphPath(name string) string {
	return filepath.Join(StateDir(), safeName(name)+".json")
}

func ViewerPath(name string) string {
	return filepath.Join(StateDir(), safeName(name)+".viewer")
}

func safeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "default"
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

func Load(name string) (graph.WorkGraph, error) {
	path := GraphPath(name)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return graph.New(""), nil
	}
	if err != nil {
		return graph.WorkGraph{}, err
	}
	var g graph.WorkGraph
	if err := json.Unmarshal(data, &g); err != nil {
		return graph.WorkGraph{}, fmt.Errorf("decode %s: %w", path, err)
	}
	graph.Normalize(&g)
	return g, nil
}

func Save(name string, g *graph.WorkGraph) error {
	graph.Normalize(g)
	if err := graph.Validate(g); err != nil {
		return err
	}
	dir := StateDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return err
	}
	path := GraphPath(name)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func Clear(name string) error {
	_ = os.Remove(ViewerPath(name))
	return os.Remove(GraphPath(name))
}

func Modify(name string, harness string, fn func(*graph.WorkGraph) error) error {
	g, err := Load(name)
	if err != nil {
		return err
	}
	if g.Harness == "" && harness != "" {
		g.Harness = harness
	}
	if err := fn(&g); err != nil {
		return err
	}
	return Save(name, &g)
}

func FileModTime(name string) (int64, error) {
	st, err := os.Stat(GraphPath(name))
	if err != nil {
		return 0, err
	}
	return st.ModTime().UnixNano(), nil
}
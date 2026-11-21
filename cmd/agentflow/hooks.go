package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/athulramesh/rex-agentflow/internal/graph"
	"github.com/athulramesh/rex-agentflow/internal/identity"
	"github.com/athulramesh/rex-agentflow/internal/ingest/claude"
	"github.com/athulramesh/rex-agentflow/internal/ingest/cursor"
	"github.com/athulramesh/rex-agentflow/internal/ingest/devin"
	"github.com/athulramesh/rex-agentflow/internal/store"
)

func handleCursorHook(graphFlag string, raw []byte) error {
	conv, sess, cwd, command := cursor.Peek(raw)
	payloadID := identity.PayloadID(conv, sess)
	if cursor.IsAgentflowShell(command) && strings.TrimSpace(graphFlag) == "" && payloadID != "" {
		_ = store.EnqueueShell(cwd, payloadID)
	}
	resolved := applyResolveEvent(graphFlag, conv, sess)
	id, err := finalizeResolve(resolved)
	if err != nil {
		return err
	}
	maybeCursorAlias(conv, sess, resolved)
	return routeHook(id, cursor.Harness, func(g *graph.WorkGraph) error {
		return cursor.ApplyHook(g, raw)
	})
}

func handleClaudeHook(graphFlag string, raw []byte) error {
	sess, cwd, command := claude.PeekSession(raw)
	if cursor.IsAgentflowShell(command) && strings.TrimSpace(graphFlag) == "" && sess != "" {
		_ = store.EnqueueShell(cwd, sess)
	}
	resolved := applyResolveEvent(graphFlag, "", sess)
	id, err := finalizeResolve(resolved)
	if err != nil {
		return err
	}
	return routeHook(id, claude.Harness, func(g *graph.WorkGraph) error {
		return claude.ApplyHook(g, raw)
	})
}

func handleDevinHook(graphFlag string, raw []byte) error {
	sess, cwd, command := devin.PeekSession(raw)
	if cursor.IsAgentflowShell(command) && strings.TrimSpace(graphFlag) == "" && sess != "" {
		_ = store.EnqueueShell(cwd, sess)
	}
	resolved := applyResolveEvent(graphFlag, "", sess)
	id, err := finalizeResolve(resolved)
	if err != nil {
		return err
	}
	return routeHook(id, devin.Harness, func(g *graph.WorkGraph) error {
		return devin.ApplyHook(g, raw)
	})
}

func applyResolveEvent(flag, conv, sess string) identity.Resolved {
	return identity.Resolve(resolveEventInput(flag, conv, sess))
}

func routeHook(id string, harness string, fn func(*graph.WorkGraph) error) error {
	id = applyAlias(id)
	var stale bool
	err := store.WithGraph(id, harness, func(g *graph.WorkGraph) error {
		before := len(g.Diagnostics)
		if err := fn(g); err != nil {
			return err
		}
		for i := before; i < len(g.Diagnostics); i++ {
			if g.Diagnostics[i].Code == "stale_session" {
				stale = true
				break
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if !stale {
		cwd, _ := os.Getwd()
		_ = store.RegisterPlace(store.Place{
			SessionID: id,
			RexBlock:  envRexBlock(),
			Cwd:       cwd,
		})
	}
	return nil
}

func maybeCursorAlias(conv, sess string, r identity.Resolved) {
	conv = strings.TrimSpace(conv)
	sess = strings.TrimSpace(sess)
	if conv == "" || sess == "" || conv == sess {
		return
	}
	canonical := conv
	_ = store.WriteAlias(sess, canonical)
	_ = store.WriteAlias(conv, canonical)
	oldExists := fileExists(store.GraphPath(sess))
	newExists := fileExists(store.GraphPath(conv))
	if oldExists && !newExists {
		_ = store.Rekey(sess, conv)
	}
	if oldExists && newExists && r.ID == conv {
		_ = store.WithGraph(conv, cursor.Harness, func(g *graph.WorkGraph) error {
			g.Diagnostics = append(g.Diagnostics, graph.Diagnostic{
				Code: "split_session",
				Msg:  "graph exists for both session_id and conversation_id",
			})
			g.Touch()
			return nil
		})
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func showAmbiguous(ids []string, below bool, ratio int, binary string) error {
	self := binary
	if self == "" {
		var err error
		self, err = os.Executable()
		if err != nil {
			return err
		}
	}
	for _, id := range ids {
		if err := showOne(id, below, ratio, self); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
	return nil
}
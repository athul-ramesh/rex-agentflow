package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/athulramesh/rex-agentflow/internal/identity"
	"github.com/athulramesh/rex-agentflow/internal/store"
)

func envGraph() string {
	return strings.TrimSpace(os.Getenv("AGENTFLOW_GRAPH"))
}

func envRexBlock() string {
	return strings.TrimSpace(os.Getenv("REX_BLOCK"))
}

func applyAlias(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return id
	}
	if c, ok := store.LookupAlias(id); ok {
		return c
	}
	return id
}

func resolveEventInput(flag, conversationID, sessionID string) identity.Input {
	return identity.Input{
		Flag:           flag,
		EnvGraph:       envGraph(),
		ConversationID: conversationID,
		SessionID:      sessionID,
		RexBlock:       envRexBlock(),
		Event:          true,
	}
}

func resolveCLIInput(flag string) identity.Input {
	cwd, _ := os.Getwd()
	queued, _ := store.LatestShell(cwd)
	adopted, _ := store.PlacesMatching(envRexBlock(), cwd)
	return identity.Input{
		Flag:     flag,
		EnvGraph: envGraph(),
		RexBlock: envRexBlock(),
		Cwd:      cwd,
		Queued:   queued,
		Adopted:  adopted,
		Event:    false,
	}
}

func finalizeResolve(r identity.Resolved) (string, error) {
	if r.Ambiguous {
		return "", fmt.Errorf("ambiguous session: %s", strings.Join(r.IDs, " "))
	}
	id := applyAlias(r.ID)
	if err := warnOrStrict(r); err != nil {
		return "", err
	}
	return id, nil
}

func warnOrStrict(r identity.Resolved) error {
	if !r.Fallback() {
		return nil
	}
	if os.Getenv("AGENTFLOW_STRICT") == "1" {
		return fmt.Errorf("no agent session id (set AGENTFLOW_GRAPH, run hooks, or pass --graph)")
	}
	if os.Getenv("AGENTFLOW_QUIET") == "1" {
		return nil
	}
	fmt.Fprintf(os.Stderr, "agentflow: using fallback graph %q (no session id in payload, queue, env, or places)\n", r.ID)
	return nil
}

func registerPlaceForCLI(r identity.Resolved, id string) {
	switch r.Source {
	case identity.SourceFlag, identity.SourceQueue, identity.SourceEnv, identity.SourceAdopted:
		cwd, _ := os.Getwd()
		_ = store.RegisterPlace(store.Place{
			SessionID: id,
			RexBlock:  envRexBlock(),
			Cwd:       cwd,
		})
	}
}
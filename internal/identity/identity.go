package identity

import "strings"

type Source string

const (
	SourceFlag     Source = "flag"
	SourceQueue    Source = "queue"
	SourceEnv      Source = "env"
	SourceAdopted  Source = "adopted"
	SourcePayload  Source = "payload"
	SourceRexBlock Source = "rex_block"
	SourceDefault  Source = "default"
)

type Input struct {
	Flag           string
	EnvGraph       string
	ConversationID string
	SessionID      string
	RexBlock       string
	Cwd            string
	Event          bool // true: event-mode order; false: CLI order

	Queued  string   // LatestShell result; CLI only
	Adopted []string // PlacesMatching raw ids; CLI only
}

type Resolved struct {
	ID        string // raw key; empty when Ambiguous
	Source    Source
	Pinned    bool // --graph on an event command
	Ambiguous bool
	IDs       []string // raw ids when Ambiguous
}

func PayloadID(conversationID, sessionID string) string {
	if c := strings.TrimSpace(conversationID); c != "" {
		return c
	}
	return strings.TrimSpace(sessionID)
}

func (r Resolved) Fallback() bool {
	return r.Source == SourceRexBlock || r.Source == SourceDefault
}

// Resolve picks the graph key. It is pure: callers supply queue and adopt results.
func Resolve(in Input) Resolved {
	flag := strings.TrimSpace(in.Flag)
	env := strings.TrimSpace(in.EnvGraph)
	rex := strings.TrimSpace(in.RexBlock)
	payload := PayloadID(in.ConversationID, in.SessionID)

	if in.Event {
		if flag != "" {
			return Resolved{ID: flag, Source: SourceFlag, Pinned: true}
		}
		if payload != "" {
			return Resolved{ID: payload, Source: SourcePayload}
		}
		if env != "" {
			return Resolved{ID: env, Source: SourceEnv}
		}
		if rex != "" {
			return Resolved{ID: rex, Source: SourceRexBlock}
		}
		return Resolved{ID: "default", Source: SourceDefault}
	}

	// CLI mode
	if flag != "" {
		return Resolved{ID: flag, Source: SourceFlag}
	}
	if q := strings.TrimSpace(in.Queued); q != "" {
		return Resolved{ID: q, Source: SourceQueue}
	}
	if env != "" {
		return Resolved{ID: env, Source: SourceEnv}
	}
	if len(in.Adopted) == 1 {
		return Resolved{ID: in.Adopted[0], Source: SourceAdopted}
	}
	if len(in.Adopted) > 1 {
		return Resolved{ID: "", Source: SourceAdopted, Ambiguous: true, IDs: append([]string(nil), in.Adopted...)}
	}
	if rex != "" {
		return Resolved{ID: rex, Source: SourceRexBlock}
	}
	return Resolved{ID: "default", Source: SourceDefault}
}
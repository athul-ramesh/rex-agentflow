package cursor

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
)

type cursorPeek struct {
	HookEventName  string   `json:"hook_event_name"`
	ConversationID string   `json:"conversation_id"`
	SessionID      string   `json:"session_id"`
	Cwd            string   `json:"cwd"`
	Command        string   `json:"command"`
	WorkspaceRoots []string `json:"workspace_roots"`
}

var agentflowShellRE = regexp.MustCompile(`(^|[;&|]\s*)agentflow(\s|$)`)

// Peek reads session identity fields from hook stdin without full translation.
func Peek(raw []byte) (conversationID, sessionID, cwd, command string) {
	var p cursorPeek
	if json.Unmarshal(raw, &p) != nil {
		return "", "", "", ""
	}
	conversationID = strings.TrimSpace(p.ConversationID)
	sessionID = strings.TrimSpace(p.SessionID)
	cwd = strings.TrimSpace(p.Cwd)
	if cwd == "" && len(p.WorkspaceRoots) > 0 {
		cwd = strings.TrimSpace(p.WorkspaceRoots[0])
	}
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	command = strings.TrimSpace(p.Command)
	return conversationID, sessionID, cwd, command
}

func IsAgentflowShell(command string) bool {
	command = strings.TrimSpace(command)
	if command == "" {
		return false
	}
	if strings.Contains(command, "--graph") {
		return false
	}
	return agentflowShellRE.MatchString(command)
}
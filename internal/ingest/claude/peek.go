package claude

import (
	"encoding/json"
	"os"
	"strings"
)

type claudePeek struct {
	SessionID string `json:"session_id"`
	Cwd       string `json:"cwd"`
	Command   string `json:"command"`
}

func PeekSession(raw []byte) (sessionID, cwd, command string) {
	var p claudePeek
	if json.Unmarshal(raw, &p) != nil {
		return "", "", ""
	}
	sessionID = strings.TrimSpace(p.SessionID)
	cwd = strings.TrimSpace(p.Cwd)
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	command = strings.TrimSpace(p.Command)
	return sessionID, cwd, command
}
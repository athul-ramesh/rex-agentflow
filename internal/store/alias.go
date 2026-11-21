package store

import (
	"os"
	"path/filepath"
	"strings"
)

func idsDir() string {
	return filepath.Join(StateDir(), "ids")
}

func aliasPath(id string) string {
	return filepath.Join(idsDir(), FileKey(id))
}

func LookupAlias(id string) (canonical string, ok bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", false
	}
	data, err := os.ReadFile(aliasPath(id))
	if err != nil {
		return "", false
	}
	c := strings.TrimSpace(string(data))
	if c == "" {
		return "", false
	}
	return c, true
}

func WriteAlias(fromID, canonical string) error {
	fromID = strings.TrimSpace(fromID)
	canonical = strings.TrimSpace(canonical)
	if fromID == "" || canonical == "" {
		return nil
	}
	if err := os.MkdirAll(idsDir(), 0o755); err != nil {
		return err
	}
	return os.WriteFile(aliasPath(fromID), []byte(canonical+"\n"), 0o644)
}

// Rekey moves graph, lock, viewer, and place files from oldID to newID.
func Rekey(oldID, newID string) error {
	oldID = strings.TrimSpace(oldID)
	newID = strings.TrimSpace(newID)
	if oldID == "" || newID == "" || oldID == newID {
		return nil
	}
	if _, err := os.Stat(GraphPath(newID)); err == nil {
		return os.ErrExist
	}
	renames := []struct{ old, neu string }{
		{GraphPath(oldID), GraphPath(newID)},
		{graphLockPath(oldID), graphLockPath(newID)},
		{ViewerPath(oldID), ViewerPath(newID)},
		{placePath(oldID), placePath(newID)},
	}
	for _, r := range renames {
		if _, err := os.Stat(r.old); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}
		if err := os.Rename(r.old, r.neu); err != nil {
			return err
		}
	}
	return nil
}
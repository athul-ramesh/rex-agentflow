package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const shellQueueTTL = 30 * time.Second

type ShellTicket struct {
	SessionID string    `json:"session_id"`
	At        time.Time `json:"at"`
}

func shellQueueDir(cwd string) string {
	return filepath.Join(StateDir(), "shellq", FileKey(cwd))
}

func EnqueueShell(cwd, sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	cwd = strings.TrimSpace(cwd)
	if sessionID == "" || cwd == "" {
		return nil
	}
	dir := shellQueueDir(cwd)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	t := ShellTicket{SessionID: sessionID, At: time.Now().UTC()}
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	name := fmt.Sprintf("%d-%d.json", time.Now().UnixNano(), os.Getpid())
	tmp := filepath.Join(dir, name+".tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, name))
}

func LatestShell(cwd string) (sessionID string, ok bool) {
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		return "", false
	}
	dir := shellQueueDir(cwd)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return "", false
	}
	if err != nil {
		return "", false
	}
	now := time.Now().UTC()
	cutoff := now.Add(-shellQueueTTL)
	type item struct {
		path string
		t    ShellTicket
	}
	var items []item
	for _, ent := range entries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, ent.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var t ShellTicket
		if json.Unmarshal(data, &t) != nil {
			_ = os.Remove(path)
			continue
		}
		if t.At.Before(cutoff) {
			_ = os.Remove(path)
			continue
		}
		items = append(items, item{path: path, t: t})
	}
	if len(items) == 0 {
		return "", false
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].t.At.After(items[j].t.At)
	})
	return strings.TrimSpace(items[0].t.SessionID), items[0].t.SessionID != ""
}
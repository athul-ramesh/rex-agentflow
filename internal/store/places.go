package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Place struct {
	SessionID string    `json:"session_id"`
	RexBlock  string    `json:"rex_block,omitempty"`
	Cwd       string    `json:"cwd,omitempty"`
	At        time.Time `json:"at"`
}

func placesDir() string {
	return filepath.Join(StateDir(), "places")
}

func placePath(id string) string {
	return filepath.Join(placesDir(), FileKey(id)+".json")
}

// FileKey is the on-disk safe name for a raw session or graph id.
func FileKey(id string) string {
	return safeName(id)
}

func RegisterPlace(p Place) error {
	p.SessionID = strings.TrimSpace(p.SessionID)
	if p.SessionID == "" {
		return nil
	}
	if p.At.IsZero() {
		p.At = time.Now().UTC()
	}
	dir := placesDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.%d.tmp", placePath(p.SessionID), os.Getpid())
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, placePath(p.SessionID))
}

func LoadPlace(id string) (Place, error) {
	data, err := os.ReadFile(placePath(id))
	if err != nil {
		return Place{}, err
	}
	var p Place
	if err := json.Unmarshal(data, &p); err != nil {
		return Place{}, err
	}
	return p, nil
}

// PlacesMatching returns raw session ids for places matching rexBlock or cwd rules.
func PlacesMatching(rexBlock, cwd string) ([]string, error) {
	rexBlock = strings.TrimSpace(rexBlock)
	cwd = strings.TrimSpace(cwd)
	entries, err := os.ReadDir(placesDir())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var matches []string
	for _, ent := range entries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(placesDir(), ent.Name()))
		if err != nil {
			continue
		}
		var p Place
		if json.Unmarshal(data, &p) != nil {
			continue
		}
		sid := strings.TrimSpace(p.SessionID)
		if sid == "" {
			continue
		}
		if rexBlock != "" {
			if strings.TrimSpace(p.RexBlock) == rexBlock {
				matches = append(matches, sid)
			}
			continue
		}
		if cwd == "" {
			continue
		}
		if strings.TrimSpace(p.Cwd) == cwd {
			matches = append(matches, sid)
		}
	}
	return matches, nil
}
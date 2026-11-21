package pane

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/athulramesh/rex-agentflow/internal/store"
)

type ShowOptions struct {
	Below  bool
	Ratio  int
	Graph  string
	Binary string
}

func viewerOpen(name string) (string, bool) {
	data, err := os.ReadFile(store.ViewerPath(name))
	if err != nil {
		return "", false
	}
	id := strings.TrimSpace(string(data))
	if id == "" {
		return "", false
	}
	if err := exec.Command("rex", "block", "inspect", id).Run(); err != nil {
		return id, false
	}
	return id, true
}

func viewerCommand(self, graph string) []string {
	// Global flags must precede the subcommand (flag.Parse stops at the first arg).
	view := []string{self, "--graph", graph, "view"}
	var settings []string
	if value := os.Getenv("XDG_STATE_HOME"); value != "" {
		settings = append(settings, "XDG_STATE_HOME="+value)
	}
	if len(settings) == 0 {
		return view
	}
	return slices.Concat([]string{"/usr/bin/env"}, settings, view)
}

func Show(opt ShowOptions) error {
	if id, ok := viewerOpen(opt.Graph); ok {
		fmt.Println(id)
		return nil
	}
	if _, err := exec.LookPath("rex"); err != nil || os.Getenv("REX_SERVER") == "" {
		return errors.New("not running inside Rex; run `agentflow view` in a Kitty terminal")
	}
	self := opt.Binary
	if self == "" {
		var err error
		self, err = os.Executable()
		if err != nil {
			return err
		}
	}
	split := "right"
	if opt.Below {
		split = "down"
	}
	ratio := opt.Ratio
	if ratio <= 0 {
		ratio = 30
	}
	args := []string{"split", "--focus=false", "--split", split, "--ratio", strconv.Itoa(ratio),
		"--shell=none", "--label", "agentflow"}
	if sess := os.Getenv("REX_SESSION"); sess != "" {
		args = append(args, "-s", sess)
	}
	if b := os.Getenv("REX_BLOCK"); b != "" {
		args = append(args, "-b", b)
	}
	args = append(args, "--")
	args = append(args, viewerCommand(self, opt.Graph)...)
	cmd := exec.Command("rex", args...)
	cmd.Stderr = os.Stderr
	outBytes, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("rex split: %w", err)
	}
	lines := strings.Fields(strings.TrimSpace(string(outBytes)))
	if len(lines) == 0 {
		return errors.New("rex split printed no block ID")
	}
	id := lines[len(lines)-1]
	if b := os.Getenv("REX_BLOCK"); b != "" {
		_ = exec.Command("rex", "block", "move", id, split, b, "--ratio", strconv.Itoa(ratio)).Run()
	}
	if err := os.MkdirAll(filepath.Dir(store.ViewerPath(opt.Graph)), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(store.ViewerPath(opt.Graph), []byte(id+"\n"), 0o644); err != nil {
		return err
	}
	fmt.Println(id)
	return nil
}

func Hide(graph string) error {
	id, ok := viewerOpen(graph)
	os.Remove(store.ViewerPath(graph))
	if !ok {
		return nil
	}
	return exec.Command("rex", "block", "close", id).Run()
}
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/athulramesh/rex-agentflow/internal/contract"
	"github.com/athulramesh/rex-agentflow/internal/graph"
	"github.com/athulramesh/rex-agentflow/internal/identity"
	"github.com/athulramesh/rex-agentflow/internal/ingest/cursor"
	"github.com/athulramesh/rex-agentflow/internal/osc7501"
	"github.com/athulramesh/rex-agentflow/internal/pane"
	"github.com/athulramesh/rex-agentflow/internal/reducer"
	"github.com/athulramesh/rex-agentflow/internal/store"
	"github.com/athulramesh/rex-agentflow/internal/view"
)

func main() {
	graphFlag := flag.String("graph", "", "graph name (pin with --graph NAME; default resolves from session id)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: agentflow [--graph NAME] COMMAND\n\n")
		fmt.Fprintf(os.Stderr, "Commands: show, hide, view, get, set, clear, emit-osc, hook, claude-hook, devin-hook, stream, ingest\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(2)
	}
	cmd := flag.Arg(0)
	eventCmd := isEventCmd(cmd)

	var name string
	var resolved identity.Resolved
	if eventCmd {
		// Event commands resolve after reading stdin (hooks) or per-line (ingest/stream).
		if cmd != "hook" && cmd != "claude-hook" && cmd != "devin-hook" && cmd != "ingest" && cmd != "stream" {
			var err error
			resolved = identity.Resolve(resolveEventInput(*graphFlag, "", ""))
			name, err = finalizeResolve(resolved)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
		}
	} else {
		var err error
		resolved = identity.Resolve(resolveCLIInput(*graphFlag))
		if resolved.Ambiguous && cmd != "show" {
			fmt.Fprintf(os.Stderr, "ambiguous session: %s\n", strings.Join(resolved.IDs, " "))
			os.Exit(2)
		}
		name, err = finalizeResolve(resolved)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		registerPlaceForCLI(resolved, name)
	}

	switch cmd {
	case "show":
		if resolved.Ambiguous {
			showFlags := flag.NewFlagSet("show", flag.ExitOnError)
			below := showFlags.Bool("below", false, "split below instead of right")
			ratio := showFlags.Int("ratio", 30, "pane size percent")
			_ = showFlags.Parse(flag.Args()[1:])
			self, _ := os.Executable()
			if err := showAmbiguous(resolved.IDs, *below, *ratio, self); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return
		}
		showFlags := flag.NewFlagSet("show", flag.ExitOnError)
		below := showFlags.Bool("below", false, "split below instead of right")
		ratio := showFlags.Int("ratio", 30, "pane size percent")
		_ = showFlags.Parse(flag.Args()[1:])
		self, _ := os.Executable()
		if err := showOne(name, *below, *ratio, self); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "hide":
		if err := pane.Hide(name); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "view":
		if err := view.Run(name); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "get":
		g, err := store.Load(name)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		getFlags := flag.NewFlagSet("get", flag.ExitOnError)
		asJSON := getFlags.Bool("json", false, "print JSON")
		_ = getFlags.Parse(flag.Args()[1:])
		if *asJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(g)
		} else {
			printHuman(g)
		}
	case "set":
		setFlags := flag.NewFlagSet("set", flag.ExitOnError)
		asJSON := setFlags.Bool("json", false, "replace from JSON stdin")
		_ = setFlags.Parse(flag.Args()[1:])
		var g graph.WorkGraph
		dec := json.NewDecoder(os.Stdin)
		if *asJSON {
			if err := dec.Decode(&g); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		} else {
			if err := dec.Decode(&g); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
		graph.Normalize(&g)
		if err := store.WithGraph(name, g.Harness, func(loaded *graph.WorkGraph) error {
			*loaded = g
			return nil
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "clear":
		if err := store.Clear(name); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "emit-osc":
		g, err := store.Load(name)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := osc7501.WriteGraph(os.Stdout, g); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "hook":
		raw, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := handleCursorHook(*graphFlag, raw); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if os.Getenv("AGENTFLOW_OSC") == "1" {
			conv, sess, _, _ := cursor.Peek(raw)
			r := identity.Resolve(resolveEventInput(*graphFlag, conv, sess))
			id, _ := finalizeResolve(r)
			g, _ := store.Load(id)
			_ = osc7501.WriteGraph(os.Stdout, g)
		}
	case "claude-hook":
		raw, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := handleClaudeHook(*graphFlag, raw); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "devin-hook":
		raw, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := handleDevinHook(*graphFlag, raw); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "stream":
		streamFlags := flag.NewFlagSet("stream", flag.ExitOnError)
		doShow := streamFlags.Bool("show", false, "open viewer pane once if inside Rex")
		_ = streamFlags.Parse(flag.Args()[1:])
		pinned := strings.TrimSpace(*graphFlag)
		fb := identity.Resolve(resolveEventInput(*graphFlag, "", ""))
		fallback, err := finalizeResolve(fb)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if err := cursor.ProcessStreamLocked(pinned, fallback, os.Stdin, func(latched string) error {
			if !*doShow || os.Getenv("REX_SERVER") == "" {
				return nil
			}
			r := identity.Resolve(resolveEventInput("", latched, latched))
			if pinned != "" {
				r = identity.Resolve(resolveEventInput(pinned, "", ""))
			}
			id, err := finalizeResolve(r)
			if err != nil {
				return err
			}
			self, _ := os.Executable()
			return showOne(id, false, 30, self)
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "ingest":
		sc := bufio.NewScanner(os.Stdin)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		var lastPayloadID string
		for sc.Scan() {
			line := bytes.TrimSpace(sc.Bytes())
			if len(line) == 0 {
				continue
			}
			ev, err := contract.Parse(line)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			sid := strings.TrimSpace(ev.SessionID)
			if sid != "" {
				lastPayloadID = sid
			}
			in := resolveEventInput(*graphFlag, "", sid)
			if sid == "" && lastPayloadID != "" {
				in.SessionID = lastPayloadID
			}
			resolved := identity.Resolve(in)
			lineID, err := finalizeResolve(resolved)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
			lineID = applyAlias(lineID)
			if err := store.WithGraph(lineID, ev.Harness, func(g *graph.WorkGraph) error {
				reducer.Apply(g, ev)
				return nil
			}); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
		if err := sc.Err(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", cmd)
		os.Exit(2)
	}
}

func isEventCmd(cmd string) bool {
	switch cmd {
	case "hook", "claude-hook", "devin-hook", "stream", "ingest":
		return true
	default:
		return false
	}
}

func showOne(name string, below bool, ratio int, binary string) error {
	return pane.Show(pane.ShowOptions{Below: below, Ratio: ratio, Graph: name, Binary: binary})
}

func printHuman(g graph.WorkGraph) {
	if g.Title != "" {
		fmt.Println(g.Title)
	}
	for _, it := range g.Plan {
		fmt.Printf("  [%s] %s\n", it.Status, it.Text)
	}
	for _, w := range g.Workers {
		fmt.Printf("  %s %s (%s) %s\n", w.State, w.Title, w.Kind, w.Msg)
	}
}
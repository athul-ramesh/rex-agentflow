package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/athulramesh/rex-agentflow/internal/contract"
	"github.com/athulramesh/rex-agentflow/internal/graph"
	"github.com/athulramesh/rex-agentflow/internal/ingest/cursor"
	"github.com/athulramesh/rex-agentflow/internal/osc7501"
	"github.com/athulramesh/rex-agentflow/internal/pane"
	"github.com/athulramesh/rex-agentflow/internal/reducer"
	"github.com/athulramesh/rex-agentflow/internal/store"
	"github.com/athulramesh/rex-agentflow/internal/view"
)

func main() {
	graphFlag := flag.String("graph", "", "graph name (default: $AGENTFLOW_GRAPH, then $REX_BLOCK, then default)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: agentflow [--graph NAME] COMMAND\n\n")
		fmt.Fprintf(os.Stderr, "Commands: show, hide, view, get, set, clear, emit-osc, hook, stream, ingest\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(2)
	}
	name := resolveGraph(*graphFlag)
	cmd := flag.Arg(0)

	switch cmd {
	case "show":
		showFlags := flag.NewFlagSet("show", flag.ExitOnError)
		below := showFlags.Bool("below", false, "split below instead of right")
		ratio := showFlags.Int("ratio", 30, "pane size percent")
		_ = showFlags.Parse(flag.Args()[1:])
		self, _ := os.Executable()
		if err := pane.Show(pane.ShowOptions{Below: *below, Ratio: *ratio, Graph: name, Binary: self}); err != nil {
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
		getFlags := flag.NewFlagSet("get", flag.ExitOnError)
		asJSON := getFlags.Bool("json", false, "print JSON")
		_ = getFlags.Parse(flag.Args()[1:])
		g, err := store.Load(name)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
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
		if err := store.WithGraph(name, cursor.Harness, func(g *graph.WorkGraph) error {
			return cursor.ApplyHook(g, raw)
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if os.Getenv("AGENTFLOW_OSC") == "1" {
			g, _ := store.Load(name)
			_ = osc7501.WriteGraph(os.Stdout, g)
		}
	case "stream":
		streamFlags := flag.NewFlagSet("stream", flag.ExitOnError)
		doShow := streamFlags.Bool("show", false, "open viewer pane once if inside Rex")
		_ = streamFlags.Parse(flag.Args()[1:])
		shown := false
		if err := cursor.ProcessStreamLocked(name, os.Stdin, func() error {
			if *doShow && !shown && os.Getenv("REX_SERVER") != "" {
				shown = true
				self, _ := os.Executable()
				_ = pane.Show(pane.ShowOptions{Graph: name, Binary: self, Ratio: 30})
			}
			return nil
		}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "ingest":
		sc := bufio.NewScanner(os.Stdin)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
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
			if err := store.WithGraph(name, ev.Harness, func(g *graph.WorkGraph) error {
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

func resolveGraph(flag string) string {
	if flag != "" {
		return flag
	}
	if v := os.Getenv("AGENTFLOW_GRAPH"); v != "" {
		return v
	}
	if v := os.Getenv("REX_BLOCK"); v != "" {
		return v
	}
	return "default"
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
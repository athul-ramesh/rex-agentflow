# rex-agentflow

`agentflow` is a Rexpad-shaped Go CLI that keeps a live **work graph** (plan + subagent workers) in an atomic JSON file, renders it in a Rex sibling pane via Kitty graphics, and maps Cursor agent events to Rex **OSC 7501** program status.

## Install

```bash
go build -o bin/agentflow ./cmd/agentflow
# optional: ln -s "$PWD/bin/agentflow" ~/bin/agentflow
```

State files live at `$XDG_STATE_HOME/agentflow/<name>.json` (default `~/.local/state/agentflow/`). The graph **name** defaults to `$AGENTFLOW_GRAPH`, then `$REX_BLOCK`, then `default`. Override with `--graph NAME`.

## Cursor hooks

Copy the example into your project or user hooks config:

```bash
cp hooks/cursor-hooks.json.example /path/to/project/.cursor/hooks.json
```

Ensure `agentflow` is on `PATH` inside the environment where hooks run. Each hook reads one JSON event from stdin, updates the graph, and exits 0.

To also print OSC sequences on hook stdout (for a wrapper that forwards to the agent TTY):

```bash
export AGENTFLOW_OSC=1
```

A wrapper can instead run `agentflow emit-osc` in the agent pane after hooks update the graph.

## Rex viewer

From a Rex terminal block:

```bash
agentflow show          # rex split right ~30%, runs agentflow view
agentflow hide          # close tracked viewer pane
agentflow view          # Kitty poll loop (~150ms) in the current terminal
```

`show` is idempotent; viewer block id is stored in `<name>.viewer` beside the graph JSON.

## CLI

| Command | Purpose |
|--------|---------|
| `get [--json]` | Print current graph |
| `set [--json]` | Replace graph from stdin |
| `clear` | Remove graph (+ viewer sidecar) |
| `emit-osc` | Print OSC 7501 for root + workers to stdout |
| `hook` | Ingest one Cursor hook JSON line |
| `stream [--show]` | Ingest Cursor `stream-json` NDJSON until EOF |

## Stream mode

Pipe Cursor CLI output:

```bash
cursor-agent -p --output-format stream-json "..." | agentflow stream --show
```

## Manual check

```bash
echo '{"hook_event_name":"subagentStart","subagent_id":"abc","subagent_type":"explore","task":"map APIs"}' | agentflow --graph demo hook
agentflow --graph demo get --json
agentflow --graph demo emit-osc | od -c | head
```

## Tests

```bash
go test ./...
```
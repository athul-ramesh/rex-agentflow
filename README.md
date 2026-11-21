# rex-agentflow

`agentflow` keeps a live **work graph** (plan + subagent workers) in an atomic JSON file, draws it in a Rex sibling pane with the Kitty graphics protocol, and maps agent lifecycle events to Rex **OSC 7501** program status. It is shaped like [rexpad](https://github.com/almonk/rexpad): hooks update the graph, the agent opens the pane.

Provider adapters (Cursor, Pi, Claude Code, Devin CLI) translate native payloads into a versioned NDJSON event contract. One reducer applies those events under a per-graph file lock.

## Requirements

- Go 1.22+
- [Rex](https://superlogical.com) (for `agentflow show`)
- A terminal that speaks the Kitty graphics protocol (Rex does)
- Optional: SF Pro Text or Inter for the same type as rexpad (falls back to Go’s bundled fonts)

## Install

```bash
git clone https://github.com/athul-ramesh/rex-agentflow.git
cd rex-agentflow
go build -o bin/agentflow ./cmd/agentflow
ln -sf "$PWD/bin/agentflow" ~/.local/bin/agentflow   # must be on PATH
```

Confirm:

```bash
agentflow --help
```

State lives at `$XDG_STATE_HOME/agentflow/<name>.json` (default `~/.local/state/agentflow/`). The graph name is `$AGENTFLOW_GRAPH`, else `$REX_BLOCK`, else `default`. Override with `--graph NAME`.

## One-time setup

### 1. Cursor hooks (ingest Task / subagent events)

User-wide (every project):

```bash
cp hooks/cursor-hooks.json.example ~/.cursor/hooks.json
```

Or one project only:

```bash
cp hooks/cursor-hooks.json.example /path/to/project/.cursor/hooks.json
```

Restart Cursor (or start a new agent) so hooks load. Each hook runs `agentflow hook`, reads one JSON event from stdin, updates the graph, and exits 0.

### 2. Agent skill (so the agent opens the pane)

```bash
mkdir -p ~/.cursor/skills
ln -sfn "$PWD/skills/agentflow" ~/.cursor/skills/agentflow

# Optional Codex mirror (same pattern as rexpad):
mkdir -p ~/.codex/skills
ln -sfn ~/.cursor/skills/agentflow ~/.codex/skills/agentflow
```

In a Rex terminal, for multi-step or multi-agent work the agent should run:

```bash
agentflow show
```

Hooks keep workers current after that. You do not need to call `hook` by hand.

## Quick start (Rex)

1. Open a Rex tab in a repo that has hooks (or with user hooks installed).
2. Start Cursor Agent (`cursor-agent` / `agent`) in that pane.
3. Ask for multi-step work that fans out with Task / subagents.
4. The agent runs `agentflow show`. A sibling pane appears (~30% width) and updates as hooks fire.

Useful commands in the agent pane:

```bash
agentflow show    # open / reuse the viewer (idempotent)
agentflow hide    # close the tracked viewer
agentflow get --json
agentflow clear   # wipe graph (+ viewer sidecar); safe if already empty
agentflow view    # run the Kitty loop in the current terminal
```

### Cursor CLI stream (no IDE hooks)

```bash
cursor-agent -f --trust -p --output-format stream-json "your prompt" \
  | agentflow stream --show
```

`--show` opens the Rex viewer once, then updates from NDJSON until EOF.

## Pi agent (optional)

The TypeScript extension emits native contract events on `agentflow ingest`:

```bash
ln -sf "$PWD/pi/agentflow.ts" ~/.pi/agent/extensions/agentflow.ts
mkdir -p ~/.pi/agent/skills
ln -sfn ~/.cursor/skills/agentflow ~/.pi/agent/skills/agentflow
```

- `AGENTFLOW_BIN` overrides the binary (default `~/.local/bin/agentflow`).
- When set, `AGENTFLOW_GRAPH` is passed as `--graph` on every ingest call.
- TUI sessions always track. Headless / RPC / print modes track when `REX_BLOCK`, `AGENTFLOW_GRAPH`, or `AGENTFLOW_ENABLE=1` is set.
- Hardened Pi hosts need `agentflow` on PATH and FS grants for `~/.local/bin`, this repo’s `bin/`, and `~/.local/state/agentflow`. Restart Pi after changing those.
- Sync delegation: `tool_call` → `worker.start`, `tool_result` → `worker.stop`. Background launch (`async: true`) emits `tool.return` on launch completion and leaves the worker working.

## Claude Code (optional)

```bash
cp hooks/claude-hooks.json.example ~/.claude/settings.json   # merge into your settings
```

Wire hooks to `agentflow claude-hook`. Keep `AGENTFLOW_OSC` unset so Claude can use hook stdout for decision JSON. Push Rex status with `agentflow emit-osc` from the agent pane when needed. Correlation uses `session_id` and `agent_id`.

## Devin CLI (optional)

```bash
cp hooks/devin-hooks.json.example .devin/hooks.v1.json   # adjust paths as needed
```

Wire hooks to `agentflow devin-hook`. Session-level support uses `session_id` and `prompt_id`. Worker-level subagent completion is not claimed yet. Devin may import Claude hooks by default. Prefer a Devin-specific hooks file so the two adapters do not fight. Keep `AGENTFLOW_OSC` unset for the same stdout reason as Claude.

## Event contract

Any process can publish newline-delimited JSON to `agentflow ingest`:

```bash
echo '{"v":1,"type":"worker.start","harness":"custom","session_id":"s1","worker":{"id":"w1","kind":"explore","title":"map APIs","state":"working"}}' \
  | agentflow --graph demo ingest
```

Event types: `session.bind`, `worker.start`, `worker.patch`, `worker.stop`, `tool.return`, `turn.settle`, `session.end`. `tool.return` means the tool call finished. It is not the same as a long-running worker finishing (`worker.stop`).

## CLI

| Command | Purpose |
|--------|---------|
| `show` | Split a Rex pane and run the Kitty viewer |
| `hide` | Close the tracked viewer pane |
| `view` | Kitty poll loop in the current terminal |
| `get [--json]` | Print the current graph |
| `set [--json]` | Replace the graph from stdin |
| `clear` | Remove graph (+ viewer sidecar) |
| `emit-osc` | Print OSC 7501 for root + workers |
| `hook` | Ingest one Cursor hook JSON object from stdin |
| `claude-hook` | Ingest one Claude Code hook JSON object from stdin |
| `devin-hook` | Ingest one Devin CLI hook JSON object from stdin |
| `stream [--show]` | Ingest Cursor `stream-json` NDJSON until EOF |
| `ingest` | Ingest NDJSON contract events until EOF |

Global flag: `--graph NAME` (before the subcommand).

### OSC on Cursor hook stdout

```bash
export AGENTFLOW_OSC=1
```

Use this only with `agentflow hook` (Cursor). Claude and Devin hooks must leave stdout free. Or run `agentflow emit-osc` in the agent pane after hooks update the graph.

## Manual check

```bash
echo '{"hook_event_name":"subagentStart","subagent_id":"abc","subagent_type":"explore","task":"map APIs"}' \
  | agentflow --graph demo hook
agentflow --graph demo get --json
agentflow --graph demo show    # from a Rex pane
```

## Display notes

- Layout is in points; the viewer picks a device-pixel scale from cell height (Retina ≈ 2×), matching rexpad.
- Override with `AGENTFLOW_SCALE=2` if needed.
- Fonts: `AGENTFLOW_FONT` / `AGENTFLOW_FONT_BOLD`, else SF Pro / Inter / Go fonts.
- Long Task prompts show as short kind labels (`explore`, `scout`) with the prompt on the muted subline.

## Tests

```bash
go test ./...
```

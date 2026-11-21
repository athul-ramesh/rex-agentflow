---
name: agentflow
description: Keep a live work graph of the plan and subagent workers in a Rex sibling pane, drawn with the kitty image protocol, and map Cursor agent events to Rex OSC 7501 status. Use at the start of any multi-step or multi-agent task in a Rex terminal (three or more steps, Task/subagent fan-out, or when the user says "show your plan", "work graph", "show progress", "what are you working on"), and keep the graph updated as you work. Drive it with the `agentflow` CLI. Cursor user hooks already ingest subagent/Task events when installed at ~/.cursor/hooks.json; Pi ingests the same events when `~/.pi/agent/extensions/agentflow.ts` is installed (symlink from rex-agentflow).
---

# agentflow: a live work graph in Rex

`agentflow` draws the current plan and worker list as a picture in a split pane
next to your own pane. Hooks update the graph from Cursor agent events. You
open the pane and seed or edit the plan with the CLI.

Each **agent session** gets its own graph file (Cursor: `conversation_id` else
`session_id`; other harnesses: `session_id`). Two agents in the same Rex pane
stay on two files and two viewer sidecars. `$REX_BLOCK` is only the Rex split
anchor, not the graph name. Run `agentflow show` with no flag; hooks and Pi set
`AGENTFLOW_GRAPH`, and Cursor's `beforeShellExecution` hook fills the shell
queue so CLI commands match hook routing. Pin with `--graph NAME`. Use
`AGENTFLOW_STRICT=1` if missing session ids should fail instead of using the
fallback file.

## Start of a task

```bash
agentflow show
```

`show` opens the pane without taking focus and is idempotent. Call it once at
the start of multi-step or multi-agent work in Rex.

If you need a fresh graph:

```bash
agentflow clear
agentflow show
```

User-level Cursor hooks (`~/.cursor/hooks.json`) already call `agentflow hook`
on `subagentStart`, `subagentStop`, Task tool use, and `stop`. You do not need
to replay those events by hand.

On Pi, the `agentflow` extension (`~/.pi/agent/extensions/agentflow.ts`) pipes
equivalent JSON into `agentflow hook` when `subagent` or Task tools run. Install
from this repo; restart Pi or start a new session after symlinking. Skill
discovery: `~/.pi/agent/skills/agentflow` → `~/.cursor/skills/agentflow`.

## While you work

- Prefer the hooks for worker lifecycle. Do not invent parallel bookkeeping.
- After you change the plan yourself, run `agentflow get --json` if you lose
  track, edit, then `agentflow set --json`.
- Keep the pane open. `agentflow hide` only when the user asks, or when you
  start unrelated work (then `show` again for the new plan).

## End of a task

Leave the pane open so the user can review. Clear only if they ask:

```bash
agentflow clear
```

## Other details

- State files live under `$XDG_STATE_HOME/agentflow/` (default
  `~/.local/state/agentflow/`).
- `agentflow emit-osc` prints OSC 7501 for root + workers. Set
  `AGENTFLOW_OSC=1` if a wrapper should also print OSC on hook stdout.
- Outside Rex, `show` fails. Run `agentflow view` in a Kitty terminal instead,
  or skip the pane and rely on hooks + `get`.

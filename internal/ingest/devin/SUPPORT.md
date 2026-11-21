# Devin CLI hook adapter

Harness id: `devin-hooks`

Configure Devin with `hooks/devin-hooks.json.example` (copy to `.devin/hooks.v1.json`). Wire the command as `agentflow devin-hook` once the CLI subcommand exists; stdin is the hook JSON documented at [Devin lifecycle hooks](https://docs.devin.ai/cli/extensibility/hooks/lifecycle-hooks).

## Support

| Level | Status |
| --- | --- |
| **Session-level** | **Yes** — `SessionStart`, `UserPromptSubmit`, `Stop`, and `SessionEnd` drive `session.bind`, main worker activity, and `session.end` using `session_id` and `prompt_id`. |
| **Worker-level** | **Not claimed** — Devin's documented hook payloads do not expose reliable subagent identity or completion for `run_subagent` / `read_subagent`. Pre/Post tool hooks are ignored for the work graph unless a future capture shows stable ids. |

## Event mapping

| Devin hook | Contract events |
| --- | --- |
| `SessionStart` | `session.bind` (`session_id`; title from `source`) |
| `UserPromptSubmit` | `session.bind` + `worker.start` on `main` (`session_id`, `prompt_id` as turn/run id on events) |
| `Stop` | `session.end` |
| `SessionEnd` | `session.end` (`reason`) |
| `PermissionRequest`, `PreToolUse`, `PostToolUse`, `PostCompaction` | No graph update (stdout reserved for Devin hook decisions) |

Devin may still load `.claude/` hooks in parallel when `read_config_from.claude` is enabled; keep agentflow on Devin-specific hooks to avoid double-ingest.
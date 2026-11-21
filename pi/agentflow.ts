// @ts-nocheck

import { spawn } from "node:child_process";
import { homedir } from "node:os";
import { join } from "node:path";

const CONTRACT_VERSION = 1;
const HARNESS = "pi";

const IGNORED_TOOLS = new Set([
	"bash",
	"read",
	"edit",
	"write",
	"grep",
	"find",
	"ls",
	"glob",
	"bg_wait",
	"contact_supervisor",
	"subagent_supervisor",
	"subagents_enable",
]);

function resolveAgentflowBin(): string {
	const fromEnv = process.env.AGENTFLOW_BIN;
	if (typeof fromEnv === "string" && fromEnv.trim()) {
		return fromEnv.trim();
	}
	return join(homedir(), ".local", "bin", "agentflow");
}

function ingestArgs(): string[] {
	const args = ["ingest"];
	const graph = process.env.AGENTFLOW_GRAPH;
	if (typeof graph === "string" && graph.trim()) {
		args.unshift("--graph", graph.trim());
	}
	return args;
}

/** Fire-and-forget: never block Pi tool execution on agentflow. */
function emitContract(payload: Record<string, unknown>): void {
	const line = JSON.stringify({ v: CONTRACT_VERSION, harness: HARNESS, ...payload });
	const bin = resolveAgentflowBin();
	const child = spawn(bin, ingestArgs(), {
		stdio: ["pipe", "ignore", "ignore"],
		env: process.env,
	});
	child.stdin.write(`${line}\n`);
	child.stdin.end();
	child.unref?.();
	child.on("error", () => {
		/* fail open */
	});
}

function isWorkerTool(toolName: string): boolean {
	if (!toolName || IGNORED_TOOLS.has(toolName)) {
		return false;
	}
	if (toolName === "Task" || toolName === "task") {
		return true;
	}
	if (toolName === "subagent" || toolName === "subagent_launch") {
		return true;
	}
	return /^subagent/i.test(toolName);
}

function isDelegationLaunch(toolName: string, input: Record<string, unknown>): boolean {
	if (!isWorkerTool(toolName)) {
		return false;
	}
	if (toolName === "subagent" || toolName === "subagent_launch") {
		const action = typeof input.action === "string" ? input.action.trim() : "";
		const hasAgent = typeof input.agent === "string" && input.agent.trim().length > 0;
		const hasTask = typeof input.task === "string" && input.task.trim().length > 0;
		const workflow = input.workflow;
		const hasWorkflow =
			workflow === true ||
			(typeof workflow === "string" && workflow.trim().length > 0);
		if (hasAgent || hasTask || hasWorkflow) {
			return true;
		}
		if (action) {
			return false;
		}
		return hasAgent || hasTask || hasWorkflow;
	}
	return true;
}

/**
 * Background/async launches: the launch tool's `tool_result` arrives while the child still runs.
 * Pi-subagents exposes this via `async: true` on `subagent` / `subagent_launch` input.
 * Workflow hosts (`workflow: true`) run children inside the parent; treat them as synchronous unless `async` is set.
 */
function isBackgroundLaunch(toolName: string, input: Record<string, unknown>): boolean {
	if (!isDelegationLaunch(toolName, input)) {
		return false;
	}
	if (input.async === true) {
		return true;
	}
	if (toolName === "subagent" || toolName === "subagent_launch") {
		const action = typeof input.action === "string" ? input.action.trim().toLowerCase() : "";
		if (action === "spawn" || action === "runs.run") {
			return input.async !== false;
		}
	}
	return false;
}

function workerKind(toolName: string, input: Record<string, unknown>): string {
	if (toolName === "Task" || toolName === "task") {
		return "task";
	}
	const agent = input.agent;
	if (typeof agent === "string" && agent.trim()) {
		return agent.trim();
	}
	return "subagent";
}

function workerTitle(toolName: string, input: Record<string, unknown>): string {
	const task = input.task;
	if (typeof task === "string" && task.trim()) {
		return task.trim().slice(0, 200);
	}
	const workflow = input.workflow;
	if (typeof workflow === "string" && workflow.trim()) {
		return `workflow: ${workflow.trim().slice(0, 160)}`;
	}
	if (workflow === true) {
		return "workflow";
	}
	if (typeof input.agent === "string" && input.agent.trim()) {
		return input.agent.trim();
	}
	if (toolName === "Task" || toolName === "task") {
		return "Task";
	}
	return "subagent";
}

function workerRef(event, input: Record<string, unknown>) {
	const id = event.toolCallId;
	const ref = {
		id,
		tool_call_id: id,
		kind: workerKind(event.toolName, input),
		title: workerTitle(event.toolName, input),
	};
	const parent =
		typeof event.parentToolCallId === "string" && event.parentToolCallId.trim()
			? event.parentToolCallId.trim()
			: "";
	if (parent) {
		ref.parent_id = parent;
	}
	return ref;
}

function firstTextLine(content: unknown[]): string {
	for (const part of content) {
		if (part && typeof part === "object" && part.type === "text") {
			const text = part.text;
			if (typeof text === "string") {
				const line = text.split(/\r?\n/).find((l) => l.trim().length > 0);
				if (line) {
					return line.trim().slice(0, 120);
				}
			}
		}
	}
	return "";
}

function shouldTrack(ctx): boolean {
	if (ctx?.mode === "tui") {
		return true;
	}
	if (process.env.AGENTFLOW_ENABLE === "1") {
		return true;
	}
	if (typeof process.env.AGENTFLOW_GRAPH === "string" && process.env.AGENTFLOW_GRAPH.trim()) {
		return true;
	}
	if (typeof process.env.REX_BLOCK === "string" && process.env.REX_BLOCK.trim()) {
		return true;
	}
	return false;
}

function sessionIdFrom(ctx): string {
	const sm = ctx?.sessionManager;
	const getId = sm?.getSessionId;
	if (typeof getId !== "function") {
		return "";
	}
	let id;
	try {
		id = getId.call(sm);
	} catch {
		return "";
	}
	if (typeof id === "string" && id.trim()) {
		return id.trim();
	}
	return "";
}

export default function (pi) {
	let rootSession = false;
	let boundSessionId = "";
	const backgroundLaunches = new Set();

	pi.on("session_start", (_event, ctx) => {
		rootSession = shouldTrack(ctx);
		if (!rootSession) {
			return;
		}
		const sid = sessionIdFrom(ctx);
		if (sid) {
			boundSessionId = sid;
			emitContract({ type: "session.bind", session_id: sid });
		}
	});

	pi.on("tool_call", (event) => {
		if (!rootSession) {
			return;
		}
		const input = event.input ?? {};
		if (!isDelegationLaunch(event.toolName, input)) {
			return;
		}
		if (isBackgroundLaunch(event.toolName, input)) {
			backgroundLaunches.add(event.toolCallId);
		}

		emitContract({
			type: "worker.start",
			session_id: boundSessionId || undefined,
			worker: { ...workerRef(event, input), state: "working" },
		});
	});

	pi.on("tool_result", (event) => {
		if (!rootSession) {
			return;
		}
		const input = event.input ?? {};
		if (!isWorkerTool(event.toolName)) {
			return;
		}
		if (event.toolName === "subagent" || event.toolName === "subagent_launch") {
			if (!isDelegationLaunch(event.toolName, input)) {
				return;
			}
		}

		const summary = firstTextLine(event.content ?? []);
		const status = event.isError ? "error" : "finished";
		const base = {
			session_id: boundSessionId || undefined,
			status,
			msg: summary || undefined,
			worker: workerRef(event, input),
		};

		if (backgroundLaunches.has(event.toolCallId)) {
			backgroundLaunches.delete(event.toolCallId);
			emitContract({ type: "tool.return", ...base });
			return;
		}

		emitContract({ type: "worker.stop", ...base });
	});

	const emitSessionEnd = () => {
		if (!rootSession) {
			return;
		}
		emitContract({
			type: "session.end",
			session_id: boundSessionId || undefined,
		});
	};

	pi.on("agent_settled", emitSessionEnd);
	pi.on("agent_end", (_event, ctx) => {
		if (ctx?.isIdle?.() === true) {
			emitSessionEnd();
		}
	});
	pi.on("session_shutdown", emitSessionEnd);
}
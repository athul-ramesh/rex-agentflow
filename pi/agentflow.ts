// Pi extension: pipe subagent/tool lifecycle into `agentflow hook` (Cursor-shaped JSON).
// @ts-nocheck

import { spawn } from "node:child_process";
import { homedir } from "node:os";
import { join } from "node:path";

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
	// Absolute path so host-side hook spawn does not depend on Pi's PATH.
	return join(homedir(), ".local", "bin", "agentflow");
}

function hookArgs(): string[] {
	const args = ["hook"];
	const graph = process.env.AGENTFLOW_GRAPH;
	if (typeof graph === "string" && graph.trim()) {
		args.unshift("--graph", graph.trim());
	}
	return args;
}

/** Fire-and-forget: never block Pi tool execution on agentflow. */
function emitHook(payload: Record<string, unknown>): void {
	let json: string;
	try {
		json = JSON.stringify(payload);
	} catch {
		return;
	}

	const bin = resolveAgentflowBin();
	const child = spawn(bin, hookArgs(), {
		stdio: ["pipe", "ignore", "ignore"],
		env: process.env,
	});
	child.stdin.write(json);
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
		// Management-only subagent calls (status, guide, validate, …) are not workers.
		if (action) {
			return false;
		}
		return hasAgent || hasTask || hasWorkflow;
	}
	return true;
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
	// Headless / RPC still feed the graph when a graph or Rex block is named.
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

export default function (pi) {
	let rootSession = false;

	pi.on("session_start", (_event, ctx) => {
		rootSession = shouldTrack(ctx);
	});

	pi.on("tool_call", (event) => {
		if (!rootSession) {
			return;
		}
		const input = event.input ?? {};
		if (!isDelegationLaunch(event.toolName, input)) {
			return;
		}

		emitHook({
			hook_event_name: "subagentStart",
			subagent_id: event.toolCallId,
			tool_call_id: event.toolCallId,
			subagent_type: workerKind(event.toolName, input),
			task: workerTitle(event.toolName, input),
			tool_name: event.toolName,
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
		emitHook({
			hook_event_name: "subagentStop",
			subagent_id: event.toolCallId,
			tool_call_id: event.toolCallId,
			status: event.isError ? "error" : "finished",
			summary,
			tool_name: event.toolName,
		});
	});

	const emitAgentStop = () => {
		if (!rootSession) {
			return;
		}
		emitHook({ hook_event_name: "stop" });
	};

	pi.on("agent_settled", emitAgentStop);
	pi.on("agent_end", (_event, ctx) => {
		if (ctx?.isIdle?.() === true) {
			emitAgentStop();
		}
	});
	pi.on("session_shutdown", emitAgentStop);
}
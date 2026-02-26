#!/usr/bin/env node

import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
} from "@modelcontextprotocol/sdk/types.js";
import { AgentHubClient } from "./api-client";

// Read configuration from environment variables.
const AGENTHUB_SERVER_URL =
  process.env.AGENTHUB_SERVER_URL || "http://localhost:8080";
const AGENTHUB_TOKEN = process.env.AGENTHUB_TOKEN || "";
const AGENTHUB_WORKSPACE_ID = process.env.AGENTHUB_WORKSPACE_ID || "";
const AGENTHUB_AGENT_ID = process.env.AGENTHUB_AGENT_ID || "";

let client: AgentHubClient | null = null;

function getClient(): AgentHubClient {
  if (!client) {
    if (!AGENTHUB_TOKEN) {
      throw new Error(
        "AGENTHUB_TOKEN is required. Set it in your environment or MCP config."
      );
    }
    if (!AGENTHUB_WORKSPACE_ID) {
      throw new Error(
        "AGENTHUB_WORKSPACE_ID is required. Set it in your environment or MCP config."
      );
    }
    client = new AgentHubClient({
      serverUrl: AGENTHUB_SERVER_URL,
      token: AGENTHUB_TOKEN,
      workspaceId: AGENTHUB_WORKSPACE_ID,
      agentId: AGENTHUB_AGENT_ID,
    });
  }
  return client;
}

function textResult(text: string, isError = false) {
  return { content: [{ type: "text" as const, text }], isError };
}

function jsonResult(obj: unknown) {
  return textResult(JSON.stringify(obj, null, 2));
}

const server = new Server(
  { name: "agenthub", version: "1.0.0" },
  { capabilities: { tools: {} } }
);

// --- List Tools ---
server.setRequestHandler(ListToolsRequestSchema, async () => ({
  tools: [
    {
      name: "claim_next_task",
      description:
        "Find and claim the highest-priority unclaimed task in the workspace. " +
        "Optionally filter by tags or priority. Returns the claimed task details.",
      inputSchema: {
        type: "object" as const,
        properties: {
          priority: {
            type: "number",
            minimum: 1,
            maximum: 5,
            description: "Only consider tasks at this priority level (1-5)",
          },
          tags: {
            type: "string",
            description: "Comma-separated tags to filter tasks by",
          },
        },
      },
    },
    {
      name: "complete_task_with_summary",
      description:
        "Mark a task as completed and attach a summary of what was accomplished. " +
        "The summary is stored in task metadata and a status_update message is broadcast.",
      inputSchema: {
        type: "object" as const,
        properties: {
          task_id: {
            type: "string",
            description: "The UUID of the task to complete",
          },
          summary: {
            type: "string",
            description:
              "A brief summary of the work done (e.g., 'Implemented user login with JWT auth')",
          },
          artifacts: {
            type: "array",
            items: { type: "string" },
            description:
              "List of artifact IDs or file paths produced while completing the task",
          },
        },
        required: ["task_id", "summary"],
      },
    },
    {
      name: "update_progress",
      description:
        "Update the progress of an in-progress task. Sets percentage complete, " +
        "updates status if needed, and broadcasts a status_update message to the workspace.",
      inputSchema: {
        type: "object" as const,
        properties: {
          task_id: {
            type: "string",
            description: "The UUID of the task to update progress on",
          },
          percent_complete: {
            type: "number",
            minimum: 0,
            maximum: 100,
            description: "Percentage of task completion (0-100)",
          },
          status_message: {
            type: "string",
            description:
              "Brief description of current progress (e.g., 'Writing unit tests')",
          },
          status: {
            type: "string",
            enum: ["assigned", "in_progress", "review", "blocked"],
            description: "Optionally update the task status",
          },
          blocked_reason: {
            type: "string",
            description: "If status is 'blocked', provide the reason",
          },
        },
        required: ["task_id", "percent_complete", "status_message"],
      },
    },
    {
      name: "generate_daily_summary",
      description:
        "Generate a daily summary report for the workspace. Gathers metrics including " +
        "tasks completed, tasks created, blocked tasks, and active agents. " +
        "Returns a structured report saved to the daily_reports table.",
      inputSchema: {
        type: "object" as const,
        properties: {},
      },
    },
  ],
}));

// --- Call Tool ---
server.setRequestHandler(CallToolRequestSchema, async (request) => {
  const { name, arguments: args } = request.params;

  switch (name) {
    case "claim_next_task":
      return handleClaimNextTask(args);
    case "complete_task_with_summary":
      return handleCompleteTaskWithSummary(args);
    case "update_progress":
      return handleUpdateProgress(args);
    case "generate_daily_summary":
      return handleGenerateDailySummary();
    default:
      return textResult(`Unknown tool: ${name}`, true);
  }
});

// --- Tool Handlers ---

async function handleClaimNextTask(args: Record<string, unknown> | undefined) {
  try {
    const api = getClient();
    const priority = args?.priority as number | undefined;
    const tags = args?.tags as string | undefined;

    const filters: Record<string, any> = { status: "pending" };
    if (priority) filters.priority = priority;

    const tasks = await api.listTasks(filters);

    if (!tasks || tasks.length === 0) {
      return textResult("No unclaimed tasks available matching the criteria.");
    }

    let candidates = tasks;
    if (tags) {
      const tagList = tags
        .split(",")
        .map((t: string) => t.trim().toLowerCase());
      candidates = tasks.filter((task: any) =>
        task.tags?.some((t: string) => tagList.includes(t.toLowerCase()))
      );
      if (candidates.length === 0) {
        return textResult(`No unclaimed tasks found with tags: ${tags}`);
      }
    }

    const target = candidates[0];
    const claimed = await api.claimTask(target.id);

    return jsonResult({
      message: "Task claimed successfully",
      task: {
        id: claimed.id,
        title: claimed.title,
        description: claimed.description,
        priority: claimed.priority,
        status: claimed.status,
        tags: claimed.tags,
        depends_on: claimed.depends_on,
        branch_name: claimed.branch_name,
        metadata: claimed.metadata,
      },
    });
  } catch (err: any) {
    return textResult(`Error claiming task: ${err.message || err}`, true);
  }
}

async function handleCompleteTaskWithSummary(
  args: Record<string, unknown> | undefined
) {
  try {
    const api = getClient();
    const taskId = args?.task_id as string;
    const summary = args?.summary as string;
    const artifacts = args?.artifacts as string[] | undefined;

    if (!taskId || !summary) {
      return textResult("task_id and summary are required", true);
    }

    const metadata: Record<string, any> = {
      completion_summary: summary,
      completed_by: api.agentId || "mcp-agent",
    };
    if (artifacts && artifacts.length > 0) {
      metadata.artifacts = artifacts;
    }

    const task = await api.completeTask(taskId, summary, metadata);

    // Broadcast a status update.
    try {
      await api.sendMessage({
        message_type: "status_update",
        payload: {
          text: `Task completed: ${task.title} - ${summary}`,
          task_id: task.id,
        },
      });
    } catch {
      // Non-critical.
    }

    return jsonResult({
      message: "Task completed successfully",
      task: {
        id: task.id,
        title: task.title,
        status: task.status,
        completed_at: task.completed_at,
        summary,
      },
    });
  } catch (err: any) {
    return textResult(`Error completing task: ${err.message || err}`, true);
  }
}

async function handleUpdateProgress(
  args: Record<string, unknown> | undefined
) {
  try {
    const api = getClient();
    const taskId = args?.task_id as string;
    const percentComplete = args?.percent_complete as number;
    const statusMessage = args?.status_message as string;
    const status = args?.status as string | undefined;
    const blockedReason = args?.blocked_reason as string | undefined;

    if (!taskId || percentComplete === undefined || !statusMessage) {
      return textResult(
        "task_id, percent_complete, and status_message are required",
        true
      );
    }

    const updates: Record<string, any> = {
      metadata: {
        percent_complete: percentComplete,
        last_progress_update: new Date().toISOString(),
        progress_message: statusMessage,
      },
    };

    if (status) {
      updates.status = status;
    }
    if (blockedReason && status === "blocked") {
      updates.metadata.blocked_reason = blockedReason;
    }

    const task = await api.updateTask(taskId, updates);

    // Broadcast progress update.
    try {
      await api.sendMessage({
        message_type: "status_update",
        payload: {
          text: `[${percentComplete}%] ${task.title}: ${statusMessage}`,
          task_id: task.id,
          percent_complete: percentComplete,
        },
      });
    } catch {
      // Non-critical.
    }

    return jsonResult({
      message: "Progress updated",
      task: {
        id: task.id,
        title: task.title,
        status: task.status,
        percent_complete: percentComplete,
        progress_message: statusMessage,
      },
    });
  } catch (err: any) {
    return textResult(`Error updating progress: ${err.message || err}`, true);
  }
}

async function handleGenerateDailySummary() {
  try {
    const api = getClient();
    const report = await api.generateDailySummary();

    return jsonResult({
      message: "Daily summary generated",
      report: {
        id: report.id,
        report_date: report.report_date,
        summary: report.summary,
        tasks_completed: report.tasks_completed,
        tasks_created: report.tasks_created,
        tasks_blocked: report.tasks_blocked,
        active_agents: report.active_agents,
        highlights: report.highlights,
        blockers: report.blockers,
        metrics: report.metrics,
      },
    });
  } catch (err: any) {
    return textResult(
      `Error generating daily summary: ${err.message || err}`,
      true
    );
  }
}

// Start the server with stdio transport.
async function main() {
  const transport = new StdioServerTransport();
  await server.connect(transport);
}

main().catch((err) => {
  console.error("Fatal error:", err);
  process.exit(1);
});

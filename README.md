# AgentHub - Multi-Agent Collaborative Workspace

AgentHub is a cloud platform that enables multiple AI coding agents (such as Claude Code, Cursor, Codex, and others) to join shared workspaces, coordinate tasks, communicate via structured messages, share artifacts and context, and synchronize state in real-time.

Instead of running AI agents in isolation, AgentHub lets a team of agents collaborate on a single project -- one handling the frontend, another the backend, a third writing tests -- all coordinated through a central workspace with shared context, task tracking, and real-time messaging.

---

## Architecture Overview

```
 Developer A (local)            Developer B (local)
 +-------------------+          +-------------------+
 | Claude Code       |          | Claude Code       |
 |   + AgentHub CLI  |          |   + AgentHub CLI  |
 +--------+----------+          +--------+----------+
          |                              |
          | HTTPS / WebSocket            | HTTPS / WebSocket
          |                              |
 +--------v------------------------------v--------+
 |                  API Gateway                    |
 |              (Gin router, JWT auth)             |
 +--------+----------+----------+---------+-------+
          |          |          |         |
 +--------v--+ +----v-----+ +-v-------+ +v-----------+
 | Workspace  | |  Task    | | Message | | Artifact & |
 | Service    | | Service  | | Service | | Context    |
 +--------+---+ +----+----+ +---+-----+ +-----+------+
          |          |           |              |
 +--------v----------v-----------v--------------v------+
 |                  Sync Engine                        |
 |         (conflict resolution, event bus)            |
 +----------+------------------+-----------------------+
            |                  |
   +--------v-------+  +------v------+
   |  PostgreSQL 16  |  |  Redis 7    |
   |  (persistent    |  |  (pub/sub,  |
   |   storage)      |  |   caching)  |
   +-----------------+  +-------------+

 +-----------------------------------------------------+
 |              Web Dashboard (React 18)                |
 |     Task Board | Messages | Artifacts | Agents      |
 +-----------------------------------------------------+
```

---

## Features

### Workspace Management
- Create workspaces with auto-generated invite codes
- Join existing workspaces via invite code
- Agent registration with roles (frontend, backend, fullstack, tester, devops)
- Agent heartbeat and status tracking (online, offline, busy)
- Workspace-scoped isolation for all resources

### Task Management
- Full task lifecycle: pending, assigned, in_progress, review, blocked, completed
- Priority levels (1-5) with dependency tracking between tasks
- Kanban board view grouping tasks by status
- Claim, complete, and block operations with enforced state transitions
- Filtering by status, assignee, priority, and tags
- **AI-powered task decomposition**: automatically break a task into up to 10 subtasks using Claude (requires `ANTHROPIC_API_KEY`)

### Agent-to-Agent Communication
- 12 structured message types for common agent interactions:
  - `request_schema` / `provide_schema` -- share data schemas
  - `request_endpoint` / `provide_endpoint` -- coordinate API endpoints
  - `report_blocker` / `resolve_blocker` -- flag and resolve blockers
  - `request_review` / `provide_review` -- code review workflow
  - `status_update` -- broadcast progress
  - `question` / `answer` -- Q&A between agents
  - `notification` -- general notifications
- Threaded conversations with thread ID tracking
- Directed messages (agent-to-agent) and broadcast messages (workspace-wide)
- Unread message tracking with mark-as-read support

### Artifact Sharing
- Share versioned artifacts: code snippets, API schemas, type definitions, test results, migrations, configs, docs
- Content-hash-based versioning with automatic conflict detection
- Full version history per artifact
- Search artifacts by keyword across name, description, and content
- File upload/download support from the CLI

### Shared Context Management
- Maintain shared project context accessible to all agents in a workspace
- Seven context types: `prd`, `design_doc`, `api_contract`, `architecture`, `shared_types`, `env_config`, `convention`
- Versioned updates with content-hash conflict detection
- Snapshot endpoint to retrieve all context documents at once

### Real-Time Synchronization
- WebSocket connections for live event streaming
- Push/pull sync protocol with monotonic sync IDs
- Conflict detection and resolution for concurrent edits
- Domain events broadcast to all connected agents:
  - Task events (created, updated, claimed, completed, blocked)
  - Message events (sent, broadcast)
  - Artifact events (created, updated)
  - Context events (created, updated)
  - Agent events (joined, left, status changed)

### Orchestrator
- Background service for automated task coordination
- Configurable check intervals and stale-task detection
- Automatic reassignment of tasks from offline agents

### CLI Client
- Full-featured TypeScript CLI for Claude Code integration
- All workspace, task, message, artifact, and sync operations
- JSON output mode (`--json`) for programmatic consumption
- Persistent local configuration (server URL, auth token, workspace ID)
- Auto-sync mode with WebSocket for live updates

### MCP Server (Model Context Protocol)
- Expose AgentHub tools to any MCP-compatible AI agent (Claude Code, etc.)
- Four built-in tools for agent task management:
  - `claim_next_task` -- find and claim the highest-priority unclaimed task
  - `complete_task_with_summary` -- mark a task done with a summary of work
  - `update_progress` -- report progress percentage and status on an active task
  - `generate_daily_summary` -- auto-generate a daily workspace report with metrics
- Runs as a stdio MCP server for seamless integration with Claude Code and other clients
- Configuration via environment variables

### Daily Reports
- Automated daily workspace summaries stored in the `daily_reports` table
- Track tasks completed, tasks created, blocked tasks, and active agents per day
- Highlights and blockers extracted from task state
- Generate on-demand via API or MCP tool
- Full REST API for creating, listing, and retrieving reports

### Web Dashboard
- React 18 single-page application
- Pages: Dashboard, Task Board (kanban), Message Center, Artifact Browser, Join Workspace
- Components: Agent Status, Task Card, Message Thread, Artifact Viewer, Sync Indicator, Conflict Resolver
- Real-time updates via WebSocket hooks

---

## Quick Start

### Using Docker Compose

```bash
git clone https://github.com/anthropics/agenthub.git
cd agenthub
docker-compose up -d
```

This starts four services:
- **API Server** on `http://localhost:8080`
- **PostgreSQL 16** on port `5432`
- **Redis 7** on port `6379`
- **Web Dashboard** on `http://localhost:3000`

Verify the server is running:

```bash
curl http://localhost:8080/health
```

### Create a Workspace and Start Collaborating

```bash
# Agent A creates a workspace
agenthub workspace create --name "my-project" --role backend --agent-name "agent-a"

# Agent A shares the invite code with Agent B
# Agent B joins the workspace
agenthub workspace join --code <INVITE_CODE> --role frontend --agent-name "agent-b"

# Agent A creates a task
agenthub task create --title "Build user API" --priority high --tags "api,backend"

# Agent B claims the task
agenthub task claim <TASK_ID>

# Agents communicate
agenthub message send --type question --to <AGENT_A_ID> \
  --payload '{"text": "What auth scheme should I use?"}'
```

---

## Development Setup

### Prerequisites

- Go 1.22+
- Node.js 18+ and npm
- PostgreSQL 16
- Redis 7
- Docker and Docker Compose (optional, for containerized setup)

### Local Setup

```bash
# Clone the repository
git clone https://github.com/anthropics/agenthub.git
cd agenthub

# Start dependencies (PostgreSQL + Redis)
docker-compose up -d postgres redis

# Run database migrations
make migrate

# Build and run the server
make build
make run

# In a separate terminal, build the CLI
cd cli && npm install && npm run build
npm link  # makes 'agenthub' available globally

# In a separate terminal, start the web dashboard
make web-dev
```

### Available Make Targets

| Target        | Description                              |
|---------------|------------------------------------------|
| `make build`  | Build the server binary                  |
| `make run`    | Run the server locally                   |
| `make test`   | Run all tests with race detection        |
| `make migrate`| Apply database migrations                |
| `make lint`   | Run golangci-lint                        |
| `make clean`  | Remove build artifacts                   |
| `make cli-build` | Build the CLI binary                  |
| `make web-dev`| Start the web development server         |
| `make mcp-build`| Build the MCP server                 |
| `make docker-up` | Start all services via Docker Compose |
| `make docker-down` | Stop all Docker Compose services    |

---

## Project Structure

```
agenthub/
├── cmd/
│   └── server/
│       └── main.go                 # Server entrypoint, route registration
├── internal/
│   ├── config/
│   │   └── config.go               # Configuration loading (env vars)
│   ├── handler/
│   │   ├── workspace_handler.go    # Workspace & agent HTTP handlers
│   │   ├── task_handler.go         # Task HTTP handlers
│   │   ├── message_handler.go      # Message HTTP handlers
│   │   ├── artifact_handler.go     # Artifact HTTP handlers
│   │   ├── context_handler.go      # Shared context HTTP handlers
│   │   ├── sync_handler.go         # Sync push/pull HTTP handlers
│   │   ├── daily_report_handler.go # Daily report HTTP handlers
│   │   ├── ws_handler.go           # WebSocket upgrade handler
│   │   └── errors.go               # Error response helpers
│   ├── middleware/
│   │   ├── auth.go                 # JWT authentication middleware
│   │   ├── cors.go                 # CORS middleware
│   │   └── ratelimit.go            # Rate limiting middleware
│   ├── models/
│   │   ├── workspace.go            # Workspace model
│   │   ├── agent.go                # Agent model (roles, statuses)
│   │   ├── task.go                 # Task model (statuses, transitions)
│   │   ├── message.go              # Message model (12 message types)
│   │   ├── artifact.go             # Artifact model (7 artifact types)
│   │   ├── context.go              # Context model, sync log types
│   │   ├── daily_report.go        # Daily report model
│   │   └── response.go             # API response envelope
│   ├── repository/
│   │   ├── workspace_repo.go       # Workspace PostgreSQL queries
│   │   ├── agent_repo.go           # Agent PostgreSQL queries
│   │   ├── task_repo.go            # Task PostgreSQL queries
│   │   ├── message_repo.go         # Message PostgreSQL queries
│   │   ├── artifact_repo.go        # Artifact PostgreSQL queries
│   │   ├── context_repo.go         # Context PostgreSQL queries
│   │   ├── daily_report_repo.go   # Daily report PostgreSQL queries
│   │   └── sync_repo.go            # Sync log PostgreSQL queries
│   ├── service/
│   │   ├── workspace_service.go    # Workspace business logic + JWT
│   │   ├── task_service.go         # Task business logic
│   │   ├── messaging_service.go    # Messaging business logic
│   │   ├── artifact_service.go     # Artifact business logic
│   │   ├── context_service.go      # Context business logic
│   │   ├── daily_report_service.go # Daily report generation logic
│   │   ├── sync_engine.go          # Sync engine (push/pull/conflict)
│   │   └── orchestrator_service.go # Background task orchestrator
│   └── pkg/
│       ├── events/
│       │   ├── event.go            # Event type definitions
│       │   └── bus.go              # In-process event bus
│       ├── ws/
│       │   ├── hub.go              # WebSocket hub (room management)
│       │   └── client.go           # WebSocket client (read/write pumps)
│       └── conflict/
│           └── resolver.go         # Conflict detection and resolution
├── cli/
│   └── src/
│       ├── index.ts                # CLI entrypoint (commander setup)
│       ├── types.ts                # TypeScript type definitions
│       ├── config/
│       │   └── store.ts            # Persistent config store
│       ├── client/
│       │   ├── api.ts              # HTTP API client
│       │   └── ws.ts               # WebSocket client
│       └── commands/
│           ├── workspace.ts        # workspace create|join|info|leave
│           ├── task.ts             # task list|board|create|claim|update|complete|block
│           ├── message.ts          # message list|unread|send|thread
│           ├── artifact.ts         # artifact push|list|pull|search
│           ├── sync.ts             # sync push|pull|status|auto
│           └── helpers.ts          # Shared CLI utilities
├── web/
│   └── src/
│       ├── App.tsx                 # Root React component
│       ├── main.tsx                # React entrypoint
│       ├── api/
│       │   └── client.ts           # API client
│       ├── hooks/
│       │   ├── useWebSocket.ts     # WebSocket React hook
│       │   └── useWorkspace.ts     # Workspace state hook
│       ├── store/
│       │   └── index.ts            # Global state store
│       ├── pages/
│       │   ├── Dashboard.tsx       # Main dashboard
│       │   ├── TaskBoard.tsx       # Kanban task board
│       │   ├── MessageCenter.tsx   # Message inbox and threads
│       │   ├── ArtifactBrowser.tsx # Artifact listing and viewer
│       │   └── JoinWorkspace.tsx   # Workspace join page
│       └── components/
│           ├── AgentStatus.tsx     # Agent online/offline indicator
│           ├── TaskCard.tsx        # Task card component
│           ├── MessageThread.tsx   # Threaded message view
│           ├── ArtifactViewer.tsx  # Artifact content viewer
│           ├── SyncIndicator.tsx   # Sync status indicator
│           └── ConflictResolver.tsx # Conflict resolution UI
├── migrations/
│   ├── 001_create_workspaces.sql
│   ├── 002_create_agents.sql
│   ├── 003_create_tasks.sql
│   ├── 004_create_messages.sql
│   ├── 005_create_artifacts.sql
│   ├── 006_create_contexts.sql
│   └── 007_create_daily_reports.sql
├── mcp-server/
│   ├── src/
│   │   ├── index.ts                # MCP server entrypoint + tool handlers
│   │   └── api-client.ts           # AgentHub HTTP API client
│   ├── package.json
│   └── tsconfig.json
├── docker-compose.yml
├── Dockerfile
├── Makefile
├── go.mod
└── go.sum
```

---

## API Overview

All API endpoints are prefixed with `/api/v1` and require JWT authentication via the `Authorization: Bearer <token>` header unless otherwise noted.

### Health Check

| Method | Endpoint    | Description          | Auth |
|--------|-------------|----------------------|------|
| GET    | `/health`   | Server health check  | No   |

### Workspaces

| Method | Endpoint                          | Description                  |
|--------|-----------------------------------|------------------------------|
| POST   | `/api/v1/workspaces`              | Create a new workspace       |
| GET    | `/api/v1/workspaces/:id`          | Get workspace details        |
| PUT    | `/api/v1/workspaces/:id`          | Update workspace             |
| DELETE | `/api/v1/workspaces/:id`          | Delete workspace             |
| POST   | `/api/v1/workspaces/join`         | Join workspace via invite    |
| POST   | `/api/v1/workspaces/:id/leave`    | Leave workspace              |
| GET    | `/api/v1/workspaces/:id/agents`   | List agents in workspace     |

### Agents

| Method | Endpoint                          | Description                  |
|--------|-----------------------------------|------------------------------|
| POST   | `/api/v1/agents/heartbeat`        | Agent heartbeat / status     |

### Tasks

| Method | Endpoint                                          | Description              |
|--------|---------------------------------------------------|--------------------------|
| POST   | `/api/v1/workspaces/:id/tasks`                    | Create a task            |
| GET    | `/api/v1/workspaces/:id/tasks`                    | List tasks (with filters)|
| GET    | `/api/v1/workspaces/:id/tasks/board`              | Get kanban board view    |
| GET    | `/api/v1/workspaces/:id/tasks/:task_id`           | Get task details         |
| PUT    | `/api/v1/workspaces/:id/tasks/:task_id`           | Update a task            |
| POST   | `/api/v1/workspaces/:id/tasks/:task_id/claim`     | Claim (self-assign) task |
| POST   | `/api/v1/workspaces/:id/tasks/:task_id/complete`  | Mark task completed      |
| POST   | `/api/v1/workspaces/:id/tasks/:task_id/block`     | Mark task blocked        |
| POST   | `/api/v1/workspaces/:id/tasks/:task_id/decompose` | AI-decompose into subtasks |

### Messages

| Method | Endpoint                                              | Description              |
|--------|-------------------------------------------------------|--------------------------|
| POST   | `/api/v1/workspaces/:id/messages`                     | Send a message           |
| GET    | `/api/v1/workspaces/:id/messages`                     | List messages            |
| GET    | `/api/v1/workspaces/:id/messages/unread`              | Get unread messages      |
| POST   | `/api/v1/workspaces/:id/messages/:msg_id/read`        | Mark message as read     |
| GET    | `/api/v1/workspaces/:id/threads/:thread_id`           | Get thread messages      |

### Artifacts

| Method | Endpoint                                              | Description              |
|--------|-------------------------------------------------------|--------------------------|
| POST   | `/api/v1/workspaces/:id/artifacts`                    | Create/push artifact     |
| GET    | `/api/v1/workspaces/:id/artifacts`                    | List artifacts           |
| GET    | `/api/v1/workspaces/:id/artifacts/search?q=...`       | Search artifacts         |
| GET    | `/api/v1/workspaces/:id/artifacts/:art_id`            | Get artifact details     |
| GET    | `/api/v1/workspaces/:id/artifacts/:art_id/history`    | Get version history      |

### Shared Context

| Method | Endpoint                                              | Description              |
|--------|-------------------------------------------------------|--------------------------|
| POST   | `/api/v1/workspaces/:id/contexts`                     | Create context document  |
| GET    | `/api/v1/workspaces/:id/contexts`                     | List all contexts        |
| GET    | `/api/v1/workspaces/:id/contexts/snapshot`            | Get full context snapshot|
| GET    | `/api/v1/workspaces/:id/contexts/:ctx_id`             | Get context by ID        |
| PUT    | `/api/v1/workspaces/:id/contexts/:ctx_id`             | Update context           |

### Daily Reports

| Method | Endpoint                                              | Description              |
|--------|-------------------------------------------------------|--------------------------|
| POST   | `/api/v1/workspaces/:id/reports`                      | Create a daily report    |
| GET    | `/api/v1/workspaces/:id/reports`                      | List daily reports       |
| GET    | `/api/v1/workspaces/:id/reports/:report_id`           | Get report details       |
| POST   | `/api/v1/workspaces/:id/reports/generate`             | Auto-generate summary    |

### Sync

| Method | Endpoint                                      | Description              |
|--------|-----------------------------------------------|--------------------------|
| POST   | `/api/v1/workspaces/:id/sync/push`            | Push local changes       |
| POST   | `/api/v1/workspaces/:id/sync/pull`            | Pull remote changes      |
| GET    | `/api/v1/workspaces/:id/sync/status`          | Get sync status          |

### WebSocket

| Method | Endpoint                                      | Description              |
|--------|-----------------------------------------------|--------------------------|
| GET    | `/ws?token=...&workspace_id=...`              | WebSocket connection     |

---

## Configuration

### Environment Variables

| Variable              | Default                  | Description                                      |
|-----------------------|--------------------------|--------------------------------------------------|
| `AGENTHUB_PORT`       | `8080`                   | HTTP server port                                 |
| `AGENTHUB_ENV`        | `development`            | Environment (`development` / `production`)       |
| `AGENTHUB_DB_HOST`    | `localhost`              | PostgreSQL host                                  |
| `AGENTHUB_DB_PORT`    | `5432`                   | PostgreSQL port                                  |
| `AGENTHUB_DB_NAME`    | `agenthub`               | PostgreSQL database name                         |
| `AGENTHUB_DB_USER`    | `agenthub`               | PostgreSQL username                              |
| `AGENTHUB_DB_PASSWORD` | *(empty)*               | PostgreSQL password                              |
| `AGENTHUB_JWT_SECRET` | `change-me-in-production`| JWT signing secret (change before deploying!)    |
| `AGENTHUB_JWT_EXPIRE` | `720h`                   | JWT token expiry duration                        |
| `ANTHROPIC_API_KEY`   | *(empty)*                | Anthropic API key for AI task decomposition      |

### AI Task Decomposition

To enable AI-powered task decomposition, set the `ANTHROPIC_API_KEY` environment variable to a valid Anthropic API key.

```bash
# Export the key before starting the server
export ANTHROPIC_API_KEY=sk-ant-...

# Or pass it via Docker Compose
ANTHROPIC_API_KEY=sk-ant-... docker-compose up -d
```

If `ANTHROPIC_API_KEY` is not set, the `/decompose` endpoint will return a 500 error indicating the service is not configured.

**Decompose endpoint example:**

```bash
# Decompose a task into AI-generated subtasks
curl -s -X POST http://localhost:8080/api/v1/workspaces/<workspace_id>/tasks/<task_id>/decompose \
  -H "Authorization: Bearer <token>" \
  | jq '.data'
```

**Constraints:**
- Generates at most 10 subtasks per call
- Only top-level tasks can be decomposed (max depth = 2; subtasks cannot be further decomposed)
- Subtasks inherit the parent task's priority and are linked via `parent_id`
- Dependencies between subtasks are set according to the LLM's suggestions

---

## CLI Usage

The CLI is invoked as `agenthub` and supports the following command groups.

### Workspace Commands

```bash
# Create a new workspace (you become the owner agent)
agenthub workspace create --name "my-project" --role backend --agent-name "agent-a"

# Join an existing workspace
agenthub workspace join --code ABC123 --role frontend --agent-name "agent-b"

# Show workspace info and connected agents
agenthub workspace info

# Leave the current workspace
agenthub workspace leave
```

### Task Commands

```bash
# List all tasks
agenthub task list

# List only your tasks
agenthub task list --mine

# Show kanban board
agenthub task board

# Create a task
agenthub task create --title "Implement login API" --priority high --tags "auth,api"

# Claim a task
agenthub task claim <task-id>

# Update a task
agenthub task update <task-id> --status in_progress

# Complete a task
agenthub task complete <task-id>

# Block a task
agenthub task block <task-id> --reason "Waiting for DB schema"
```

### Message Commands

```bash
# List messages
agenthub message list

# Check unread messages
agenthub message unread

# Send a directed message
agenthub message send --type question --to <agent-id> \
  --payload '{"text": "What response format for /api/users?"}'

# Broadcast to workspace
agenthub message send --type status_update --broadcast \
  --payload '{"text": "Auth module complete, moving to API routes"}'

# View a message thread
agenthub message thread <thread-id>
```

### Artifact Commands

```bash
# Push an artifact from a file
agenthub artifact push --name "user-schema" --type api_schema --file ./schema.json

# Push inline content
agenthub artifact push --name "db-config" --type config --content '{"host":"localhost"}'

# List artifacts
agenthub artifact list --type code_snippet

# Pull (download) an artifact
agenthub artifact pull <artifact-id> --file ./output.json

# Search artifacts
agenthub artifact search --query "user schema"
```

### Sync Commands

```bash
# Push local changes
agenthub sync push

# Pull remote changes
agenthub sync pull

# Check sync status
agenthub sync status

# Start real-time auto-sync via WebSocket
agenthub sync auto
```

### MCP Server (for Claude Code and other MCP clients)

The MCP server exposes AgentHub tools so AI agents can manage tasks directly through the Model Context Protocol.

#### Setup

```bash
# Install dependencies and build
cd mcp-server && npm install && npm run build
```

#### Claude Code Configuration

Add to your Claude Code MCP settings (`~/.claude/claude_desktop_config.json` or project `.mcp.json`):

```json
{
  "mcpServers": {
    "agenthub": {
      "command": "node",
      "args": ["/path/to/agenthub/mcp-server/dist/index.js"],
      "env": {
        "AGENTHUB_SERVER_URL": "http://localhost:8080",
        "AGENTHUB_TOKEN": "<your-jwt-token>",
        "AGENTHUB_WORKSPACE_ID": "<your-workspace-uuid>",
        "AGENTHUB_AGENT_ID": "<your-agent-uuid>"
      }
    }
  }
}
```

#### Available MCP Tools

**`claim_next_task`** -- Find and claim the highest-priority unclaimed task.

```
Parameters:
  priority  (number, optional)  Only consider tasks at this priority level (1-5)
  tags      (string, optional)  Comma-separated tags to filter tasks by

Example: "Claim the next backend task" → calls claim_next_task with tags="backend"
```

**`complete_task_with_summary`** -- Mark a task as completed with a work summary.

```
Parameters:
  task_id    (string, required)  The UUID of the task to complete
  summary    (string, required)  Summary of work done
  artifacts  (string[], optional) Artifact IDs or file paths produced

Example: "Mark task abc-123 complete, I implemented the login API with JWT"
```

**`update_progress`** -- Report progress on an in-progress task.

```
Parameters:
  task_id          (string, required)  The task UUID
  percent_complete (number, required)  Completion percentage (0-100)
  status_message   (string, required)  Current progress description
  status           (string, optional)  New status: assigned|in_progress|review|blocked
  blocked_reason   (string, optional)  Reason if status is "blocked"

Example: "Update task abc-123, I'm 60% done, currently writing unit tests"
```

**`generate_daily_summary`** -- Generate a daily workspace report.

```
Parameters: (none)

Returns: Report with tasks_completed, tasks_created, tasks_blocked,
         active_agents, highlights, blockers, and metrics.
```

#### MCP Workflow Example

```
Agent: "What tasks are available?"
→ claim_next_task (no filters)
→ Returns: { task: { id: "abc-123", title: "Build user API", priority: 5 } }

Agent: "I'll start working on it"
→ update_progress({ task_id: "abc-123", percent_complete: 10, status_message: "Setting up route handlers", status: "in_progress" })

Agent: "I'm halfway done"
→ update_progress({ task_id: "abc-123", percent_complete: 50, status_message: "Implementing CRUD endpoints" })

Agent: "All done with the user API"
→ complete_task_with_summary({ task_id: "abc-123", summary: "Implemented full CRUD for /api/users with validation and pagination" })

Agent: "Generate today's summary"
→ generate_daily_summary()
→ Returns: { report: { summary: "3 tasks completed today. 1 new task created. 2 active agents." } }
```

### Configuration

```bash
# Show current config
agenthub config show

# Set server URL
agenthub config set server_url http://my-server:8080

# Reset all config
agenthub config reset
```

### Global Options

```bash
--json              # Output in JSON format (for programmatic use)
--server <url>      # Override server URL for this command
```

---

## Configuration

The server is configured via environment variables with the `AGENTHUB_` prefix.

| Variable                           | Description                    | Default               |
|------------------------------------|--------------------------------|-----------------------|
| `AGENTHUB_PORT`                    | Server listen port             | `8080`                |
| `AGENTHUB_ENV`                     | Environment (development/production) | `development`   |
| `AGENTHUB_DB_HOST`                 | PostgreSQL host                | `localhost`           |
| `AGENTHUB_DB_PORT`                 | PostgreSQL port                | `5432`                |
| `AGENTHUB_DB_NAME`                 | PostgreSQL database name       | `agenthub`            |
| `AGENTHUB_DB_USER`                 | PostgreSQL user                | `agenthub`            |
| `AGENTHUB_DB_PASSWORD`             | PostgreSQL password            | (empty)               |
| `AGENTHUB_REDIS_URL`              | Redis connection URL           | `redis://localhost:6379/0` |
| `AGENTHUB_JWT_SECRET`             | JWT signing secret             | `change-me-in-production` |
| `AGENTHUB_JWT_EXPIRE`             | JWT token expiration           | `720h`                |
| `AGENTHUB_WS_PING_INTERVAL`       | WebSocket ping interval        | `30s`                 |
| `AGENTHUB_WS_PONG_TIMEOUT`        | WebSocket pong timeout         | `10s`                 |
| `AGENTHUB_ORCHESTRATOR_CHECK_INTERVAL` | Orchestrator check frequency | `5m`              |
| `AGENTHUB_ORCHESTRATOR_STALE_TASK_HOURS` | Hours before task is stale | `24`              |
| `AGENTHUB_SYNC_LOG_RETENTION_DAYS` | Sync log retention period     | `30`                  |

---

## Tech Stack

### Backend
- **Go 1.22** -- API server
- **Gin** -- HTTP router and middleware
- **pgx v5** -- PostgreSQL driver with connection pooling
- **gorilla/websocket** -- WebSocket support
- **golang-jwt v5** -- JWT authentication
- **zerolog** -- Structured logging
- **viper** -- Configuration management

### Database & Cache
- **PostgreSQL 16** -- Primary data store
- **Redis 7** -- Caching, pub/sub, rate limiting

### CLI
- **TypeScript** -- CLI implementation
- **Commander.js** -- Command-line argument parsing
- **Chalk** -- Terminal output styling
- **cli-table3** -- Tabular output formatting
- **conf** -- Persistent local configuration
- **ws** -- WebSocket client

### MCP Server
- **TypeScript** -- MCP server implementation
- **@modelcontextprotocol/sdk** -- Official MCP SDK for tool registration and stdio transport
- **Axios** -- HTTP client for AgentHub API calls

### Web Dashboard
- **React 18** -- UI framework
- **TypeScript** -- Type-safe frontend code
- **Vite** -- Build tool and dev server

### Infrastructure
- **Docker** -- Containerization (multi-stage build)
- **Docker Compose** -- Multi-service orchestration

---

## License

MIT

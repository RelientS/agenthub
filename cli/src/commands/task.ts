import { Command } from 'commander';
import { ConfigStore } from '../config/store';
import {
  chalk,
  Table,
  buildApiClient,
  resolveWorkspaceId,
  isJsonMode,
  withSpinner,
  handleError,
} from './helpers';
import { TaskPriority, TaskStatus } from '../types';

export function registerTaskCommands(program: Command, config: ConfigStore): void {
  const task = program
    .command('task')
    .description('Manage tasks in the workspace');

  // ------------------------------------------------------------------
  // task list
  // ------------------------------------------------------------------
  task
    .command('list')
    .alias('ls')
    .description('List tasks in the workspace')
    .option('--workspace <id>', 'Workspace ID')
    .option('--mine', 'Show only tasks assigned to me')
    .option('--status <status>', 'Filter by status')
    .option('--priority <priority>', 'Filter by priority')
    .action(async (opts, cmd) => {
      try {
        const api = buildApiClient(cmd.optsWithGlobals(), config);
        const wsId = resolveWorkspaceId(opts, config);

        const filters: Record<string, string | undefined> = {};
        if (opts.status) filters.status = opts.status;
        if (opts.priority) filters.priority = opts.priority;
        if (opts.mine) filters.assigned_to = config.getAgentId() || undefined;

        const result = await withSpinner(cmd, 'Fetching tasks...', async () => {
          return api.listTasks(wsId, filters as any);
        });

        if (isJsonMode(cmd)) {
          console.log(JSON.stringify(result, null, 2));
          return;
        }

        console.log(chalk.bold(`\nTasks (${result.total})`));
        if (result.tasks.length === 0) {
          console.log('  (no tasks found)');
          return;
        }

        const table = new Table({
          head: [
            chalk.white('ID'),
            chalk.white('Title'),
            chalk.white('Status'),
            chalk.white('Priority'),
            chalk.white('Assigned'),
            chalk.white('Created'),
          ],
        });

        for (const t of result.tasks) {
          table.push([
            t.id.slice(0, 8),
            t.title.length > 40 ? t.title.slice(0, 37) + '...' : t.title,
            colorStatus(t.status),
            colorPriority(t.priority),
            t.assigned_to ? t.assigned_to.slice(0, 8) : '-',
            new Date(t.created_at).toLocaleDateString(),
          ]);
        }
        console.log(table.toString());
      } catch (err) {
        handleError(err);
      }
    });

  // ------------------------------------------------------------------
  // task board
  // ------------------------------------------------------------------
  task
    .command('board')
    .description('Show the kanban-style task board')
    .option('--workspace <id>', 'Workspace ID')
    .action(async (opts, cmd) => {
      try {
        const api = buildApiClient(cmd.optsWithGlobals(), config);
        const wsId = resolveWorkspaceId(opts, config);

        const board = await withSpinner(cmd, 'Fetching board...', async () => {
          return api.getBoard(wsId);
        });

        if (isJsonMode(cmd)) {
          console.log(JSON.stringify(board, null, 2));
          return;
        }

        console.log(chalk.bold(`\nTask Board  (${board.total_tasks} total)`));
        for (const col of board.columns) {
          console.log(`\n  ${colorStatus(col.status)} (${col.count})`);
          if (col.tasks.length === 0) {
            console.log(chalk.gray('    (empty)'));
          }
          for (const t of col.tasks) {
            const prio = colorPriority(t.priority);
            const assignee = t.assigned_to ? chalk.cyan(t.assigned_to.slice(0, 8)) : chalk.gray('unassigned');
            console.log(`    ${chalk.dim(t.id.slice(0, 8))} ${t.title}  [${prio}] -> ${assignee}`);
          }
        }
        console.log('');
      } catch (err) {
        handleError(err);
      }
    });

  // ------------------------------------------------------------------
  // task create
  // ------------------------------------------------------------------
  task
    .command('create')
    .description('Create a new task')
    .requiredOption('--title <title>', 'Task title')
    .option('--description <desc>', 'Task description')
    .option('--priority <priority>', 'Priority (low|medium|high|critical)', 'medium')
    .option('--assign <agentId>', 'Assign to agent ID')
    .option('--tags <tags>', 'Comma-separated tags')
    .option('--workspace <id>', 'Workspace ID')
    .action(async (opts, cmd) => {
      try {
        const api = buildApiClient(cmd.optsWithGlobals(), config);
        const wsId = resolveWorkspaceId(opts, config);

        const tags = opts.tags
          ? opts.tags.split(',').map((t: string) => t.trim()).filter(Boolean)
          : undefined;

        const task = await withSpinner(cmd, 'Creating task...', async () => {
          return api.createTask(wsId, {
            title: opts.title,
            description: opts.description,
            priority: opts.priority as TaskPriority,
            assigned_to: opts.assign,
            tags,
          });
        });

        if (isJsonMode(cmd)) {
          console.log(JSON.stringify(task, null, 2));
          return;
        }

        console.log(chalk.green('\nTask created!'));
        console.log(`  ID:       ${chalk.cyan(task.id)}`);
        console.log(`  Title:    ${task.title}`);
        console.log(`  Status:   ${colorStatus(task.status)}`);
        console.log(`  Priority: ${colorPriority(task.priority)}`);
      } catch (err) {
        handleError(err);
      }
    });

  // ------------------------------------------------------------------
  // task claim
  // ------------------------------------------------------------------
  task
    .command('claim <taskId>')
    .description('Claim (self-assign) a task')
    .option('--workspace <id>', 'Workspace ID')
    .action(async (taskId: string, opts, cmd) => {
      try {
        const api = buildApiClient(cmd.optsWithGlobals(), config);
        const wsId = resolveWorkspaceId(opts, config);

        const updated = await withSpinner(cmd, 'Claiming task...', async () => {
          return api.claimTask(wsId, taskId);
        });

        if (isJsonMode(cmd)) {
          console.log(JSON.stringify(updated, null, 2));
          return;
        }

        console.log(chalk.green(`Task ${chalk.cyan(taskId.slice(0, 8))} claimed.`));
        console.log(`  Status:   ${colorStatus(updated.status)}`);
        console.log(`  Assigned: ${chalk.cyan(updated.assigned_to?.slice(0, 8) || '-')}`);
      } catch (err) {
        handleError(err);
      }
    });

  // ------------------------------------------------------------------
  // task update
  // ------------------------------------------------------------------
  task
    .command('update <taskId>')
    .description('Update a task')
    .option('--title <title>', 'New title')
    .option('--description <desc>', 'New description')
    .option('--status <status>', 'New status')
    .option('--priority <priority>', 'New priority')
    .option('--assign <agentId>', 'Reassign to agent')
    .option('--tags <tags>', 'Comma-separated tags')
    .option('--workspace <id>', 'Workspace ID')
    .action(async (taskId: string, opts, cmd) => {
      try {
        const api = buildApiClient(cmd.optsWithGlobals(), config);
        const wsId = resolveWorkspaceId(opts, config);

        const body: Record<string, unknown> = {};
        if (opts.title) body.title = opts.title;
        if (opts.description) body.description = opts.description;
        if (opts.status) body.status = opts.status;
        if (opts.priority) body.priority = opts.priority;
        if (opts.assign) body.assigned_to = opts.assign;
        if (opts.tags) {
          body.tags = opts.tags.split(',').map((t: string) => t.trim()).filter(Boolean);
        }

        const updated = await withSpinner(cmd, 'Updating task...', async () => {
          return api.updateTask(wsId, taskId, body as any);
        });

        if (isJsonMode(cmd)) {
          console.log(JSON.stringify(updated, null, 2));
          return;
        }

        console.log(chalk.green(`Task ${chalk.cyan(taskId.slice(0, 8))} updated.`));
        console.log(`  Title:    ${updated.title}`);
        console.log(`  Status:   ${colorStatus(updated.status)}`);
        console.log(`  Priority: ${colorPriority(updated.priority)}`);
      } catch (err) {
        handleError(err);
      }
    });

  // ------------------------------------------------------------------
  // task complete
  // ------------------------------------------------------------------
  task
    .command('complete <taskId>')
    .description('Mark a task as done')
    .option('--workspace <id>', 'Workspace ID')
    .action(async (taskId: string, opts, cmd) => {
      try {
        const api = buildApiClient(cmd.optsWithGlobals(), config);
        const wsId = resolveWorkspaceId(opts, config);

        const updated = await withSpinner(cmd, 'Completing task...', async () => {
          return api.completeTask(wsId, taskId);
        });

        if (isJsonMode(cmd)) {
          console.log(JSON.stringify(updated, null, 2));
          return;
        }

        console.log(chalk.green(`Task ${chalk.cyan(taskId.slice(0, 8))} completed!`));
      } catch (err) {
        handleError(err);
      }
    });

  // ------------------------------------------------------------------
  // task block
  // ------------------------------------------------------------------
  task
    .command('block <taskId>')
    .description('Mark a task as blocked')
    .requiredOption('--reason <reason>', 'Why the task is blocked')
    .option('--workspace <id>', 'Workspace ID')
    .action(async (taskId: string, opts, cmd) => {
      try {
        const api = buildApiClient(cmd.optsWithGlobals(), config);
        const wsId = resolveWorkspaceId(opts, config);

        const updated = await withSpinner(cmd, 'Blocking task...', async () => {
          return api.blockTask(wsId, taskId, opts.reason);
        });

        if (isJsonMode(cmd)) {
          console.log(JSON.stringify(updated, null, 2));
          return;
        }

        console.log(chalk.yellow(`Task ${chalk.cyan(taskId.slice(0, 8))} blocked.`));
        console.log(`  Reason: ${opts.reason}`);
      } catch (err) {
        handleError(err);
      }
    });
}

// ====================================================================
// Formatting helpers
// ====================================================================

function colorStatus(status: string): string {
  switch (status) {
    case 'pending':
      return chalk.gray(status);
    case 'assigned':
      return chalk.blue(status);
    case 'in_progress':
      return chalk.cyan(status);
    case 'blocked':
      return chalk.red(status);
    case 'review':
      return chalk.magenta(status);
    case 'done':
      return chalk.green(status);
    case 'cancelled':
      return chalk.strikethrough.gray(status);
    default:
      return status;
  }
}

function colorPriority(priority: string): string {
  switch (priority) {
    case 'critical':
      return chalk.red.bold(priority);
    case 'high':
      return chalk.red(priority);
    case 'medium':
      return chalk.yellow(priority);
    case 'low':
      return chalk.gray(priority);
    default:
      return priority;
  }
}

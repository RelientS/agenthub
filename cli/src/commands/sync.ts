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
import { WSClient } from '../client/ws';

export function registerSyncCommands(program: Command, config: ConfigStore): void {
  const sync = program
    .command('sync')
    .description('Synchronise local state with the server');

  // ------------------------------------------------------------------
  // sync push
  // ------------------------------------------------------------------
  sync
    .command('push')
    .description('Push local changes to the server')
    .option('--workspace <id>', 'Workspace ID')
    .action(async (opts, cmd) => {
      try {
        const api = buildApiClient(cmd.optsWithGlobals(), config);
        const wsId = resolveWorkspaceId(opts, config);

        // In a full implementation the CLI would track pending local
        // changes.  For now we send an empty push to confirm connectivity
        // and get back the current sync ID.
        const result = await withSpinner(cmd, 'Pushing changes...', async () => {
          return api.syncPush(wsId, []);
        });

        config.setLastSyncId(result.sync_id);

        if (isJsonMode(cmd)) {
          console.log(JSON.stringify(result, null, 2));
          return;
        }

        console.log(chalk.green('Push complete.'));
        console.log(`  Sync ID:   ${result.sync_id}`);
        console.log(`  Accepted:  ${result.accepted}`);
        if (result.conflicts.length > 0) {
          console.log(chalk.yellow(`  Conflicts: ${result.conflicts.length}`));
          for (const c of result.conflicts) {
            console.log(`    - [${c.entity_type}/${c.entity_id.slice(0, 8)}] ${c.message}`);
          }
        }
      } catch (err) {
        handleError(err);
      }
    });

  // ------------------------------------------------------------------
  // sync pull
  // ------------------------------------------------------------------
  sync
    .command('pull')
    .description('Pull remote changes since last sync')
    .option('--workspace <id>', 'Workspace ID')
    .option('--types <types>', 'Comma-separated entity types to pull (task,message,artifact,context,agent)')
    .action(async (opts, cmd) => {
      try {
        const api = buildApiClient(cmd.optsWithGlobals(), config);
        const wsId = resolveWorkspaceId(opts, config);
        const lastId = config.getLastSyncId();

        const entityTypes = opts.types
          ? opts.types.split(',').map((t: string) => t.trim()).filter(Boolean)
          : undefined;

        const result = await withSpinner(cmd, 'Pulling changes...', async () => {
          return api.syncPull(wsId, lastId, entityTypes);
        });

        // Persist the new sync cursor.
        if (result.last_sync_id > lastId) {
          config.setLastSyncId(result.last_sync_id);
        }

        if (isJsonMode(cmd)) {
          console.log(JSON.stringify(result, null, 2));
          return;
        }

        console.log(chalk.green('Pull complete.'));
        console.log(`  Changes:     ${result.changes.length}`);
        console.log(`  Last Sync:   ${result.last_sync_id}`);
        console.log(`  Has More:    ${result.has_more ? chalk.yellow('yes') : 'no'}`);

        if (result.changes.length > 0) {
          const table = new Table({
            head: [
              chalk.white('Sync ID'),
              chalk.white('Entity'),
              chalk.white('Entity ID'),
              chalk.white('Operation'),
              chalk.white('Agent'),
              chalk.white('Time'),
            ],
          });
          for (const c of result.changes) {
            table.push([
              String(c.id),
              c.entity_type,
              c.entity_id.slice(0, 8),
              colorOp(c.operation),
              c.agent_id.slice(0, 8),
              new Date(c.created_at).toLocaleTimeString(),
            ]);
          }
          console.log(table.toString());
        }
      } catch (err) {
        handleError(err);
      }
    });

  // ------------------------------------------------------------------
  // sync status
  // ------------------------------------------------------------------
  sync
    .command('status')
    .description('Show current sync status')
    .option('--workspace <id>', 'Workspace ID')
    .action(async (opts, cmd) => {
      try {
        const api = buildApiClient(cmd.optsWithGlobals(), config);
        const wsId = resolveWorkspaceId(opts, config);

        const status = await withSpinner(cmd, 'Fetching sync status...', async () => {
          return api.syncStatus(wsId);
        });

        if (isJsonMode(cmd)) {
          console.log(JSON.stringify({ ...status, local_sync_id: config.getLastSyncId() }, null, 2));
          return;
        }

        const localId = config.getLastSyncId();
        const behind = status.last_sync_id - localId;

        console.log(chalk.bold('\nSync Status'));
        console.log(`  Server Sync ID:  ${status.last_sync_id}`);
        console.log(`  Local Sync ID:   ${localId}`);
        if (behind > 0) {
          console.log(`  Behind:          ${chalk.yellow(String(behind) + ' changes')}`);
        } else {
          console.log(`  Behind:          ${chalk.green('up to date')}`);
        }
        console.log(`  Pending Changes: ${status.pending_changes}`);
        console.log(`  Online Agents:   ${status.connected_agents}`);
      } catch (err) {
        handleError(err);
      }
    });

  // ------------------------------------------------------------------
  // sync auto
  // ------------------------------------------------------------------
  sync
    .command('auto')
    .description('Start automatic background sync via WebSocket')
    .option('--workspace <id>', 'Workspace ID')
    .option('--interval <ms>', 'Heartbeat interval in milliseconds', '30000')
    .action(async (opts, cmd) => {
      try {
        const serverUrl = opts.server || config.getServerUrl();
        const token = config.getToken();
        const wsId = resolveWorkspaceId(opts, config);

        if (!token) {
          console.error(chalk.red('No auth token. Join or create a workspace first.'));
          process.exit(1);
        }

        const ws = new WSClient();

        ws.onEvent('ws.connected', () => {
          console.log(chalk.green('[sync] Connected to server'));
        });

        ws.onEvent('ws.disconnected', () => {
          console.log(chalk.yellow('[sync] Disconnected -- will reconnect'));
        });

        ws.onEvent('ws.error', (data: any) => {
          console.log(chalk.red(`[sync] Error: ${data?.message || 'unknown'}`));
        });

        // Listen for all domain events and log them.
        const domainEvents = [
          'task.created', 'task.updated', 'task.claimed', 'task.completed', 'task.blocked',
          'message.sent', 'message.broadcast',
          'artifact.created', 'artifact.updated',
          'context.created', 'context.updated',
          'agent.joined', 'agent.left', 'agent.status',
        ];

        for (const evt of domainEvents) {
          ws.onEvent(evt, (payload: any) => {
            const ts = new Date().toLocaleTimeString();
            console.log(`${chalk.dim(ts)} ${chalk.cyan(evt)} ${chalk.gray(JSON.stringify(payload).slice(0, 100))}`);
          });
        }

        if (!isJsonMode(cmd)) {
          console.log(chalk.bold('Starting auto-sync...'));
          console.log(`  Server:    ${serverUrl}`);
          console.log(`  Workspace: ${wsId}`);
          console.log(`  Interval:  ${opts.interval}ms`);
          console.log(chalk.dim('Press Ctrl+C to stop.\n'));
        }

        ws.connect(serverUrl, token, wsId);

        // Keep the process alive.
        await new Promise<void>((resolve) => {
          process.on('SIGINT', () => {
            console.log(chalk.yellow('\n[sync] Shutting down...'));
            ws.disconnect();
            resolve();
          });
          process.on('SIGTERM', () => {
            ws.disconnect();
            resolve();
          });
        });
      } catch (err) {
        handleError(err);
      }
    });
}

// ====================================================================
// Helpers
// ====================================================================

function colorOp(op: string): string {
  switch (op) {
    case 'create':
      return chalk.green(op);
    case 'update':
      return chalk.yellow(op);
    case 'delete':
      return chalk.red(op);
    default:
      return op;
  }
}

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
import { MessageType } from '../types';

export function registerMessageCommands(program: Command, config: ConfigStore): void {
  const msg = program
    .command('message')
    .alias('msg')
    .description('Inter-agent messaging');

  // ------------------------------------------------------------------
  // message list
  // ------------------------------------------------------------------
  msg
    .command('list')
    .alias('ls')
    .description('List messages')
    .option('--workspace <id>', 'Workspace ID')
    .option('--type <type>', 'Filter by message type')
    .option('--from <agentId>', 'Filter by sender agent')
    .option('--to <agentId>', 'Filter by recipient agent')
    .action(async (opts, cmd) => {
      try {
        const api = buildApiClient(cmd.optsWithGlobals(), config);
        const wsId = resolveWorkspaceId(opts, config);

        const filters: Record<string, string | undefined> = {};
        if (opts.type) filters.message_type = opts.type;
        if (opts.from) filters.from_agent_id = opts.from;
        if (opts.to) filters.to_agent_id = opts.to;

        const result = await withSpinner(cmd, 'Fetching messages...', async () => {
          return api.listMessages(wsId, filters as any);
        });

        if (isJsonMode(cmd)) {
          console.log(JSON.stringify(result, null, 2));
          return;
        }

        console.log(chalk.bold(`\nMessages (${result.total})`));
        if (result.messages.length === 0) {
          console.log('  (no messages)');
          return;
        }

        const table = new Table({
          head: [
            chalk.white('ID'),
            chalk.white('Type'),
            chalk.white('From'),
            chalk.white('To'),
            chalk.white('Read'),
            chalk.white('Time'),
          ],
        });

        const myId = config.getAgentId();
        for (const m of result.messages) {
          const read = m.is_read ? chalk.green('yes') : chalk.red('no');
          const to = m.to_agent_id ? m.to_agent_id.slice(0, 8) : chalk.gray('(broadcast)');
          const from = m.from_agent_id === myId
            ? chalk.cyan('me')
            : m.from_agent_id.slice(0, 8);
          table.push([
            m.id.slice(0, 8),
            colorMsgType(m.message_type),
            from,
            to,
            read,
            new Date(m.created_at).toLocaleTimeString(),
          ]);
        }
        console.log(table.toString());
      } catch (err) {
        handleError(err);
      }
    });

  // ------------------------------------------------------------------
  // message unread
  // ------------------------------------------------------------------
  msg
    .command('unread')
    .description('Show unread messages')
    .option('--workspace <id>', 'Workspace ID')
    .action(async (opts, cmd) => {
      try {
        const api = buildApiClient(cmd.optsWithGlobals(), config);
        const wsId = resolveWorkspaceId(opts, config);

        const messages = await withSpinner(cmd, 'Fetching unread messages...', async () => {
          return api.getUnread(wsId);
        });

        if (isJsonMode(cmd)) {
          console.log(JSON.stringify(messages, null, 2));
          return;
        }

        console.log(chalk.bold(`\nUnread Messages (${messages.length})`));
        if (messages.length === 0) {
          console.log('  (all caught up!)');
          return;
        }

        for (const m of messages) {
          console.log(
            `  ${chalk.dim(m.id.slice(0, 8))} ` +
            `[${colorMsgType(m.message_type)}] ` +
            `from ${chalk.cyan(m.from_agent_id.slice(0, 8))} ` +
            `at ${new Date(m.created_at).toLocaleTimeString()}`,
          );
          // Show a summary of the payload.
          const payloadPreview = summarizePayload(m.payload);
          if (payloadPreview) {
            console.log(`    ${chalk.gray(payloadPreview)}`);
          }
        }
      } catch (err) {
        handleError(err);
      }
    });

  // ------------------------------------------------------------------
  // message send
  // ------------------------------------------------------------------
  msg
    .command('send')
    .description('Send a message to an agent or broadcast')
    .requiredOption('--type <type>', 'Message type (e.g. question, status_update, notification)')
    .option('--to <agentId>', 'Recipient agent ID (omit for broadcast)')
    .option('--broadcast', 'Explicitly broadcast to entire workspace')
    .option('--payload <json>', 'JSON payload string', '{}')
    .option('--workspace <id>', 'Workspace ID')
    .action(async (opts, cmd) => {
      try {
        const api = buildApiClient(cmd.optsWithGlobals(), config);
        const wsId = resolveWorkspaceId(opts, config);

        let payload: Record<string, unknown>;
        try {
          payload = JSON.parse(opts.payload);
        } catch {
          console.error(chalk.red('Invalid JSON in --payload'));
          process.exit(1);
        }

        const message = await withSpinner(cmd, 'Sending message...', async () => {
          return api.sendMessage(wsId, {
            to_agent_id: opts.broadcast ? undefined : opts.to,
            message_type: opts.type as MessageType,
            payload,
          });
        });

        if (isJsonMode(cmd)) {
          console.log(JSON.stringify(message, null, 2));
          return;
        }

        const target = message.to_agent_id
          ? `agent ${chalk.cyan(message.to_agent_id.slice(0, 8))}`
          : chalk.gray('(broadcast)');
        console.log(chalk.green(`\nMessage sent to ${target}`));
        console.log(`  ID:   ${chalk.cyan(message.id)}`);
        console.log(`  Type: ${colorMsgType(message.message_type)}`);
      } catch (err) {
        handleError(err);
      }
    });

  // ------------------------------------------------------------------
  // message thread
  // ------------------------------------------------------------------
  msg
    .command('thread <threadId>')
    .description('View all messages in a thread')
    .option('--workspace <id>', 'Workspace ID')
    .action(async (threadId: string, opts, cmd) => {
      try {
        const api = buildApiClient(cmd.optsWithGlobals(), config);
        const wsId = resolveWorkspaceId(opts, config);

        const messages = await withSpinner(cmd, 'Fetching thread...', async () => {
          return api.getThread(wsId, threadId);
        });

        if (isJsonMode(cmd)) {
          console.log(JSON.stringify(messages, null, 2));
          return;
        }

        console.log(chalk.bold(`\nThread ${chalk.cyan(threadId.slice(0, 8))} (${messages.length} messages)`));
        const myId = config.getAgentId();
        for (const m of messages) {
          const who = m.from_agent_id === myId
            ? chalk.cyan('me')
            : chalk.yellow(m.from_agent_id.slice(0, 8));
          const time = new Date(m.created_at).toLocaleTimeString();
          console.log(`  ${chalk.dim(time)} ${who} [${colorMsgType(m.message_type)}]`);
          const preview = summarizePayload(m.payload);
          if (preview) {
            console.log(`    ${preview}`);
          }
        }
      } catch (err) {
        handleError(err);
      }
    });
}

// ====================================================================
// Helpers
// ====================================================================

function colorMsgType(t: string): string {
  if (t.startsWith('request_')) return chalk.yellow(t);
  if (t.startsWith('provide_')) return chalk.green(t);
  if (t.startsWith('report_')) return chalk.red(t);
  if (t.startsWith('resolve_')) return chalk.green(t);
  if (t === 'question') return chalk.yellow(t);
  if (t === 'answer') return chalk.green(t);
  if (t === 'status_update') return chalk.blue(t);
  return chalk.gray(t);
}

function summarizePayload(payload: Record<string, unknown>): string {
  const keys = Object.keys(payload);
  if (keys.length === 0) return '';
  // If there is a "message" or "text" or "content" key, show its value.
  for (const k of ['message', 'text', 'content', 'description', 'body']) {
    if (typeof payload[k] === 'string') {
      const val = payload[k] as string;
      return val.length > 80 ? val.slice(0, 77) + '...' : val;
    }
  }
  return `{${keys.join(', ')}}`;
}

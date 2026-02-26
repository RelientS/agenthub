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
import { AgentRole } from '../types';

export function registerWorkspaceCommands(program: Command, config: ConfigStore): void {
  const ws = program
    .command('workspace')
    .alias('ws')
    .description('Manage workspaces');

  // ------------------------------------------------------------------
  // workspace create
  // ------------------------------------------------------------------
  ws.command('create')
    .description('Create a new workspace and register as owner agent')
    .requiredOption('--name <name>', 'Workspace name')
    .option('--role <role>', 'Agent role (frontend|backend|fullstack|tester|devops)', 'fullstack')
    .option('--capabilities <caps>', 'Comma-separated capabilities', '')
    .option('--agent-name <agentName>', 'Your agent display name', 'cli-agent')
    .action(async (opts, cmd) => {
      try {
        const api = buildApiClient(cmd.optsWithGlobals(), config);
        const caps = opts.capabilities
          ? opts.capabilities.split(',').map((c: string) => c.trim()).filter(Boolean)
          : [];

        const result = await withSpinner(cmd, 'Creating workspace...', async () => {
          return api.createWorkspace({
            name: opts.name,
            agent_name: opts.agentName,
            agent_role: opts.role as AgentRole,
            agent_capabilities: caps,
          });
        });

        // Persist session.
        config.setToken(result.token);
        config.setAgentId(result.agent.id);
        config.setWorkspaceId(result.workspace.id);
        config.setLastSyncId(0);

        if (isJsonMode(cmd)) {
          console.log(JSON.stringify(result, null, 2));
          return;
        }

        console.log(chalk.green('\nWorkspace created successfully!'));
        console.log(`  ID:          ${chalk.cyan(result.workspace.id)}`);
        console.log(`  Name:        ${result.workspace.name}`);
        console.log(`  Invite Code: ${chalk.yellow(result.workspace.invite_code)}`);
        console.log(`  Agent ID:    ${chalk.cyan(result.agent.id)}`);
        console.log(`  Role:        ${result.agent.role}`);
      } catch (err) {
        handleError(err);
      }
    });

  // ------------------------------------------------------------------
  // workspace join
  // ------------------------------------------------------------------
  ws.command('join')
    .description('Join an existing workspace with an invite code')
    .requiredOption('--code <code>', 'Workspace invite code')
    .option('--role <role>', 'Agent role (frontend|backend|fullstack|tester|devops)', 'fullstack')
    .option('--capabilities <caps>', 'Comma-separated capabilities', '')
    .option('--agent-name <agentName>', 'Your agent display name', 'cli-agent')
    .action(async (opts, cmd) => {
      try {
        const api = buildApiClient(cmd.optsWithGlobals(), config);
        const caps = opts.capabilities
          ? opts.capabilities.split(',').map((c: string) => c.trim()).filter(Boolean)
          : [];

        const result = await withSpinner(cmd, 'Joining workspace...', async () => {
          return api.joinWorkspace({
            invite_code: opts.code,
            agent_name: opts.agentName,
            agent_role: opts.role as AgentRole,
            agent_capabilities: caps,
          });
        });

        config.setToken(result.token);
        config.setAgentId(result.agent.id);
        config.setWorkspaceId(result.workspace.id);
        config.setLastSyncId(0);

        if (isJsonMode(cmd)) {
          console.log(JSON.stringify(result, null, 2));
          return;
        }

        console.log(chalk.green('\nJoined workspace successfully!'));
        console.log(`  Workspace: ${result.workspace.name} (${chalk.cyan(result.workspace.id)})`);
        console.log(`  Agent ID:  ${chalk.cyan(result.agent.id)}`);
        console.log(`  Role:      ${result.agent.role}`);
      } catch (err) {
        handleError(err);
      }
    });

  // ------------------------------------------------------------------
  // workspace info
  // ------------------------------------------------------------------
  ws.command('info')
    .description('Show workspace details and connected agents')
    .option('--workspace <id>', 'Workspace ID (defaults to current)')
    .action(async (opts, cmd) => {
      try {
        const api = buildApiClient(cmd.optsWithGlobals(), config);
        const wsId = resolveWorkspaceId(opts, config);

        const [workspace, agents] = await withSpinner(
          cmd,
          'Fetching workspace info...',
          async () => {
            return Promise.all([
              api.getWorkspace(wsId),
              api.listAgents(wsId),
            ]);
          },
        );

        if (isJsonMode(cmd)) {
          console.log(JSON.stringify({ workspace, agents }, null, 2));
          return;
        }

        console.log(chalk.bold('\nWorkspace'));
        console.log(`  ID:          ${chalk.cyan(workspace.id)}`);
        console.log(`  Name:        ${workspace.name}`);
        console.log(`  Description: ${workspace.description || '(none)'}`);
        console.log(`  Status:      ${workspace.status}`);
        console.log(`  Invite Code: ${chalk.yellow(workspace.invite_code)}`);
        console.log(`  Created:     ${workspace.created_at}`);

        console.log(chalk.bold(`\nAgents (${agents.length})`));
        if (agents.length === 0) {
          console.log('  (no agents connected)');
        } else {
          const table = new Table({
            head: [
              chalk.white('ID'),
              chalk.white('Name'),
              chalk.white('Role'),
              chalk.white('Status'),
              chalk.white('Capabilities'),
            ],
          });
          for (const agent of agents) {
            const statusColor =
              agent.status === 'online'
                ? chalk.green
                : agent.status === 'busy'
                  ? chalk.yellow
                  : chalk.gray;
            table.push([
              agent.id.slice(0, 8),
              agent.name,
              agent.role,
              statusColor(agent.status),
              (agent.capabilities || []).join(', ') || '-',
            ]);
          }
          console.log(table.toString());
        }
      } catch (err) {
        handleError(err);
      }
    });

  // ------------------------------------------------------------------
  // workspace leave
  // ------------------------------------------------------------------
  ws.command('leave')
    .description('Leave the current workspace')
    .option('--workspace <id>', 'Workspace ID (defaults to current)')
    .action(async (opts, cmd) => {
      try {
        const api = buildApiClient(cmd.optsWithGlobals(), config);
        const wsId = resolveWorkspaceId(opts, config);

        await withSpinner(cmd, 'Leaving workspace...', async () => {
          return api.leaveWorkspace(wsId);
        });

        // Clear local session if we left the current workspace.
        const currentWs = config.getWorkspaceId();
        if (currentWs === wsId) {
          config.clear();
        }

        if (isJsonMode(cmd)) {
          console.log(JSON.stringify({ success: true }, null, 2));
          return;
        }

        console.log(chalk.green('Left workspace successfully.'));
      } catch (err) {
        handleError(err);
      }
    });
}

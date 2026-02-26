#!/usr/bin/env node

import { Command } from 'commander';
import { ConfigStore } from './config/store';
import { registerWorkspaceCommands } from './commands/workspace';
import { registerTaskCommands } from './commands/task';
import { registerMessageCommands } from './commands/message';
import { registerArtifactCommands } from './commands/artifact';
import { registerSyncCommands } from './commands/sync';

const config = new ConfigStore();

const program = new Command();

program
  .name('agenthub')
  .description('AgentHub CLI - Multi-Agent Collaborative Workspace')
  .version('1.0.0')
  .option('--json', 'Output in JSON format')
  .option('--server <url>', 'Override server URL');

// Register all command groups.
registerWorkspaceCommands(program, config);
registerTaskCommands(program, config);
registerMessageCommands(program, config);
registerArtifactCommands(program, config);
registerSyncCommands(program, config);

// -----------------------------------------------------------------
// config subcommand -- lightweight key/value management.
// -----------------------------------------------------------------
const cfg = program
  .command('config')
  .description('View or update CLI configuration');

cfg
  .command('show')
  .description('Display current configuration')
  .action(() => {
    const data = {
      server_url: config.getServerUrl(),
      token: config.getToken() ? '(set)' : '(not set)',
      agent_id: config.getAgentId() || '(not set)',
      workspace_id: config.getWorkspaceId() || '(not set)',
      last_sync_id: config.getLastSyncId(),
    };

    const globals = program.opts();
    if (globals.json) {
      // In JSON mode show actual token presence, not the token itself.
      console.log(JSON.stringify(data, null, 2));
    } else {
      console.log('\nAgentHub CLI Configuration');
      for (const [k, v] of Object.entries(data)) {
        console.log(`  ${k.padEnd(16)} ${v}`);
      }
      console.log('');
    }
  });

cfg
  .command('set <key> <value>')
  .description('Set a configuration value (server_url)')
  .action((key: string, value: string) => {
    switch (key) {
      case 'server_url':
        config.setServerUrl(value);
        console.log(`server_url set to ${value}`);
        break;
      default:
        console.error(`Unknown config key: ${key}. Supported keys: server_url`);
        process.exit(1);
    }
  });

cfg
  .command('reset')
  .description('Clear all stored configuration')
  .action(() => {
    config.clear();
    const globals = program.opts();
    if (globals.json) {
      console.log(JSON.stringify({ success: true }, null, 2));
    } else {
      console.log('Configuration cleared.');
    }
  });

// Parse and execute.
program.parseAsync(process.argv).catch((err) => {
  console.error(err);
  process.exit(1);
});

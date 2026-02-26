import * as fs from 'fs';
import * as path from 'path';
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
import { ArtifactType } from '../types';

export function registerArtifactCommands(program: Command, config: ConfigStore): void {
  const art = program
    .command('artifact')
    .alias('art')
    .description('Manage shared artifacts');

  // ------------------------------------------------------------------
  // artifact push
  // ------------------------------------------------------------------
  art
    .command('push')
    .description('Push a new artifact to the workspace')
    .requiredOption('--name <name>', 'Artifact name')
    .requiredOption('--type <type>', 'Artifact type (code_snippet|api_schema|type_definition|test_result|migration|config|doc)')
    .option('--file <path>', 'Read content from file')
    .option('--content <text>', 'Inline content (alternative to --file)')
    .option('--description <desc>', 'Artifact description')
    .option('--language <lang>', 'Programming language')
    .option('--tags <tags>', 'Comma-separated tags')
    .option('--workspace <id>', 'Workspace ID')
    .action(async (opts, cmd) => {
      try {
        const api = buildApiClient(cmd.optsWithGlobals(), config);
        const wsId = resolveWorkspaceId(opts, config);

        // Resolve content from --file or --content.
        let content: string;
        let filePath: string | undefined;
        if (opts.file) {
          const absPath = path.resolve(opts.file);
          if (!fs.existsSync(absPath)) {
            console.error(chalk.red(`File not found: ${absPath}`));
            process.exit(1);
          }
          content = fs.readFileSync(absPath, 'utf-8');
          filePath = opts.file;
        } else if (opts.content) {
          content = opts.content;
        } else {
          console.error(chalk.red('Either --file or --content is required.'));
          process.exit(1);
          return; // TypeScript control flow.
        }

        const tags = opts.tags
          ? opts.tags.split(',').map((t: string) => t.trim()).filter(Boolean)
          : undefined;

        const artifact = await withSpinner(cmd, 'Pushing artifact...', async () => {
          return api.createArtifact(wsId, {
            artifact_type: opts.type as ArtifactType,
            name: opts.name,
            description: opts.description,
            content,
            file_path: filePath,
            language: opts.language,
            tags,
          });
        });

        if (isJsonMode(cmd)) {
          console.log(JSON.stringify(artifact, null, 2));
          return;
        }

        console.log(chalk.green('\nArtifact pushed!'));
        console.log(`  ID:      ${chalk.cyan(artifact.id)}`);
        console.log(`  Name:    ${artifact.name}`);
        console.log(`  Type:    ${artifact.artifact_type}`);
        console.log(`  Version: ${artifact.version}`);
        console.log(`  Hash:    ${chalk.dim(artifact.content_hash)}`);
      } catch (err) {
        handleError(err);
      }
    });

  // ------------------------------------------------------------------
  // artifact list
  // ------------------------------------------------------------------
  art
    .command('list')
    .alias('ls')
    .description('List artifacts in the workspace')
    .option('--workspace <id>', 'Workspace ID')
    .option('--type <type>', 'Filter by artifact type')
    .option('--name <name>', 'Filter by name (substring match)')
    .action(async (opts, cmd) => {
      try {
        const api = buildApiClient(cmd.optsWithGlobals(), config);
        const wsId = resolveWorkspaceId(opts, config);

        const filters: Record<string, string | undefined> = {};
        if (opts.type) filters.artifact_type = opts.type;
        if (opts.name) filters.name = opts.name;

        const result = await withSpinner(cmd, 'Fetching artifacts...', async () => {
          return api.listArtifacts(wsId, filters as any);
        });

        if (isJsonMode(cmd)) {
          console.log(JSON.stringify(result, null, 2));
          return;
        }

        console.log(chalk.bold(`\nArtifacts (${result.total})`));
        if (result.artifacts.length === 0) {
          console.log('  (no artifacts found)');
          return;
        }

        const table = new Table({
          head: [
            chalk.white('ID'),
            chalk.white('Name'),
            chalk.white('Type'),
            chalk.white('Version'),
            chalk.white('Language'),
            chalk.white('Created'),
          ],
        });

        for (const a of result.artifacts) {
          table.push([
            a.id.slice(0, 8),
            a.name.length > 30 ? a.name.slice(0, 27) + '...' : a.name,
            colorArtifactType(a.artifact_type),
            String(a.version),
            a.language || '-',
            new Date(a.created_at).toLocaleDateString(),
          ]);
        }
        console.log(table.toString());
      } catch (err) {
        handleError(err);
      }
    });

  // ------------------------------------------------------------------
  // artifact pull
  // ------------------------------------------------------------------
  art
    .command('pull <artifactId>')
    .description('Pull (download) an artifact. Prints content or writes to file.')
    .option('--file <path>', 'Write content to file instead of stdout')
    .option('--workspace <id>', 'Workspace ID')
    .action(async (artifactId: string, opts, cmd) => {
      try {
        const api = buildApiClient(cmd.optsWithGlobals(), config);
        const wsId = resolveWorkspaceId(opts, config);

        const artifact = await withSpinner(cmd, 'Pulling artifact...', async () => {
          return api.getArtifact(wsId, artifactId);
        });

        if (isJsonMode(cmd)) {
          console.log(JSON.stringify(artifact, null, 2));
          return;
        }

        if (opts.file) {
          const absPath = path.resolve(opts.file);
          fs.writeFileSync(absPath, artifact.content, 'utf-8');
          console.log(chalk.green(`Artifact written to ${absPath}`));
          console.log(`  Name:    ${artifact.name}`);
          console.log(`  Version: ${artifact.version}`);
          return;
        }

        // Print metadata header then content.
        console.log(chalk.bold(`\n${artifact.name}`));
        console.log(`  ID:      ${chalk.cyan(artifact.id)}`);
        console.log(`  Type:    ${colorArtifactType(artifact.artifact_type)}`);
        console.log(`  Version: ${artifact.version}`);
        console.log(`  Hash:    ${chalk.dim(artifact.content_hash)}`);
        console.log(chalk.dim('--- content ---'));
        console.log(artifact.content);
      } catch (err) {
        handleError(err);
      }
    });

  // ------------------------------------------------------------------
  // artifact search
  // ------------------------------------------------------------------
  art
    .command('search')
    .description('Search artifacts by keyword')
    .requiredOption('--query <q>', 'Search query')
    .option('--workspace <id>', 'Workspace ID')
    .action(async (opts, cmd) => {
      try {
        const api = buildApiClient(cmd.optsWithGlobals(), config);
        const wsId = resolveWorkspaceId(opts, config);

        const artifacts = await withSpinner(cmd, 'Searching...', async () => {
          return api.searchArtifacts(wsId, opts.query);
        });

        if (isJsonMode(cmd)) {
          console.log(JSON.stringify(artifacts, null, 2));
          return;
        }

        console.log(chalk.bold(`\nSearch results for "${opts.query}" (${artifacts.length})`));
        if (artifacts.length === 0) {
          console.log('  (no matches)');
          return;
        }

        const table = new Table({
          head: [
            chalk.white('ID'),
            chalk.white('Name'),
            chalk.white('Type'),
            chalk.white('Version'),
          ],
        });
        for (const a of artifacts) {
          table.push([
            a.id.slice(0, 8),
            a.name,
            colorArtifactType(a.artifact_type),
            String(a.version),
          ]);
        }
        console.log(table.toString());
      } catch (err) {
        handleError(err);
      }
    });
}

// ====================================================================
// Helpers
// ====================================================================

function colorArtifactType(t: string): string {
  switch (t) {
    case 'code_snippet':
      return chalk.cyan(t);
    case 'api_schema':
      return chalk.yellow(t);
    case 'type_definition':
      return chalk.magenta(t);
    case 'test_result':
      return chalk.green(t);
    case 'migration':
      return chalk.blue(t);
    case 'config':
      return chalk.red(t);
    case 'doc':
      return chalk.white(t);
    default:
      return t;
  }
}

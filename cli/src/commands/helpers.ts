import chalk from 'chalk';
import ora, { Ora } from 'ora';
import Table from 'cli-table3';
import { Command } from 'commander';
import { ConfigStore } from '../config/store';
import { ApiClient } from '../client/api';

// Re-export for convenience in command modules.
export { chalk, ora, Table };
export type { Ora };

/**
 * Resolve the workspace ID from the --workspace option or the stored config.
 * Exits with an error if neither is available.
 */
export function resolveWorkspaceId(opts: { workspace?: string }, config: ConfigStore): string {
  const id = opts.workspace || config.getWorkspaceId();
  if (!id) {
    console.error(
      chalk.red('No workspace selected. Use --workspace <id> or run "agenthub workspace join" first.'),
    );
    process.exit(1);
  }
  return id;
}

/**
 * Build an ApiClient from the current ConfigStore, optionally overriding
 * the server URL via --server.
 */
export function buildApiClient(opts: { server?: string }, config: ConfigStore): ApiClient {
  const url = opts.server || config.getServerUrl();
  const token = config.getToken() || undefined;
  return new ApiClient(url, token);
}

/**
 * If --json is set on the command's root program, output data as JSON and
 * return true. Otherwise return false so the caller can render human output.
 */
export function outputJson(cmd: Command, data: unknown): boolean {
  const root = cmd.parent ?? cmd;
  if ((root as any).opts?.().json || (cmd as any).optsWithGlobals?.().json) {
    console.log(JSON.stringify(data, null, 2));
    return true;
  }
  return false;
}

/**
 * Check whether the --json flag is set anywhere in the command chain.
 */
export function isJsonMode(cmd: Command): boolean {
  try {
    const globals = (cmd as any).optsWithGlobals?.() ?? {};
    return !!globals.json;
  } catch {
    return false;
  }
}

/**
 * Run an async action wrapped in an ora spinner. If --json mode is active
 * the spinner is suppressed.
 */
export async function withSpinner<T>(
  cmd: Command,
  text: string,
  fn: (spinner: Ora) => Promise<T>,
): Promise<T> {
  if (isJsonMode(cmd)) {
    // Suppress spinner in JSON mode.
    const noop: Ora = ora({ isSilent: true });
    return fn(noop);
  }
  const spinner = ora(text).start();
  try {
    const result = await fn(spinner);
    spinner.succeed();
    return result;
  } catch (err: any) {
    spinner.fail(err.message || String(err));
    throw err;
  }
}

/**
 * Generic error handler wrapping command actions.
 */
export function handleError(err: unknown): never {
  if (err instanceof Error) {
    console.error(chalk.red(`Error: ${err.message}`));
  } else {
    console.error(chalk.red(`Error: ${String(err)}`));
  }
  process.exit(1);
}

import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';

/**
 * Configuration data persisted to ~/.agenthub/config.json.
 */
interface ConfigData {
  server_url: string;
  token: string | null;
  agent_id: string | null;
  workspace_id: string | null;
  last_sync_id: number;
}

const DEFAULT_CONFIG: ConfigData = {
  server_url: 'http://localhost:8080',
  token: null,
  agent_id: null,
  workspace_id: null,
  last_sync_id: 0,
};

/**
 * ConfigStore manages local CLI configuration stored as a JSON file
 * at ~/.agenthub/config.json. All reads and writes are synchronous
 * to keep the CLI simple and predictable.
 */
export class ConfigStore {
  private configDir: string;
  private configPath: string;
  private data: ConfigData;

  constructor() {
    this.configDir = path.join(os.homedir(), '.agenthub');
    this.configPath = path.join(this.configDir, 'config.json');
    this.data = this.load();
  }

  // ------------------------------------------------------------------
  // server_url
  // ------------------------------------------------------------------

  getServerUrl(): string {
    return this.data.server_url;
  }

  setServerUrl(url: string): void {
    this.data.server_url = url;
    this.save();
  }

  // ------------------------------------------------------------------
  // token
  // ------------------------------------------------------------------

  getToken(): string | null {
    return this.data.token;
  }

  setToken(token: string): void {
    this.data.token = token;
    this.save();
  }

  // ------------------------------------------------------------------
  // agent_id
  // ------------------------------------------------------------------

  getAgentId(): string | null {
    return this.data.agent_id;
  }

  setAgentId(id: string): void {
    this.data.agent_id = id;
    this.save();
  }

  // ------------------------------------------------------------------
  // workspace_id
  // ------------------------------------------------------------------

  getWorkspaceId(): string | null {
    return this.data.workspace_id;
  }

  setWorkspaceId(id: string): void {
    this.data.workspace_id = id;
    this.save();
  }

  // ------------------------------------------------------------------
  // last_sync_id
  // ------------------------------------------------------------------

  getLastSyncId(): number {
    return this.data.last_sync_id;
  }

  setLastSyncId(id: number): void {
    this.data.last_sync_id = id;
    this.save();
  }

  // ------------------------------------------------------------------
  // clear
  // ------------------------------------------------------------------

  clear(): void {
    this.data = { ...DEFAULT_CONFIG };
    this.save();
  }

  // ------------------------------------------------------------------
  // Internal helpers
  // ------------------------------------------------------------------

  private load(): ConfigData {
    try {
      if (fs.existsSync(this.configPath)) {
        const raw = fs.readFileSync(this.configPath, 'utf-8');
        const parsed = JSON.parse(raw) as Partial<ConfigData>;
        return { ...DEFAULT_CONFIG, ...parsed };
      }
    } catch {
      // Corrupted config -- fall through to defaults.
    }
    return { ...DEFAULT_CONFIG };
  }

  private save(): void {
    if (!fs.existsSync(this.configDir)) {
      fs.mkdirSync(this.configDir, { recursive: true });
    }
    fs.writeFileSync(this.configPath, JSON.stringify(this.data, null, 2), 'utf-8');
  }
}

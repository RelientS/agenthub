import WebSocket from 'ws';
import { WSMessageEnvelope } from '../types';

type EventCallback = (data: unknown) => void;

/**
 * WSClient manages a WebSocket connection to the AgentHub server
 * with automatic reconnection, heartbeat pings, and typed event
 * dispatching.
 */
export class WSClient {
  private ws: WebSocket | null = null;
  private listeners: Map<string, EventCallback[]> = new Map();
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null;
  private shouldReconnect = true;
  private reconnectDelay = 1000; // Start at 1 s, back off to 30 s max.
  private maxReconnectDelay = 30_000;

  private serverUrl = '';
  private token = '';
  private workspaceId = '';

  // ================================================================
  // Public API
  // ================================================================

  /**
   * Open a WebSocket connection. If a connection is already open it
   * will be closed first.
   */
  connect(serverUrl: string, token: string, workspaceId: string): void {
    this.serverUrl = serverUrl;
    this.token = token;
    this.workspaceId = workspaceId;
    this.shouldReconnect = true;
    this.doConnect();
  }

  /** Gracefully close the connection and stop reconnection attempts. */
  disconnect(): void {
    this.shouldReconnect = false;
    this.clearTimers();
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  /**
   * Register a callback for a specific event type. The callback
   * receives the parsed payload from the server envelope.
   */
  onEvent(eventType: string, callback: EventCallback): void {
    const existing = this.listeners.get(eventType) || [];
    existing.push(callback);
    this.listeners.set(eventType, existing);
  }

  /** Remove all listeners for a given event type. */
  offEvent(eventType: string): void {
    this.listeners.delete(eventType);
  }

  /**
   * Send an agent heartbeat through the open WebSocket so the server
   * keeps the agent marked as online.
   */
  sendHeartbeat(status: string, currentTask?: string): void {
    this.send({
      type: 'agent.heartbeat',
      workspace_id: this.workspaceId,
      payload: { status, current_task: currentTask },
    });
  }

  /** Returns true when the underlying WebSocket is in OPEN state. */
  isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }

  // ================================================================
  // Internals
  // ================================================================

  private doConnect(): void {
    if (this.ws) {
      this.ws.removeAllListeners();
      this.ws.close();
      this.ws = null;
    }

    const wsUrl = this.serverUrl.replace(/^http/, 'ws');
    const url = `${wsUrl}/api/v1/ws?token=${encodeURIComponent(this.token)}&workspace_id=${encodeURIComponent(this.workspaceId)}`;
    this.ws = new WebSocket(url);

    this.ws.on('open', () => {
      this.reconnectDelay = 1000; // Reset backoff.
      this.emit('ws.connected', null);
      this.startHeartbeatLoop();
    });

    this.ws.on('message', (raw: WebSocket.Data) => {
      try {
        const envelope = JSON.parse(raw.toString()) as WSMessageEnvelope;
        this.emit(envelope.type, envelope.payload);
      } catch {
        // Ignore malformed frames.
      }
    });

    this.ws.on('close', () => {
      this.emit('ws.disconnected', null);
      this.stopHeartbeatLoop();
      this.scheduleReconnect();
    });

    this.ws.on('error', (err: Error) => {
      this.emit('ws.error', { message: err.message });
      // The 'close' event will fire next and handle reconnection.
    });
  }

  private send(envelope: WSMessageEnvelope): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(envelope));
    }
  }

  private emit(eventType: string, data: unknown): void {
    const cbs = this.listeners.get(eventType);
    if (cbs) {
      for (const cb of cbs) {
        try {
          cb(data);
        } catch {
          // Swallow listener errors to keep the client alive.
        }
      }
    }
  }

  private scheduleReconnect(): void {
    if (!this.shouldReconnect) return;
    this.reconnectTimer = setTimeout(() => {
      this.doConnect();
    }, this.reconnectDelay);
    // Exponential backoff with cap.
    this.reconnectDelay = Math.min(this.reconnectDelay * 2, this.maxReconnectDelay);
  }

  private startHeartbeatLoop(): void {
    this.stopHeartbeatLoop();
    this.heartbeatTimer = setInterval(() => {
      this.sendHeartbeat('online');
    }, 30_000);
  }

  private stopHeartbeatLoop(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }

  private clearTimers(): void {
    this.stopHeartbeatLoop();
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }
}

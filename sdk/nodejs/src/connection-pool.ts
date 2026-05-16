import { Agent } from 'undici';
import { HttpsAgent as HttpsKeepAliveAgent } from 'agentkeepalive';
import * as https from 'https';
import * as http from 'http';

export interface ConnectionPoolConfig {
  maxConnections?: number;
  maxFreeSockets?: number;
  timeout?: number;
  keepAliveTimeout?: number;
}

const DEFAULT_CONFIG: ConnectionPoolConfig = {
  maxConnections: 100,
  maxFreeSockets: 10,
  timeout: 30000,
  keepAliveTimeout: 60000,
};

export class ConnectionPool {
  private httpAgent: http.Agent | undefined;
  private httpsAgent: https.Agent | undefined;
  private undiciAgent: Agent | undefined;
  private config: ConnectionPoolConfig;

  constructor(config?: ConnectionPoolConfig) {
    this.config = { ...DEFAULT_CONFIG, ...config };
  }

  getHttpAgent(): http.Agent {
    if (!this.httpAgent) {
      this.httpAgent = new http.Agent({
        keepAlive: true,
        maxSockets: this.config.maxConnections,
        maxFreeSockets: this.config.maxFreeSockets,
        timeout: this.config.timeout,
        keepAliveTimeout: this.config.keepAliveTimeout,
      });
    }
    return this.httpAgent;
  }

  getHttpsAgent(): https.Agent {
    if (!this.httpsAgent) {
      this.httpsAgent = new HttpsKeepAliveAgent({
        keepAlive: true,
        maxSockets: this.config.maxConnections,
        maxFreeSockets: this.config.maxFreeSockets,
        timeout: this.config.timeout,
        keepAliveTimeout: this.config.keepAliveTimeout,
      });
    }
    return this.httpsAgent;
  }

  getUndiciAgent(): Agent {
    if (!this.undiciAgent) {
      this.undiciAgent = new Agent({
        connections: this.config.maxConnections,
        keepAliveTimeout: this.config.keepAliveTimeout,
        keepAliveMaxTimeout: this.config.keepAliveTimeout,
      });
    }
    return this.undiciAgent;
  }

  getAgentForUrl(url: string): http.Agent | https.Agent | Agent {
    const parsedUrl = new URL(url);
    return parsedUrl.protocol === 'https:'
      ? this.getHttpsAgent()
      : this.getHttpAgent();
  }

  async destroy(): Promise<void> {
    if (this.httpAgent) {
      this.httpAgent.destroy();
    }
    if (this.httpsAgent) {
      (this.httpsAgent as any).destroy?.();
    }
    if (this.undiciAgent) {
      await this.undiciAgent.close();
    }
  }
}

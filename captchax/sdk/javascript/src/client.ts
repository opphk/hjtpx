import { HttpClient } from './http';
import {
  CaptchaXConfig,
  SliderCaptchaResult,
  SliderVerifyRequest,
  SliderVerifyResult,
  ClickCaptchaResult,
  ClickVerifyRequest,
  ClickVerifyResult,
  PuzzleCaptchaResult,
  PuzzleVerifyRequest,
  PuzzleVerifyResult,
  Scenario,
  Webhook,
  BatchVerifyRequest,
  BatchVerifyResponse,
  HealthStatus,
  CaptchaType,
  ClientInfo,
} from './types';

export class CaptchaXClient {
  private http: HttpClient;
  private appId?: string;
  private apiVersion: 'v1' | 'v2';

  constructor(config: CaptchaXConfig) {
    if (!config.baseUrl) {
      throw new Error('baseUrl is required');
    }

    this.http = new HttpClient(
      config.baseUrl,
      config.timeout,
      config.retryTimes
    );
    this.appId = config.appId;
    this.apiVersion = config.apiVersion || 'v1';

    if (this.appId) {
      this.http.setHeader('X-App-ID', this.appId);
    }
  }

  public setAppId(appId: string): void {
    this.appId = appId;
    this.http.setHeader('X-App-ID', appId);
  }

  public setApiVersion(version: 'v1' | 'v2'): void {
    this.apiVersion = version;
  }

  public getApiVersion(): 'v1' | 'v2' {
    return this.apiVersion;
  }

  private getApiPrefix(): string {
    return `/api/${this.apiVersion}`;
  }

  public async healthCheck(): Promise<HealthStatus> {
    return this.http.get<HealthStatus>('/health');
  }

  public async generateSliderCaptcha(
    options?: {
      width?: number;
      height?: number;
      clientInfo?: string;
      scenarioId?: string;
    }
  ): Promise<SliderCaptchaResult> {
    if (!this.appId) {
      throw new Error('appId is required for captcha generation');
    }

    const endpoint = `${this.getApiPrefix()}/captcha/slider`;
    const body: Record<string, unknown> = {
      app_id: this.appId,
    };

    if (options?.width) body.width = options.width;
    if (options?.height) body.height = options.height;
    if (options?.clientInfo) body.client_info = options.clientInfo;
    if (options?.scenarioId) body.scenario_id = options.scenarioId;

    return this.http.post<SliderCaptchaResult>(endpoint, body);
  }

  public async verifySliderCaptcha(
    captchaId: string,
    targetX: number,
    targetY?: number
  ): Promise<SliderVerifyResult> {
    const endpoint = `${this.getApiPrefix()}/captcha/slider/verify`;
    const body: SliderVerifyRequest = {
      captcha_id: captchaId,
      target_x: targetX,
      target_y: targetY,
    };

    return this.http.post<SliderVerifyResult>(endpoint, body);
  }

  public async generateClickCaptcha(
    options?: {
      charCount?: number;
      clientInfo?: string;
      scenarioId?: string;
    }
  ): Promise<ClickCaptchaResult> {
    if (!this.appId) {
      throw new Error('appId is required for captcha generation');
    }

    const endpoint = `${this.getApiPrefix()}/captcha/click`;
    const body: Record<string, unknown> = {
      app_id: this.appId,
    };

    if (options?.charCount) body.char_count = options.charCount;
    if (options?.clientInfo) body.client_info = options.clientInfo;
    if (options?.scenarioId) body.scenario_id = options.scenarioId;

    return this.http.post<ClickCaptchaResult>(endpoint, body);
  }

  public async verifyClickCaptcha(
    captchaId: string,
    clicks: ClickVerifyRequest['clicks']
  ): Promise<ClickVerifyResult> {
    const endpoint = `${this.getApiPrefix()}/captcha/click/verify`;
    const body: ClickVerifyRequest = {
      captcha_id: captchaId,
      clicks,
    };

    return this.http.post<ClickVerifyResult>(endpoint, body);
  }

  public async generatePuzzleCaptcha(
    options?: {
      width?: number;
      height?: number;
      clientInfo?: string;
      scenarioId?: string;
    }
  ): Promise<PuzzleCaptchaResult> {
    if (!this.appId) {
      throw new Error('appId is required for captcha generation');
    }

    const endpoint = `${this.getApiPrefix()}/captcha/puzzle`;
    const body: Record<string, unknown> = {
      app_id: this.appId,
    };

    if (options?.width) body.width = options.width;
    if (options?.height) body.height = options.height;
    if (options?.clientInfo) body.client_info = options.clientInfo;
    if (options?.scenarioId) body.scenario_id = options.scenarioId;

    return this.http.post<PuzzleCaptchaResult>(endpoint, body);
  }

  public async verifyPuzzleCaptcha(
    captchaId: string,
    targetX: number,
    targetY?: number
  ): Promise<PuzzleVerifyResult> {
    const endpoint = `${this.getApiPrefix()}/captcha/puzzle/verify`;
    const body: PuzzleVerifyRequest = {
      captcha_id: captchaId,
      target_x: targetX,
      target_y: targetY,
    };

    return this.http.post<PuzzleVerifyResult>(endpoint, body);
  }

  public async batchVerify(
    items: BatchVerifyRequest['items'],
    options?: {
      deduplicationId?: string;
    }
  ): Promise<BatchVerifyResponse> {
    const endpoint = `${this.getApiPrefix()}/captcha/batch/verify`;
    const body: BatchVerifyRequest = { items };

    return this.http.post<BatchVerifyResponse>(endpoint, body, {
      deduplicationId: options?.deduplicationId,
    });
  }

  public async listScenarios(): Promise<{ scenarios: Scenario[]; total: number }> {
    return this.http.get<{ scenarios: Scenario[]; total: number }>(
      `${this.getApiPrefix()}/captcha/scenarios`
    );
  }

  public async createScenario(
    scenario: Omit<Scenario, 'id' | 'created_at' | 'updated_at'>
  ): Promise<Scenario> {
    return this.http.post<Scenario>(
      `${this.getApiPrefix()}/captcha/scenarios`,
      scenario
    );
  }

  public async getScenario(id: string): Promise<Scenario> {
    return this.http.get<Scenario>(
      `${this.getApiPrefix()}/captcha/scenarios/${id}`
    );
  }

  public async updateScenario(
    id: string,
    updates: Partial<Omit<Scenario, 'id' | 'created_at' | 'updated_at'>>
  ): Promise<Scenario> {
    return this.http.put<Scenario>(
      `${this.getApiPrefix()}/captcha/scenarios/${id}`,
      updates
    );
  }

  public async deleteScenario(id: string): Promise<{ deleted: boolean }> {
    return this.http.delete<{ deleted: boolean }>(
      `${this.getApiPrefix()}/captcha/scenarios/${id}`
    );
  }

  public async registerWebhook(
    webhook: Omit<Webhook, 'id' | 'created_at' | 'updated_at' | 'enabled'>
  ): Promise<Webhook> {
    return this.http.post<Webhook>(
      `${this.getApiPrefix()}/captcha/webhook/register`,
      webhook
    );
  }

  public async listWebhooks(
    options?: { appId?: string }
  ): Promise<{ webhooks: Webhook[]; total: number }> {
    const endpoint = options?.appId
      ? `${this.getApiPrefix()}/captcha/webhook?app_id=${encodeURIComponent(options.appId)}`
      : `${this.getApiPrefix()}/captcha/webhook`;

    return this.http.get<{ webhooks: Webhook[]; total: number }>(endpoint);
  }

  public async updateWebhook(
    id: string,
    updates: Partial<Omit<Webhook, 'id' | 'created_at' | 'updated_at'>>
  ): Promise<Webhook> {
    return this.http.put<Webhook>(
      `${this.getApiPrefix()}/captcha/webhook/${id}`,
      updates
    );
  }

  public async unregisterWebhook(id: string): Promise<{ deleted: boolean }> {
    return this.http.delete<{ deleted: boolean }>(
      `${this.getApiPrefix()}/captcha/webhook/${id}`
    );
  }

  public createClientInfo(info?: Partial<ClientInfo>): string {
    return JSON.stringify({
      userAgent: typeof navigator !== 'undefined' ? navigator.userAgent : undefined,
      platform: typeof navigator !== 'undefined' ? navigator.platform : undefined,
      timestamp: Date.now(),
      ...info,
    });
  }

  public async generateAndVerifySlider(
    options?: {
      width?: number;
      height?: number;
      clientInfo?: string;
      scenarioId?: string;
    },
    verifyCallback?: (result: SliderVerifyResult) => boolean | Promise<boolean>
  ): Promise<{ captcha: SliderCaptchaResult; verifyResult: SliderVerifyResult }> {
    const captcha = await this.generateSliderCaptcha(options);
    const verifyResult = await this.verifySliderCaptcha(
      captcha.id,
      captcha.target_x,
      captcha.target_y
    );

    if (verifyCallback && !(await verifyCallback(verifyResult))) {
      throw new Error('Slider verification failed validation');
    }

    return { captcha, verifyResult };
  }

  public async generateAndVerifyClick(
    options?: {
      charCount?: number;
      clientInfo?: string;
      scenarioId?: string;
    },
    verifyCallback?: (result: ClickVerifyResult) => boolean | Promise<boolean>
  ): Promise<{ captcha: ClickCaptchaResult; verifyResult: ClickVerifyResult }> {
    const captcha = await this.generateClickCaptcha(options);
    const clicks = captcha.char_positions.map((pos) => ({
      char: pos.char,
      x: pos.x,
      y: pos.y,
    }));
    const verifyResult = await this.verifyClickCaptcha(captcha.id, clicks);

    if (verifyCallback && !(await verifyCallback(verifyResult))) {
      throw new Error('Click verification failed validation');
    }

    return { captcha, verifyResult };
  }
}

export default CaptchaXClient;

/**
 * System 资源类
 * 系统配置和管理功能
 */

import { BaseResource, ClientOptions } from "./base";

export interface SystemConfig {
  key: string;
  value: any;
  updated_at?: string;
}

export interface SystemInfo {
  version: string;
  go_version: string;
  os: string;
  arch: string;
}

export interface SystemHealth {
  status: string;
  uptime: number;
  memory_mb: number;
}

export interface SystemStats {
  requests_total: number;
  requests_success: number;
  requests_error: number;
  avg_response_time_ms: number;
}

/**
 * System 资源类
 */
export class SystemResource extends BaseResource {
  constructor(options: ClientOptions) {
    super(options);
  }

  // Config Management
  async listConfig(): Promise<SystemConfig[]> {
    return this.request<SystemConfig[]>("/v1/system/config");
  }

  async getConfig(key: string): Promise<SystemConfig> {
    return this.request<SystemConfig>(`/v1/system/config/${key}`);
  }

  async updateConfig(key: string, value: any): Promise<SystemConfig> {
    return this.request<SystemConfig>(`/v1/system/config/${key}`, {
      method: "PUT",
      body: { value },
    });
  }

  async deleteConfig(key: string): Promise<void> {
    await this.request<void>(`/v1/system/config/${key}`, {
      method: "DELETE",
    });
  }

  // System Operations
  async getInfo(): Promise<SystemInfo> {
    return this.request<SystemInfo>("/v1/system/info");
  }

  async getHealth(): Promise<SystemHealth> {
    return this.request<SystemHealth>("/v1/system/health");
  }

  async getStats(): Promise<SystemStats> {
    return this.request<SystemStats>("/v1/system/stats");
  }

  async reload(): Promise<{ reloaded: boolean }> {
    return this.request("/v1/system/reload", { method: "POST" });
  }

  async runGC(): Promise<{ gc_completed: boolean; freed_mb: number }> {
    return this.request("/v1/system/gc", { method: "POST" });
  }

  async backup(): Promise<{ backup_started: boolean; backup_id: string }> {
    return this.request("/v1/system/backup", { method: "POST" });
  }
}

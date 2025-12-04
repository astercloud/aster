/**
 * BaseResource - 所有资源类的基类
 * 提供统一的 HTTP 请求、重试、错误处理等功能
 */

/**
 * 客户端选项
 */
export interface ClientOptions {
  /** API 基础 URL */
  baseUrl: string;
  /** API Key（可选） */
  apiKey?: string;
  /** 超时时间（毫秒） */
  timeout?: number;
  /** 重试配置 */
  retry?: RetryOptions;
  /** 自定义 headers */
  headers?: Record<string, string>;
  /** fetch 实现（用于测试） */
  fetchImpl?: typeof fetch;
}

/**
 * 重试选项
 */
export interface RetryOptions {
  /** 最大重试次数 */
  maxRetries?: number;
  /** 可重试的状态码 */
  retryableStatusCodes?: number[];
  /** 退避倍数 */
  backoffMultiplier?: number;
  /** 最大退避时间（毫秒） */
  maxBackoffTime?: number;
}

/**
 * 请求选项
 */
export interface RequestOptions {
  /** HTTP 方法 */
  method?: "GET" | "POST" | "PUT" | "DELETE" | "PATCH";
  /** 请求体 */
  body?: any;
  /** 查询参数 */
  params?: Record<string, any>;
  /** 自定义 headers */
  headers?: Record<string, string>;
  /** 超时时间（覆盖全局配置） */
  timeout?: number;
  /** AbortSignal */
  signal?: AbortSignal;
}

/**
 * BaseResource 基类
 * 所有资源类都继承此类
 */
export class BaseResource {
  protected readonly options: ClientOptions;
  protected readonly fetchImpl: typeof fetch;

  // 默认配置
  private readonly defaultRetry: Required<RetryOptions> = {
    maxRetries: 3,
    retryableStatusCodes: [408, 429, 500, 502, 503, 504],
    backoffMultiplier: 2,
    maxBackoffTime: 30000, // 30秒
  };

  constructor(options: ClientOptions) {
    this.options = {
      timeout: 120000, // 默认 120 秒
      ...options,
      retry: {
        ...this.defaultRetry,
        ...options.retry,
      },
    };

    this.fetchImpl = options.fetchImpl ?? fetch;
  }

  /**
   * 发送 HTTP 请求
   * 自动处理重试、超时、错误
   */
  protected async request<T>(
    path: string,
    options: RequestOptions = {},
  ): Promise<T> {
    const url = this.buildUrl(path, options.params);
    const timeout = options.timeout ?? this.options.timeout;

    let lastError: Error | null = null;
    const maxRetries = this.options.retry!.maxRetries!;

    // 重试循环
    for (let attempt = 0; attempt <= maxRetries; attempt++) {
      try {
        // 创建 AbortController 用于超时
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), timeout);

        try {
          const response = await this.fetchImpl(url, {
            method: options.method ?? "GET",
            headers: this.buildHeaders(options.headers),
            body: options.body ? JSON.stringify(options.body) : undefined,
            signal: options.signal ?? controller.signal,
          });

          clearTimeout(timeoutId);

          // 检查是否需要重试
          if (
            !response.ok &&
            this.shouldRetry(response.status) &&
            attempt < maxRetries
          ) {
            lastError = new Error(
              `HTTP ${response.status}: ${response.statusText}`,
            );
            await this.delay(this.getBackoffDelay(attempt));
            continue;
          }

          // 处理响应
          return await this.handleResponse<T>(response);
        } finally {
          clearTimeout(timeoutId);
        }
      } catch (error: any) {
        lastError = error;

        // 超时或网络错误，可以重试
        if (attempt < maxRetries && this.isRetryableError(error)) {
          await this.delay(this.getBackoffDelay(attempt));
          continue;
        }

        // 不可重试，直接抛出
        throw this.wrapError(error, path);
      }
    }

    // 所有重试都失败了
    throw this.wrapError(lastError!, path);
  }

  /**
   * 构建完整 URL
   */
  private buildUrl(path: string, params?: Record<string, any>): string {
    const baseUrl = this.options.baseUrl.replace(/\/+$/, "");
    const fullPath = path.startsWith("/") ? path : `/${path}`;
    let url = `${baseUrl}${fullPath}`;

    // 添加查询参数
    if (params) {
      const searchParams = new URLSearchParams();
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          searchParams.append(key, String(value));
        }
      });
      const queryString = searchParams.toString();
      if (queryString) {
        url += `?${queryString}`;
      }
    }

    return url;
  }

  /**
   * 构建请求 headers
   */
  private buildHeaders(
    customHeaders?: Record<string, string>,
  ): Record<string, string> {
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      ...this.options.headers,
      ...customHeaders,
    };

    // 添加 API Key
    if (this.options.apiKey) {
      headers["Authorization"] = `Bearer ${this.options.apiKey}`;
    }

    return headers;
  }

  /**
   * 处理 HTTP 响应
   */
  private async handleResponse<T>(response: Response): Promise<T> {
    // 处理 204 No Content
    if (response.status === 204) {
      return undefined as T;
    }

    // 尝试解析 JSON
    const text = await response.text();
    let data: any;

    try {
      data = text ? JSON.parse(text) : undefined;
    } catch (error) {
      // 不是有效的 JSON
      if (!response.ok) {
        throw new Error(
          `HTTP ${response.status}: ${text || response.statusText}`,
        );
      }
      return text as T;
    }

    // 检查错误
    if (!response.ok) {
      const errorMessage = data?.error || data?.message || response.statusText;
      throw new Error(`HTTP ${response.status}: ${errorMessage}`);
    }

    // 解包 {success: true, data: {...}} 格式的响应
    if (
      data &&
      typeof data === "object" &&
      "success" in data &&
      "data" in data
    ) {
      return data.data as T;
    }

    return data as T;
  }

  /**
   * 判断状态码是否应该重试
   */
  private shouldRetry(statusCode: number): boolean {
    return this.options.retry!.retryableStatusCodes!.includes(statusCode);
  }

  /**
   * 判断错误是否可重试
   */
  private isRetryableError(error: any): boolean {
    // 超时、网络错误等
    return (
      error.name === "AbortError" ||
      error.name === "TimeoutError" ||
      error.message?.includes("fetch failed") ||
      error.message?.includes("network") ||
      error.message?.includes("ECONNREFUSED")
    );
  }

  /**
   * 计算退避延迟
   * 使用指数退避算法
   */
  private getBackoffDelay(attempt: number): number {
    const { backoffMultiplier, maxBackoffTime } = this.options.retry!;
    const delay = Math.min(
      1000 * Math.pow(backoffMultiplier!, attempt),
      maxBackoffTime!,
    );
    // 添加随机抖动（±25%）
    const jitter = delay * (0.75 + Math.random() * 0.5);
    return Math.floor(jitter);
  }

  /**
   * 延迟函数
   */
  private delay(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  /**
   * 包装错误
   */
  private wrapError(error: Error, path: string): Error {
    const message = `[AsterClient] Request failed for ${path}: ${error.message}`;
    const wrappedError = new Error(message);
    // 保存原始错误信息（兼容旧版 TypeScript）
    (wrappedError as any).originalError = error;
    return wrappedError;
  }

  /**
   * 获取基础 URL
   */
  protected getBaseUrl(): string {
    return this.options.baseUrl;
  }
}

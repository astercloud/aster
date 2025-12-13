import type {
  ApiResponse,
  OverviewStats,
  TraceListResult,
  TraceDetail,
  TokenUsageStats,
  CostBreakdown,
  PerformanceStats,
  Insight,
  EventsResult,
  ModelPricing,
  TraceQueryOpts,
  TokenQueryOpts,
  CostQueryOpts,
  SessionListResult,
  SessionDetail,
  SessionQueryOpts,
  AgentRecord,
  CreateAgentRequest,
  UpdateAgentRequest,
  EvalRecord,
  EvalListResult,
  EvalQueryOpts,
  RunTextEvalRequest,
  RunSessionEvalRequest,
  RunBatchEvalRequest,
  RunCustomEvalRequest,
  BenchmarkRecord,
  BenchmarkListResult,
  CreateBenchmarkRequest,
  RunBenchmarkRequest,
} from './types';

// API base URL - uses relative path when embedded, absolute for dev
const API_BASE = import.meta.env.VITE_API_URL || '/v1';

class ApiClient {
  private baseUrl: string;

  constructor(baseUrl: string = API_BASE) {
    this.baseUrl = baseUrl;
  }

  private async fetch<T>(path: string, options?: RequestInit): Promise<T> {
    const url = `${this.baseUrl}${path}`;

    const response = await fetch(url, {
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers,
      },
      ...options,
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({
        error: { code: 'unknown', message: response.statusText }
      }));
      throw new Error(error.error?.message || 'Request failed');
    }

    const result: ApiResponse<T> = await response.json();

    if (!result.success) {
      throw new Error(result.error?.message || 'Request failed');
    }

    return result.data as T;
  }

  private buildQueryString(params: object): string {
    const searchParams = new URLSearchParams();
    for (const [key, value] of Object.entries(params)) {
      if (value !== undefined && value !== null && value !== '') {
        searchParams.append(key, String(value));
      }
    }
    const queryString = searchParams.toString();
    return queryString ? `?${queryString}` : '';
  }

  // Overview
  async getOverview(period: string = '24h'): Promise<OverviewStats> {
    return this.fetch<OverviewStats>(`/dashboard/overview?period=${period}`);
  }

  // Traces
  async getTraces(opts: TraceQueryOpts = {}): Promise<TraceListResult> {
    const query = this.buildQueryString(opts);
    return this.fetch<TraceListResult>(`/dashboard/traces${query}`);
  }

  async getTrace(id: string): Promise<TraceDetail> {
    return this.fetch<TraceDetail>(`/dashboard/traces/${id}`);
  }

  // Metrics
  async getTokenUsage(opts: TokenQueryOpts = {}): Promise<TokenUsageStats> {
    const query = this.buildQueryString(opts);
    return this.fetch<TokenUsageStats>(`/dashboard/metrics/tokens${query}`);
  }

  async getCosts(opts: CostQueryOpts = {}): Promise<CostBreakdown> {
    const query = this.buildQueryString(opts);
    return this.fetch<CostBreakdown>(`/dashboard/metrics/costs${query}`);
  }

  async getPerformance(period: string = '24h'): Promise<PerformanceStats> {
    return this.fetch<PerformanceStats>(`/dashboard/metrics/performance?period=${period}`);
  }

  // Events
  async getRecentEvents(limit: number = 100): Promise<EventsResult> {
    return this.fetch<EventsResult>(`/dashboard/events?limit=${limit}`);
  }

  async getEventsSince(cursor: number): Promise<EventsResult> {
    return this.fetch<EventsResult>(`/dashboard/events/since/${cursor}`);
  }

  // Insights
  async getInsights(): Promise<Insight[]> {
    return this.fetch<Insight[]>('/dashboard/insights');
  }

  // Pricing
  async getPricing(): Promise<Record<string, ModelPricing>> {
    return this.fetch<Record<string, ModelPricing>>('/dashboard/pricing');
  }

  async updatePricing(pricing: Partial<ModelPricing> & { model: string }): Promise<{ model: string; pricing: ModelPricing }> {
    return this.fetch('/dashboard/pricing', {
      method: 'PUT',
      body: JSON.stringify(pricing),
    });
  }

  // Sessions
  async getSessions(opts: SessionQueryOpts = {}): Promise<SessionListResult> {
    const query = this.buildQueryString(opts);
    return this.fetch<SessionListResult>(`/dashboard/sessions${query}`);
  }

  async getSession(id: string): Promise<SessionDetail> {
    return this.fetch<SessionDetail>(`/dashboard/sessions/${id}`);
  }

  // Agents
  async getAgents(): Promise<AgentRecord[]> {
    return this.fetch<AgentRecord[]>('/agents');
  }

  async getAgent(id: string): Promise<AgentRecord> {
    return this.fetch<AgentRecord>(`/agents/${id}`);
  }

  async createAgent(req: CreateAgentRequest): Promise<AgentRecord> {
    return this.fetch<AgentRecord>('/agents', {
      method: 'POST',
      body: JSON.stringify(req),
    });
  }

  async updateAgent(id: string, req: UpdateAgentRequest): Promise<AgentRecord> {
    return this.fetch<AgentRecord>(`/agents/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(req),
    });
  }

  async deleteAgent(id: string): Promise<void> {
    await fetch(`${this.baseUrl}/agents/${id}`, {
      method: 'DELETE',
    });
  }

  // Evaluations
  async getEvals(opts: EvalQueryOpts = {}): Promise<EvalListResult> {
    const query = this.buildQueryString(opts);
    return this.fetch<EvalListResult>(`/eval/evals${query}`);
  }

  async getEval(id: string): Promise<EvalRecord> {
    return this.fetch<EvalRecord>(`/eval/evals/${id}`);
  }

  async runTextEval(req: RunTextEvalRequest): Promise<EvalRecord> {
    return this.fetch<EvalRecord>('/eval/text', {
      method: 'POST',
      body: JSON.stringify(req),
    });
  }

  async runSessionEval(req: RunSessionEvalRequest): Promise<EvalRecord> {
    return this.fetch<EvalRecord>('/eval/session', {
      method: 'POST',
      body: JSON.stringify(req),
    });
  }

  async runBatchEval(req: RunBatchEvalRequest): Promise<EvalRecord> {
    return this.fetch<EvalRecord>('/eval/batch', {
      method: 'POST',
      body: JSON.stringify(req),
    });
  }

  async runCustomEval(req: RunCustomEvalRequest): Promise<EvalRecord> {
    return this.fetch<EvalRecord>('/eval/custom', {
      method: 'POST',
      body: JSON.stringify(req),
    });
  }

  async deleteEval(id: string): Promise<void> {
    await fetch(`${this.baseUrl}/eval/evals/${id}`, {
      method: 'DELETE',
    });
  }

  // Benchmarks
  async getBenchmarks(): Promise<BenchmarkListResult> {
    return this.fetch<BenchmarkListResult>('/eval/benchmarks');
  }

  async getBenchmark(id: string): Promise<BenchmarkRecord> {
    return this.fetch<BenchmarkRecord>(`/eval/benchmarks/${id}`);
  }

  async createBenchmark(req: CreateBenchmarkRequest): Promise<BenchmarkRecord> {
    return this.fetch<BenchmarkRecord>('/eval/benchmarks', {
      method: 'POST',
      body: JSON.stringify(req),
    });
  }

  async runBenchmark(id: string, req: RunBenchmarkRequest = {}): Promise<BenchmarkRecord> {
    return this.fetch<BenchmarkRecord>(`/eval/benchmarks/${id}/run`, {
      method: 'POST',
      body: JSON.stringify(req),
    });
  }

  async deleteBenchmark(id: string): Promise<void> {
    await fetch(`${this.baseUrl}/eval/benchmarks/${id}`, {
      method: 'DELETE',
    });
  }
}

// Export singleton instance
export const api = new ApiClient();

// Export class for custom instances
export { ApiClient };

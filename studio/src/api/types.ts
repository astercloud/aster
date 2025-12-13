// API Response types
export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
  };
}

// Overview types
export interface OverviewStats {
  active_agents: number;
  active_sessions: number;
  total_requests: number;
  token_usage: TokenCount;
  cost: CostAmount;
  error_rate: number;
  avg_latency_ms: number;
  period: string;
  updated_at: string;
}

export interface TokenCount {
  input: number;
  output: number;
  total: number;
}

export interface CostAmount {
  amount: number;
  currency: string;
}

// Trace types
export type TraceStatus = 'ok' | 'error' | 'running';
export type TraceNodeType = 'agent' | 'llm' | 'tool' | 'middleware';

export interface TraceSummary {
  id: string;
  name: string;
  agent_id?: string;
  agent_name?: string;
  start_time: string;
  duration_ms: number;
  status: TraceStatus;
  span_count: number;
  token_usage: TokenCount;
  error_message?: string;
}

export interface TraceNode {
  id: string;
  name: string;
  type: TraceNodeType;
  start_time: string;
  end_time?: string;
  duration_ms: number;
  status: TraceStatus;
  attributes?: Record<string, unknown>;
  children?: TraceNode[];
}

export interface TraceDetail extends TraceSummary {
  root_span?: TraceNode;
  cost: CostAmount;
}

export interface TraceListResult {
  traces: TraceSummary[];
  total: number;
  has_more: boolean;
}

// Token usage types
export interface TokenUsageStats {
  period: string;
  total: TokenCount;
  by_agent?: Record<string, TokenCount>;
  by_model?: Record<string, TokenCount>;
  trend?: TokenTrendPoint[];
  cost: CostAmount;
}

export interface TokenTrendPoint {
  timestamp: string;
  input: number;
  output: number;
}

// Cost types
export interface CostBreakdown {
  period: string;
  total: CostAmount;
  by_agent?: Record<string, CostAmount>;
  by_model?: Record<string, CostAmount>;
  trend?: CostTrendPoint[];
}

export interface CostTrendPoint {
  timestamp: string;
  amount: number;
}

// Performance types
export interface PerformanceStats {
  period: string;
  ttft: LatencyPercentiles;
  tpot: LatencyPercentiles;
  tool_latency: Record<string, LatencyPercentiles>;
  avg_loop_count: number;
  request_count: number;
  error_count: number;
  error_rate: number;
}

export interface LatencyPercentiles {
  p50: number;
  p95: number;
  p99: number;
  avg: number;
  max: number;
}

// Insight types
export type InsightType = 'performance' | 'cost' | 'reliability' | 'usage';

export interface Insight {
  id: string;
  type: InsightType;
  severity: 'info' | 'warning' | 'critical';
  title: string;
  description: string;
  suggestion: string;
  data?: Record<string, unknown>;
  created_at: string;
}

// Event types
export interface EventItem {
  cursor: number;
  timestamp: string;
  event: unknown;
}

export interface EventsResult {
  events: EventItem[];
  cursor: number;
  next_cursor?: number;
  message?: string;
}

// WebSocket Event Stream types
export type EventChannel = 'progress' | 'control' | 'monitor';

export interface StreamEventFilters {
  channels?: EventChannel[];
  event_types?: string[];
  agent_ids?: string[];
  min_level?: 'debug' | 'info' | 'warn' | 'error';
}

export interface StreamEvent {
  type: 'event' | 'heartbeat' | 'subscribed' | 'filtered' | 'stats' | 'error';
  timestamp: string;
  agent_id?: string;
  channel?: EventChannel;
  event_type?: string;
  data?: Record<string, unknown>;
  message?: string;
  stats?: EventStreamStats;
}

export interface EventStreamStats {
  connected_at: string;
  subscribed_agents: number;
  events_received: number;
  events_filtered: number;
}

// Pricing types
export interface ModelPricing {
  model: string;
  input_price_per_m: number;
  output_price_per_m: number;
  currency: string;
}

// Query options
export interface TraceQueryOpts {
  start?: string;
  end?: string;
  status?: string;
  agent_id?: string;
  limit?: number;
  offset?: number;
}

export interface TokenQueryOpts {
  period?: string;
  start?: string;
  end?: string;
  agent_id?: string;
  model?: string;
}

export interface CostQueryOpts {
  period?: string;
  start?: string;
  end?: string;
  agent_id?: string;
}

// Session types
export type SessionStatus = 'active' | 'completed' | 'suspended';

export interface SessionSummary {
  id: string;
  agent_id?: string;
  agent_name?: string;
  status: SessionStatus;
  message_count: number;
  token_usage: TokenCount;
  created_at: string;
  updated_at: string;
  metadata?: Record<string, unknown>;
}

export interface SessionMessage {
  id?: string;
  role: string;
  content: string;
  timestamp: string;
  metadata?: Record<string, unknown>;
}

export interface SessionDetail extends SessionSummary {
  messages: SessionMessage[];
}

export interface SessionListResult {
  sessions: SessionSummary[];
  total: number;
  has_more: boolean;
}

export interface SessionQueryOpts {
  status?: string;
  agent_id?: string;
  limit?: number;
  offset?: number;
}

// Agent types
export type AgentStatus = 'active' | 'disabled' | 'archived';

export interface ModelConfig {
  provider?: string;
  model?: string;
  api_key?: string;
  base_url?: string;
  temperature?: number;
  max_tokens?: number;
  system_prompt?: string;
}

export interface SandboxConfig {
  type?: string;
  working_dir?: string;
  allowed_paths?: string[];
  env?: Record<string, string>;
}

export interface AgentConfig {
  agent_id?: string;
  template_id: string;
  template_version?: string;
  model_config?: ModelConfig;
  sandbox?: SandboxConfig;
  tools?: string[];
  middlewares?: string[];
  middleware_config?: Record<string, Record<string, unknown>>;
  expose_thinking?: boolean;
  routing_profile?: string;
  metadata?: Record<string, unknown>;
}

export interface AgentRecord {
  id: string;
  config: AgentConfig;
  status: AgentStatus;
  created_at: string;
  updated_at: string;
  metadata?: Record<string, unknown>;
}

export interface AgentListResult {
  agents: AgentRecord[];
}

export interface CreateAgentRequest {
  template_id: string;
  name?: string;
  model_config?: ModelConfig;
  sandbox?: SandboxConfig;
  middlewares?: string[];
  middleware_config?: Record<string, Record<string, unknown>>;
  metadata?: Record<string, unknown>;
}

export interface UpdateAgentRequest {
  name?: string;
  metadata?: Record<string, unknown>;
}

// Evaluation types
export type EvalType = 'text' | 'session' | 'batch' | 'custom';
export type EvalStatus = 'pending' | 'running' | 'completed' | 'failed';

export interface EvalMetrics {
  accuracy?: number;
  precision?: number;
  recall?: number;
  f1_score?: number;
  latency_ms?: number;
  token_count?: number;
  cost?: number;
  custom?: Record<string, number>;
}

export interface EvalRecord {
  id: string;
  name: string;
  type: EvalType;
  status: EvalStatus;
  input: Record<string, unknown>;
  output?: Record<string, unknown>;
  metrics?: EvalMetrics;
  score?: number;
  started_at: string;
  completed_at?: string;
  duration_ms?: number;
  error?: string;
  metadata?: Record<string, unknown>;
}

export interface EvalListResult {
  evals: EvalRecord[];
  total: number;
  has_more: boolean;
}

export interface EvalQueryOpts {
  type?: EvalType;
  status?: EvalStatus;
  limit?: number;
  offset?: number;
}

export interface RunTextEvalRequest {
  name?: string;
  prompt: string;
  expected?: string;
  agent_id?: string;
  model_config?: ModelConfig;
}

export interface RunSessionEvalRequest {
  name?: string;
  session_id: string;
  criteria?: string[];
}

export interface RunBatchEvalRequest {
  name?: string;
  items: Array<{
    prompt: string;
    expected?: string;
  }>;
  agent_id?: string;
  model_config?: ModelConfig;
}

export interface RunCustomEvalRequest {
  name?: string;
  evaluator: string;
  input: Record<string, unknown>;
  config?: Record<string, unknown>;
}

// Benchmark types
export interface BenchmarkRun {
  id: string;
  input: Record<string, unknown>;
  output?: Record<string, unknown>;
  metrics?: EvalMetrics;
  score?: number;
  duration_ms?: number;
  error?: string;
}

export interface BenchmarkSummary {
  total_runs: number;
  successful_runs: number;
  failed_runs: number;
  avg_score?: number;
  avg_latency_ms?: number;
  total_tokens?: number;
  total_cost?: number;
}

export interface BenchmarkRecord {
  id: string;
  name: string;
  runs?: BenchmarkRun[];
  results?: Record<string, unknown>;
  summary?: BenchmarkSummary;
  created_at: string;
  metadata?: Record<string, unknown>;
}

export interface BenchmarkListResult {
  benchmarks: BenchmarkRecord[];
  total: number;
  has_more: boolean;
}

export interface CreateBenchmarkRequest {
  name: string;
  description?: string;
  test_cases: Array<{
    input: Record<string, unknown>;
    expected?: Record<string, unknown>;
  }>;
  evaluator_config?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
}

export interface RunBenchmarkRequest {
  agent_id?: string;
  model_config?: ModelConfig;
  parallel?: boolean;
}

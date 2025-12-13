import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api';
import type {
  TraceQueryOpts,
  TokenQueryOpts,
  CostQueryOpts,
  ModelPricing,
  SessionQueryOpts,
  CreateAgentRequest,
  UpdateAgentRequest,
  EvalQueryOpts,
  RunTextEvalRequest,
  RunSessionEvalRequest,
  RunBatchEvalRequest,
  RunCustomEvalRequest,
  CreateBenchmarkRequest,
  RunBenchmarkRequest,
} from '../api';

// Query keys
export const queryKeys = {
  overview: (period: string) => ['overview', period] as const,
  traces: (opts: TraceQueryOpts) => ['traces', opts] as const,
  trace: (id: string) => ['trace', id] as const,
  tokenUsage: (opts: TokenQueryOpts) => ['tokenUsage', opts] as const,
  costs: (opts: CostQueryOpts) => ['costs', opts] as const,
  performance: (period: string) => ['performance', period] as const,
  events: (limit: number) => ['events', limit] as const,
  eventsSince: (cursor: number) => ['eventsSince', cursor] as const,
  insights: ['insights'] as const,
  pricing: ['pricing'] as const,
  sessions: (opts: SessionQueryOpts) => ['sessions', opts] as const,
  session: (id: string) => ['session', id] as const,
  agents: ['agents'] as const,
  agent: (id: string) => ['agent', id] as const,
  evals: (opts: EvalQueryOpts) => ['evals', opts] as const,
  eval: (id: string) => ['eval', id] as const,
  benchmarks: ['benchmarks'] as const,
  benchmark: (id: string) => ['benchmark', id] as const,
};

// Overview
export function useOverview(period: string = '24h') {
  return useQuery({
    queryKey: queryKeys.overview(period),
    queryFn: () => api.getOverview(period),
    refetchInterval: 30000, // Refresh every 30 seconds
  });
}

// Traces
export function useTraces(opts: TraceQueryOpts = {}) {
  return useQuery({
    queryKey: queryKeys.traces(opts),
    queryFn: () => api.getTraces(opts),
  });
}

export function useTrace(id: string) {
  return useQuery({
    queryKey: queryKeys.trace(id),
    queryFn: () => api.getTrace(id),
    enabled: !!id,
  });
}

// Metrics
export function useTokenUsage(opts: TokenQueryOpts = {}) {
  return useQuery({
    queryKey: queryKeys.tokenUsage(opts),
    queryFn: () => api.getTokenUsage(opts),
    refetchInterval: 60000, // Refresh every minute
  });
}

export function useCosts(opts: CostQueryOpts = {}) {
  return useQuery({
    queryKey: queryKeys.costs(opts),
    queryFn: () => api.getCosts(opts),
    refetchInterval: 60000,
  });
}

export function usePerformance(period: string = '24h') {
  return useQuery({
    queryKey: queryKeys.performance(period),
    queryFn: () => api.getPerformance(period),
    refetchInterval: 60000,
  });
}

// Events
export function useRecentEvents(limit: number = 100) {
  return useQuery({
    queryKey: queryKeys.events(limit),
    queryFn: () => api.getRecentEvents(limit),
    refetchInterval: 5000, // Refresh every 5 seconds for real-time feel
  });
}

export function useEventsSince(cursor: number) {
  return useQuery({
    queryKey: queryKeys.eventsSince(cursor),
    queryFn: () => api.getEventsSince(cursor),
    enabled: cursor > 0,
  });
}

// Insights
export function useInsights() {
  return useQuery({
    queryKey: queryKeys.insights,
    queryFn: () => api.getInsights(),
    refetchInterval: 300000, // Refresh every 5 minutes
  });
}

// Pricing
export function usePricing() {
  return useQuery({
    queryKey: queryKeys.pricing,
    queryFn: () => api.getPricing(),
  });
}

export function useUpdatePricing() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (pricing: Partial<ModelPricing> & { model: string }) =>
      api.updatePricing(pricing),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.pricing });
    },
  });
}

// Sessions
export function useSessions(opts: SessionQueryOpts = {}) {
  return useQuery({
    queryKey: queryKeys.sessions(opts),
    queryFn: () => api.getSessions(opts),
  });
}

export function useSession(id: string) {
  return useQuery({
    queryKey: queryKeys.session(id),
    queryFn: () => api.getSession(id),
    enabled: !!id,
  });
}

// Agents
export function useAgents() {
  return useQuery({
    queryKey: queryKeys.agents,
    queryFn: () => api.getAgents(),
  });
}

export function useAgent(id: string) {
  return useQuery({
    queryKey: queryKeys.agent(id),
    queryFn: () => api.getAgent(id),
    enabled: !!id,
  });
}

export function useCreateAgent() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (req: CreateAgentRequest) => api.createAgent(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.agents });
    },
  });
}

export function useUpdateAgent() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, req }: { id: string; req: UpdateAgentRequest }) =>
      api.updateAgent(id, req),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.agents });
      queryClient.invalidateQueries({ queryKey: queryKeys.agent(variables.id) });
    },
  });
}

export function useDeleteAgent() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => api.deleteAgent(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.agents });
    },
  });
}

// Evaluations
export function useEvals(opts: EvalQueryOpts = {}) {
  return useQuery({
    queryKey: queryKeys.evals(opts),
    queryFn: () => api.getEvals(opts),
  });
}

export function useEval(id: string) {
  return useQuery({
    queryKey: queryKeys.eval(id),
    queryFn: () => api.getEval(id),
    enabled: !!id,
  });
}

export function useRunTextEval() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (req: RunTextEvalRequest) => api.runTextEval(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['evals'] });
    },
  });
}

export function useRunSessionEval() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (req: RunSessionEvalRequest) => api.runSessionEval(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['evals'] });
    },
  });
}

export function useRunBatchEval() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (req: RunBatchEvalRequest) => api.runBatchEval(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['evals'] });
    },
  });
}

export function useRunCustomEval() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (req: RunCustomEvalRequest) => api.runCustomEval(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['evals'] });
    },
  });
}

export function useDeleteEval() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => api.deleteEval(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['evals'] });
    },
  });
}

// Benchmarks
export function useBenchmarks() {
  return useQuery({
    queryKey: queryKeys.benchmarks,
    queryFn: () => api.getBenchmarks(),
  });
}

export function useBenchmark(id: string) {
  return useQuery({
    queryKey: queryKeys.benchmark(id),
    queryFn: () => api.getBenchmark(id),
    enabled: !!id,
  });
}

export function useCreateBenchmark() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (req: CreateBenchmarkRequest) => api.createBenchmark(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.benchmarks });
    },
  });
}

export function useRunBenchmark() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, req }: { id: string; req?: RunBenchmarkRequest }) =>
      api.runBenchmark(id, req),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.benchmarks });
      queryClient.invalidateQueries({ queryKey: queryKeys.benchmark(variables.id) });
    },
  });
}

export function useDeleteBenchmark() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => api.deleteBenchmark(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.benchmarks });
    },
  });
}

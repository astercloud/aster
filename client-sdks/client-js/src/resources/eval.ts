/**
 * Eval 资源类
 * Agent 评估和测试系统
 */

import { BaseResource, ClientOptions } from "./base";
import { EvalRequest, EvalInfo, EvalResult, EvalFilter, PaginatedEvalResponse, TestCase, TestCaseSet, BenchmarkConfig, BenchmarkResult, ABTestConfig, ABTestResult, EvalReportRequest, EvalReportResult, ScorerConfig } from "../types/eval";

/**
 * Eval 资源类
 */
export class EvalResource extends BaseResource {
  constructor(options: ClientOptions) {
    super(options);
  }

  // ==========================================================================
  // Eval Execution
  // ==========================================================================

  /**
   * 创建并执行 Eval
   * @param request Eval 请求
   * @returns Eval 信息
   */
  async create(request: EvalRequest): Promise<EvalInfo> {
    return this.request<EvalInfo>("/v1/evals", {
      method: "POST",
      body: request,
    });
  }

  /**
   * 运行文本评估（便捷方法）
   * @param request 文本评估请求
   * @returns Eval 结果
   */
  async runTextEval(request: { prompt: string; expected?: string; scorer?: string; [key: string]: any }): Promise<EvalResult> {
    const evalInfo = await this.create({
      type: "text",
      ...request,
    } as unknown as EvalRequest);
    return this.waitForCompletion(evalInfo.id);
  }

  /**
   * 运行批量评估（便捷方法）
   * @param request 批量评估请求
   * @returns Eval 结果
   */
  async runBatchEval(request: { items: Array<{ prompt: string; expected?: string; [key: string]: any }>; scorer?: string; [key: string]: any }): Promise<EvalResult> {
    const evalInfo = await this.create({
      type: "batch",
      ...request,
    } as unknown as EvalRequest);
    return this.waitForCompletion(evalInfo.id);
  }

  /**
   * 获取 Eval 详情
   * @param evalId Eval ID
   * @returns Eval 信息
   */
  async get(evalId: string): Promise<EvalInfo> {
    return this.request<EvalInfo>(`/v1/evals/${evalId}`);
  }

  /**
   * 列出所有 Evals
   * @param filter 过滤条件
   * @returns Eval 列表
   */
  async list(filter?: EvalFilter): Promise<PaginatedEvalResponse> {
    return this.request<PaginatedEvalResponse>("/v1/evals", {
      params: filter,
    });
  }

  /**
   * 取消 Eval
   * @param evalId Eval ID
   * @returns 更新后的 Eval
   */
  async cancel(evalId: string): Promise<EvalInfo> {
    return this.request<EvalInfo>(`/v1/evals/${evalId}/cancel`, {
      method: "POST",
    });
  }

  /**
   * 删除 Eval
   * @param evalId Eval ID
   */
  async delete(evalId: string): Promise<void> {
    await this.request(`/v1/evals/${evalId}`, {
      method: "DELETE",
    });
  }

  // ==========================================================================
  // Eval Results
  // ==========================================================================

  /**
   * 获取 Eval 结果
   * @param evalId Eval ID
   * @returns Eval 结果
   */
  async getResult(evalId: string): Promise<EvalResult> {
    return this.request<EvalResult>(`/v1/evals/${evalId}/result`);
  }

  /**
   * 等待 Eval 完成
   * @param evalId Eval ID
   * @param pollInterval 轮询间隔（毫秒），默认 1000
   * @param maxWaitTime 最大等待时间（毫秒），默认 300000 (5分钟)
   * @returns Eval 结果
   */
  async waitForCompletion(evalId: string, pollInterval: number = 1000, maxWaitTime: number = 300000): Promise<EvalResult> {
    const startTime = Date.now();

    while (Date.now() - startTime < maxWaitTime) {
      const evalInfo = await this.get(evalId);

      if (evalInfo.status === "completed") {
        return this.getResult(evalId);
      }

      if (evalInfo.status === "failed" || evalInfo.status === "cancelled") {
        throw new Error(`Eval ${evalId} ${evalInfo.status}${evalInfo.error ? `: ${evalInfo.error}` : ""}`);
      }

      // 等待下次轮询
      await new Promise((resolve) => setTimeout(resolve, pollInterval));
    }

    throw new Error(`Eval ${evalId} did not complete within ${maxWaitTime}ms`);
  }

  // ==========================================================================
  // Test Cases Management
  // ==========================================================================

  /**
   * 创建测试用例集
   * @param name 集合名称
   * @param testCases 测试用例列表
   * @param description 描述
   * @returns 测试用例集
   */
  async createTestCaseSet(name: string, testCases: TestCase[], description?: string): Promise<TestCaseSet> {
    return this.request<TestCaseSet>("/v1/evals/test-cases", {
      method: "POST",
      body: { name, testCases, description },
    });
  }

  /**
   * 获取测试用例集
   * @param testCaseSetId 集合 ID
   * @returns 测试用例集
   */
  async getTestCaseSet(testCaseSetId: string): Promise<TestCaseSet> {
    return this.request<TestCaseSet>(`/v1/evals/test-cases/${testCaseSetId}`);
  }

  /**
   * 列出所有测试用例集
   * @returns 测试用例集列表
   */
  async listTestCaseSets(): Promise<TestCaseSet[]> {
    const result = await this.request<{ items: TestCaseSet[] }>("/v1/evals/test-cases");
    return result.items;
  }

  /**
   * 更新测试用例集
   * @param testCaseSetId 集合 ID
   * @param updates 更新内容
   * @returns 更新后的测试用例集
   */
  async updateTestCaseSet(
    testCaseSetId: string,
    updates: {
      name?: string;
      description?: string;
      testCases?: TestCase[];
    },
  ): Promise<TestCaseSet> {
    return this.request<TestCaseSet>(`/v1/evals/test-cases/${testCaseSetId}`, {
      method: "PATCH",
      body: updates,
    });
  }

  /**
   * 删除测试用例集
   * @param testCaseSetId 集合 ID
   */
  async deleteTestCaseSet(testCaseSetId: string): Promise<void> {
    await this.request(`/v1/evals/test-cases/${testCaseSetId}`, {
      method: "DELETE",
    });
  }

  // ==========================================================================
  // Benchmark
  // ==========================================================================

  /**
   * 创建并执行 Benchmark
   * @param config Benchmark 配置
   * @returns Benchmark 结果
   */
  async createBenchmark(config: BenchmarkConfig): Promise<BenchmarkResult> {
    return this.request<BenchmarkResult>("/v1/evals/benchmark", {
      method: "POST",
      body: config,
    });
  }

  /**
   * 获取 Benchmark 结果
   * @param benchmarkId Benchmark ID
   * @returns Benchmark 结果
   */
  async getBenchmark(benchmarkId: string): Promise<BenchmarkResult> {
    return this.request<BenchmarkResult>(`/v1/evals/benchmark/${benchmarkId}`);
  }

  /**
   * 删除 Benchmark
   * @param benchmarkId Benchmark ID
   */
  async deleteBenchmark(benchmarkId: string): Promise<void> {
    await this.request(`/v1/evals/benchmark/${benchmarkId}`, {
      method: "DELETE",
    });
  }

  /**
   * 等待 Benchmark 完成
   * @param benchmarkId Benchmark ID
   * @param pollInterval 轮询间隔（毫秒），默认 2000
   * @param maxWaitTime 最大等待时间（毫秒），默认 600000 (10分钟)
   * @returns Benchmark 结果
   */
  async waitForBenchmarkCompletion(benchmarkId: string, pollInterval: number = 2000, maxWaitTime: number = 600000): Promise<BenchmarkResult> {
    const startTime = Date.now();

    while (Date.now() - startTime < maxWaitTime) {
      const result = await this.getBenchmark(benchmarkId);

      if (result.status === "completed") {
        return result;
      }

      if (result.status === "failed" || result.status === "cancelled") {
        throw new Error(`Benchmark ${benchmarkId} ${result.status}`);
      }

      await new Promise((resolve) => setTimeout(resolve, pollInterval));
    }

    throw new Error(`Benchmark ${benchmarkId} did not complete within ${maxWaitTime}ms`);
  }

  // ==========================================================================
  // A/B Test
  // ==========================================================================

  /**
   * 创建并执行 A/B 测试
   * @param config A/B 测试配置
   * @returns A/B 测试结果
   */
  async createABTest(config: ABTestConfig): Promise<ABTestResult> {
    return this.request<ABTestResult>("/v1/evals/ab-test", {
      method: "POST",
      body: config,
    });
  }

  /**
   * 获取 A/B 测试结果
   * @param abTestId A/B 测试 ID
   * @returns A/B 测试结果
   */
  async getABTest(abTestId: string): Promise<ABTestResult> {
    return this.request<ABTestResult>(`/v1/evals/ab-test/${abTestId}`);
  }

  /**
   * 等待 A/B 测试完成
   * @param abTestId A/B 测试 ID
   * @param pollInterval 轮询间隔（毫秒），默认 2000
   * @param maxWaitTime 最大等待时间（毫秒），默认 600000 (10分钟)
   * @returns A/B 测试结果
   */
  async waitForABTestCompletion(abTestId: string, pollInterval: number = 2000, maxWaitTime: number = 600000): Promise<ABTestResult> {
    const startTime = Date.now();

    while (Date.now() - startTime < maxWaitTime) {
      const result = await this.getABTest(abTestId);

      if (result.status === "completed") {
        return result;
      }

      if (result.status === "failed" || result.status === "cancelled") {
        throw new Error(`A/B Test ${abTestId} ${result.status}`);
      }

      await new Promise((resolve) => setTimeout(resolve, pollInterval));
    }

    throw new Error(`A/B Test ${abTestId} did not complete within ${maxWaitTime}ms`);
  }

  // ==========================================================================
  // Reports
  // ==========================================================================

  /**
   * 生成 Eval 报告
   * @param request 报告请求
   * @returns 报告结果
   */
  async generateReport(request: EvalReportRequest): Promise<EvalReportResult> {
    return this.request<EvalReportResult>("/v1/evals/reports", {
      method: "POST",
      body: request,
    });
  }

  /**
   * 导出 Eval 结果
   * @param evalId Eval ID
   * @param format 导出格式
   * @returns 导出内容
   */
  async exportResult(evalId: string, format: "json" | "csv"): Promise<string> {
    const result = await this.request<{ content: string }>(`/v1/evals/${evalId}/export`, {
      params: { format },
    });
    return result.content;
  }

  // ==========================================================================
  // Quick Eval Helpers
  // ==========================================================================

  /**
   * 快速单次评估
   * @param agentId Agent ID
   * @param input 输入
   * @param expectedOutput 期望输出
   * @param scorers Scorer 配置列表
   * @returns Eval 结果
   */
  async quickEval(agentId: string, input: string, expectedOutput: string, scorers: ScorerConfig[]): Promise<EvalResult> {
    const evalInfo = await this.create({
      name: `Quick Eval - ${new Date().toISOString()}`,
      type: "single",
      agentId,
      testCases: [
        {
          id: "quick-test-1",
          name: "Quick Test",
          input,
          expectedOutput,
        },
      ],
      scorers,
    });

    return this.waitForCompletion(evalInfo.id);
  }

  /**
   * 批量评估
   * @param agentId Agent ID
   * @param testCases 测试用例列表
   * @param scorers Scorer 配置列表
   * @param concurrency 并发数
   * @returns Eval 结果
   */
  async batchEval(agentId: string, testCases: TestCase[], scorers: ScorerConfig[], concurrency?: number): Promise<EvalResult> {
    const evalInfo = await this.create({
      name: `Batch Eval - ${new Date().toISOString()}`,
      type: "batch",
      agentId,
      testCases,
      scorers,
      concurrency,
    });

    return this.waitForCompletion(evalInfo.id);
  }

  /**
   * 比较两个 Agents
   * @param agentAId Agent A ID
   * @param agentBId Agent B ID
   * @param testCaseSetId 测试用例集 ID
   * @param scorers Scorer 配置列表
   * @returns A/B 测试结果
   */
  async compareAgents(agentAId: string, agentBId: string, testCaseSetId: string, scorers: ScorerConfig[]): Promise<ABTestResult> {
    const abTest = await this.createABTest({
      name: `Compare ${agentAId} vs ${agentBId}`,
      agentAId,
      agentBId,
      testCaseSetId,
      scorers,
    });

    return this.waitForABTestCompletion(abTest.id);
  }
}

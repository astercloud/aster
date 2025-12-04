/**
 * AsterClient 客户端集成测试
 * 测试所有模块的交互和集成
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { AsterClient } from "../../src/client";

// Mock fetch
global.fetch = vi.fn();

// 辅助函数：创建 mock response
function mockResponse(data: any) {
  return {
    ok: true,
    json: async () => data,
    text: async () => JSON.stringify(data),
  };
}

describe("AsterClient Client Integration", () => {
  let client: AsterClient;

  beforeEach(() => {
    vi.clearAllMocks();
    client = new AsterClient({
      baseUrl: "http://localhost:8080",
      apiKey: "test-api-key",
      timeout: 30000,
    });
  });

  // ==========================================================================
  // 客户端初始化测试
  // ==========================================================================

  describe("Client Initialization", () => {
    it("should initialize with all resource modules", () => {
      expect(client.memory).toBeDefined();
      expect(client.sessions).toBeDefined();
      expect(client.workflows).toBeDefined();
      expect(client.mcp).toBeDefined();
      expect(client.middleware).toBeDefined();
      expect(client.tools).toBeDefined();
      expect(client.telemetry).toBeDefined();
    });

    it("should have convenience methods", () => {
      expect(typeof client.healthCheck).toBe("function");
      expect(typeof client.getStatus).toBe("function");
      expect(typeof client.getUsageStatistics).toBe("function");
      expect(typeof client.getPerformanceMetrics).toBe("function");
    });
  });

  // ==========================================================================
  // Memory + Session 集成测试
  // ==========================================================================

  describe("Memory + Session Integration", () => {
    it("should create session and use working memory", async () => {
      // Mock session creation
      (global.fetch as any).mockResolvedValueOnce(
        mockResponse({
          id: "session-123",
          agentId: "agent-1",
          status: "active",
          createdAt: new Date().toISOString(),
        }),
      );

      const session = await client.sessions.create({
        agentId: "agent-1",
        templateId: "chat",
        userId: "user-1",
      });

      expect(session.id).toBe("session-123");

      // Mock working memory set
      (global.fetch as any).mockResolvedValueOnce(mockResponse({}));

      await client.memory.working.set(
        "session_context",
        {
          sessionId: session.id,
          userId: "user-1",
        },
        {
          scope: "thread",
        },
      );

      expect(global.fetch).toHaveBeenCalledTimes(2);
    });

    it("should use semantic memory for session context", async () => {
      // Mock semantic memory store
      (global.fetch as any).mockResolvedValueOnce(
        mockResponse({
          id: "chunk-1",
          content: "User preference: dark theme",
        }),
      );

      await client.memory.semantic.store("User prefers dark theme", {
        sessionId: "session-123",
      });

      // Mock semantic search
      (global.fetch as any).mockResolvedValueOnce(
        mockResponse({
          chunks: [
            {
              id: "chunk-1",
              content: "User preference: dark theme",
              similarity: 0.95,
            },
          ],
        }),
      );

      const results = await client.memory.semantic.search("What is the user preference?", { limit: 5 });

      expect(results.length).toBeGreaterThan(0);
      expect(global.fetch).toHaveBeenCalledTimes(2);
    });
  });

  // ==========================================================================
  // Workflow + Tool 集成测试
  // ==========================================================================

  describe("Workflow + Tool Integration", () => {
    it("should create workflow and execute tools", async () => {
      // Mock workflow creation
      (global.fetch as any).mockResolvedValueOnce(
        mockResponse({
          id: "workflow-1",
          type: "sequential",
          name: "Tool Pipeline",
          status: "active",
        }),
      );

      const workflow = await client.workflows.create({
        type: "sequential",
        name: "Tool Pipeline",
        description: "Execute tools in sequence",
        steps: [
          { agent: "reader", action: "read_file" },
          { agent: "processor", action: "process_data" },
        ],
      });

      expect(workflow.id).toBe("workflow-1");

      // Mock tool execution
      (global.fetch as any).mockResolvedValueOnce(
        mockResponse({
          success: true,
          result: "file content",
          executionTime: 150,
        }),
      );

      const toolResult = await client.tools.execute("file_read", {
        path: "/tmp/test.txt",
      });

      expect(toolResult.success).toBe(true);
      expect(global.fetch).toHaveBeenCalledTimes(2);
    });
  });

  // ==========================================================================
  // MCP + Middleware 集成测试
  // ==========================================================================

  describe("MCP + Middleware Integration", () => {
    it("should add MCP server and configure middleware", async () => {
      // Mock MCP server addition
      (global.fetch as any).mockResolvedValueOnce(
        mockResponse({
          serverId: "mcp-1",
          name: "Test Server",
          status: "connected",
          toolCount: 5,
        }),
      );

      const server = await client.mcp.addServer({
        serverId: "mcp-1",
        name: "Test Server",
        endpoint: "http://localhost:9000/mcp",
      });

      expect(server.serverId).toBe("mcp-1");

      // Mock middleware config update
      (global.fetch as any).mockResolvedValueOnce(
        mockResponse({
          name: "tool_approval",
          enabled: true,
          config: {
            approvalRequired: ["bash", "file_delete"],
          },
        }),
      );

      await client.middleware.updateConfig("tool_approval", {
        approvalRequired: ["bash", "file_delete"],
      });

      expect(global.fetch).toHaveBeenCalledTimes(2);
    });
  });

  // ==========================================================================
  // Telemetry 监控测试
  // ==========================================================================

  describe("Telemetry Monitoring", () => {
    it("should check health and get performance metrics", async () => {
      // Mock health check
      (global.fetch as any).mockResolvedValueOnce(
        mockResponse({
          status: "healthy",
          timestamp: new Date().toISOString(),
          components: {
            database: { status: "healthy" },
            memory: { status: "healthy" },
          },
        }),
      );

      const health = await client.healthCheck();
      expect(health.status).toBe("healthy");

      // Mock performance metrics
      (global.fetch as any).mockResolvedValueOnce(
        mockResponse({
          timeRange: {
            start: "2024-01-01T00:00:00Z",
            end: "2024-01-02T00:00:00Z",
          },
          requests: {
            total: 1000,
            successful: 950,
            failed: 50,
            avgLatency: 120,
            p95Latency: 250,
            p99Latency: 500,
          },
        }),
      );

      const perf = await client.getPerformanceMetrics();
      expect(perf.requests.total).toBe(1000);
      expect(global.fetch).toHaveBeenCalledTimes(2);
    });

    it("should get usage statistics", async () => {
      // Mock usage statistics
      (global.fetch as any).mockResolvedValueOnce(
        mockResponse({
          timeRange: {
            start: "2024-01-01T00:00:00Z",
            end: "2024-01-08T00:00:00Z",
          },
          sessions: {
            total: 500,
            active: 50,
            avgDuration: 3600,
          },
          workflows: {
            total: 200,
            successful: 180,
            failed: 20,
          },
          tools: {
            total: 1000,
            topTools: [
              { toolName: "bash", callCount: 300 },
              { toolName: "http_request", callCount: 250 },
            ],
          },
        }),
      );

      const usage = await client.getUsageStatistics();
      expect(usage.sessions?.total).toBe(500);
      expect(usage.tools?.topTools).toHaveLength(2);
    });
  });

  // ==========================================================================
  // 完整工作流测试
  // ==========================================================================

  describe("Complete Workflow", () => {
    it("should execute a complete workflow from start to finish", async () => {
      // 1. Create session
      (global.fetch as any).mockResolvedValueOnce(
        mockResponse({
          id: "session-1",
          agentId: "agent-1",
          status: "active",
        }),
      );

      const session = await client.sessions.create({
        agentId: "agent-1",
        templateId: "chat",
        userId: "user-1",
        enableCheckpoints: true,
      });

      // 2. Set working memory
      (global.fetch as any).mockResolvedValueOnce(mockResponse({}));

      await client.memory.working.set("context", {
        sessionId: session.id,
      });

      // 3. Create and execute workflow
      (global.fetch as any).mockResolvedValueOnce(
        mockResponse({
          id: "workflow-1",
          type: "parallel",
          name: "Research",
          status: "active",
        }),
      );

      const workflow = await client.workflows.create({
        type: "parallel",
        name: "Research",
        agents: [
          { id: "researcher-1", task: "Research AI" },
          { id: "researcher-2", task: "Research ML" },
        ],
        maxConcurrency: 2,
      });

      (global.fetch as any).mockResolvedValueOnce(
        mockResponse({
          id: "run-1",
          workflowId: workflow.id,
          status: "running",
          progress: 0,
        }),
      );

      const run = await client.workflows.execute(workflow.id, {
        input: "Start research",
      });

      // 4. Get telemetry
      (global.fetch as any).mockResolvedValueOnce(
        mockResponse({
          status: "healthy",
          timestamp: new Date().toISOString(),
          components: {},
        }),
      );

      await client.healthCheck();

      // 5. Complete session
      (global.fetch as any).mockResolvedValueOnce(
        mockResponse({
          id: session.id,
          status: "completed",
        }),
      );

      await client.sessions.complete(session.id);

      expect(global.fetch).toHaveBeenCalledTimes(6);
    });
  });
});

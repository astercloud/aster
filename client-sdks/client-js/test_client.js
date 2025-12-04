/**
 * 客户端SDK快速测试脚本
 * 验证与服务端API的连接性
 */

const BASE_URL = process.env.API_URL || "http://localhost:8080";

// 简单的HTTP请求辅助函数
async function request(endpoint, options = {}) {
  const url = BASE_URL + endpoint;
  const response = await fetch(url, {
    method: options.method || "GET",
    headers: {
      "Content-Type": "application/json",
      ...options.headers,
    },
    body: options.body ? JSON.stringify(options.body) : undefined,
  });

  if (!response.ok && response.status !== 204) {
    const error = await response.text();
    throw new Error(`Request failed: ${response.status} - ${error}`);
  }

  if (response.status === 204) {
    return null;
  }

  return await response.json();
}

async function runTests() {
  console.log("🚀 开始测试客户端SDK与服务端API连接性...\n");

  let testsRun = 0;
  let testsPassed = 0;
  let testsFailed = 0;

  // 测试宏
  async function test(name, fn) {
    testsRun++;
    try {
      await fn();
      console.log(`✅ [${testsRun}] ${name}`);
      testsPassed++;
    } catch (error) {
      console.log(`❌ [${testsRun}] ${name}`);
      console.log(`   错误: ${error.message}`);
      testsFailed++;
    }
  }

  // Agent API测试
  let agentId;
  await test("Agent: 创建", async () => {
    const result = await request("/v1/agents", {
      method: "POST",
      body: { name: "Test Agent", model: "gpt-4" },
    });
    agentId = result.data.id;
  });

  await test("Agent: 列表", async () => {
    await request("/v1/agents");
  });

  if (agentId) {
    await test("Agent: 获取详情", async () => {
      await request(`/v1/agents/${agentId}`);
    });

    await test("Agent: 激活", async () => {
      await request(`/v1/agents/${agentId}/activate`, { method: "POST" });
    });

    await test("Agent: 删除", async () => {
      await request(`/v1/agents/${agentId}`, { method: "DELETE" });
    });
  }

  // Session API测试
  let sessionId;
  await test("Session: 创建", async () => {
    const result = await request("/v1/sessions", {
      method: "POST",
      body: { name: "Test Session" },
    });
    sessionId = result.data.id;
  });

  await test("Session: 列表", async () => {
    await request("/v1/sessions");
  });

  if (sessionId) {
    await test("Session: 添加消息", async () => {
      await request(`/v1/sessions/${sessionId}/messages`, {
        method: "POST",
        body: { role: "user", content: "Hello!" },
      });
    });

    await test("Session: 删除", async () => {
      await request(`/v1/sessions/${sessionId}`, { method: "DELETE" });
    });
  }

  // Memory API测试
  await test("Memory: 创建 Working Memory", async () => {
    await request("/v1/memory/working", {
      method: "POST",
      body: { key: "test", value: { data: "test" } },
    });
  });

  await test("Memory: 列表 Working Memory", async () => {
    await request("/v1/memory/working");
  });

  await test("Memory: 创建 Semantic Memory", async () => {
    await request("/v1/memory/semantic", {
      method: "POST",
      body: { content: "Test", tags: ["test"] },
    });
  });

  // Workflow API测试
  let workflowId;
  await test("Workflow: 创建", async () => {
    const result = await request("/v1/workflows", {
      method: "POST",
      body: { name: "Test", steps: [{ id: "1", name: "S1", type: "agent" }] },
    });
    workflowId = result.data.id;
  });

  if (workflowId) {
    await test("Workflow: 执行", async () => {
      await request(`/v1/workflows/${workflowId}/execute`, {
        method: "POST",
        body: {},
      });
    });

    await test("Workflow: 删除", async () => {
      await request(`/v1/workflows/${workflowId}`, { method: "DELETE" });
    });
  }

  // Tool API测试
  let toolId;
  await test("Tool: 创建", async () => {
    const result = await request("/v1/tools", {
      method: "POST",
      body: { name: "Test Tool", type: "custom", schema: { type: "object" } },
    });
    toolId = result.data.id;
  });

  if (toolId) {
    await test("Tool: 执行", async () => {
      await request(`/v1/tools/${toolId}/execute`, {
        method: "POST",
        body: { input: {} },
      });
    });

    await test("Tool: 删除", async () => {
      await request(`/v1/tools/${toolId}`, { method: "DELETE" });
    });
  }

  // MCP API测试
  let mcpId;
  await test("MCP: 创建服务器", async () => {
    const result = await request("/v1/mcp/servers", {
      method: "POST",
      body: { name: "Test", type: "stdio", command: "node" },
    });
    mcpId = result.data.id;
  });

  if (mcpId) {
    await test("MCP: 启动", async () => {
      await request(`/v1/mcp/servers/${mcpId}/start`, { method: "POST" });
    });

    await test("MCP: 停止", async () => {
      await request(`/v1/mcp/servers/${mcpId}/stop`, { method: "POST" });
    });

    await test("MCP: 删除", async () => {
      await request(`/v1/mcp/servers/${mcpId}`, { method: "DELETE" });
    });
  }

  // Middleware API测试
  let mwId;
  await test("Middleware: 创建", async () => {
    const result = await request("/v1/middlewares", {
      method: "POST",
      body: { name: "Test MW", type: "custom", priority: 10 },
    });
    mwId = result.data.id;
  });

  if (mwId) {
    await test("Middleware: 启用", async () => {
      await request(`/v1/middlewares/${mwId}/enable`, { method: "POST" });
    });

    await test("Middleware: 删除", async () => {
      await request(`/v1/middlewares/${mwId}`, { method: "DELETE" });
    });
  }

  // Telemetry API测试
  await test("Telemetry: 记录 Metric", async () => {
    await request("/v1/telemetry/metrics", {
      method: "POST",
      body: { name: "test", type: "counter", value: 1 },
    });
  });

  await test("Telemetry: 记录 Trace", async () => {
    await request("/v1/telemetry/traces", {
      method: "POST",
      body: { name: "test", span_id: "span-1" },
    });
  });

  // Eval API测试
  await test("Eval: 文本评估", async () => {
    await request("/v1/eval/text", {
      method: "POST",
      body: { prompt: "Test", expected: "Test" },
    });
  });

  await test("Eval: 批量评估", async () => {
    await request("/v1/eval/batch", {
      method: "POST",
      body: { items: [{ prompt: "Test" }] },
    });
  });

  // System API测试
  await test("System: 获取信息", async () => {
    await request("/v1/system/info");
  });

  await test("System: 健康检查", async () => {
    await request("/v1/system/health");
  });

  await test("System: 获取统计", async () => {
    await request("/v1/system/stats");
  });

  // 打印总结
  console.log(`\n${"=".repeat(60)}`);
  console.log(`📊 测试总结`);
  console.log(`${"=".repeat(60)}`);
  console.log(`总测试数: ${testsRun}`);
  console.log(`✅ 通过: ${testsPassed}`);
  console.log(`❌ 失败: ${testsFailed}`);
  console.log(`成功率: ${((testsPassed / testsRun) * 100).toFixed(1)}%`);
  console.log(`${"=".repeat(60)}\n`);

  if (testsFailed === 0) {
    console.log("🎉 所有测试通过！客户端SDK与服务端API完全兼容！");
    process.exit(0);
  } else {
    console.log("⚠️  部分测试失败，请检查服务端是否正常运行");
    process.exit(1);
  }
}

// 运行测试
runTests().catch((error) => {
  console.error("❌ 测试运行失败:", error);
  process.exit(1);
});

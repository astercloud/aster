/**
 * MCP + Middleware + Tool 使用示例
 */

import { MCPResource, MiddlewareResource, ToolResource } from "@aster/client-js";

async function main() {
  console.log("=".repeat(60));
  console.log("AsterClient MCP + Middleware + Tool 演示");
  console.log("=".repeat(60));

  // 创建资源实例
  const mcp = new MCPResource({
    baseUrl: "http://localhost:8080",
    apiKey: process.env.ASTER_API_KEY,
  });

  const middleware = new MiddlewareResource({
    baseUrl: "http://localhost:8080",
    apiKey: process.env.ASTER_API_KEY,
  });

  const tool = new ToolResource({
    baseUrl: "http://localhost:8080",
    apiKey: process.env.ASTER_API_KEY,
  });

  // ========================================================================
  // 1. MCP 协议演示
  // ========================================================================
  console.log("\n🔌 1. MCP 协议（Model Context Protocol）");
  console.log("-".repeat(60));

  // 添加 MCP Server
  const mcpServer = await mcp.addServer({
    serverId: "my-mcp-server",
    name: "My MCP Server",
    endpoint: "http://localhost:8090/mcp",
    accessKeyId: "key",
    accessKeySecret: "secret",
    enabled: true,
  });
  console.log("✅ 添加 MCP Server:", mcpServer.serverId);
  console.log("   状态:", mcpServer.status);
  console.log("   工具数:", mcpServer.toolCount);

  // 列出所有 Servers
  const servers = await mcp.listServers();
  console.log(`📋 总共 ${servers.length} 个 MCP Servers`);

  // 连接到 Server
  try {
    await mcp.connectServer("my-mcp-server");
    console.log("✅ 已连接到 Server");
  } catch (error: any) {
    console.log("⚠️  连接失败:", error.message);
  }

  // 获取 Server 的工具列表
  try {
    const tools = await mcp.getServerTools("my-mcp-server");
    console.log(`🔧 Server 提供 ${tools.length} 个工具:`);
    tools.forEach((tool, index) => {
      console.log(`   ${index + 1}. ${tool.name} - ${tool.description}`);
    });

    // 调用 MCP 工具
    if (tools.length > 0) {
      const result = await mcp.call("my-mcp-server", tools[0].name, {
        // 参数示例
        input: "test",
      });
      console.log("📤 工具调用结果:");
      console.log("   成功:", result.success);
      console.log("   耗时:", result.executionTime, "ms");
      if (result.result) {
        console.log("   结果:", JSON.stringify(result.result).substring(0, 100));
      }
    }
  } catch (error: any) {
    console.log("⚠️  获取工具失败:", error.message);
  }

  // MCP 统计信息
  try {
    const stats = await mcp.getStats();
    console.log("📊 MCP 统计:");
    console.log("   连接的 Servers:", stats.connectedServers, "/", stats.totalServers);
    console.log("   总工具数:", stats.totalTools);
    console.log("   总调用次数:", stats.totalCalls);
    console.log("   成功率:", ((stats.successfulCalls / stats.totalCalls) * 100).toFixed(1), "%");
  } catch (error: any) {
    console.log("⚠️  获取统计失败:", error.message);
  }

  // ========================================================================
  // 2. Middleware 系统演示
  // ========================================================================
  console.log("\n🧅 2. Middleware 系统（洋葱模型）");
  console.log("-".repeat(60));

  // 列出所有 Middleware
  const middlewares = await middleware.list();
  console.log(`📋 总共 ${middlewares.length} 个 Middlewares:`);
  middlewares.forEach((mw, index) => {
    const status = mw.enabled ? "✅" : "⏸️ ";
    console.log(`   ${status} ${index + 1}. [P${mw.priority}] ${mw.displayName} - ${mw.description}`);
  });

  // 配置 Summarization Middleware（上下文压缩）
  console.log("\n📝 配置 Summarization Middleware:");
  const summarization = await middleware.updateConfig("summarization", {
    threshold: 170000, // 170K tokens 后触发总结
    keepMessages: 6, // 保留最近 6 条消息
    llmProvider: "anthropic",
    llmModel: "claude-sonnet-4",
  });
  console.log("✅ Summarization 已配置");
  console.log("   阈值:", summarization.config?.threshold, "tokens");
  console.log("   保留消息数:", summarization.config?.keepMessages);

  // 配置 Tool Approval Middleware（工具审批）
  console.log("\n🔐 配置 Tool Approval Middleware:");
  await middleware.updateConfig("tool_approval", {
    approvalRequired: ["file_delete", "bash", "database_query"],
    autoApprove: ["file_read", "http_request"],
    approvalTimeout: 300, // 5分钟
    timeoutBehavior: "reject",
  });
  console.log("✅ Tool Approval 已配置");
  console.log("   需要审批的工具: file_delete, bash, database_query");
  console.log("   自动批准的工具: file_read, http_request");

  // 配置 PII Redaction Middleware（敏感信息脱敏）
  console.log("\n🔒 配置 PII Redaction Middleware:");
  await middleware.updateConfig("pii_redaction", {
    enabledTypes: ["email", "phone", "ssn", "credit_card"],
    strategy: "mask", // 遮蔽策略
    partial: true, // 保留部分信息
  });
  console.log("✅ PII Redaction 已配置");

  // 配置 Cost Tracker Middleware（成本追踪）
  console.log("\n💰 配置 Cost Tracker Middleware:");
  await middleware.updateConfig("cost_tracker", {
    enabled: true,
    costModel: "token_based",
    pricing: {
      promptTokenPrice: 0.003, // $0.003 / 1K tokens
      completionTokenPrice: 0.015, // $0.015 / 1K tokens
      currency: "USD",
    },
    budget: {
      daily: 100, // $100/天
      monthly: 2000, // $2000/月
    },
  });
  console.log("✅ Cost Tracker 已配置");
  console.log("   每日预算: $100");
  console.log("   每月预算: $2000");

  // 获取 Middleware 执行顺序
  const executionOrder = await middleware.getExecutionOrder();
  console.log("\n🔄 Middleware 执行顺序:");
  executionOrder.forEach((name, index) => {
    console.log(`   ${index + 1}. ${name}`);
  });

  // 获取 Middleware 统计信息
  try {
    const allStats = await middleware.getAllStats();
    console.log("\n📊 Middleware 统计:");
    allStats.slice(0, 3).forEach((stat) => {
      console.log(`   ${stat.name}:`);
      console.log(`     执行: ${stat.executionCount} 次`);
      console.log(`     成功率: ${((stat.successCount / stat.executionCount) * 100).toFixed(1)}%`);
      console.log(`     平均耗时: ${stat.avgExecutionTime.toFixed(2)} ms`);
    });
  } catch (error: any) {
    console.log("⚠️  获取统计失败:", error.message);
  }

  // ========================================================================
  // 3. Tool 系统演示
  // ========================================================================
  console.log("\n🔧 3. Tool 系统");
  console.log("-".repeat(60));

  // 列出所有工具
  const tools = await tool.list();
  console.log(`📋 总共 ${tools.length} 个工具:`);

  // 按分类统计
  const categoryCount: Record<string, number> = {};
  tools.forEach((t) => {
    categoryCount[t.category] = (categoryCount[t.category] || 0) + 1;
  });
  Object.entries(categoryCount).forEach(([category, count]) => {
    console.log(`   ${category}: ${count} 个`);
  });

  // 列出内置工具
  const builtinTools = tools.filter((t) => t.type === "builtin");
  console.log(`\n🛠️  内置工具 (${builtinTools.length} 个):`);
  builtinTools.forEach((t, index) => {
    const status = t.enabled ? "✅" : "⏸️ ";
    const approval = t.requiresApproval ? "🔒" : "";
    console.log(`   ${status}${approval} ${index + 1}. ${t.name} - ${t.description}`);
  });

  // 执行 Bash 工具（同步）
  console.log("\n💻 执行 Bash 工具:");
  try {
    const result = await tool.execute("bash", {
      command: 'echo "Hello from AsterClient!"',
      workDir: "/tmp",
      timeout: 10,
    });
    console.log("✅ 执行成功:");
    console.log("   耗时:", result.executionTime, "ms");
    console.log("   结果:", result.result);
  } catch (error: any) {
    console.log("⚠️  执行失败:", error.message);
  }

  // 执行 HTTP 请求工具
  console.log("\n🌐 执行 HTTP 请求工具:");
  try {
    const result = await tool.execute("http_request", {
      url: "https://api.github.com/zen",
      method: "GET",
      timeout: 10,
    });
    console.log("✅ 执行成功:");
    console.log("   耗时:", result.executionTime, "ms");
    console.log("   响应:", result.result);
  } catch (error: any) {
    console.log("⚠️  执行失败:", error.message);
  }

  // 执行长时运行工具（异步）
  console.log("\n⏱️  执行 Web Scraper（长时运行工具）:");
  try {
    const asyncResult = await tool.executeAsync("web_scraper", {
      url: "https://example.com",
      selectors: ["h1", "p"],
      executeJs: true,
      waitTime: 2000,
    });
    console.log("✅ 任务已创建:", asyncResult.taskId);
    console.log("   状态:", asyncResult.status);

    // 等待任务完成
    console.log("⏳ 等待任务完成...");
    const task = await tool.waitForTask(asyncResult.taskId, {
      pollInterval: 2000,
      timeout: 60000,
    });
    console.log("✅ 任务完成:");
    console.log("   状态:", task.status);
    console.log("   进度:", task.progress, "%");
    if (task.result) {
      console.log("   结果:", JSON.stringify(task.result).substring(0, 200));
    }
  } catch (error: any) {
    console.log("⚠️  执行失败:", error.message);
  }

  // 列出所有任务
  try {
    const tasks = await tool.listTasks({
      status: "running",
    });
    console.log(`\n📊 运行中的任务: ${tasks.length} 个`);
    tasks.forEach((t, index) => {
      console.log(`   ${index + 1}. [${t.toolName}] ${t.status} - ${t.progress}%`);
    });
  } catch (error: any) {
    console.log("⚠️  获取任务列表失败:", error.message);
  }

  // 工具统计信息
  try {
    const allStats = await tool.getAllStats();
    console.log("\n📊 工具使用统计 (Top 5):");
    allStats
      .sort((a, b) => b.totalCalls - a.totalCalls)
      .slice(0, 5)
      .forEach((stat, index) => {
        console.log(`   ${index + 1}. ${stat.toolName}:`);
        console.log(`      调用: ${stat.totalCalls} 次`);
        console.log(`      成功率: ${((stat.successCount / stat.totalCalls) * 100).toFixed(1)}%`);
        console.log(`      平均耗时: ${stat.avgExecutionTime.toFixed(2)} ms`);
      });
  } catch (error: any) {
    console.log("⚠️  获取统计失败:", error.message);
  }

  // 工具使用报告
  try {
    const report = await tool.getUsageReport({
      start: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(), // 7天前
      end: new Date().toISOString(),
    });
    console.log("\n📈 工具使用报告（过去7天）:");
    console.log("   总调用次数:", report.totalCalls);
    console.log("   最常用工具:");
    report.topTools.slice(0, 3).forEach((t, index) => {
      console.log(`      ${index + 1}. ${t.toolName} - ${t.callCount} 次 (${t.percentage.toFixed(1)}%)`);
    });
  } catch (error: any) {
    console.log("⚠️  获取报告失败:", error.message);
  }

  // ========================================================================
  // 4. 清理
  // ========================================================================
  console.log("\n🧹 4. 清理");
  console.log("-".repeat(60));

  // 断开 MCP Server
  try {
    await mcp.disconnectServer("my-mcp-server");
    console.log("✅ MCP Server 已断开");
  } catch (error: any) {
    console.log("⚠️  断开失败:", error.message);
  }

  console.log("\n" + "=".repeat(60));
  console.log("✅ 演示完成！");
  console.log("=".repeat(60));
}

// 运行示例
main().catch((error) => {
  console.error("❌ 错误:", error);
  process.exit(1);
});

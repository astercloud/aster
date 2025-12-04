# Aster Agent UI 组件库

**Version 0.15.0** | 专为 AI Agent 应用设计的 Vue 3 组件库

## 🤖 什么是 Aster Agent UI？

Aster Agent UI 是一个专门为 AI Agent 应用设计的组件库，提供了构建 Agent 管理、对话、工作流等功能所需的所有 UI 组件。既可以作为销售/合作演示，也可以直接供前端团队参考 SDK 与代码结构。

## ✨ 核心特性

- 🤖 **67 个组件** - 包含 46 个 ChatUI 组件 + 21 个 Agent 专属组件
- 💬 **对话界面** - 完整的 Agent 对话体验
- 🔄 **工作流** - Agent 工作流可视化
- 👥 **多 Agent** - 支持多 Agent 协作
- 🧠 **思考过程** - Agent 推理过程可视化
- 🎨 **现代设计** - 简洁美观的界面
- 🌙 **深色模式** - 完整的暗色主题
- 💪 **TypeScript** - 完整的类型定义

## 🚀 快速开始

### 前提条件

- Node.js 16+
- Go 1.21+（如果需要运行后端）

### 1. 启动后端服务器

在项目根目录：

```bash
PROVIDER=deepseek \
MODEL=deepseek-chat \
DEEPSEEK_API_KEY=your-api-key \
go run ./cmd/aster-server
```

### 2. 启动前端 UI

```bash
cd ui
npm install
npm run dev
```

### 3. 访问应用

打开浏览器访问 http://localhost:3001

**注意：** 前端需要配置正确的 API Key 才能连接后端。详见 [配置指南](./SETUP_GUIDE.md)。

### 构建生产包

```bash
npm run build
```

发布时会生成 `dist/`，其中包含 `aster-ui.es.js`、`aster-ui.umd.js`、`style.css` 与类型声明。

## 🎯 快速导航

启动开发服务器后，访问以下页面：

- **[首页](http://localhost:3000/)** - 组件库导航和快速入口
- **[Agent 聊天演示](http://localhost:3000/agent-demo)** - 完整的 Agent 对话体验
- **[交互式文档](http://localhost:3000/docs)** - 组件文档 + 实时 Demo
- **[组件展示](http://localhost:3000/components)** - 所有组件的视觉效果
- **[Agent 管理](http://localhost:3000/agents)** - Agent 创建和配置
- **[工作流](http://localhost:3000/workflows)** - 工作流可视化
- **[协作房间](http://localhost:3000/rooms)** - 多 Agent 协作
- **[项目管理](http://localhost:3000/projects)** - AI 写作项目管理
- **[Landing Page](http://localhost:3000/landing)** - ChatUI 风格的产品展示页

## 📦 组件分类

### 🤖 Agent 组件

- **AgentCard** - Agent 信息卡片
- **AgentDashboard** - Agent 管理仪表板
- **AgentChatSession** - Agent 对话会话
- **ThinkingBlock** - 思考过程可视化（含人工审批）
- **WorkflowTimeline** - 工作流时间线（含快捷操作）

### 📁 项目组件

- **ProjectCard** - 项目信息卡片
- **ProjectList** - 项目列表（含筛选）

### ✏️ 编辑器组件

- **EditorPanel** - Markdown 编辑器（含预览）

### 💬 对话组件

- **Chat** - 聊天容器
- **Bubble** - 消息气泡
- **MultimodalInput** - 多模态输入
- **MessageStatus** - 消息状态

### 🎨 基础组件

- **Button** - 按钮
- **Avatar** - 头像
- **Icon** - 图标
- **Card** - 卡片

## 📖 文档资源

### 🚀 快速开始

- [状态更新](./STATUS_UPDATE.md) - 最新状态和问题解决 🆕
- [配置指南](./SETUP_GUIDE.md) - 完整的环境配置和启动说明 ⭐
- [快速测试](./QUICK_TEST.md) - 验证系统是否正常工作 ⭐
- [快速入门指南](./QUICK_START.md) - 5 分钟上手
- [故障排除](./TROUBLESHOOTING.md) - 常见问题解决方案

### 📚 学习资源

- [完整使用示例](./COMPLETE_EXAMPLE.md) - 构建 AI 写作助手
- [组件文档](./src/docs/README.md) - 完整的组件 API
- [ChatUI 组件指南](./CHATUI_GUIDE.md) - 对话组件使用

### 📊 项目状态

- [最终完成报告](./FINAL_REPORT.md) - 项目总览和成就 🎉
- [开发进度报告](./PROGRESS_REPORT.md) - 87.5% 完成
- [实现总结](./IMPLEMENTATION_SUMMARY.md) - 技术细节
- [缺失功能清单](./MISSING_FEATURES.md) - 待实现功能

## 🔧 与 Aster 后端集成

```vue
<script setup>
import { useAsterClient } from "@/composables/useAsterClient";

const { client } = useAsterClient();

// 获取 Agent 列表
const agents = await client.agents.list();

// 与 Agent 对话
const response = await client.agents.chat(agentId, {
  message: "Hello",
  stream: false,
});
</script>
```

## 📊 组件统计

- **总组件数：** 33+
- **Agent 专属：** 6 个
- **对话组件：** 9 个
- **基础组件：** 4 个
- **表单组件：** 4 个
- **布局组件：** 8 个
- **反馈组件：** 6 个

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！提交前请运行：

```bash
npm run build
npm run type-check
```

## 📄 许可证

MIT License © Aster Cloud

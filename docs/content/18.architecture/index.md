# 架构文档

欢迎来到 aster 架构文档！这里详细介绍了 aster 的整体设计和各层实现。

---

## 📚 文档导航

### [1. 架构概览](./1.overview.md)
aster 的四层架构设计，包括核心 SDK、Server 层、HTTP 层和客户端 SDK 的整体架构。

**关键内容**:
- 整体架构图
- 核心设计原则（框架无关、分层架构、模块化、事件驱动）
- 15 个功能域
- 使用方式（Go SDK、生产服务器、HTTP 服务、客户端 SDK）

### [2. 核心 SDK 架构](./2.core-sdk.md)
`pkg/` 目录下的核心 SDK 详细设计，零 HTTP 框架依赖。

**关键内容**:
- Agent 类型（Basic、Workflow、SubAgent）
- Middleware 系统（洋葱模型）
- Backend 抽象层（4 种实现）
- Session Store（PostgreSQL/MySQL 支持）
- 性能指标（~27M ops/s）

### [3. Server 层架构](./server-architecture.md) ✨ 新增
`server/` 目录下的生产级服务器实现，企业级特性。

**关键内容**:
- 认证与授权（API Key、JWT、RBAC）
- Prometheus Metrics 监控
- OpenTelemetry 分布式追踪
- 智能速率限制（令牌桶/滑动窗口）
- 增强健康检查
- 8 个核心 Handler
- 生产部署就绪

### [4. HTTP 层架构](./3.http-layer.md)
`cmd/aster` 目录下的开发 HTTP 服务实现，可替换框架。

**关键内容**:
- 100+ REST API 端点
- 与核心 SDK 的解耦设计
- 框架灵活性（Gin/Echo/Chi/标准库）
- 依赖管理策略
- 响应格式规范
- 开发模式快速启动

### [5. 客户端 SDK 架构](./4.client-sdk.md)
`client-sdks` 目录下的多语言客户端 SDK 设计。

**关键内容**:
- 15 个资源模块
- 事件驱动架构（三通道设计）
- BaseResource 基类
- 完整的 TypeScript 类型系统
- React/Vercel AI SDK 集成

### [6. Human-in-the-Loop 架构](./hitl-overview.md) 🔐 安全特性
人工在环审核系统，控制 Agent 敏感操作。

**关键内容**:
- 三种决策类型（批准/拒绝/编辑）
- 智能风险评估
- 多种审核处理器（CLI/Web UI/MQ）
- 与 Telemetry、Memory 集成
- 安全最佳实践

---

## 🎯 快速开始

### 理解整体架构
从 [架构概览](./1.overview.md) 开始，了解 aster 的整体设计理念。

### 深入核心实现
阅读 [核心 SDK 架构](./2.core-sdk.md)，理解 Agent、Middleware 和 Backend 的设计。

### 了解 HTTP 服务
查看 [HTTP 层架构](./3.http-layer.md)，了解如何使用或替换 HTTP 实现。

### 使用客户端 SDK
参考 [客户端 SDK 架构](./4.client-sdk.md)，学习如何在应用中集成 aster。

---

## 🔑 核心概念

### 框架无关 (Framework Agnostic)
- ✅ 核心 SDK 零 HTTP 框架依赖
- ✅ 用户可以使用任何 HTTP 框架
- ✅ 避免依赖冲突

### 分层架构 (Layered Architecture)
```
Client → Server → HTTP → Core
展示层 → 服务层 → 接口层 → 业务层
```

**四层架构**:
- **Client 层**: TypeScript/React 客户端
- **Server 层**: 生产级服务器（认证、监控、限流）
- **HTTP 层**: 开发工具（快速启动）
- **Core 层**: 核心业务逻辑（零依赖）

### 模块化设计 (Modular Design)
- 15 个功能域
- 每个模块独立
- 按需使用

### 事件驱动 (Event-Driven)
- Progress Channel（数据流）
- Control Channel（审批流）
- Monitor Channel（治理流）

---

## 📊 对比其他框架

| 特性 | aster | LangChain | CrewAI |
|------|----------|-----------|--------|
| **语言** | Go + TypeScript | Python | Python |
| **架构** | 三层解耦 | 单体 | 单体 |
| **框架依赖** | 零依赖 (核心) | 强依赖 | 强依赖 |
| **性能** | 极高 (Go) | 中等 | 中等 |
| **事件驱动** | ✅ 三通道 | ❌ | ❌ |
| **Middleware** | ✅ 洋葱模型 | ✅ | ❌ |
| **类型安全** | ✅ 完整 | ❌ | ❌ |
| **可扩展性** | ✅ 极强 | ✅ | ⚠️ |

---

## 🚀 设计亮点

### 1. 生产级 Server 层 ✨ 新增
完整的企业级服务器实现：
- **认证授权**: API Key + JWT + RBAC
- **可观测性**: Prometheus + OpenTelemetry
- **速率限制**: 令牌桶 + 滑动窗口
- **健康检查**: 增强的健康监控
- **优雅关闭**: 生产就绪

### 2. 零框架依赖
核心 SDK 不依赖任何 HTTP 框架，用户可以：
- 使用 Gin、Echo、Chi 或标准库
- 避免版本冲突
- 更小的依赖树

### 3. 事件驱动
三通道设计支持：
- 实时进度更新
- 工具审批流程
- 成本和合规监控

### 4. 极致性能
- Middleware Stack: ~27M ops/s
- Backend Write: ~3.8M ops/s
- HTTP 吞吐量: 10,000+ req/s
- 基于 Go 的高并发

### 5. 完整类型系统
- 100+ TypeScript 接口
- 20+ 事件类型
- 完整的错误类型层次
- 端到端类型安全

### 6. 框架集成
- React Hooks
- Vercel AI SDK
- Prometheus + Grafana
- Jaeger + Zipkin
- Vue Composables (未来)

---

## 📖 相关文档

### 核心概念
- [Agents](../02.core-concepts/1.agents.md) - Agent 基础
- [Memory](../04.memory/) - 记忆系统
- [Middleware](../06.middleware/) - 中间件系统
- [Human-in-the-Loop](../06.middleware/2.builtin/human-in-the-loop.md) - 人工审核
- [Workflows](../07.workflows/) - 工作流编排

### API 参考
- [REST API](../14.api-reference/) - HTTP API 文档
- [Client SDKs](../14.api-reference/6.client-sdks.md) - 客户端 SDK 文档

### 部署指南
- [Docker 部署](../09.deployment/1.docker.md)
- [Kubernetes](../09.deployment/2.kubernetes.md)
- [生产最佳实践](../09.deployment/4.production.md)

---

## 🎓 学习路径

### 初学者
1. 阅读 [架构概览](./1.overview.md)
2. 查看 [快速开始](../01.introduction/2.quick-start.md)
3. 尝试 [示例代码](../12.examples/)

### 进阶开发者
1. 深入 [核心 SDK 架构](./2.core-sdk.md)
2. 学习 [Middleware 系统](../06.middleware/)
3. 探索 [工作流编排](../07.workflows/)

### 架构师
1. 研究所有架构文档
2. 了解 [部署架构](../09.deployment/)
3. 参考 [最佳实践](../15.best-practices/)

---

## 💡 设计理念

aster 的架构设计基于以下核心理念：

### 1. 简单优于复杂
- 清晰的分层
- 明确的职责
- 最小化依赖

### 2. 灵活优于强制
- 用户可选择框架
- 模块化设计
- 可替换组件

### 3. 性能优于便利
- Go 语言实现
- 零拷贝设计
- 高效并发

### 4. 类型安全优于动态
- 完整的类型定义
- 编译时检查
- IDE 友好

---

**最后更新**: 2024年11月17日  
**版本**: v2.0

---

## 反馈和贡献

如果你对架构有任何疑问或建议，欢迎：
- 提交 Issue
- 发起 Discussion
- 贡献 PR

aster 是一个开源项目，我们欢迎社区的反馈和贡献！

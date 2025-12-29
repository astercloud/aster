# Changelog

## [v0.35.0] - 2025-12-29

### Added

- **UI Protocol A2UI v0.9 对齐增强**
  - `createSurface` 消息类型：支持创建 Surface 并指定 `catalogId` 组件目录
  - `DataModelOperation` 操作类型：支持 `add`（追加/合并）、`replace`（替换）、`remove`（删除）三种数据操作
  - `actionContext` 按钮属性：点击时自动解析路径引用，收集表单数据到 `UIActionEvent.context`
  - `ValidationError` 标准格式：JSON Pointer 路径指向错误位置，便于 LLM 自我纠正
  - `ClientMessage` 类型：`UserActionMessage` 和 `ProtocolError` 客户端到服务端消息
  - 简化 `PropertyValue` 格式：支持 A2UI 风格直接字面值（string/number/boolean）
  - `UIActionEvent` 增强：添加 `timestamp`（ISO 8601）和 `context` 字段

- **Go 类型定义** (`pkg/types/ui_protocol.go`)
  - `DataModelOperation` 枚举类型
  - `CreateSurfaceMessage` 结构体
  - `ClientMessage`、`UserActionMessage` 类型
  - `ProtocolError`、`ValidationError` 错误类型
  - `ButtonProps.ActionContext` 字段
  - `UIActionEvent.Timestamp`、`UIActionEvent.Context` 字段

- **TypeScript 类型定义** (`ui/src/types/ui-protocol.ts`)
  - 完整的 A2UI 对齐类型定义
  - `parseExtendedPropertyValue()`、`normalizePropertyValue()` 工具函数
  - `createUIActionEvent()`、`createUserActionMessage()` 工厂函数
  - `createValidationError()`、`createGenericError()` 错误创建函数

- **MessageProcessor 增强** (`ui/src/protocol/message-processor.ts`)
  - `processCreateSurface()` 方法
  - `validateMessage()` 消息验证方法
  - `addDataToSurface()`、`removeDataFromSurface()` 数据操作方法
  - `resolveActionContext()` 上下文解析方法
  - Surface `catalogId` 存储支持

- **Client Message 模块** (`ui/src/protocol/client-message.ts`)
  - `createUserActionMessage()` 用户动作消息创建
  - `createErrorMessage()` 错误消息创建
  - `resolveActionContext()` 路径引用解析
  - `serializeClientMessage()`、`parseClientMessage()` 序列化函数

- **Path Resolver 增强** (`ui/src/protocol/path-resolver.ts`)
  - `addData()` 函数：数组追加、对象合并、路径创建
  - `removeData()` 函数：值删除、数组元素删除并重新索引

### Changed

- **AsterButton 组件**：支持 `actionContext` 属性，点击时自动解析并包含在事件中
- **useUIAction composable**：使用 `createUIActionEvent()` 生成标准格式事件

### Tests

- 新增 8 个测试文件，共 173+ 新测试用例
  - `data-operations.spec.ts` - 数据模型操作单元测试
  - `action-context.spec.ts` - Action Context 解析测试
  - `property-value-parsing.spec.ts` - 简化属性值格式测试
  - `create-surface.spec.ts` - CreateSurface 消息测试
  - `validation-error.spec.ts` - 验证错误格式测试
  - `data-operations.property.spec.ts` - 数据操作属性测试 (Property 17, 18)
  - `action-context.property.spec.ts` - Action Context 属性测试 (Property 19)
  - `property-value.property.spec.ts` - 属性值兼容性测试 (Property 20)
- Go 类型测试：12 个新测试用例覆盖所有 A2UI 类型

### Documentation

- 更新 UI Protocol 文档 (`docs/content/02.core-concepts/19.ui-protocol.md`)
  - 添加 `createSurface` 消息说明
  - 添加数据模型操作类型 (add/replace/remove) 说明
  - 添加 Action Context 使用示例
  - 添加简化属性值格式说明
  - 添加验证错误格式说明
  - 添加 Client-to-Server 消息说明

## [v0.33.0] - 2025-12-28

### Added

- **消息元数据与可见性控制**
  - 新增 `MessageMetadata` 结构体，支持 `UserVisible` 和 `AgentVisible` 字段
  - 工厂方法：`NewMessageMetadata()`、`AgentOnly()`、`UserOnly()`、`Invisible()`
  - 消息过滤函数：`FilterMessagesForAgent()`、`FilterMessagesForUser()`、`FilterMessagesBySource()`、`FilterMessagesByTag()`
  - 支持消息来源标识和自定义标签

- **工具注解系统**
  - 新增 `ToolAnnotations` 结构体，描述工具安全特征（ReadOnly、Destructive、Idempotent、OpenWorld、RiskLevel）
  - 预定义注解模板：`AnnotationsSafeReadOnly`、`AnnotationsSafeWrite`、`AnnotationsDestructiveWrite`、`AnnotationsExecution`、`AnnotationsNetworkRead` 等
  - 新增 `AnnotatedTool` 接口和辅助函数 `GetAnnotations()`、`IsToolSafeForAutoApproval()`、`GetToolRiskLevel()`
  - 新增 `PermissionModeSmartApprove` 权限模式：只读工具自动批准
  - 为内置工具（Read、Write、Glob、Grep、Bash、WebFetch、WebSearch）添加注解

- **上下文压缩增强**
  - 新增 `CompactionStrategy` 配置，支持渐进式压缩策略
  - 渐进式删除工具响应：0% → 10% → 20% → 50% → 100%
  - 新增 `UseMetadataVisibility` 选项：使用元数据控制可见性而非删除消息
  - 新方法：`progressiveCompact()`、`removeToolResponses()`、`summarizeWithMetadata()`

### Documentation

- 新增工具注解系统文档 (`docs/content/05.tools/1.overview/annotations.md`)
- 新增消息元数据文档 (`docs/content/02.core-concepts/18.message-metadata.md`)
- 更新 Summarization 中间件文档，添加渐进式压缩策略章节

### Tests

- 新增消息元数据单元测试 (`pkg/types/message_test.go`)
- 新增工具注解单元测试 (`pkg/tools/annotations_test.go`)
- 修复 `TestExitPlanModeTool_ConcurrentAccess` 测试

## [v0.32.0] - 2025-12-13

### Added

- **Aster Studio 可观测性增强**
  - 新增 MySQL 存储后端 (`pkg/store/mysql.go`)，支持持久化 Agent 数据
  - 支持三种存储类型：JSON（默认）、MySQL、Redis
  - 通过环境变量配置存储：`ASTER_STORE_TYPE`、`ASTER_MYSQL_DSN`、`ASTER_REDIS_ADDR`

- **事件流筛选功能**
  - 支持按通道筛选（progress/control/monitor）
  - 支持按事件类型筛选（error/token_usage/state_changed 等）
  - 修复远程 Agent 事件的筛选逻辑

- **远程 Agent 事件改进**
  - 修复 `map[string]any` 类型事件的 channel/type 提取
  - 改进 `shouldForward` 函数支持远程 Agent 事件筛选

- **文档更新**
  - 新增 Aster Studio 文档 (`docs/content/10.observability/5.studio/`)
  - 包含：概览、事件流、远程 Agent 集成指南

### Changed

- **Store Factory**: 添加 MySQL 存储类型支持
- **main.go**: 支持通过环境变量配置存储后端
- **dashboard_events.go**: 增强事件筛选逻辑

### Fixed

- **aggregator.go**: 修复 `Cost` 类型名为 `CostAmount`
- **事件筛选**: 修复远程 Agent 事件无法筛选的问题

## [v0.28.0] - 2025-12-10

### Added

- **LocalSandbox 安全增强**: 全面提升本地沙箱的安全性
  - `SecurityLevel`: 四级安全级别 (None, Basic, Strict, Paranoid)
  - 增强危险命令检测: 70+ 正则模式，覆盖文件破坏、权限提升、系统控制、远程代码执行、网络攻击等
  - `ResourceLimits`: 资源限制配置 (CPU时间、内存、文件大小、进程数、输出大小)
  - `AuditEntry`: 完整审计日志，记录所有命令执行
  - `CommandStats`: 命令统计，追踪调用次数和执行时间
  - 命令白名单: 严格模式下只允许预定义的安全命令
  - 动态阻止列表: 运行时添加/移除阻止命令
  - 安全环境变量: 过滤危险环境变量 (LD_PRELOAD, DYLD_INSERT_LIBRARIES 等)
  - 命令注入检测: 检测反引号、命令替换、换行符注入等
  - 路径安全检查: 检测敏感路径访问和路径遍历攻击
  - 关键命令保护: 即使排除命令也检查最危险的模式

### Changed

- **randomString**: 使用 crypto/rand 替代 time-based 实现，提高安全性
- **execDirect**: 排除命令仍执行关键安全检查

### Tests

- 添加 8 个新的安全测试用例
  - TestLocalSandbox_SecurityLevels
  - TestLocalSandbox_EnhancedDangerousPatterns (16 个危险命令)
  - TestLocalSandbox_AuditLog
  - TestLocalSandbox_CommandStats
  - TestLocalSandbox_BlockedCommands
  - TestLocalSandbox_DynamicBlockedCommands
  - TestLocalSandbox_SetSecurityLevel

## [v0.27.0] - 2025-12-10

### Added

- **Claude Agent SDK 风格沙箱系统**: 实现与 Claude Agent SDK 对齐的沙箱和权限系统
  - `SandboxSettings`: 细粒度沙箱安全配置，支持 `AutoAllowBashIfSandboxed`、`ExcludedCommands`、`AllowUnsandboxedCommands`
  - `NetworkSandboxSettings`: 网络隔离配置，支持主机白名单/黑名单、Unix Socket 控制、代理端口配置
  - `SandboxIgnoreViolations`: 按模式忽略特定文件/网络违规
  - `SandboxPermissionMode`: 四种权限模式 (default, acceptEdits, bypassPermissions, plan)

- **CanUseTool 回调机制**: 自定义权限检查回调，允许应用层完全控制工具权限
  - `CanUseToolFunc`: 权限检查函数类型
  - `CanUseToolOptions`: 权限检查选项，包含沙箱状态和绕过请求信息
  - `PermissionResult`: 权限检查结果，支持修改输入参数和动态权限更新

- **EnhancedInspector 增强权限检查器**: 整合 CanUseTool 回调、沙箱配置和规则管理
  - 会话级规则支持（临时规则，会话结束自动清除）
  - 动态权限更新（运行时添加/移除规则）
  - 违规记录和查询功能
  - 与 Agent 深度集成

- **dangerouslyDisableSandbox 机制**: 允许模型请求绕过沙箱，需要权限审批

- **sandbox-permission 示例**: 完整演示新沙箱权限系统的使用

### Changed

- **LocalSandbox 增强**: 添加 `isExcludedCommand()`、`execDirect()`、`CheckNetworkAccess()`、`CheckUnixSocketAccess()`、`ShouldIgnoreViolation()`、`GetSettings()`、`IsEnabled()` 方法
- **SandboxFactory 更新**: 支持传递 Settings 到 LocalSandbox
- **Agent 集成**: 添加 `permissionInspector` 字段，在工具执行前进行权限检查

### Documentation

- 更新 `docs/content/02.core-concepts/5.sandbox.md` 沙箱系统文档
- 更新 `docs/content/08.security/2.permission.md` 权限系统文档
- 更新 `docs/content/12.examples/9.desktop/index.md` 桌面应用示例文档

## [v0.24.0] - 2025-12-09

### Added

- **EventBus 内存泄漏修复**: 添加 `NewEventBusWithConfig()` 构造函数，支持可配置的清理设置
- **EventBus 自动清理**: 实现后台清理 worker，按时间（默认1小时）和数量（默认10000）自动清理旧事件
- **EventBus 优雅关闭**: 添加 `Close()` 方法支持优雅关闭
- **ComponentLogger**: 在 `pkg/logging` 添加组件级别日志记录器，支持 `ForComponent()` 创建
- **Printf 兼容方法**: 添加 `Printf` 方法便于从 `log.Printf` 迁移

### Changed

- **结构化日志迁移**: 将 `pkg/` 下所有 `log.Printf` 调用迁移到结构化日志
  - `pkg/agent/` - agent.go, actor.go, streaming.go, processor.go, model_fallback.go, tool_manager.go, session.go
  - `pkg/provider/` - anthropic.go, deepseek.go, glm.go, openai_compatible.go
  - `pkg/middleware/` - 14个中间件文件
  - `pkg/events/` - bus.go
  - `pkg/actor/` - system.go
  - `pkg/skills/` - injector.go, manager.go
  - `pkg/workflow/` - actor_engine.go
  - `pkg/mcpserver/` - docstools.go
  - `pkg/a2a/` - server.go
  - `pkg/tools/builtin/` - ask_user.go, write.go
  - `pkg/core/` - scheduler.go, room.go
  - `pkg/memory/` - lineage.go
  - `pkg/memory/logic/` - consolidation.go

### Fixed

- **randomString 性能优化**: 修复 `pkg/events/bus.go` 中低效的 `randomString()` 实现，从 `time.Sleep` 循环改为 `crypto/rand`

### Tests

- 添加 EventBus 清理功能的 8 个测试用例

## [v0.15.0] - 2025-11-24

### Added

- 完成15个工具的全面测试验证
- 添加工具测试文档和报告

### Fixed

- **WebSearch工具**: 修复search_depth参数问题，从"general"改为"basic"以符合Tavily API规范
- 验证多轮推理功能正常工作（streaming.go）

### Tested

- 网络工具: HttpRequest ✅, WebSearch ✅
- 文件工具: Read ✅, Write ✅, Edit ✅, Glob ✅, Grep ✅
- 执行工具: Bash ✅, BashOutput ✅, KillShell ✅
- 任务管理: TodoWrite ✅, Task ✅, ExitPlanMode ✅
- 高级工具: DemoLongTask ✅, Skill ⚠️, SemanticSearch ⚠️

### Notes

- Skill和SemanticSearch工具需要额外配置才能完整使用
- 所有核心工具调用功能已验证正常
- 系统可投入实际使用

## [v0.14.0] - Previous Release

- 基础工具实现
- 多轮推理支持

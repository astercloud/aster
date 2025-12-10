# Changelog

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

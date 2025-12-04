# Changelog

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

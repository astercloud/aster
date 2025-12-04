---
title: 评估系统
description: 自动化评估 Agent 的质量和性能
---

# 评估系统

Evals 系统帮助你衡量和改进 Agent 的表现。

## 📚 分类

### [内置评分器](/evals/builtin-scorers)

- 答案质量评分
- 工具使用评分
- 上下文相关性

### [自定义评分器](/evals/custom-scorers)

- 创建评分器
- 评分逻辑
- 结果分析

### [CI 集成](/evals/ci-integration)

- GitHub Actions
- GitLab CI
- 回归测试

## 🚀 快速开始

```go
// 创建评估
eval := evals.New("my-eval")
eval.AddScorer(evals.AnswerQualityScorer)
eval.AddTestCase(testCase)

// 运行评估
results, err := eval.Run(ctx, agent)
```

## 📖 相关文档

- [最佳实践：测试](/best-practices/testing)

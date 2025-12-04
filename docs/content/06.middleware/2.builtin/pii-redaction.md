---
title: PII Auto-Redaction
description: PII自动脱敏中间件 - 自动检测和脱敏个人敏感信息
navigation:
  icon: i-lucide-shield-check
---

# PII 自动脱敏 (PII Auto-Redaction)

## 概述

PII（Personally Identifiable Information，个人身份信息）自动脱敏是 aster 的安全功能，可以在将消息发送到 LLM 之前自动检测并脱敏敏感信息，防止数据泄漏。

## 功能特性

- **自动检测**: 使用正则表达式和验证器自动检测多种 PII 类型
- **多种脱敏策略**: 支持掩码、替换、哈希和自适应策略
- **中间件集成**: 无缝集成到 Agent 工作流中
- **PII 追踪**: 可选的 PII 匹配追踪功能
- **敏感度分级**: 根据数据敏感度自动选择合适的脱敏策略

## 支持的 PII 类型

### 基本类型

- **邮箱地址** (`PIIEmail`): 如 `user@example.com`
- **电话号码**:
  - 美国电话 (`PIIPhone`): 如 `(555) 123-4567`
  - 中国手机 (`PIIChinesePhone`): 如 `13812345678`
- **IP 地址** (`PIIIPAddress`): 如 `192.168.1.1`
- **出生日期** (`PIIDateOfBirth`): 如 `1990-01-01`

### 高敏感类型

- **信用卡号** (`PIICreditCard`): 支持 Visa、MasterCard、Amex 等
  - 使用 Luhn 算法验证
  - 支持带破折号/空格格式
- **美国社会安全号** (`PIISSNus`): 如 `123-45-6789`
  - 验证区域号、组号、序列号
- **中国身份证** (`PIIChineseID`): 18位身份证号
  - 验证校验码
- **护照号** (`PIIPassport`): 通用格式

## 快速开始

### 1. 创建 PII 脱敏中间件

```go
package main

import (
    "github.com/astercloud/aster/pkg/security"
    "github.com/astercloud/aster/pkg/agent"
)

func main() {
    // 使用默认配置
    piiMiddleware := security.NewDefaultPIIMiddleware()

    // 创建 Agent 并添加中间件
    agent := agent.NewAgent(agent.Config{
        Name: "my-agent",
    })

    agent.AddMiddleware(piiMiddleware)
}
```

### 2. 自定义配置

```go
// 创建检测器
detector := security.NewRegexPIIDetector()

// 选择脱敏策略
strategy := security.NewAdaptiveStrategy() // 自适应策略

// 配置中间件
piiMiddleware := security.NewPIIRedactionMiddleware(security.PIIMiddlewareConfig{
    Detector:       detector,
    Strategy:       strategy,
    EnableTracking: true, // 启用 PII 追踪
    Priority:       200,  // 中间件优先级
})
```

## 脱敏策略

### MaskStrategy - 掩码策略

部分掩码，保留部分信息以供识别：

```go
strategy := security.NewMaskStrategy()

// 邮箱: john.doe@example.com  -> j*******@example.com
// 电话: 13812345678           -> 138****5678
// 信用卡: 4532148803436464    -> 4532********6464
```

**配置选项**:

```go
strategy := &security.MaskStrategy{
    MaskChar:      '*',  // 掩码字符
    KeepPrefix:    3,    // 保留前缀长度
    KeepSuffix:    4,    // 保留后缀长度
    MinMaskLength: 4,    // 最小掩码长度
}
```

### ReplaceStrategy - 替换策略

完全替换为类型标签：

```go
strategy := security.NewReplaceStrategy()

// 邮箱: user@example.com      -> [EMAIL]
// 电话: 13812345678           -> [CHINESE_PHONE]
// 信用卡: 4532148803436464    -> [CREDIT_CARD]
```

**自定义标签**:

```go
strategy := &security.ReplaceStrategy{
    UseTypeLabel: true,
    CustomLabels: map[security.PIIType]string{
        security.PIIEmail: "[用户邮箱]",
        security.PIIChinesePhone: "[手机号]",
    },
}
```

### HashStrategy - 哈希策略

使用 SHA256 单向加密：

```go
strategy := security.NewHashStrategy()

// 任何 PII -> [HASH:a3f58b1d...]
```

**配置选项**:

```go
strategy := &security.HashStrategy{
    ShowPrefix:   true,
    PrefixLength: 8,
    Salt:         "your-secret-salt", // 生产环境使用随机盐值
}
```

### AdaptiveStrategy - 自适应策略

根据敏感度自动选择策略：

```go
strategy := security.NewAdaptiveStrategy()

// 低敏感（如邮箱）   -> MaskStrategy
// 中等敏感（如电话） -> MaskStrategy
// 高敏感（如信用卡） -> ReplaceStrategy
```

**自定义策略**:

```go
strategy := &security.AdaptiveStrategy{
    LowStrategy:    security.NewMaskStrategy(),
    MediumStrategy: security.NewMaskStrategy(),
    HighStrategy:   security.NewReplaceStrategy(),
}
```

## PII 追踪和分析

启用追踪后，可以查看检测到的 PII 信息：

```go
// 启用追踪
piiMiddleware := security.NewPIIRedactionMiddleware(security.PIIMiddlewareConfig{
    EnableTracking: true,
    // ...
})

// 获取 PII 匹配记录
matches := piiMiddleware.GetPIIMatches("agent-id")
for _, match := range matches {
    fmt.Printf("检测到 %s: %s (置信度: %.2f, 敏感度: %d)\n",
        match.Type, match.Value, match.Confidence, match.Severity)
}

// 获取 PII 摘要
summary := piiMiddleware.GetPIISummary("agent-id")
fmt.Printf("检测到 %d 个 PII\n", summary.TotalMatches)
fmt.Printf("最高风险级别: %d\n", summary.HighestRisk)
for piiType, count := range summary.TypeCounts {
    fmt.Printf("  %s: %d 个\n", piiType, count)
}

// 清除追踪信息
piiMiddleware.ClearTracking("agent-id")
```

## 条件脱敏

根据上下文条件决定是否脱敏：

```go
// 创建条件中间件
conditionalMw := security.NewConditionalPIIMiddleware(security.ConditionalPIIConfig{
    Detector: security.NewRegexPIIDetector(),
    Strategy: security.NewAdaptiveStrategy(),
    Condition: func(ctx context.Context, req *middleware.ModelRequest) bool {
        // 只对特定 Agent 脱敏
        if req.Metadata != nil {
            if agentType, ok := req.Metadata["agent_type"].(string); ok {
                return agentType == "public-facing"
            }
        }
        return false
    },
})
```

## 高级用法

### 1. 自定义 PII 模式

```go
import "regexp"

// 添加自定义模式
customPattern := security.PIIPattern{
    Type:        security.PIICustom,
    Description: "Employee ID",
    Regex:       regexp.MustCompile(`\bEMP-\d{6}\b`),
    Validator:   func(value string) bool {
        // 自定义验证逻辑
        return len(value) == 10
    },
}

security.AddCustomPattern(customPattern)
```

### 2. 按类型检测

```go
detector := security.NewRegexPIIDetector()

// 只检测邮箱和电话
matches, err := detector.DetectTypes(ctx, text,
    security.PIIEmail,
    security.PIIChinesePhone,
)
```

### 3. 上下文过滤

```go
// 创建上下文
piiContext := &security.PIIContext{
    MinConfidence:  0.8,                   // 最低置信度
    AllowedTypes:   []security.PIIType{security.PIIEmail}, // 白名单
    IgnorePatterns: []string{"@company.com"}, // 忽略公司邮箱
}

// 过滤匹配结果
filtered := security.FilterMatchesByContext(matches, piiContext)
```

### 4. 直接使用 Redactor

不使用中间件，直接脱敏文本：

```go
detector := security.NewRegexPIIDetector()
strategy := security.NewMaskStrategy()
redactor := security.NewRedactor(detector, strategy)

// 简单脱敏
redacted, err := redactor.Redact(ctx, "My phone is 13812345678")
// -> "My phone is 138****5678"

// 带报告的脱敏
redacted, report, err := redactor.RedactWithReport(ctx, text)
fmt.Printf("脱敏了 %d 个字符\n", report.RedactedCharacters)
fmt.Printf("共 %d 个匹配\n", report.TotalMatches)
```

## 性能考虑

1. **正则表达式优化**: 模式按特异性排序，更具体的模式优先匹配
2. **去重逻辑**: 避免同一位置的重复匹配
3. **上下文检查**: 使用 context.Context 支持取消和超时
4. **追踪开销**: 如不需要追踪功能，设置 `EnableTracking: false`

## 安全最佳实践

1. **生产环境盐值**: HashStrategy 使用随机盐值
2. **定期清理**: 及时清除 PII 追踪信息
3. **审计日志**: 记录 PII 检测和脱敏操作
4. **测试覆盖**: 确保所有 PII 类型都被正确检测和脱敏
5. **敏感度分级**: 根据业务需求调整敏感度级别

## 示例场景

### 场景 1: 客服 Agent

```go
// 客服 Agent 处理用户咨询，需要脱敏用户信息
piiMiddleware := security.NewPIIRedactionMiddleware(security.PIIMiddlewareConfig{
    Detector:       security.NewRegexPIIDetector(),
    Strategy:       security.NewAdaptiveStrategy(), // 根据敏感度自适应
    EnableTracking: true,  // 启用追踪用于审计
    Priority:       100,   // 高优先级
})
```

### 场景 2: 内部 Agent

```go
// 内部 Agent 不需要严格脱敏，使用条件脱敏
conditionalMw := security.NewConditionalPIIMiddleware(security.ConditionalPIIConfig{
    Detector: security.NewRegexPIIDetector(),
    Strategy: security.NewMaskStrategy(), // 仅掩码
    Condition: func(ctx context.Context, req *middleware.ModelRequest) bool {
        // 只脱敏外部 API 调用
        return req.Metadata["external_api"] == true
    },
})
```

### 场景 3: 合规要求

```go
// 严格合规，完全替换所有 PII
piiMiddleware := security.NewPIIRedactionMiddleware(security.PIIMiddlewareConfig{
    Detector:       security.NewRegexPIIDetector(),
    Strategy:       security.NewReplaceStrategy(), // 完全替换
    EnableTracking: true,
    Priority:       50, // 最高优先级
})
```

## 测试

运行 PII 脱敏测试：

```bash
go test ./pkg/security/ -v
```

## 相关文档

- [Middleware 架构](/middleware) - 中间件系统概述
- [内置中间件](/middleware/builtin) - 其他内置中间件
- [Memory Provenance](/memory/provenance) - 内存溯源

## 参考资源

- [OWASP PII Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Protecting_PII_Cheat_Sheet.html)
- [GDPR合规指南](https://gdpr.eu/)
- [Luhn Algorithm](https://en.wikipedia.org/wiki/Luhn_algorithm)

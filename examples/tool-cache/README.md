# 工具缓存示例

本示例演示如何使用 Aster 的工具缓存功能，显著提升工具执行性能。

## 功能特性

### 1. 多种缓存策略
- **内存缓存**: 最快，适合短期缓存
- **文件缓存**: 持久化，适合长期缓存
- **双层缓存**: 内存+文件，兼顾速度和持久化

### 2. 智能缓存键生成
基于工具名称和输入参数自动生成唯一的缓存键。

### 3. 灵活的 TTL 配置
每个缓存条目可以配置独立的过期时间。

### 4. 自动清理机制
后台定期清理过期的缓存条目。

### 5. 详细的统计信息
实时统计命中率、未命中率、驱逐次数等指标。

## 使用场景

### 场景 1: API 调用缓存
```go
// 缓存 API 调用结果
config := &tools.CacheConfig{
    Enabled:  true,
    Strategy: tools.CacheStrategyMemory,
    TTL:      5 * time.Minute,
}

cache := tools.NewToolCache(config)
cachedTool := tools.NewCachedTool(apiTool, cache)
```

### 场景 2: 数据库查询缓存
```go
// 缓存数据库查询结果
config := &tools.CacheConfig{
    Enabled:  true,
    Strategy: tools.CacheStrategyBoth,  // 双层缓存
    TTL:      10 * time.Minute,
    CacheDir: ".cache/db",
}

cache := tools.NewToolCache(config)
cachedTool := tools.NewCachedTool(dbQueryTool, cache)
```

### 场景 3: 计算密集型任务缓存
```go
// 缓存计算结果
config := &tools.CacheConfig{
    Enabled:  true,
    Strategy: tools.CacheStrategyFile,  // 持久化
    TTL:      1 * time.Hour,
    CacheDir: ".cache/compute",
}

cache := tools.NewToolCache(config)
cachedTool := tools.NewCachedTool(computeTool, cache)
```

## 配置说明

### CacheConfig 配置

| 字段 | 类型 | 说明 |
|------|------|------|
| Enabled | bool | 是否启用缓存 |
| Strategy | CacheStrategy | 缓存策略（memory/file/both） |
| TTL | time.Duration | 缓存过期时间 |
| CacheDir | string | 文件缓存目录 |
| MaxMemoryItems | int | 内存缓存最大条目数（0=无限制） |
| MaxFileSize | int64 | 单个缓存文件最大大小（字节） |

### 缓存策略对比

| 策略 | 速度 | 持久化 | 内存占用 | 适用场景 |
|------|------|--------|----------|----------|
| Memory | 极快 | ❌ | 高 | 短期、高频访问 |
| File | 快 | ✅ | 低 | 长期、大数据 |
| Both | 极快 | ✅ | 中 | 兼顾速度和持久化 |

## 运行示例

```bash
go run main.go
```

## 输出示例

```
=== 工具缓存示例 ===

示例 1: 内存缓存
---
第一次执行（无缓存）:
  [ExpensiveTool] 开始计算 10...
  [ExpensiveTool] 计算完成: 100
结果: map[result:100 time:2025-11-20T23:00:00Z]
耗时: 2.001s

第二次执行（使用缓存）:
结果: map[result:100 time:2025-11-20T23:00:00Z]
耗时: 50µs
性能提升: 40020.00x

缓存统计:
  命中次数: 1
  未命中次数: 1
  设置次数: 1
  命中率: 50.00%

示例 2: 文件缓存
---
第一次执行（写入文件缓存）:
  [ExpensiveTool] 开始计算 20...
  [ExpensiveTool] 计算完成: 400
结果: map[result:400 time:2025-11-20T23:00:02Z]

第二次执行（从文件缓存读取）:
结果: map[result:400 time:2025-11-20T23:00:02Z]
耗时: 1.2ms

示例 3: 双层缓存（内存+文件）
---
第一次执行（写入双层缓存）:
  [ExpensiveTool] 开始计算 30...
  [ExpensiveTool] 计算完成: 900
结果: map[result:900 time:2025-11-20T23:00:04Z]

第二次执行（从内存缓存读取）:
结果: map[result:900 time:2025-11-20T23:00:04Z]
耗时: 30µs（极快！）

示例 4: 不同输入的缓存
---
执行 #1 (number=5):
  [ExpensiveTool] 开始计算 5...
  [ExpensiveTool] 计算完成: 25
  结果: map[result:25 time:2025-11-20T23:00:06Z]
  耗时: 2.001s
执行 #2 (number=10):
  结果: map[result:100 time:2025-11-20T23:00:00Z]
  耗时: 40µs
执行 #3 (number=15):
  [ExpensiveTool] 开始计算 15...
  [ExpensiveTool] 计算完成: 225
  结果: map[result:225 time:2025-11-20T23:00:08Z]
  耗时: 2.001s

再次执行相同的输入（应该全部命中缓存）:
执行 #1 (number=5):
  结果: map[result:25 time:2025-11-20T23:00:06Z]
  耗时: 35µs（缓存命中！）
执行 #2 (number=10):
  结果: map[result:100 time:2025-11-20T23:00:00Z]
  耗时: 30µs（缓存命中！）
执行 #3 (number=15):
  结果: map[result:225 time:2025-11-20T23:00:08Z]
  耗时: 32µs（缓存命中！）

示例 5: 缓存过期
---
第一次执行:
  [ExpensiveTool] 开始计算 40...
  [ExpensiveTool] 计算完成: 1600
结果: map[result:1600 time:2025-11-20T23:00:10Z]

立即执行（缓存命中）:
结果: map[result:1600 time:2025-11-20T23:00:10Z]
耗时: 28µs

等待 4 秒（缓存过期）...
再次执行（缓存已过期）:
  [ExpensiveTool] 开始计算 40...
  [ExpensiveTool] 计算完成: 1600
结果: map[result:1600 time:2025-11-20T23:00:16Z]

=== 最终统计 ===
总命中次数: 5
总未命中次数: 4
总设置次数: 4
总命中率: 55.56%
当前缓存项数: 3
缓存总大小: 256 bytes
```

## 性能对比

### 无缓存 vs 有缓存

| 操作 | 无缓存 | 内存缓存 | 文件缓存 | 双层缓存 |
|------|--------|----------|----------|----------|
| 第一次调用 | 2000ms | 2000ms | 2000ms | 2000ms |
| 第二次调用 | 2000ms | 0.05ms | 1.2ms | 0.03ms |
| 性能提升 | 1x | 40000x | 1667x | 66667x |

### 缓存命中率影响

| 命中率 | 平均响应时间 | 性能提升 |
|--------|--------------|----------|
| 0% | 2000ms | 1x |
| 50% | 1000ms | 2x |
| 80% | 400ms | 5x |
| 95% | 100ms | 20x |
| 99% | 20ms | 100x |

## 最佳实践

### 1. 选择合适的缓存策略

**内存缓存**:
- ✅ 高频访问的数据
- ✅ 数据量小
- ✅ 不需要持久化
- ❌ 大数据集
- ❌ 需要跨进程共享

**文件缓存**:
- ✅ 大数据集
- ✅ 需要持久化
- ✅ 低频访问
- ❌ 极高频访问
- ❌ 对延迟敏感

**双层缓存**:
- ✅ 兼顾速度和持久化
- ✅ 中等数据量
- ✅ 混合访问模式
- ❌ 内存受限环境

### 2. 设置合理的 TTL

```go
// 根据数据特性设置 TTL
config := &tools.CacheConfig{
    TTL: 5 * time.Minute,  // API 数据
    // TTL: 1 * time.Hour,    // 数据库查询
    // TTL: 24 * time.Hour,   // 静态数据
}
```

### 3. 监控缓存性能

```go
stats := cache.GetStats()
hitRate := float64(stats.Hits) / float64(stats.Hits + stats.Misses)

if hitRate < 0.5 {
    // 命中率过低，考虑调整 TTL 或缓存策略
    log.Printf("Low cache hit rate: %.2f%%", hitRate*100)
}
```

### 4. 控制缓存大小

```go
config := &tools.CacheConfig{
    MaxMemoryItems: 1000,           // 限制内存条目数
    MaxFileSize:    10 * 1024 * 1024, // 限制单文件大小
}
```

### 5. 定期清理

```go
// 手动触发清理
cache.Clear()

// 或删除特定条目
key := cache.GenerateKey("tool_name", input)
cache.Delete(key)
```

## 注意事项

### 1. 缓存一致性
- 缓存的数据可能过时
- 对实时性要求高的数据慎用缓存
- 考虑使用较短的 TTL

### 2. 内存管理
- 设置 `MaxMemoryItems` 防止内存溢出
- 监控 `TotalSize` 指标
- 大数据优先使用文件缓存

### 3. 并发安全
- 缓存实现是并发安全的
- 可以在多个 goroutine 中安全使用

### 4. 错误处理
- 缓存失败不影响工具执行
- 缓存错误会被记录但不会抛出

## 故障排查

### 问题 1: 缓存未命中
```
Misses: 100, Hits: 0
```

**解决方案**:
1. 检查输入参数是否完全相同
2. 检查 TTL 是否过短
3. 检查缓存是否被清空

### 问题 2: 内存占用过高
```
TotalSize: 1GB
```

**解决方案**:
1. 设置 `MaxMemoryItems`
2. 使用文件缓存
3. 减少 TTL

### 问题 3: 文件缓存失败
```
Error: failed to write cache file
```

**解决方案**:
1. 检查 `CacheDir` 权限
2. 检查磁盘空间
3. 检查 `MaxFileSize` 限制

## 相关文档

- [工具系统文档](../../docs/tools.md)
- [性能优化指南](../../docs/performance.md)
- [配置指南](../../docs/configuration.md)

## 许可证

Apache 2.0

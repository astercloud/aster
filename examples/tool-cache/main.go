// ToolCache 演示工具执行结果缓存功能，通过缓存耗时工具的输出来提升性能。
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/astercloud/aster/pkg/tools"
)

// ExpensiveTool 模拟一个耗时的工具
type ExpensiveTool struct{}

func (t *ExpensiveTool) Name() string {
	return "expensive_calculation"
}

func (t *ExpensiveTool) Description() string {
	return "执行耗时的计算"
}

func (t *ExpensiveTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"number": map[string]any{
				"type":        "integer",
				"description": "要计算的数字",
			},
		},
		"required": []string{"number"},
	}
}

func (t *ExpensiveTool) Prompt() string {
	return "使用此工具执行复杂的数学计算"
}

func (t *ExpensiveTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	number, ok := input["number"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid input: number must be a number")
	}

	fmt.Printf("  [ExpensiveTool] 开始计算 %v...\n", number)

	// 模拟耗时操作
	time.Sleep(2 * time.Second)

	result := number * number

	fmt.Printf("  [ExpensiveTool] 计算完成: %v\n", result)

	return map[string]any{
		"result": result,
		"time":   time.Now().Format(time.RFC3339),
	}, nil
}

func main() {
	fmt.Println("=== 工具缓存示例 ===")
	fmt.Println()

	// 示例 1: 内存缓存
	fmt.Println("示例 1: 内存缓存")
	fmt.Println("---")

	memoryConfig := &tools.CacheConfig{
		Enabled:        true,
		Strategy:       tools.CacheStrategyMemory,
		TTL:            5 * time.Minute,
		MaxMemoryItems: 100,
	}

	memoryCache := tools.NewToolCache(memoryConfig)
	expensiveTool := &ExpensiveTool{}
	cachedTool := tools.NewCachedTool(expensiveTool, memoryCache)

	ctx := context.Background()
	input := map[string]any{"number": 10.0}

	// 第一次执行（无缓存）
	fmt.Println("第一次执行（无缓存）:")
	start := time.Now()
	result1, err := cachedTool.Execute(ctx, input, nil)
	if err != nil {
		log.Fatalf("Execute failed: %v", err)
	}
	duration1 := time.Since(start)
	fmt.Printf("结果: %v\n", result1)
	fmt.Printf("耗时: %v\n\n", duration1)

	// 第二次执行（使用缓存）
	fmt.Println("第二次执行（使用缓存）:")
	start = time.Now()
	result2, err := cachedTool.Execute(ctx, input, nil)
	if err != nil {
		log.Fatalf("Execute failed: %v", err)
	}
	duration2 := time.Since(start)
	fmt.Printf("结果: %v\n", result2)
	fmt.Printf("耗时: %v\n", duration2)
	fmt.Printf("性能提升: %.2fx\n\n", float64(duration1)/float64(duration2))

	// 查看统计信息
	stats := memoryCache.GetStats()
	fmt.Println("缓存统计:")
	fmt.Printf("  命中次数: %d\n", stats.Hits)
	fmt.Printf("  未命中次数: %d\n", stats.Misses)
	fmt.Printf("  设置次数: %d\n", stats.Sets)
	fmt.Printf("  命中率: %.2f%%\n\n", float64(stats.Hits)/float64(stats.Hits+stats.Misses)*100)

	// 示例 2: 文件缓存
	fmt.Println("示例 2: 文件缓存")
	fmt.Println("---")

	fileConfig := &tools.CacheConfig{
		Enabled:  true,
		Strategy: tools.CacheStrategyFile,
		TTL:      1 * time.Hour,
		CacheDir: ".cache/tools",
	}

	fileCache := tools.NewToolCache(fileConfig)
	cachedTool2 := tools.NewCachedTool(expensiveTool, fileCache)

	input2 := map[string]any{"number": 20.0}

	// 第一次执行
	fmt.Println("第一次执行（写入文件缓存）:")
	result3, err := cachedTool2.Execute(ctx, input2, nil)
	if err != nil {
		log.Fatalf("Execute failed: %v", err)
	}
	fmt.Printf("结果: %v\n\n", result3)

	// 第二次执行（从文件读取）
	fmt.Println("第二次执行（从文件缓存读取）:")
	start = time.Now()
	result4, err := cachedTool2.Execute(ctx, input2, nil)
	if err != nil {
		log.Fatalf("Execute failed: %v", err)
	}
	duration := time.Since(start)
	fmt.Printf("结果: %v\n", result4)
	fmt.Printf("耗时: %v\n\n", duration)

	// 示例 3: 双层缓存
	fmt.Println("示例 3: 双层缓存（内存+文件）")
	fmt.Println("---")

	bothConfig := &tools.CacheConfig{
		Enabled:        true,
		Strategy:       tools.CacheStrategyBoth,
		TTL:            10 * time.Minute,
		CacheDir:       ".cache/tools",
		MaxMemoryItems: 50,
	}

	bothCache := tools.NewToolCache(bothConfig)
	cachedTool3 := tools.NewCachedTool(expensiveTool, bothCache)

	input3 := map[string]any{"number": 30.0}

	// 第一次执行
	fmt.Println("第一次执行（写入双层缓存）:")
	result5, err := cachedTool3.Execute(ctx, input3, nil)
	if err != nil {
		log.Fatalf("Execute failed: %v", err)
	}
	fmt.Printf("结果: %v\n\n", result5)

	// 第二次执行（从内存读取）
	fmt.Println("第二次执行（从内存缓存读取）:")
	start = time.Now()
	result6, err := cachedTool3.Execute(ctx, input3, nil)
	if err != nil {
		log.Fatalf("Execute failed: %v", err)
	}
	duration = time.Since(start)
	fmt.Printf("结果: %v\n", result6)
	fmt.Printf("耗时: %v（极快！）\n\n", duration)

	// 示例 4: 不同输入的缓存
	fmt.Println("示例 4: 不同输入的缓存")
	fmt.Println("---")

	inputs := []map[string]any{
		{"number": 5.0},
		{"number": 10.0},
		{"number": 15.0},
	}

	for i, input := range inputs {
		fmt.Printf("执行 #%d (number=%v):\n", i+1, input["number"])
		start := time.Now()
		result, err := cachedTool.Execute(ctx, input, nil)
		if err != nil {
			log.Fatalf("Execute failed: %v", err)
		}
		duration := time.Since(start)
		fmt.Printf("  结果: %v\n", result)
		fmt.Printf("  耗时: %v\n", duration)
	}
	fmt.Println()

	// 再次执行相同的输入（应该全部命中缓存）
	fmt.Println("再次执行相同的输入（应该全部命中缓存）:")
	for i, input := range inputs {
		fmt.Printf("执行 #%d (number=%v):\n", i+1, input["number"])
		start := time.Now()
		result, err := cachedTool.Execute(ctx, input, nil)
		if err != nil {
			log.Fatalf("Execute failed: %v", err)
		}
		duration := time.Since(start)
		fmt.Printf("  结果: %v\n", result)
		fmt.Printf("  耗时: %v（缓存命中！）\n", duration)
	}
	fmt.Println()

	// 示例 5: 缓存过期
	fmt.Println("示例 5: 缓存过期")
	fmt.Println("---")

	shortTTLConfig := &tools.CacheConfig{
		Enabled:  true,
		Strategy: tools.CacheStrategyMemory,
		TTL:      3 * time.Second,
	}

	shortTTLCache := tools.NewToolCache(shortTTLConfig)
	cachedTool4 := tools.NewCachedTool(expensiveTool, shortTTLCache)

	input4 := map[string]any{"number": 40.0}

	// 第一次执行
	fmt.Println("第一次执行:")
	result7, err := cachedTool4.Execute(ctx, input4, nil)
	if err != nil {
		log.Fatalf("Execute failed: %v", err)
	}
	fmt.Printf("结果: %v\n\n", result7)

	// 立即执行（缓存命中）
	fmt.Println("立即执行（缓存命中）:")
	start = time.Now()
	result8, err := cachedTool4.Execute(ctx, input4, nil)
	if err != nil {
		log.Fatalf("Execute failed: %v", err)
	}
	duration = time.Since(start)
	fmt.Printf("结果: %v\n", result8)
	fmt.Printf("耗时: %v\n\n", duration)

	// 等待缓存过期
	fmt.Println("等待 4 秒（缓存过期）...")
	time.Sleep(4 * time.Second)

	// 再次执行（缓存未命中）
	fmt.Println("再次执行（缓存已过期）:")
	result9, err := cachedTool4.Execute(ctx, input4, nil)
	if err != nil {
		log.Fatalf("Execute failed: %v", err)
	}
	fmt.Printf("结果: %v\n\n", result9)

	// 最终统计
	fmt.Println("=== 最终统计 ===")
	finalStats := memoryCache.GetStats()
	fmt.Printf("总命中次数: %d\n", finalStats.Hits)
	fmt.Printf("总未命中次数: %d\n", finalStats.Misses)
	fmt.Printf("总设置次数: %d\n", finalStats.Sets)
	fmt.Printf("总命中率: %.2f%%\n", float64(finalStats.Hits)/float64(finalStats.Hits+finalStats.Misses)*100)
	fmt.Printf("当前缓存项数: %d\n", finalStats.ItemCount)
	fmt.Printf("缓存总大小: %d bytes\n", finalStats.TotalSize)
}

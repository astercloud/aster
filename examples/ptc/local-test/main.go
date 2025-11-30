package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/bridge"
	"github.com/astercloud/aster/pkg/tools/builtin"
)

// 本地测试示例
// 不需要 API Key,直接测试 PTC 基础设施
func main() {
	fmt.Println("=== Aster PTC 本地测试 ===")

	// 1. 创建工具注册表
	registry := tools.NewRegistry()
	builtin.RegisterAll(registry)

	// 2. 创建 ToolBridge
	toolBridge := bridge.NewToolBridge(registry)

	// 3. 启动 HTTP 桥接服务器
	server := bridge.NewHTTPBridgeServer(toolBridge, "localhost:18080")

	fmt.Println("启动 HTTP 桥接服务器...")
	if err := server.StartAsync(); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	// 等待服务器完全启动
	time.Sleep(200 * time.Millisecond)
	fmt.Println("✅ 服务器启动成功: http://localhost:18080")

	// 4. 创建 Python 运行时
	runtime := bridge.NewPythonRuntime(nil)
	runtime.SetTools([]string{"Read", "Write", "Glob"})
	runtime.SetBridgeURL("http://localhost:18080")

	// 5. 测试 Python 代码执行
	testCases := []struct {
		name string
		code string
		desc string
	}{
		{
			name: "基础工具调用",
			desc: "测试单个工具调用",
			code: `
import asyncio

async def main():
    # 调用 Glob 工具查找 Go 文件
    files = await Glob(pattern="*.go", path=".")
    print(f"找到 {len(files)} 个 Go 文件")
    for f in files[:5]:  # 只显示前5个
        print(f"  - {f}")

asyncio.run(main())
`,
		},
		{
			name: "并发调用",
			desc: "测试并发调用多个工具",
			code: `
import asyncio

async def main():
    # 并发查找不同类型的文件
    tasks = [
        Glob(pattern="*.go", path="."),
        Glob(pattern="*.md", path="."),
        Glob(pattern="*.json", path="."),
    ]

    results = await asyncio.gather(*tasks)

    print("文件统计:")
    print(f"  Go 文件: {len(results[0])}")
    print(f"  Markdown 文件: {len(results[1])}")
    print(f"  JSON 文件: {len(results[2])}")

asyncio.run(main())
`,
		},
		{
			name: "错误处理",
			desc: "测试错误处理机制",
			code: `
import asyncio

async def main():
    try:
        # 尝试读取不存在的文件
        content = await Read(path="nonexistent_file_12345.txt")
        print(f"内容: {content}")
    except Exception as e:
        print(f"✅ 捕获到预期错误: {e}")

    # 继续执行其他任务
    files = await Glob(pattern="*.go", path=".")
    print(f"✅ 错误后继续执行成功,找到 {len(files)} 个文件")

asyncio.run(main())
`,
		},
		{
			name: "复杂数据处理",
			desc: "测试复杂的数据处理逻辑",
			code: `
import asyncio

async def main():
    # 获取所有 Go 文件
    go_files = await Glob(pattern="*.go", path=".")

    # 按目录分组
    by_dir = {}
    for f in go_files:
        parts = f.split("/")
        if len(parts) > 1:
            dir_name = "/".join(parts[:-1])
        else:
            dir_name = "."

        if dir_name not in by_dir:
            by_dir[dir_name] = []
        by_dir[dir_name].append(parts[-1])

    # 输出统计
    print(f"文件分布:")
    for dir_name, files in sorted(by_dir.items()):
        print(f"  {dir_name}: {len(files)} 个文件")

asyncio.run(main())
`,
		},
	}

	// 6. 执行测试用例
	ctx := context.Background()

	for i, tc := range testCases {
		fmt.Printf("\n[测试 %d/%d] %s\n", i+1, len(testCases), tc.name)
		fmt.Printf("说明: %s\n", tc.desc)
		fmt.Println("执行中...")

		start := time.Now()
		result, err := runtime.Execute(ctx, tc.code, map[string]any{})
		duration := time.Since(start)

		if err != nil {
			fmt.Printf("❌ 执行失败: %v\n", err)
			continue
		}

		if !result.Success {
			fmt.Printf("❌ 代码执行失败: %s\n", result.Error)
			if result.Stderr != "" {
				fmt.Printf("错误输出:\n%s\n", result.Stderr)
			}
			continue
		}

		fmt.Printf("✅ 执行成功 (耗时: %v)\n", duration)
		if result.Stdout != "" {
			fmt.Printf("\n输出:\n%s\n", result.Stdout)
		}
	}

	// 7. 性能测试
	fmt.Println("\n=== 性能测试 ===")
	fmt.Println("测试 10 次连续调用的平均延迟...")

	totalDuration := time.Duration(0)
	iterations := 10
	simpleCode := `
import asyncio
async def main():
    files = await Glob(pattern="*.go", path=".")
    print(len(files))
asyncio.run(main())
`

	for i := 0; i < iterations; i++ {
		start := time.Now()
		_, _ = runtime.Execute(ctx, simpleCode, map[string]any{})
		totalDuration += time.Since(start)
	}

	avgDuration := totalDuration / time.Duration(iterations)
	fmt.Printf("平均延迟: %v\n", avgDuration)
	fmt.Printf("QPS: %.2f 次/秒\n", 1000.0/float64(avgDuration.Milliseconds()))

	fmt.Println("\n=== 测试完成 ===")
	fmt.Println("PTC 基础设施工作正常!")
}

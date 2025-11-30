// Subagent 演示 SubAgentMiddleware 实现的任务委派功能，支持将任务分派给
// 专业化的子代理如研究员、编码员和审查员等角色。
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/astercloud/aster/pkg/backends"
	"github.com/astercloud/aster/pkg/middleware"
)

func main() {
	fmt.Println("=== SubAgent Middleware 示例 ===")

	// 1. 创建 Backend
	backend := backends.NewStateBackend()
	fmt.Println("✓ 创建 StateBackend")

	// 2. 定义子代理规格
	specs := []middleware.SubAgentSpec{
		{
			Name:        "researcher",
			Description: "深度研究和分析专家",
			Prompt:      "你是一个专注于深度研究的 AI。仔细分析问题,提供详细的研究报告。",
		},
		{
			Name:        "coder",
			Description: "代码编写和重构专家",
			Prompt:      "你是一个专业的程序员。编写高质量、可维护的代码。",
		},
		{
			Name:        "reviewer",
			Description: "代码审查和质量检查专家",
			Prompt:      "你是一个严格的代码审查员。识别潜在问题,提出改进建议。",
		},
	}
	fmt.Printf("✓ 定义了 %d 个子代理规格\n", len(specs))

	// 3. 创建子代理工厂
	factory := func(ctx context.Context, spec middleware.SubAgentSpec) (middleware.SubAgent, error) {
		// 这里使用 SimpleSubAgent 作为演示
		// 实际应用中应该创建真正的 Agent 实例
		execFn := func(ctx context.Context, description string, parentContext map[string]any) (string, error) {
			return fmt.Sprintf("[%s] 已执行任务: %s\n系统提示: %s\n上下文: %v",
				spec.Name, description, spec.Prompt, parentContext), nil
		}
		return middleware.NewSimpleSubAgent(spec.Name, spec.Prompt, execFn), nil
	}
	fmt.Println("✓ 创建子代理工厂")

	// 4. 创建 SubAgentMiddleware
	subagentMiddleware, err := middleware.NewSubAgentMiddleware(&middleware.SubAgentMiddlewareConfig{
		Specs:          specs,
		Factory:        factory,
		EnableParallel: true,
	})
	if err != nil {
		log.Fatalf("创建 SubAgentMiddleware 失败: %v", err)
	}
	fmt.Println("✓ 创建 SubAgentMiddleware")

	// 5. 创建 FilesystemMiddleware
	fsMiddleware := middleware.NewFilesystemMiddleware(&middleware.FilesystemMiddlewareConfig{
		Backend:        backend,
		EnableEviction: true,
	})
	fmt.Println("✓ 创建 FilesystemMiddleware")

	// 6. 创建 Middleware Stack
	stack := middleware.NewStack([]middleware.Middleware{
		fsMiddleware,
		subagentMiddleware,
	})
	fmt.Printf("✓ 创建 Middleware Stack (共 %d 个中间件)\n\n", len(stack.Middlewares()))

	// 7. 演示工具收集
	allTools := stack.Tools()
	fmt.Printf("=== 工具清单 ===\n")
	fmt.Printf("总计: %d 个工具\n\n", len(allTools))

	for i, tool := range allTools {
		fmt.Printf("%d. %s\n", i+1, tool.Name())
		fmt.Printf("   描述: %s\n", tool.Description())
		if tool.Name() == "task" {
			fmt.Printf("   可用子代理: %v\n", subagentMiddleware.ListSubAgents())
		}
		fmt.Println()
	}

	// 8. 演示 task 工具执行
	fmt.Println("=== 演示 Task 工具 ===")

	// 获取 task 工具
	var taskTool *middleware.TaskTool
	for _, tool := range allTools {
		if tool.Name() == "task" {
			taskTool = tool.(*middleware.TaskTool)
			break
		}
	}

	if taskTool != nil {
		// 测试 1: 研究任务
		fmt.Println("测试 1: 委托研究任务给 'researcher' 子代理")
		result1, err := taskTool.Execute(context.Background(), map[string]any{
			"description":   "分析 DeepAgents 项目的核心设计模式",
			"subagent_type": "researcher",
			"context": map[string]any{
				"project_path": "/Users/coso/Documents/dev/python/deepagents",
			},
		}, nil)
		if err != nil {
			fmt.Printf("错误: %v\n", err)
		} else {
			fmt.Printf("结果: %+v\n\n", result1)
		}

		// 测试 2: 代码任务
		fmt.Println("测试 2: 委托编码任务给 'coder' 子代理")
		result2, err := taskTool.Execute(context.Background(), map[string]any{
			"description":   "实现一个简单的 HTTP 服务器",
			"subagent_type": "coder",
		}, nil)
		if err != nil {
			fmt.Printf("错误: %v\n", err)
		} else {
			fmt.Printf("结果: %+v\n\n", result2)
		}

		// 测试 3: 审查任务
		fmt.Println("测试 3: 委托代码审查给 'reviewer' 子代理")
		result3, err := taskTool.Execute(context.Background(), map[string]any{
			"description":   "审查 pkg/middleware/subagent.go 的代码质量",
			"subagent_type": "reviewer",
		}, nil)
		if err != nil {
			fmt.Printf("错误: %v\n", err)
		} else {
			fmt.Printf("结果: %+v\n\n", result3)
		}
	}

	// 9. 演示文件系统工具
	fmt.Println("=== 演示文件系统工具 ===")

	// 写入测试文件
	_, _ = backend.Write(context.Background(), "/test/demo.txt", "Hello from SubAgent example!\nLine 2\nLine 3")
	fmt.Println("✓ 写入测试文件 /test/demo.txt")

	// 列出文件
	files, _ := backend.ListInfo(context.Background(), "/")
	fmt.Printf("✓ 根目录文件数: %d\n", len(files))

	// 搜索文件
	globResults, _ := backend.GlobInfo(context.Background(), "*.txt", "/")
	fmt.Printf("✓ 找到 %d 个 .txt 文件\n", len(globResults))

	// 10. 清理
	fmt.Println("\n=== 清理资源 ===")
	if err := stack.OnAgentStop(context.Background(), "demo-agent"); err != nil {
		fmt.Printf("清理失败: %v\n", err)
	} else {
		fmt.Println("✓ 所有中间件已清理")
	}

	fmt.Println("\n=== 示例完成 ===")
}

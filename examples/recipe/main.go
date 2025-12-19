// Recipe 演示声明式 Agent 配置系统，通过 YAML 文件定义可复用的 Agent 模板。
// Recipe 系统借鉴自 Goose 项目，支持参数化、MCP 扩展和权限配置。
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/astercloud/aster/pkg/recipe"
)

func main() {
	fmt.Println("📖 Recipe System 示例")
	fmt.Println("================================")

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "aster-recipe-demo")
	if err != nil {
		log.Fatalf("创建临时目录失败: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// 演示各种 Recipe 功能
	demonstrateBasicRecipe(tmpDir)
	demonstrateBuilderPattern()
	demonstrateMCPExtensions(tmpDir)
	demonstrateParameters(tmpDir)
	demonstratePermissions(tmpDir)

	fmt.Println("\n✅ Recipe System 示例完成!")
}

// 演示基础 Recipe
func demonstrateBasicRecipe(tmpDir string) {
	fmt.Println("\n📋 基础 Recipe")
	fmt.Println(repeatStr("-", 50))

	// 创建一个简单的 Recipe YAML
	yamlContent := `
version: "1.0"
title: Code Review Assistant
description: 帮助进行代码审查的 AI 助手
instructions: |
  你是一个专业的代码审查助手。你的职责是：
  1. 分析代码质量
  2. 发现潜在问题
  3. 提供改进建议

  请遵循以下原则：
  - 友好但专业的语气
  - 给出具体的代码示例
  - 解释为什么某些做法更好

prompt: 请审查我的代码，指出潜在的问题和改进建议。

tools:
  - Read
  - List
  - Search
  - Bash

activities:
  - 审查这个文件的代码质量
  - 检查是否有安全漏洞
  - 优化性能瓶颈
  - 检查测试覆盖率

author:
  name: Aster Team
  url: https://github.com/astercloud/aster
`

	// 保存 YAML 文件
	yamlPath := filepath.Join(tmpDir, "code-review.yaml")
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		log.Fatalf("写入 YAML 失败: %v", err)
	}
	fmt.Printf("  ✓ 创建 Recipe 文件: %s\n", yamlPath)

	// 加载 Recipe
	r, err := recipe.LoadFromFile(yamlPath)
	if err != nil {
		log.Fatalf("加载 Recipe 失败: %v", err)
	}

	fmt.Printf("\n  📖 Recipe 信息:\n")
	fmt.Printf("    标题: %s\n", r.Title)
	fmt.Printf("    版本: %s\n", r.Version)
	fmt.Printf("    描述: %s\n", r.Description)
	fmt.Printf("    工具: %v\n", r.Tools)
	fmt.Printf("    活动建议: %d 条\n", len(r.Activities))

	// 验证 Recipe
	if err := r.Validate(); err != nil {
		fmt.Printf("    ⚠️ 验证警告: %v\n", err)
	} else {
		fmt.Printf("    ✓ Recipe 验证通过\n")
	}
}

// 演示 Builder 模式
func demonstrateBuilderPattern() {
	fmt.Println("\n📋 Builder 模式创建 Recipe")
	fmt.Println(repeatStr("-", 50))

	// 使用 Builder 创建 Recipe
	r, err := recipe.NewBuilder().
		Title("Writing Assistant").
		Description("帮助撰写和编辑文档的 AI 助手").
		Instructions(`你是一个专业的写作助手。你擅长：
- 文章撰写和润色
- 语法和风格检查
- 结构优化建议
- 多语言翻译`).
		Prompt("请帮我改进这篇文章的结构和表达。").
		Tools("Read", "Write", "Search").
		PermissionMode(recipe.PermissionSmartApprove).
		Build()

	if err != nil {
		log.Fatalf("创建 Recipe 失败: %v", err)
	}

	fmt.Printf("  ✓ 使用 Builder 创建 Recipe\n")
	fmt.Printf("\n  📖 Recipe 信息:\n")
	fmt.Printf("    标题: %s\n", r.Title)
	fmt.Printf("    版本: %s\n", r.Version)
	fmt.Printf("    工具: %v\n", r.Tools)
	fmt.Printf("    权限模式: %s\n", r.PermissionMode)
}

// 演示 MCP 扩展
func demonstrateMCPExtensions(tmpDir string) {
	fmt.Println("\n📋 MCP 扩展配置")
	fmt.Println(repeatStr("-", 50))

	yamlContent := `
version: "1.0"
title: GitHub Assistant
description: 集成 GitHub 的代码助手

extensions:
  - type: stdio
    name: github
    description: GitHub API 集成
    cmd: npx
    args:
      - "-y"
      - "@anthropics/mcp-github"
    env:
      GITHUB_TOKEN: "${GITHUB_TOKEN}"
    timeout: 30
    enabled: true

  - type: sse
    name: search
    description: 搜索服务
    url: http://localhost:3000/mcp
    timeout: 10
    enabled: true

  - type: builtin
    name: filesystem
    description: 文件系统工具
    enabled: true

tools:
  - Read
  - Write
  - Search
`

	yamlPath := filepath.Join(tmpDir, "github-assistant.yaml")
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		log.Fatalf("写入 YAML 失败: %v", err)
	}

	r, err := recipe.LoadFromFile(yamlPath)
	if err != nil {
		log.Fatalf("加载 Recipe 失败: %v", err)
	}

	fmt.Printf("  ✓ 加载带 MCP 扩展的 Recipe\n")
	fmt.Printf("\n  🔌 MCP 扩展:\n")
	for _, ext := range r.Extensions {
		enabled := "✓"
		if ext.Enabled != nil && !*ext.Enabled {
			enabled = "✗"
		}
		fmt.Printf("    [%s] %s (%s) - %s\n", enabled, ext.Name, ext.Type, ext.Description)
	}
}

// 演示参数化
func demonstrateParameters(tmpDir string) {
	fmt.Println("\n📋 参数化 Recipe")
	fmt.Println(repeatStr("-", 50))

	yamlContent := `
version: "1.0"
title: Project Generator
description: 生成项目模板的 AI 助手

parameters:
  - key: project_name
    input_type: string
    requirement: required
    description: 项目名称
    default: my-project

  - key: language
    input_type: select
    requirement: required
    description: 编程语言
    default: go
    options:
      - go
      - python
      - typescript
      - rust

  - key: with_tests
    input_type: boolean
    requirement: optional
    description: 是否包含测试模板
    default: "true"

  - key: license
    input_type: select
    requirement: optional
    description: 开源许可证
    options:
      - MIT
      - Apache-2.0
      - GPL-3.0

prompt: |
  请为我创建一个名为 {{project_name}} 的 {{language}} 项目。
  {{#if with_tests}}包含测试模板。{{/if}}
  {{#if license}}使用 {{license}} 许可证。{{/if}}

tools:
  - Write
  - Bash
`

	yamlPath := filepath.Join(tmpDir, "project-generator.yaml")
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		log.Fatalf("写入 YAML 失败: %v", err)
	}

	r, err := recipe.LoadFromFile(yamlPath)
	if err != nil {
		log.Fatalf("加载 Recipe 失败: %v", err)
	}

	fmt.Printf("  ✓ 加载参数化 Recipe\n")
	fmt.Printf("\n  📝 参数列表:\n")
	for _, param := range r.Parameters {
		required := "可选"
		if param.Requirement == recipe.ParamRequired {
			required = "必填"
		}
		fmt.Printf("    - %s (%s, %s): %s\n", param.Key, param.Type, required, param.Description)
		if param.Default != "" {
			fmt.Printf("      默认值: %s\n", param.Default)
		}
		if len(param.Options) > 0 {
			fmt.Printf("      选项: %v\n", param.Options)
		}
	}
}

// 演示权限配置
func demonstratePermissions(tmpDir string) {
	fmt.Println("\n📋 权限模式配置")
	fmt.Println(repeatStr("-", 50))

	modes := []struct {
		mode recipe.PermissionMode
		desc string
	}{
		{recipe.PermissionAutoApprove, "自动批准所有工具执行"},
		{recipe.PermissionSmartApprove, "根据风险级别智能决策"},
		{recipe.PermissionAlwaysAsk, "所有工具都需要用户确认"},
	}

	fmt.Println("  支持的权限模式:")
	for _, m := range modes {
		fmt.Printf("    - %s: %s\n", m.mode, m.desc)
	}

	// 创建不同权限模式的 Recipe
	r, err := recipe.NewBuilder().
		Title("Secure Assistant").
		Description("高安全性助手").
		PermissionMode(recipe.PermissionAlwaysAsk).
		Tools("Bash", "Write", "Delete").
		Build()

	if err != nil {
		log.Fatalf("创建 Recipe 失败: %v", err)
	}

	fmt.Printf("\n  创建高安全性 Recipe:\n")
	fmt.Printf("    权限模式: %s\n", r.PermissionMode)
	fmt.Printf("    工具: %v\n", r.Tools)

	// 保存 Recipe
	yamlPath := filepath.Join(tmpDir, "secure-assistant.yaml")
	yamlData, err := r.ToYAML()
	if err != nil {
		log.Fatalf("序列化 Recipe 失败: %v", err)
	}
	if err := os.WriteFile(yamlPath, yamlData, 0644); err != nil {
		log.Fatalf("保存 Recipe 失败: %v", err)
	}
	fmt.Printf("    ✓ 保存到: %s\n", yamlPath)
}

func repeatStr(s string, n int) string {
	result := ""
	var resultSb325 strings.Builder
	for range n {
		resultSb325.WriteString(s)
	}
	result += resultSb325.String()
	return result
}

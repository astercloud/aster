// Permission 演示权限系统的三种模式：auto_approve、smart_approve 和 always_ask。
// 权限系统用于控制工具执行的审批流程，支持基于风险的智能决策。
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/astercloud/aster/pkg/permission"
	"github.com/astercloud/aster/pkg/types"
)

func main() {
	ctx := context.Background()

	fmt.Println("🔐 Permission System 示例")
	fmt.Println("================================")

	// 创建临时目录存储权限配置
	tmpDir, err := os.MkdirTemp("", "aster-permission-demo")
	if err != nil {
		log.Fatalf("创建临时目录失败: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// 演示三种模式
	demonstrateAutoApprove(ctx, tmpDir)
	demonstrateSmartApprove(ctx, tmpDir)
	demonstrateAlwaysAsk(ctx, tmpDir)
	demonstrateRules(ctx, tmpDir)

	fmt.Println("\n✅ Permission System 示例完成!")
}

// 演示自动审批模式
func demonstrateAutoApprove(ctx context.Context, tmpDir string) {
	fmt.Println("\n📋 模式 1: Auto Approve (自动审批)")
	fmt.Println(repeatStr("-", 50))

	inspector := permission.NewInspector(
		permission.ModeAutoApprove,
		permission.WithPersistPath(filepath.Join(tmpDir, "auto_permissions.json")),
		permission.WithAutoLoad(false),
	)

	// 测试各种工具
	toolCalls := []struct {
		name string
		args map[string]any
	}{
		{"read_file", map[string]any{"path": "/etc/passwd"}},
		{"write_file", map[string]any{"path": "/tmp/test.txt", "content": "hello"}},
		{"bash", map[string]any{"command": "rm -rf /important"}},
	}

	fmt.Println("  自动审批模式会批准所有工具执行:")
	for _, tc := range toolCalls {
		call := &types.ToolCallSnapshot{
			ID:        "call-1",
			Name:      tc.name,
			Arguments: tc.args,
		}
		event, _ := inspector.Check(ctx, call)
		riskLevel := inspector.GetToolRisk(tc.name)

		status := "✅ 自动批准"
		if event != nil {
			status = "⏳ 需要审批"
		}
		fmt.Printf("    %s: %s (风险: %s)\n", tc.name, status, riskLevel)
	}
}

// 演示智能审批模式
func demonstrateSmartApprove(ctx context.Context, tmpDir string) {
	fmt.Println("\n📋 模式 2: Smart Approve (智能审批)")
	fmt.Println(repeatStr("-", 50))

	inspector := permission.NewInspector(
		permission.ModeSmartApprove,
		permission.WithPersistPath(filepath.Join(tmpDir, "smart_permissions.json")),
		permission.WithAutoLoad(false),
	)

	// 测试不同风险级别的工具
	tests := []struct {
		riskName string
		toolName string
		args     map[string]any
		desc     string
	}{
		{"低风险", "read_file", map[string]any{"path": "main.go"}, "读取文件"},
		{"低风险", "list_dir", map[string]any{"path": "."}, "列出目录"},
		{"中风险", "write_file", map[string]any{"path": "test.txt", "content": "hello"}, "写入文件"},
		{"高风险", "bash", map[string]any{"command": "echo hello"}, "执行命令"},
		{"高风险", "bash", map[string]any{"command": "rm -rf /"}, "危险命令"},
	}

	fmt.Println("  智能审批模式根据风险级别决定:")
	fmt.Println("    - 低风险 (只读) → 自动批准")
	fmt.Println("    - 中风险 (写操作) → 需要审批")
	fmt.Println("    - 高风险 (系统命令) → 需要审批")
	fmt.Println()

	for _, test := range tests {
		call := &types.ToolCallSnapshot{
			ID:        "call-1",
			Name:      test.toolName,
			Arguments: test.args,
		}
		event, _ := inspector.Check(ctx, call)

		status := "✅ 自动批准"
		if event != nil {
			status = "⏳ 需要审批"
		}

		fmt.Printf("    [%s] %s (%s): %s\n", test.riskName, test.toolName, test.desc, status)
	}
}

// 演示总是询问模式
func demonstrateAlwaysAsk(ctx context.Context, tmpDir string) {
	fmt.Println("\n📋 模式 3: Always Ask (总是询问)")
	fmt.Println(repeatStr("-", 50))

	inspector := permission.NewInspector(
		permission.ModeAlwaysAsk,
		permission.WithPersistPath(filepath.Join(tmpDir, "ask_permissions.json")),
		permission.WithAutoLoad(false),
	)

	toolNames := []string{"read_file", "write_file", "bash", "list_dir"}

	fmt.Println("  总是询问模式会要求所有工具都需要审批:")
	for _, toolName := range toolNames {
		call := &types.ToolCallSnapshot{
			ID:        "call-1",
			Name:      toolName,
			Arguments: map[string]any{},
		}
		event, _ := inspector.Check(ctx, call)

		status := "⏳ 需要审批"
		if event == nil {
			status = "✅ 已批准"
		}
		fmt.Printf("    %s: %s\n", toolName, status)
	}
}

// 演示规则系统
func demonstrateRules(ctx context.Context, tmpDir string) {
	fmt.Println("\n📋 规则系统演示")
	fmt.Println(repeatStr("-", 50))

	inspector := permission.NewInspector(
		permission.ModeSmartApprove,
		permission.WithPersistPath(filepath.Join(tmpDir, "rules_permissions.json")),
		permission.WithAutoLoad(false),
	)

	// 添加自定义规则
	fmt.Println("  添加自定义规则...")

	// 规则 1: 允许所有 read_file 操作
	inspector.AddRule(permission.Rule{
		Pattern:   "read_file",
		Decision:  permission.DecisionAllowAlways,
		RiskLevel: permission.RiskLevelLow,
		Note:      "允许所有读取操作",
	})
	fmt.Println("    ✓ 规则 1: 允许所有 read_file 操作")

	// 规则 2: 禁止危险的 bash 命令
	inspector.AddRule(permission.Rule{
		Pattern:   "bash",
		Decision:  permission.DecisionDenyAlways,
		RiskLevel: permission.RiskLevelHigh,
		Conditions: []permission.Condition{
			{
				Field:    "command",
				Operator: "contains",
				Value:    "rm -rf",
			},
		},
		Note: "禁止危险的删除命令",
	})
	fmt.Println("    ✓ 规则 2: 禁止包含 'rm -rf' 的命令")

	// 规则 3: 允许写入 /tmp 目录
	inspector.AddRule(permission.Rule{
		Pattern:   "write_file",
		Decision:  permission.DecisionAllowAlways,
		RiskLevel: permission.RiskLevelMedium,
		Conditions: []permission.Condition{
			{
				Field:    "path",
				Operator: "prefix",
				Value:    "/tmp/",
			},
		},
		Note: "允许写入临时目录",
	})
	fmt.Println("    ✓ 规则 3: 允许写入 /tmp/ 目录")

	// 测试规则
	fmt.Println("\n  测试规则效果:")
	testCases := []struct {
		tool string
		args map[string]any
		desc string
	}{
		{"read_file", map[string]any{"path": "/etc/passwd"}, "读取系统文件"},
		{"bash", map[string]any{"command": "rm -rf /home"}, "危险删除命令"},
		{"bash", map[string]any{"command": "echo hello"}, "安全命令"},
		{"write_file", map[string]any{"path": "/tmp/test.txt"}, "写入临时目录"},
		{"write_file", map[string]any{"path": "/etc/hosts"}, "写入系统目录"},
	}

	for _, tc := range testCases {
		call := &types.ToolCallSnapshot{
			ID:        "call-1",
			Name:      tc.tool,
			Arguments: tc.args,
		}
		event, err := inspector.Check(ctx, call)

		var status string
		if err != nil {
			status = "❌ 拒绝"
		} else if event == nil {
			status = "✅ 允许"
		} else {
			status = "⏳ 需要审批"
		}

		fmt.Printf("    %s %v → %s\n", tc.tool, tc.args, status)
	}

	// 列出所有规则
	fmt.Println("\n  当前规则列表:")
	rules := inspector.GetRules()
	for i, rule := range rules {
		fmt.Printf("    %d. [%s] %s - %s\n", i+1, rule.RiskLevel, rule.Pattern, rule.Note)
	}
}

func repeatStr(s string, n int) string {
	result := ""
	var resultSb254 strings.Builder
	for i := 0; i < n; i++ {
		resultSb254.WriteString(s)
	}
	result += resultSb254.String()
	return result
}

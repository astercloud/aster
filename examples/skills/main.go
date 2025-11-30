// Skills 演示 Agent 技能系统的使用，支持通过命令行参数配置工作目录和
// 调试模式。
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/provider"
	"github.com/astercloud/aster/pkg/sandbox"
	"github.com/astercloud/aster/pkg/store"
	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/builtin"
	"github.com/astercloud/aster/pkg/types"
)

func main() {
	var (
		message   = flag.String("message", "", "要发送给Agent的消息")
		workspace = flag.String("workspace", "./workspace", "工作目录路径")
		debug     = flag.Bool("debug", false, "启用调试模式")
	)
	flag.Parse()

	fmt.Println("=== Agent Skills 调试模式 ===")
	if *debug {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	ctx := context.Background()

	// 创建带有 Skills 支持的 Agent
	deps := createDependencies()

	agentConfig := &types.AgentConfig{
		TemplateID: "assistant",
		ModelConfig: &types.ModelConfig{
			Provider:      "deepseek",
			Model:         "deepseek-chat",
			APIKey:        os.Getenv("DEEPSEEK_API_KEY"),
			ExecutionMode: types.ExecutionModeNonStreaming, // 🚀 非流式快速模式（小segment不会超时）
		},
		Sandbox: &types.SandboxConfig{
			Kind:    types.SandboxKindLocal,
			WorkDir: *workspace,
		},
		SkillsPackage: &types.SkillsPackageConfig{
			Source:      "local",
			Path:        ".", // 相对于 Sandbox.WorkDir，即 ./workspace
			CommandsDir: "commands",
			SkillsDir:   "skills",
			EnabledCommands: []string{
				"write",
				"analyze",
				"plan",
			},
			EnabledSkills: []string{
				"consistency-checker",
				"pdfmd",
				"pdf",
				"markdown-segment-translator",
			},
		},
	}

	ag, err := agent.Create(ctx, agentConfig, deps)
	if err != nil {
		log.Fatalf("创建 Agent 失败: %v", err)
	}
	defer func() { _ = ag.Close() }()

	fmt.Printf("Agent 创建成功: %s\n\n", ag.ID())

	// 使用命令行参数消息或默认PDF触发消息
	targetMessage := *message
	if targetMessage == "" {
		targetMessage = "请pdf处理2407.14333v5.pdf文档，提取其内容并转换为markdown格式"
	}

	fmt.Printf("--- 发送消息 ---\n")
	fmt.Printf("消息内容: %s\n\n", targetMessage)

	result, err := ag.Chat(ctx, targetMessage)
	if err != nil {
		log.Printf("AI 处理失败: %v", err)
	} else {
		fmt.Printf("AI 处理成功！\n")
		if result != nil && result.Text != "" {
			fmt.Printf("AI 响应: %s\n", result.Text)
		}
	}

	fmt.Println("\n=== 所有示例完成 ===")
}

// createDependencies 创建依赖（简化版本）
func createDependencies() *agent.Dependencies {
	// 创建基本的依赖项
	store, _ := store.NewJSONStore(".aster-store")
	toolRegistry := tools.NewRegistry()
	builtin.RegisterAll(toolRegistry)

	// 注册基本模板
	templateRegistry := agent.NewTemplateRegistry()
	templateRegistry.Register(&types.AgentTemplateDefinition{
		ID:    "assistant",
		Model: "deepseek-chat",
		SystemPrompt: "" +
			"⚠️ 关键规则 ⚠️\n" +
			"1. 当某个 Skill 的 SKILL.md 中包含明确要求使用 Bash 工具执行 Python 脚本时，你必须严格按照这些指令执行，不得擅自修改流程。\n" +
			"2. 不要尝试自己直接翻译或处理 PDF 内容，而是使用 Bash 调用相应的 Python 脚本，根据脚本输出再进行后续处理。\n" +
			"3. 如果 SKILL.md 写明“使用 Bash”，你的第一个工具调用必须是 Bash，而不是 Read 或 Write。\n\n" +
			"📦 Skills 使用规则 📦\n" +
			"- 在处理任务之前，先阅读系统提示中的 Active Skills / Skills Overview 区域。\n" +
			"- 当某个 Skill 看起来与当前任务相关时，首先使用文件类工具（例如 Read 工具，或 Bash+cat 命令）\n" +
			"  打开它的 SKILL.md 路径，并且路径要与提示中给出的完全一致（例如：`skills/pdfmd/SKILL.md`）。\n" +
			"- 阅读 SKILL.md 后，严格按照其中给出的步骤、工具选择和示例命令执行，不要自己发明流程。\n" +
			"- 对于给出明确 Bash+Python 命令的 SKILL.md，不能忽略或跳过这些命令。\n\n" +
			"📄 本示例中的 PDF → Markdown 规则 📄\n" +
			"- 当用户提到某个 PDF 文件（例如 `.pdf` 路径），并要求“转成 Markdown/MD/文本/文档”等时，必须认为 `pdfmd` 这个 Skill 是相关的。\n" +
			"- 首先使用 Read 或 Bash+cat 打开 `skills/pdfmd/SKILL.md`，看清楚需要执行的 Python 命令和具体步骤。\n" +
			"- 然后使用 Bash 工具执行 pdfmd Skill 中的 Python 脚本，例如：\n" +
			"    `python skills/pdfmd/pdf_extract.py --input \"<PDF 文件路径>\"`\n" +
			"- 将 Bash 输出视为“原始 PDF 文本”，再根据 SKILL.md 的要求进行翻译和 Markdown 结构化。\n" +
			"- 只有在成功运行 Bash 命令并拿到 PDF 文本之后，才开始翻译和排版，不要在此之前直接翻译文件。\n\n" +
			"🚀 效率规则 🚀\n" +
			"- 在保证正确性的前提下，尽量用最少的步骤完成任务。\n" +
			"- 当你已经清楚应该做什么时，直接去做，不必先解释每一步。\n" +
			"- 对于简单流程（读取→处理→写入），尽量在一次响应中完成。\n" +
			"- 避免不必要的中间确认或多轮对话。\n" +
			"- 例如：当被要求“把文件 A 翻译成 B”时，可以用 Read→翻译→Write 一次性完成。\n\n" +
			"你是一个可以访问文件系统和记忆工具的智能助手，应在合适的时候使用这些工具读取/写入文件或管理长期记忆。",
		Tools: []any{"Read", "Write", "Bash"},
	})

	return &agent.Dependencies{
		Store:            store,
		ToolRegistry:     toolRegistry,
		SandboxFactory:   sandbox.NewFactory(),
		ProviderFactory:  provider.NewMultiProviderFactory(),
		TemplateRegistry: templateRegistry,
	}
}

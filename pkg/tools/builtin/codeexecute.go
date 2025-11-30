package builtin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/tools/bridge"
)

// CodeExecuteTool 代码执行工具
// 支持 LLM 生成代码并执行，用于程序化工具调用场景
type CodeExecuteTool struct {
	runtimeManager *bridge.RuntimeManager
	toolBridge     *bridge.ToolBridge

	// PTC 支持
	httpServer    *bridge.HTTPBridgeServer
	bridgeURL     string
	serverStarted bool
	mu            sync.Mutex
}

// NewCodeExecuteTool 创建代码执行工具
func NewCodeExecuteTool(config map[string]any) (tools.Tool, error) {
	runtimeConfig := bridge.DefaultRuntimeConfig()

	// 解析配置
	if timeout, ok := config["timeout"].(float64); ok {
		runtimeConfig.Timeout = time.Duration(timeout) * time.Second
	}
	if workDir, ok := config["work_dir"].(string); ok {
		runtimeConfig.WorkDir = workDir
	}

	return &CodeExecuteTool{
		runtimeManager: bridge.NewRuntimeManager(runtimeConfig),
	}, nil
}

// NewCodeExecuteToolWithBridge 创建带桥接器的代码执行工具
func NewCodeExecuteToolWithBridge(toolBridge *bridge.ToolBridge) *CodeExecuteTool {
	return &CodeExecuteTool{
		runtimeManager: bridge.NewRuntimeManager(nil),
		toolBridge:     toolBridge,
		bridgeURL:      "http://localhost:8080", // 默认桥接 URL
	}
}

// SetBridgeURL 设置 HTTP 桥接服务器地址
func (t *CodeExecuteTool) SetBridgeURL(url string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.bridgeURL = url
}

func (t *CodeExecuteTool) Name() string {
	return "CodeExecute"
}

func (t *CodeExecuteTool) Description() string {
	return "Execute code in Python, Node.js, or Bash to perform complex operations or call tools programmatically"
}

func (t *CodeExecuteTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"language": map[string]any{
				"type":        "string",
				"enum":        []string{"python", "nodejs", "bash"},
				"description": "Programming language to use",
			},
			"code": map[string]any{
				"type":        "string",
				"description": "Code to execute. Use _input variable to access input data.",
			},
			"input": map[string]any{
				"type":        "object",
				"description": "Input data passed to the code as _input variable (Python/Node.js) or INPUT_JSON (Bash)",
			},
		},
		"required": []string{"language", "code"},
	}
}

// ensureBridgeServer 确保 HTTP 桥接服务器已启动 (PTC 支持)
func (t *CodeExecuteTool) ensureBridgeServer(tc *tools.ToolContext) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// 如果已经启动,直接返回
	if t.serverStarted {
		return nil
	}

	// 如果没有 toolBridge,不启动服务器(非 PTC 模式)
	if t.toolBridge == nil {
		return nil
	}

	// 创建并启动 HTTP 桥接服务器
	t.httpServer = bridge.NewHTTPBridgeServer(t.toolBridge, t.bridgeURL)

	// 设置工具上下文工厂
	t.httpServer.SetContextFactory(func() *tools.ToolContext {
		return tc
	})

	// 异步启动服务器
	if err := t.httpServer.StartAsync(); err != nil {
		return fmt.Errorf("failed to start HTTP bridge server: %w", err)
	}

	// 获取可用工具列表
	availableTools := t.toolBridge.ListAvailableTools()

	// 设置 RuntimeManager 的工具列表和桥接 URL
	t.runtimeManager.SetPythonTools(availableTools)
	t.runtimeManager.SetPythonBridgeURL(t.bridgeURL)

	t.serverStarted = true
	return nil
}

func (t *CodeExecuteTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	// 确保 HTTP 桥接服务器已启动 (PTC 支持)
	if err := t.ensureBridgeServer(tc); err != nil {
		return map[string]any{
			"success": false,
			"error":   fmt.Sprintf("failed to start bridge server: %v", err),
		}, nil
	}

	// 解析参数
	langStr, ok := input["language"].(string)
	if !ok {
		return nil, fmt.Errorf("language must be a string")
	}

	code, ok := input["code"].(string)
	if !ok || code == "" {
		return nil, fmt.Errorf("code must be a non-empty string")
	}

	// 转换语言类型
	var lang bridge.Language
	switch langStr {
	case "python":
		lang = bridge.LangPython
	case "nodejs":
		lang = bridge.LangNodeJS
	case "bash":
		lang = bridge.LangBash
	default:
		return map[string]any{
			"success": false,
			"error":   fmt.Sprintf("unsupported language: %s", langStr),
		}, nil
	}

	// 获取输入数据
	codeInput := make(map[string]any)
	if inputData, ok := input["input"].(map[string]any); ok {
		codeInput = inputData
	}

	// 执行代码
	result, err := t.runtimeManager.Execute(ctx, lang, code, codeInput)
	if err != nil {
		return map[string]any{
			"success": false,
			"error":   err.Error(),
		}, nil
	}

	return map[string]any{
		"success":     result.Success,
		"output":      result.Output,
		"stdout":      result.Stdout,
		"stderr":      result.Stderr,
		"error":       result.Error,
		"exit_code":   result.ExitCode,
		"duration_ms": result.Duration,
	}, nil
}

func (t *CodeExecuteTool) Prompt() string {
	return `Execute code in Python, Node.js, or Bash.

This tool allows you to run code for complex operations like:
- Data transformation and processing
- Mathematical calculations
- File manipulation
- API calls and web requests
- Programmatic tool orchestration

Available languages:
- python: Full Python 3 environment
- nodejs: Node.js runtime
- bash: Bash shell scripting

Input data is accessible via:
- Python: _input dictionary
- Node.js: _input object
- Bash: INPUT_JSON environment variable (use jq to parse)

Examples:

1. Python - Process data:
{
  "language": "python",
  "code": "result = sum(_input['numbers'])\nprint(result)",
  "input": {"numbers": [1, 2, 3, 4, 5]}
}

2. Node.js - Transform JSON:
{
  "language": "nodejs",
  "code": "const result = _input.items.map(x => x * 2);\nconsole.log(JSON.stringify(result));",
  "input": {"items": [1, 2, 3]}
}

3. Bash - System command:
{
  "language": "bash",
  "code": "echo $INPUT_URL | xargs curl -s",
  "input": {"url": "https://api.example.com"}
}

Best practices:
- Keep code simple and focused on a single task
- Print output as JSON when returning structured data
- Handle errors gracefully within the code
- Use appropriate language for the task (Python for data, Bash for system ops)`
}

// Examples 返回使用示例
func (t *CodeExecuteTool) Examples() []tools.ToolExample {
	return []tools.ToolExample{
		{
			Description: "使用 Python 计算数据",
			Input: map[string]any{
				"language": "python",
				"code":     "import json\nresult = {'sum': sum(_input['numbers']), 'avg': sum(_input['numbers'])/len(_input['numbers'])}\nprint(json.dumps(result))",
				"input": map[string]any{
					"numbers": []int{10, 20, 30, 40, 50},
				},
			},
		},
		{
			Description: "使用 Node.js 处理 JSON",
			Input: map[string]any{
				"language": "nodejs",
				"code":     "const filtered = _input.users.filter(u => u.age >= 18);\nconsole.log(JSON.stringify({adults: filtered}));",
				"input": map[string]any{
					"users": []map[string]any{
						{"name": "Alice", "age": 25},
						{"name": "Bob", "age": 15},
						{"name": "Charlie", "age": 30},
					},
				},
			},
		},
		{
			Description: "使用 Bash 执行系统命令",
			Input: map[string]any{
				"language": "bash",
				"code":     "echo \"Current directory: $(pwd)\"\necho \"Files: $(ls -la | wc -l)\"",
				"input":    map[string]any{},
			},
		},
	}
}

// AvailableLanguages 返回可用的语言列表
func (t *CodeExecuteTool) AvailableLanguages() []string {
	langs := t.runtimeManager.AvailableLanguages()
	result := make([]string, len(langs))
	for i, l := range langs {
		result[i] = string(l)
	}
	return result
}

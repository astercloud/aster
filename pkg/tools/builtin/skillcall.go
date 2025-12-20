package builtin

import (
	"context"
	"errors"

	"github.com/astercloud/aster/pkg/skills"
	"github.com/astercloud/aster/pkg/tools"
)

// SkillTool 通用的 Skill 执行工具
//
// 注意：本工具不会自动注册到默认内置工具列表中，需要在 Agent 模板中显式声明，
// 并在 ToolContext.Services 中注入 *skills.Runtime 实例，键名为 "skills_runtime"。
type SkillTool struct{}

// NewSkillTool 创建 SkillTool 实例
func NewSkillTool(config map[string]any) (tools.Tool, error) {
	return &SkillTool{}, nil
}

func (t *SkillTool) Name() string {
	return "Skill"
}

func (t *SkillTool) Description() string {
	return "Execute an executable Skill by name using the Skills Runtime"
}

func (t *SkillTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"skill": map[string]any{
				"type":        "string",
				"description": "Skill 的名称（例如 \"pdf-to-markdown\"）",
			},
			"params": map[string]any{
				"type":        "object",
				"description": "传递给 Skill 的参数键值对，将作为环境变量注入脚本。",
				"additionalProperties": map[string]any{
					"type": "string",
				},
			},
		},
		"required": []string{"skill"},
	}
}

func (t *SkillTool) Execute(ctx context.Context, input map[string]any, tc *tools.ToolContext) (any, error) {
	// 获取 Runtime
	if tc == nil || tc.Services == nil {
		return nil, errors.New("skill_call: ToolContext.Services is nil; skills runtime not available")
	}

	rtAny, ok := tc.Services["skills_runtime"]
	if !ok {
		return nil, errors.New("skill_call: skills runtime not found in ToolContext.Services (key: \"skills_runtime\")")
	}

	rt, ok := rtAny.(*skills.Runtime)
	if !ok || rt == nil {
		return nil, errors.New("skill_call: invalid skills runtime type in ToolContext.Services")
	}

	// 解析输入
	skillName, ok := input["skill"].(string)
	if !ok || skillName == "" {
		return nil, errors.New("skill_call: \"skill\" must be a non-empty string")
	}

	// params 期望为 map[string]string，但 JSON 反序列化会是 map[string]any
	params := make(map[string]string)
	if rawParams, ok := input["params"].(map[string]any); ok {
		for k, v := range rawParams {
			if s, ok := v.(string); ok {
				params[k] = s
			}
		}
	}

	// 执行 Skill
	res, err := rt.Execute(ctx, skillName, params)
	if err != nil {
		return map[string]any{
			"ok":    false,
			"error": err.Error(),
		}, nil
	}

	return map[string]any{
		"ok":        res.ExitCode == 0,
		"skill":     res.SkillName,
		"command":   res.Command,
		"exit_code": res.ExitCode,
		"stdout":    res.Stdout,
		"stderr":    res.Stderr,
		"duration":  res.Duration.String(),
	}, nil
}

func (t *SkillTool) Prompt() string {
	return `Use this tool to execute an executable Skill by name.

Usage:
- Provide "skill": the Skill name, for example "pdf-to-markdown".
- Optionally provide "params": a JSON object of key/value pairs.
  These will be exposed to the Skill script as environment variables:
  SKILL_PARAM_<UPPERCASE_KEY> = "<value>".

Requirements:
- The Agent must inject a *skills.Runtime instance into ToolContext.Services
  with key "skills_runtime".
- The target Skill must define an "executable" block in SKILL.md (runtime, entry, timeout).

Typical example:
{
  "skill": "pdf-to-markdown",
  "params": {
    "file_path": "inputs/report.pdf",
    "output_path": "outputs/report.md",
    "pages": "1-3,5"
  }
}`
}

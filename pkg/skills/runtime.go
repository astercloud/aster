package skills

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/astercloud/aster/pkg/sandbox"
)

// ExecutionResult 可执行 Skill 的运行结果
type ExecutionResult struct {
	SkillName string        // 技能名
	Command   string        // 实际执行的命令
	ExitCode  int           // 退出码
	Stdout    string        // 标准输出
	Stderr    string        // 标准错误
	Duration  time.Duration // 执行时长
}

// Runtime Skill 运行时：负责根据 SkillDefinition.Executable 调用底层 Sandbox 执行脚本
type Runtime struct {
	loader  *SkillLoader
	sandbox sandbox.Sandbox
}

// NewRuntime 创建运行时
func NewRuntime(loader *SkillLoader, sb sandbox.Sandbox) *Runtime {
	return &Runtime{
		loader:  loader,
		sandbox: sb,
	}
}

// Execute 按名称执行一个可执行 Skill。
// params 作为环境变量传入脚本（键会被转换为大写，并加上 SKILL_PARAM_ 前缀）。
// 具体参数到命令行的映射由脚本自身负责，Runtime 不做 Skill 特定解析。
func (r *Runtime) Execute(ctx context.Context, skillName string, params map[string]string) (*ExecutionResult, error) {
	if r.loader == nil || r.sandbox == nil {
		return nil, errors.New("skills runtime not properly initialized")
	}

	// 从 loader 加载 Skill 定义
	skill, err := r.loader.Load(ctx, skillName)
	if err != nil {
		return nil, fmt.Errorf("load skill %q: %w", skillName, err)
	}

	if skill.Executable == nil {
		return nil, fmt.Errorf("skill %q is not executable (missing executable config)", skillName)
	}

	// 构造命令
	cmd := buildCommand(r.sandbox.WorkDir(), skill)
	if cmd == "" {
		return nil, fmt.Errorf("skill %q has empty command/entry", skillName)
	}

	// 构造环境变量
	env := make(map[string]string)
	for k, v := range params {
		key := "SKILL_PARAM_" + strings.ToUpper(strings.ReplaceAll(k, " ", "_"))
		env[key] = v
	}

	// 设置超时
	timeout := time.Duration(0)
	if skill.Executable.TimeoutSeconds > 0 {
		timeout = time.Duration(skill.Executable.TimeoutSeconds) * time.Second
	}

	start := time.Now()
	res, err := r.sandbox.Exec(ctx, cmd, &sandbox.ExecOptions{
		Timeout: timeout,
		WorkDir: r.sandbox.WorkDir(),
		Env:     env,
	})
	dur := time.Since(start)

	if err != nil {
		return nil, fmt.Errorf("execute skill %q: %w", skillName, err)
	}

	return &ExecutionResult{
		SkillName: skillName,
		Command:   cmd,
		ExitCode:  res.Code,
		Stdout:    res.Stdout,
		Stderr:    res.Stderr,
		Duration:  dur,
	}, nil
}

// buildCommand 根据 Executable 配置构造命令字符串。
// 推荐的 runtime 值:
//   - "python": 在沙箱中执行 `python <entry>`
//   - "node" / "nodejs": 执行 `node <entry>`
//   - "bash" / "sh": 执行 `bash`/`sh <entry>`
//
// 其它 runtime 将视为直接命令行前缀，由上层自行保证环境。
func buildCommand(workDir string, skill *SkillDefinition) string {
	if skill == nil || skill.Executable == nil {
		return ""
	}

	cfg := skill.Executable

	// 如果 Command 已在 YAML 中显式指定，可以在未来扩展支持。
	// 当前实现优先使用 Runtime + Entry 的组合。

	entry := strings.TrimSpace(cfg.Entry)

	switch strings.ToLower(strings.TrimSpace(cfg.Runtime)) {
	case "python":
		if entry == "" {
			return ""
		}
		return "python " + entry
	case "node", "nodejs":
		if entry == "" {
			return ""
		}
		return "node " + entry
	case "bash":
		if entry == "" {
			return ""
		}
		return "bash " + entry
	case "sh":
		if entry == "" {
			return ""
		}
		return "sh " + entry
	default:
		// 未知 runtime：如果 entry 不是空，就直接当作命令执行
		if entry != "" {
			return entry
		}
	}

	return ""
}

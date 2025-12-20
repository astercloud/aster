package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/astercloud/aster/pkg/tools"
	"github.com/astercloud/aster/pkg/types"
)

// PromptModule 表示一个可组合的 prompt 模块
type PromptModule interface {
	Name() string
	Build(ctx *PromptContext) (string, error)
	Priority() int                     // 模块优先级，决定注入顺序
	Condition(ctx *PromptContext) bool // 是否应该注入此模块
}

// PromptContext 构建上下文
type PromptContext struct {
	Agent       *Agent
	Template    *types.AgentTemplateDefinition
	Environment *EnvironmentInfo
	Sandbox     *SandboxInfo
	Tools       map[string]tools.Tool
	Metadata    map[string]any
}

// EnvironmentInfo 环境信息
type EnvironmentInfo struct {
	WorkingDir string
	Platform   string
	OSVersion  string
	Date       time.Time
	GitRepo    *GitRepoInfo
}

// GitRepoInfo Git 仓库信息
type GitRepoInfo struct {
	IsRepo        bool
	CurrentBranch string
	MainBranch    string
	Status        string
	RecentCommits []string
}

// SandboxInfo 沙箱信息
type SandboxInfo struct {
	Kind       types.SandboxKind
	WorkDir    string
	AllowPaths []string
}

// PromptBuilder System Prompt 构建器
type PromptBuilder struct {
	modules    []PromptModule
	compressor *EnhancedPromptCompressor
}

// NewPromptBuilder 创建构建器
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{
		modules: []PromptModule{},
	}
}

// NewPromptBuilderWithCompression 创建带压缩功能的构建器
func NewPromptBuilderWithCompression(compressor *EnhancedPromptCompressor) *PromptBuilder {
	return &PromptBuilder{
		modules:    []PromptModule{},
		compressor: compressor,
	}
}

// SetCompressor 设置压缩器
func (pb *PromptBuilder) SetCompressor(compressor *EnhancedPromptCompressor) {
	pb.compressor = compressor
}

// AddModule 添加模块
func (pb *PromptBuilder) AddModule(module PromptModule) {
	pb.modules = append(pb.modules, module)
}

// Build 构建完整的 System Prompt
func (pb *PromptBuilder) Build(ctx *PromptContext) (string, error) {
	// 按优先级排序
	sort.Slice(pb.modules, func(i, j int) bool {
		return pb.modules[i].Priority() < pb.modules[j].Priority()
	})

	var sections []string

	// 获取禁用的模块列表
	var disabledModules []string
	if ctx.Template != nil && ctx.Template.Runtime != nil {
		disabledModules = ctx.Template.Runtime.DisabledPromptModules
	}

	for _, module := range pb.modules {
		// 检查是否被禁用
		moduleName := module.Name()
		isDisabled := slices.Contains(disabledModules, moduleName)
		if isDisabled {
			continue
		}

		// 检查条件
		if !module.Condition(ctx) {
			continue
		}

		// 构建模块内容
		content, err := module.Build(ctx)
		if err != nil {
			return "", fmt.Errorf("build module %s: %w", module.Name(), err)
		}

		if content != "" {
			sections = append(sections, content)
		}
	}

	systemPrompt := strings.Join(sections, "\n\n")

	// 检查是否需要压缩
	if pb.shouldCompress(systemPrompt, ctx) {
		fmt.Printf("[PromptBuilder] 🔄 Compression triggered: prompt length=%d chars, threshold=%d\n",
			len(systemPrompt), ctx.Template.Runtime.PromptCompression.MaxLength)

		// 输出原始内容（截取前 1000 字符）
		originalPreview := systemPrompt
		if len(originalPreview) > 1000 {
			originalPreview = originalPreview[:1000] + "\n... (truncated)"
		}
		fmt.Printf("[PromptBuilder] 📄 ORIGINAL PROMPT:\n%s\n", originalPreview)
		fmt.Println("------- END ORIGINAL -------")

		compressed, err := pb.compress(context.Background(), systemPrompt, ctx)
		if err != nil {
			// 压缩失败，使用原始内容
			fmt.Printf("[PromptBuilder] ❌ Compression failed: %v, using original\n", err)
			return systemPrompt, nil
		}

		// 输出压缩后的完整内容
		fmt.Printf("[PromptBuilder] 📄 COMPRESSED PROMPT:\n%s\n", compressed)
		fmt.Println("------- END COMPRESSED -------")

		fmt.Printf("[PromptBuilder] ✅ Compression complete: %d -> %d chars (%.1f%% reduction)\n",
			len(systemPrompt), len(compressed), float64(len(systemPrompt)-len(compressed))/float64(len(systemPrompt))*100)
		return compressed, nil
	}

	return systemPrompt, nil
}

// shouldCompress 判断是否需要压缩
func (pb *PromptBuilder) shouldCompress(prompt string, ctx *PromptContext) bool {
	if pb.compressor == nil {
		return false
	}

	// 检查模板配置
	if ctx.Template == nil || ctx.Template.Runtime == nil || ctx.Template.Runtime.PromptCompression == nil {
		return false
	}

	config := ctx.Template.Runtime.PromptCompression
	if !config.Enabled {
		return false
	}

	// 检查长度阈值
	maxLength := config.MaxLength
	if maxLength == 0 {
		maxLength = 5000 // 默认阈值
	}

	return len(prompt) > maxLength
}

// compress 执行压缩
func (pb *PromptBuilder) compress(ctx context.Context, prompt string, pCtx *PromptContext) (string, error) {
	if pb.compressor == nil {
		return prompt, nil
	}

	config := pCtx.Template.Runtime.PromptCompression

	// 构建压缩选项
	opts := &CompressOptions{
		TargetLength:     config.TargetLength,
		PreserveSections: config.PreserveSections,
	}

	// 设置默认值
	if opts.TargetLength == 0 {
		opts.TargetLength = 3000
	}
	if len(opts.PreserveSections) == 0 {
		opts.PreserveSections = []string{"Tools Manual", "Security Guidelines"}
	}

	// 设置压缩模式
	switch config.Mode {
	case "simple":
		opts.Mode = CompressionModeSimple
	case "llm":
		opts.Mode = CompressionModeLLM
	case "hybrid":
		opts.Mode = CompressionModeHybrid
	default:
		opts.Mode = CompressionModeHybrid
	}

	// 设置压缩级别
	switch config.Level {
	case 1:
		opts.Level = CompressionLevelLight
	case 2:
		opts.Level = CompressionLevelModerate
	case 3:
		opts.Level = CompressionLevelAggressive
	default:
		opts.Level = CompressionLevelModerate
	}

	result, err := pb.compressor.Compress(ctx, prompt, opts)
	if err != nil {
		return prompt, err
	}

	return result.Compressed, nil
}

// collectEnvironmentInfo 收集环境信息
// sessionTime 参数用于 KV-Cache 优化：在同一会话中保持时间戳稳定，避免缓存失效
// 如果 sessionTime 为零值，则使用当前时间（向后兼容）
func collectEnvironmentInfo(ctx context.Context, workDir string, sessionTime time.Time) *EnvironmentInfo {
	var date time.Time
	if !sessionTime.IsZero() {
		// 使用会话固定时间（只保留日期部分，提高缓存命中率）
		date = sessionTime.Truncate(24 * time.Hour)
	} else {
		date = time.Now()
	}

	env := &EnvironmentInfo{
		WorkingDir: workDir,
		Platform:   runtime.GOOS,
		OSVersion:  getOSVersion(),
		Date:       date,
	}

	// 检查是否是 Git 仓库
	if gitInfo := detectGitRepo(ctx, workDir); gitInfo != nil {
		env.GitRepo = gitInfo
	}

	return env
}

// getOSVersion 获取 OS 版本
func getOSVersion() string {
	// 基础版本信息
	version := fmt.Sprintf("%s %s", runtime.GOOS, runtime.GOARCH)

	// 可以根据需要扩展获取更详细的版本信息
	// 例如使用 syscall 或执行 uname 命令

	return version
}

// detectGitRepo 检测 Git 仓库信息
func detectGitRepo(ctx context.Context, workDir string) *GitRepoInfo {
	// 检查是否是 Git 仓库
	gitDir := workDir + "/.git"
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return nil
	}

	info := &GitRepoInfo{
		IsRepo: true,
	}

	// 获取当前分支
	if branch, err := execGitCommand(ctx, workDir, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		info.CurrentBranch = strings.TrimSpace(branch)
	}

	// 尝试获取主分支（main 或 master）
	if _, err := execGitCommand(ctx, workDir, "rev-parse", "--verify", "main"); err == nil {
		info.MainBranch = "main"
	} else if _, err := execGitCommand(ctx, workDir, "rev-parse", "--verify", "master"); err == nil {
		info.MainBranch = "master"
	}

	// 获取 git status
	if status, err := execGitCommand(ctx, workDir, "status", "--short"); err == nil {
		info.Status = strings.TrimSpace(status)
	}

	// 获取最近的提交（最多 5 条）
	if commits, err := execGitCommand(ctx, workDir, "log", "--oneline", "-5"); err == nil {
		lines := strings.Split(strings.TrimSpace(commits), "\n")
		info.RecentCommits = make([]string, 0, len(lines))
		for _, line := range lines {
			if line != "" {
				info.RecentCommits = append(info.RecentCommits, line)
			}
		}
	}

	return info
}

// execGitCommand 执行 git 命令
func execGitCommand(ctx context.Context, workDir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = workDir

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

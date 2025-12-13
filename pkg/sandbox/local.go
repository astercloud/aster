package sandbox

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/types"
	"github.com/fsnotify/fsnotify"
)

var sandboxLogger = logging.ForComponent("sandbox")

// getShell returns the preferred shell for command execution.
// Prefers bash (which supports all ulimit options), falls back to /bin/sh.
func getShell() string {
	if shell, err := exec.LookPath("bash"); err == nil {
		return shell
	}
	return "/bin/sh"
}

// SecurityLevel 安全级别
type SecurityLevel int

const (
	// SecurityLevelNone 无安全限制
	SecurityLevelNone SecurityLevel = iota
	// SecurityLevelBasic 基础安全（危险命令检测）
	SecurityLevelBasic
	// SecurityLevelStrict 严格安全（路径限制+资源限制）
	SecurityLevelStrict
	// SecurityLevelParanoid 偏执安全（最严格）
	SecurityLevelParanoid
)

// 危险命令模式 - 增强版，更难绕过
var dangerousPatterns = []*regexp.Regexp{
	// 文件系统破坏
	regexp.MustCompile(`(?i)\brm\s+(-[a-z]*r[a-z]*\s+)*(/|/\*|~|\$HOME)`),            // rm -rf / 及变体
	regexp.MustCompile(`(?i)\brm\s+(-[a-z]*f[a-z]*\s+)*(-[a-z]*r[a-z]*\s+)*(/|/\*)`), // rm -fr /
	regexp.MustCompile(`(?i)\brmdir\s+(/|/\*)`),                                      // rmdir /
	regexp.MustCompile(`(?i)\bfind\s+/\s+.*-delete`),                                 // find / -delete
	regexp.MustCompile(`(?i)\bfind\s+/\s+.*-exec\s+rm`),                              // find / -exec rm

	// 权限提升
	regexp.MustCompile(`(?i)(^|\s|;|&&|\|\||\|)(sudo|doas|pkexec)\s`), // sudo 及替代品
	regexp.MustCompile(`(?i)(^|\s|;|&&|\|\||\|)/usr/(s)?bin/sudo\s`),  // 绝对路径 sudo
	regexp.MustCompile(`(?i)\bsu\s+(-\s+)?root`),                      // su root
	regexp.MustCompile(`(?i)\bchmod\s+[0-7]*[4-7][0-7]{2}\s+/`),       // chmod 危险权限到根目录
	regexp.MustCompile(`(?i)\bchown\s+.*\s+/`),                        // chown 根目录
	regexp.MustCompile(`(?i)\bsetuid\b`),                              // setuid
	regexp.MustCompile(`(?i)\bsetcap\b`),                              // setcap

	// 系统控制
	regexp.MustCompile(`(?i)(^|\s|;|&&|\|\||\|)(shutdown|poweroff|halt|reboot|init\s+[06])\b`),
	regexp.MustCompile(`(?i)\bsystemctl\s+(poweroff|reboot|halt)`),
	regexp.MustCompile(`(?i)\btelinit\s+[06]`),

	// 磁盘/分区操作
	regexp.MustCompile(`(?i)\b(mkfs|mke2fs|mkswap|fdisk|parted|gdisk)\b`),
	regexp.MustCompile(`(?i)\bdd\s+.*\bof=/dev/`),                 // dd 写入设备
	regexp.MustCompile(`(?i)\b(>\s*|tee\s+)/dev/(sd|hd|nvme|vd)`), // 重定向到磁盘设备
	regexp.MustCompile(`(?i)\bswapon\b`),
	regexp.MustCompile(`(?i)\bswapoff\b`),
	regexp.MustCompile(`(?i)\bmount\s`),
	regexp.MustCompile(`(?i)\bumount\s`),

	// Fork 炸弹和资源耗尽
	regexp.MustCompile(`:\s*\(\s*\)\s*\{\s*:\s*\|\s*:\s*&\s*\}\s*;\s*:`), // fork bomb
	regexp.MustCompile(`(?i)\byes\s*\|`),                                 // yes | 无限输出
	regexp.MustCompile(`(?i)\bwhile\s+true\s*;\s*do`),                    // 无限循环
	regexp.MustCompile(`(?i)\bfor\s*\(\s*;\s*;\s*\)`),                    // C风格无限循环

	// 远程代码执行
	regexp.MustCompile(`(?i)\bcurl\s+.*\|\s*(ba)?sh`),                       // curl | sh
	regexp.MustCompile(`(?i)\bwget\s+.*\|\s*(ba)?sh`),                       // wget | sh
	regexp.MustCompile(`(?i)\bcurl\s+.*-o\s*/tmp/.*&&.*sh\s+/tmp/`),         // curl下载执行
	regexp.MustCompile(`(?i)\beval\s+\$\(`),                                 // eval $(...)
	regexp.MustCompile(`(?i)\bsource\s+<\(`),                                // source <(...)
	regexp.MustCompile(`(?i)\b\.\s+<\(`),                                    // . <(...)
	regexp.MustCompile(`(?i)(python|perl|ruby|node)\s+-e\s+.*\bexec\b`),     // 脚本语言 exec
	regexp.MustCompile(`(?i)(python|perl|ruby|node)\s+-c\s+.*\bos\.system`), // 脚本语言系统调用

	// 网络攻击
	regexp.MustCompile(`(?i)\bnc\s+(-[a-z]*l[a-z]*\s+)*-[a-z]*e\s`), // nc -e 反向shell
	regexp.MustCompile(`(?i)\bnetcat\s+.*-e\s`),
	regexp.MustCompile(`(?i)\bnmap\s`),
	regexp.MustCompile(`(?i)\biptables\s`),
	regexp.MustCompile(`(?i)\bip6tables\s`),
	regexp.MustCompile(`(?i)\bufw\s`),
	regexp.MustCompile(`(?i)\bfirewall-cmd\s`),

	// 敏感文件访问
	regexp.MustCompile(`(?i)\bcat\s+.*/etc/(passwd|shadow|sudoers)`),
	regexp.MustCompile(`(?i)\bcp\s+.*/etc/(passwd|shadow)`),
	regexp.MustCompile(`(?i)\b(vi|vim|nano|emacs)\s+/etc/(passwd|shadow|sudoers)`),

	// 内核/系统修改
	regexp.MustCompile(`(?i)\binsmod\b`),
	regexp.MustCompile(`(?i)\brmmod\b`),
	regexp.MustCompile(`(?i)\bmodprobe\b`),
	regexp.MustCompile(`(?i)\bsysctl\s+-w`),
	regexp.MustCompile(`(?i)\becho\s+.*>\s*/proc/`),
	regexp.MustCompile(`(?i)\becho\s+.*>\s*/sys/`),

	// 容器逃逸
	regexp.MustCompile(`(?i)\bdocker\s+run\s+.*--privileged`),
	regexp.MustCompile(`(?i)\bdocker\s+run\s+.*-v\s+/:/`),
	regexp.MustCompile(`(?i)\bnsenter\b`),
	regexp.MustCompile(`(?i)\bunshare\b`),

	// 历史/日志清除
	regexp.MustCompile(`(?i)\bhistory\s+-c`),
	regexp.MustCompile(`(?i)\brm\s+.*\.(bash_history|zsh_history)`),
	regexp.MustCompile(`(?i)\b>\s*/var/log/`),
	regexp.MustCompile(`(?i)\btruncate\s+.*(/var/log/|\.history)`),
}

// 敏感路径前缀
var sensitivePaths = []string{
	"/etc/passwd",
	"/etc/shadow",
	"/etc/sudoers",
	"/etc/ssh",
	"/root",
	"/proc",
	"/sys",
	"/dev",
	"/boot",
	"/var/log",
	"/var/run",
	"/run",
}

// 允许的命令白名单（严格模式）
var allowedCommands = map[string]bool{
	// 文件操作
	"ls": true, "cat": true, "head": true, "tail": true, "less": true, "more": true,
	"cp": true, "mv": true, "mkdir": true, "touch": true, "echo": true,
	"find": true, "grep": true, "awk": true, "sed": true, "sort": true, "uniq": true,
	"wc": true, "diff": true, "file": true, "stat": true, "du": true, "df": true,
	// 开发工具
	"go": true, "python": true, "python3": true, "node": true, "npm": true, "npx": true,
	"yarn": true, "pnpm": true, "pip": true, "pip3": true, "cargo": true, "rustc": true,
	"java": true, "javac": true, "mvn": true, "gradle": true,
	"git": true, "make": true, "cmake": true,
	// 网络工具（受限）
	"curl": true, "wget": true, "ping": true,
	// 系统信息
	"pwd": true, "whoami": true, "id": true, "date": true, "uname": true, "env": true,
	"which": true, "whereis": true, "type": true,
	// 文本处理
	"tr": true, "cut": true, "paste": true, "join": true, "split": true,
	"xargs": true, "tee": true,
	// 压缩
	"tar": true, "gzip": true, "gunzip": true, "zip": true, "unzip": true,
	"bzip2": true, "bunzip2": true, "xz": true,
	// 进程（只读）
	"ps": true, "top": true, "htop": true,
}

// LocalSandbox 本地沙箱实现
type LocalSandbox struct {
	workDir         string
	enforceBoundary bool
	allowPaths      []string
	watchEnabled    bool
	fs              *LocalFS
	watchers        map[string]*fileWatcher
	watcherMu       sync.Mutex

	// Claude Agent SDK 风格的安全配置
	settings         *types.SandboxSettings
	networkConfig    *types.NetworkSandboxSettings
	ignoreViolations *types.SandboxIgnoreViolations
	excludedCommands []string

	// 增强安全配置
	securityLevel   SecurityLevel
	auditLog        []AuditEntry
	auditMu         sync.RWMutex
	maxAuditEntries int
	resourceLimits  *ResourceLimits
	blockedCommands map[string]bool
	commandStats    map[string]*CommandStats
	statsMu         sync.RWMutex
}

// AuditEntry 审计日志条目
type AuditEntry struct {
	Timestamp   time.Time         `json:"timestamp"`
	Command     string            `json:"command"`
	WorkDir     string            `json:"work_dir"`
	ExitCode    int               `json:"exit_code"`
	Duration    time.Duration     `json:"duration"`
	Blocked     bool              `json:"blocked"`
	BlockReason string            `json:"block_reason,omitempty"`
	UserID      string            `json:"user_id,omitempty"`
	SessionID   string            `json:"session_id,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ResourceLimits 资源限制配置
type ResourceLimits struct {
	MaxCPUTime     time.Duration // 最大 CPU 时间
	MaxMemoryMB    int           // 最大内存 (MB)
	MaxFileSizeMB  int           // 最大文件大小 (MB)
	MaxProcesses   int           // 最大进程数
	MaxOpenFiles   int           // 最大打开文件数
	MaxOutputBytes int           // 最大输出字节数
}

// CommandStats 命令统计
type CommandStats struct {
	TotalCalls   int64
	BlockedCalls int64
	TotalTime    time.Duration
	LastCall     time.Time
}

// DefaultResourceLimits 默认资源限制
var DefaultResourceLimits = &ResourceLimits{
	MaxCPUTime:     5 * time.Minute,
	MaxMemoryMB:    512,
	MaxFileSizeMB:  100,
	MaxProcesses:   50,
	MaxOpenFiles:   1024,
	MaxOutputBytes: 10 * 1024 * 1024, // 10MB
}

// fileWatcher 文件监听器
type fileWatcher struct {
	paths    []string
	listener FileChangeListener
	watcher  *fsnotify.Watcher
	done     chan struct{}
}

// LocalSandboxConfig 本地沙箱配置
type LocalSandboxConfig struct {
	WorkDir         string
	EnforceBoundary bool
	AllowPaths      []string
	WatchFiles      bool

	// Claude Agent SDK 风格的安全配置
	Settings *types.SandboxSettings

	// 增强安全配置
	SecurityLevel   SecurityLevel
	ResourceLimits  *ResourceLimits
	BlockedCommands []string
	MaxAuditEntries int
}

// NewLocalSandbox 创建本地沙箱
func NewLocalSandbox(config *LocalSandboxConfig) (*LocalSandbox, error) {
	if config == nil {
		config = &LocalSandboxConfig{}
	}

	// 解析工作目录
	workDir := config.WorkDir
	if workDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("get working directory: %w", err)
		}
		workDir = wd
	}

	workDir, err := filepath.Abs(workDir)
	if err != nil {
		return nil, fmt.Errorf("resolve work directory: %w", err)
	}

	// 解析允许路径
	allowPaths := make([]string, 0, len(config.AllowPaths))
	for _, p := range config.AllowPaths {
		abs, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		allowPaths = append(allowPaths, abs)
	}

	// 设置默认值
	securityLevel := config.SecurityLevel
	if securityLevel == 0 {
		securityLevel = SecurityLevelBasic
	}

	resourceLimits := config.ResourceLimits
	if resourceLimits == nil {
		resourceLimits = DefaultResourceLimits
	}

	maxAuditEntries := config.MaxAuditEntries
	if maxAuditEntries == 0 {
		maxAuditEntries = 1000
	}

	// 构建阻止命令映射
	blockedCommands := make(map[string]bool)
	for _, cmd := range config.BlockedCommands {
		blockedCommands[cmd] = true
	}

	ls := &LocalSandbox{
		workDir:         workDir,
		enforceBoundary: config.EnforceBoundary,
		allowPaths:      allowPaths,
		watchEnabled:    config.WatchFiles,
		watchers:        make(map[string]*fileWatcher),
		settings:        config.Settings,
		securityLevel:   securityLevel,
		auditLog:        make([]AuditEntry, 0),
		maxAuditEntries: maxAuditEntries,
		resourceLimits:  resourceLimits,
		blockedCommands: blockedCommands,
		commandStats:    make(map[string]*CommandStats),
	}

	// 应用 Claude Agent SDK 风格的安全配置
	if config.Settings != nil {
		ls.networkConfig = config.Settings.Network
		ls.ignoreViolations = config.Settings.IgnoreViolations
		ls.excludedCommands = config.Settings.ExcludedCommands
	}

	ls.fs = &LocalFS{
		workDir:         workDir,
		enforceBoundary: config.EnforceBoundary,
		allowPaths:      allowPaths,
	}

	sandboxLogger.Info(context.Background(), "LocalSandbox created", map[string]any{
		"workDir":         workDir,
		"securityLevel":   securityLevel,
		"enforceBoundary": config.EnforceBoundary,
	})

	return ls, nil
}

// Kind 返回沙箱类型
func (ls *LocalSandbox) Kind() string {
	return "local"
}

// WorkDir 返回工作目录
func (ls *LocalSandbox) WorkDir() string {
	return ls.workDir
}

// FS 返回文件系统接口
func (ls *LocalSandbox) FS() SandboxFS {
	return ls.fs
}

// Exec 执行命令
func (ls *LocalSandbox) Exec(ctx context.Context, cmd string, opts *ExecOptions) (*ExecResult, error) {
	startTime := time.Now()
	cmdName := ls.extractCommandName(cmd)

	// 1. 检查是否为排除命令（直接执行，但仍有关键安全检查）
	if ls.isExcludedCommand(cmd) {
		result, err := ls.execDirect(ctx, cmd, opts)
		if err != nil {
			return nil, err
		}
		ls.recordAudit(cmd, opts, result, startTime, false, "excluded_command")
		return result, nil
	}

	// 2. 检查是否在阻止列表
	if ls.blockedCommands[cmdName] {
		ls.recordAudit(cmd, opts, nil, startTime, true, "command in blocklist")
		return &ExecResult{
			Code:   1,
			Stdout: "",
			Stderr: fmt.Sprintf("Command '%s' is blocked by security policy", cmdName),
		}, nil
	}

	// 3. 严格模式：检查命令白名单
	if ls.securityLevel >= SecurityLevelStrict {
		if !allowedCommands[cmdName] && !ls.isExcludedCommand(cmd) {
			ls.recordAudit(cmd, opts, nil, startTime, true, "command not in whitelist")
			return &ExecResult{
				Code:   1,
				Stdout: "",
				Stderr: fmt.Sprintf("Command '%s' is not in the allowed list (strict mode)", cmdName),
			}, nil
		}
	}

	// 4. 安全检查：阻止危险命令
	if blockReason := ls.checkDangerousCommand(cmd); blockReason != "" {
		ls.recordAudit(cmd, opts, nil, startTime, true, blockReason)
		return &ExecResult{
			Code:   1,
			Stdout: "",
			Stderr: fmt.Sprintf("Dangerous command blocked: %s", blockReason),
		}, nil
	}

	// 5. 路径安全检查
	if ls.securityLevel >= SecurityLevelStrict {
		if pathIssue := ls.checkPathSecurity(cmd); pathIssue != "" {
			ls.recordAudit(cmd, opts, nil, startTime, true, pathIssue)
			return &ExecResult{
				Code:   1,
				Stdout: "",
				Stderr: fmt.Sprintf("Path security violation: %s", pathIssue),
			}, nil
		}
	}

	// 6. 执行命令（带资源限制）
	result := ls.execWithLimits(ctx, cmd, opts)

	// 7. 记录审计日志
	ls.recordAudit(cmd, opts, result, startTime, false, "")

	// 8. 更新统计
	ls.updateStats(cmdName, result, time.Since(startTime))

	return result, nil
}

// execWithLimits 带资源限制执行命令
func (ls *LocalSandbox) execWithLimits(ctx context.Context, cmd string, opts *ExecOptions) *ExecResult {
	// 设置超时
	timeout := 120 * time.Second
	if opts != nil && opts.Timeout > 0 {
		timeout = opts.Timeout
	}
	if ls.resourceLimits != nil && ls.resourceLimits.MaxCPUTime > 0 && ls.resourceLimits.MaxCPUTime < timeout {
		timeout = ls.resourceLimits.MaxCPUTime
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 构建命令（带资源限制）
	shellCmd := ls.buildSecureCommand(cmd)
	command := exec.CommandContext(execCtx, getShell(), "-c", shellCmd)

	// 设置工作目录
	workDir := ls.workDir
	if opts != nil && opts.WorkDir != "" {
		workDir = ls.fs.Resolve(opts.WorkDir)
	}

	// 验证工作目录是否存在，如果不存在则尝试创建
	if _, err := os.Stat(workDir); os.IsNotExist(err) {
		sandboxLogger.Warn(ctx, "Work directory does not exist, attempting to create", map[string]any{
			"workDir": workDir,
		})
		if mkErr := os.MkdirAll(workDir, 0755); mkErr != nil {
			return &ExecResult{
				Code:   1,
				Stdout: "",
				Stderr: fmt.Sprintf("work directory does not exist and cannot be created: %s (error: %v)", workDir, mkErr),
			}
		}
		sandboxLogger.Info(ctx, "Work directory created successfully", map[string]any{
			"workDir": workDir,
		})
	}

	command.Dir = workDir

	// 设置安全环境变量
	env := ls.buildSecureEnv(opts)
	command.Env = env

	// 执行并捕获输出
	output, err := command.CombinedOutput()

	// 限制输出大小
	if ls.resourceLimits != nil && ls.resourceLimits.MaxOutputBytes > 0 {
		if len(output) > ls.resourceLimits.MaxOutputBytes {
			output = append(output[:ls.resourceLimits.MaxOutputBytes],
				[]byte("\n... [output truncated due to size limit]")...)
		}
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return &ExecResult{
				Code:   exitErr.ExitCode(),
				Stdout: string(output),
				Stderr: string(output),
			}
		}
		return &ExecResult{
			Code:   1,
			Stdout: "",
			Stderr: err.Error(),
		}
	}

	return &ExecResult{
		Code:   0,
		Stdout: string(output),
		Stderr: "",
	}
}

// buildSecureCommand 构建带资源限制的命令
func (ls *LocalSandbox) buildSecureCommand(cmd string) string {
	if ls.resourceLimits == nil || runtime.GOOS == "windows" {
		return cmd
	}

	// 在 Unix 系统上使用 ulimit 限制资源
	var limits []string

	if ls.resourceLimits.MaxFileSizeMB > 0 {
		// 限制文件大小 (KB)
		limits = append(limits, fmt.Sprintf("ulimit -f %d", ls.resourceLimits.MaxFileSizeMB*1024))
	}

	if ls.resourceLimits.MaxProcesses > 0 {
		// 限制进程数
		limits = append(limits, fmt.Sprintf("ulimit -u %d", ls.resourceLimits.MaxProcesses))
	}

	if ls.resourceLimits.MaxOpenFiles > 0 {
		// 限制打开文件数
		limits = append(limits, fmt.Sprintf("ulimit -n %d", ls.resourceLimits.MaxOpenFiles))
	}

	if len(limits) > 0 {
		return strings.Join(limits, " && ") + " && " + cmd
	}

	return cmd
}

// buildSecureEnv 构建安全环境变量
func (ls *LocalSandbox) buildSecureEnv(opts *ExecOptions) []string {
	// 构建 PATH：包含常用路径，支持 macOS (Intel/Apple Silicon) 和 Linux
	// 优先级：用户本地 > Homebrew > 系统路径
	pathDirs := []string{
		"/usr/local/bin",
		"/usr/local/sbin",
		"/opt/homebrew/bin",  // Apple Silicon Mac Homebrew
		"/opt/homebrew/sbin", // Apple Silicon Mac Homebrew
		"/usr/bin",
		"/usr/sbin",
		"/bin",
		"/sbin",
	}
	pathValue := strings.Join(pathDirs, ":")

	// 基础安全环境变量
	env := []string{
		"PATH=" + pathValue,
		"HOME=" + ls.workDir,
		"LANG=en_US.UTF-8",
		"LC_ALL=en_US.UTF-8",
	}

	// 添加工作目录
	env = append(env, "PWD="+ls.workDir)

	// 在严格模式下，不继承主机环境变量
	if ls.securityLevel < SecurityLevelStrict {
		// 继承部分安全的环境变量
		safeEnvVars := []string{"TERM", "SHELL", "USER", "LOGNAME", "GOPATH", "GOROOT", "NODE_PATH"}
		for _, key := range safeEnvVars {
			if val := os.Getenv(key); val != "" {
				env = append(env, key+"="+val)
			}
		}
	}

	// 添加用户指定的环境变量
	if opts != nil && len(opts.Env) > 0 {
		for k, v := range opts.Env {
			// 过滤危险环境变量
			if !ls.isDangerousEnvVar(k) {
				env = append(env, fmt.Sprintf("%s=%s", k, v))
			}
		}
	}

	return env
}

// isDangerousEnvVar 检查是否为危险环境变量
func (ls *LocalSandbox) isDangerousEnvVar(key string) bool {
	dangerousVars := map[string]bool{
		"LD_PRELOAD":            true,
		"LD_LIBRARY_PATH":       true,
		"DYLD_INSERT_LIBRARIES": true,
		"DYLD_LIBRARY_PATH":     true,
		"PYTHONPATH":            true, // 可能被滥用
		"RUBYLIB":               true,
		"PERL5LIB":              true,
		"CLASSPATH":             true,
		"BASH_ENV":              true,
		"ENV":                   true,
		"CDPATH":                true,
		"IFS":                   true,
	}
	return dangerousVars[key]
}

// checkDangerousCommand 检查危险命令
func (ls *LocalSandbox) checkDangerousCommand(cmd string) string {
	// 规范化命令（移除多余空格）
	normalizedCmd := strings.Join(strings.Fields(cmd), " ")

	for _, pattern := range dangerousPatterns {
		if pattern.MatchString(normalizedCmd) {
			return fmt.Sprintf("matches dangerous pattern: %s", pattern.String()[:min(50, len(pattern.String()))])
		}
	}

	// 检查命令注入
	if ls.hasCommandInjection(cmd) {
		return "potential command injection detected"
	}

	return ""
}

// hasCommandInjection 检测命令注入
func (ls *LocalSandbox) hasCommandInjection(cmd string) bool {
	// 检测常见的命令注入模式
	injectionPatterns := []string{
		"`",    // 反引号命令替换
		"$(",   // 命令替换
		"$((",  // 算术扩展
		"${",   // 参数扩展（可能危险）
		"\n",   // 换行符注入
		"\r",   // 回车符注入
		"\x00", // 空字节注入
	}

	for _, pattern := range injectionPatterns {
		// 允许在引号内使用这些字符
		if strings.Contains(cmd, pattern) {
			// 简单检查：如果不在引号内，则可能是注入
			if !ls.isInQuotes(cmd, strings.Index(cmd, pattern)) {
				// 对于 $( 和 ${ 做更宽松的检查，因为它们在脚本中很常见
				if pattern == "$(" || pattern == "${" {
					// 只在偏执模式下阻止
					if ls.securityLevel >= SecurityLevelParanoid {
						return true
					}
				} else {
					return true
				}
			}
		}
	}

	return false
}

// isInQuotes 检查位置是否在引号内
func (ls *LocalSandbox) isInQuotes(s string, pos int) bool {
	if pos < 0 || pos >= len(s) {
		return false
	}

	inSingle := false
	inDouble := false

	for i := 0; i < pos; i++ {
		switch s[i] {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '\\':
			if i+1 < pos {
				i++ // 跳过转义字符
			}
		}
	}

	return inSingle || inDouble
}

// checkPathSecurity 检查路径安全
func (ls *LocalSandbox) checkPathSecurity(cmd string) string {
	// 提取命令中的路径
	paths := ls.extractPaths(cmd)

	for _, path := range paths {
		// 规范化路径
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}

		// 检查是否访问敏感路径
		for _, sensitive := range sensitivePaths {
			if strings.HasPrefix(absPath, sensitive) {
				return fmt.Sprintf("access to sensitive path: %s", sensitive)
			}
		}

		// 检查路径遍历攻击
		if strings.Contains(path, "..") {
			// 检查规范化后是否超出工作目录
			if ls.enforceBoundary && !ls.fs.IsInside(absPath) {
				return fmt.Sprintf("path traversal detected: %s", path)
			}
		}
	}

	return ""
}

// extractPaths 从命令中提取路径
func (ls *LocalSandbox) extractPaths(cmd string) []string {
	var paths []string

	// 简单的路径提取：查找以 / 或 ./ 或 ../ 开头的词
	pathPattern := regexp.MustCompile(`(?:^|\s)((?:/|\.\.?/)[^\s;|&<>]+)`)
	matches := pathPattern.FindAllStringSubmatch(cmd, -1)

	for _, match := range matches {
		if len(match) > 1 {
			paths = append(paths, strings.TrimSpace(match[1]))
		}
	}

	return paths
}

// extractCommandName 提取命令名
func (ls *LocalSandbox) extractCommandName(cmd string) string {
	// 移除前导空格和管道前的部分
	cmd = strings.TrimSpace(cmd)

	// 处理环境变量前缀 (如 VAR=value cmd)
	for strings.Contains(cmd, "=") {
		parts := strings.SplitN(cmd, " ", 2)
		if len(parts) < 2 {
			break
		}
		if strings.Contains(parts[0], "=") {
			cmd = strings.TrimSpace(parts[1])
		} else {
			break
		}
	}

	// 提取第一个词
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return ""
	}

	cmdName := fields[0]

	// 移除路径前缀
	cmdName = filepath.Base(cmdName)

	return cmdName
}

// recordAudit 记录审计日志
func (ls *LocalSandbox) recordAudit(cmd string, opts *ExecOptions, result *ExecResult, startTime time.Time, blocked bool, blockReason string) {
	ls.auditMu.Lock()
	defer ls.auditMu.Unlock()

	entry := AuditEntry{
		Timestamp:   startTime,
		Command:     truncate(cmd, 500),
		WorkDir:     ls.workDir,
		Duration:    time.Since(startTime),
		Blocked:     blocked,
		BlockReason: blockReason,
		Metadata:    make(map[string]string),
	}

	if opts != nil && opts.WorkDir != "" {
		entry.WorkDir = opts.WorkDir
	}

	if result != nil {
		entry.ExitCode = result.Code
	}

	// 添加到审计日志
	ls.auditLog = append(ls.auditLog, entry)

	// 限制审计日志大小
	if len(ls.auditLog) > ls.maxAuditEntries {
		ls.auditLog = ls.auditLog[len(ls.auditLog)-ls.maxAuditEntries:]
	}

	// 记录到结构化日志
	if blocked {
		sandboxLogger.Warn(context.Background(), "Command blocked", map[string]any{
			"command": truncate(cmd, 100),
			"reason":  blockReason,
		})
	} else {
		sandboxLogger.Debug(context.Background(), "Command executed", map[string]any{
			"command":  truncate(cmd, 100),
			"exitCode": entry.ExitCode,
			"duration": entry.Duration,
		})
	}
}

// updateStats 更新命令统计
func (ls *LocalSandbox) updateStats(cmdName string, result *ExecResult, duration time.Duration) {
	ls.statsMu.Lock()
	defer ls.statsMu.Unlock()

	stats, ok := ls.commandStats[cmdName]
	if !ok {
		stats = &CommandStats{}
		ls.commandStats[cmdName] = stats
	}

	stats.TotalCalls++
	stats.TotalTime += duration
	stats.LastCall = time.Now()

	if result != nil && result.Code != 0 {
		stats.BlockedCalls++
	}
}

// GetAuditLog 获取审计日志
func (ls *LocalSandbox) GetAuditLog() []AuditEntry {
	ls.auditMu.RLock()
	defer ls.auditMu.RUnlock()

	log := make([]AuditEntry, len(ls.auditLog))
	copy(log, ls.auditLog)
	return log
}

// GetCommandStats 获取命令统计
func (ls *LocalSandbox) GetCommandStats() map[string]*CommandStats {
	ls.statsMu.RLock()
	defer ls.statsMu.RUnlock()

	stats := make(map[string]*CommandStats)
	for k, v := range ls.commandStats {
		statsCopy := *v
		stats[k] = &statsCopy
	}
	return stats
}

// SetSecurityLevel 设置安全级别
func (ls *LocalSandbox) SetSecurityLevel(level SecurityLevel) {
	ls.securityLevel = level
	sandboxLogger.Info(context.Background(), "Security level changed", map[string]any{
		"level": level,
	})
}

// GetSecurityLevel 获取安全级别
func (ls *LocalSandbox) GetSecurityLevel() SecurityLevel {
	return ls.securityLevel
}

// AddBlockedCommand 添加阻止命令
func (ls *LocalSandbox) AddBlockedCommand(cmd string) {
	ls.blockedCommands[cmd] = true
}

// RemoveBlockedCommand 移除阻止命令
func (ls *LocalSandbox) RemoveBlockedCommand(cmd string) {
	delete(ls.blockedCommands, cmd)
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Watch 监听文件变更
func (ls *LocalSandbox) Watch(paths []string, listener FileChangeListener) (string, error) {
	if !ls.watchEnabled {
		return fmt.Sprintf("watch-disabled-%d", time.Now().UnixNano()), nil
	}

	ls.watcherMu.Lock()
	defer ls.watcherMu.Unlock()

	// 创建fsnotify watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return "", fmt.Errorf("create file watcher: %w", err)
	}

	// 生成watchID
	watchID := fmt.Sprintf("watch-%d-%s", time.Now().UnixNano(), randomString(8))

	// 添加监听路径
	for _, path := range paths {
		resolved := ls.fs.Resolve(path)
		if !ls.fs.IsInside(resolved) {
			continue
		}
		if err := watcher.Add(resolved); err != nil {
			// 忽略单个路径的错误
			continue
		}
	}

	// 创建fileWatcher
	fw := &fileWatcher{
		paths:    paths,
		listener: listener,
		watcher:  watcher,
		done:     make(chan struct{}),
	}

	ls.watchers[watchID] = fw

	// 启动监听goroutine
	go ls.watchLoop(watchID, fw)

	return watchID, nil
}

// watchLoop 文件监听循环
func (ls *LocalSandbox) watchLoop(watchID string, fw *fileWatcher) {
	defer func() { _ = fw.watcher.Close() }()

	for {
		select {
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}
			// 只处理写入和创建事件
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				// 获取文件修改时间
				var mtime time.Time
				if stat, err := os.Stat(event.Name); err == nil {
					mtime = stat.ModTime()
				} else {
					mtime = time.Now()
				}

				fw.listener(FileChangeEvent{
					Path:  event.Name,
					Mtime: mtime,
				})
			}
		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			// 记录错误但继续运行
			_ = err
		case <-fw.done:
			return
		}
	}
}

// Unwatch 取消监听
func (ls *LocalSandbox) Unwatch(watchID string) error {
	ls.watcherMu.Lock()
	defer ls.watcherMu.Unlock()

	fw, ok := ls.watchers[watchID]
	if !ok {
		return nil
	}

	close(fw.done)
	delete(ls.watchers, watchID)
	return nil
}

// Dispose 释放资源
func (ls *LocalSandbox) Dispose() error {
	ls.watcherMu.Lock()
	defer ls.watcherMu.Unlock()

	for _, fw := range ls.watchers {
		close(fw.done)
	}
	ls.watchers = make(map[string]*fileWatcher)
	return nil
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// randomString 生成随机字符串（使用 crypto/rand）
func randomString(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// fallback to timestamp-based
		for i := range b {
			b[i] = byte('a' + (time.Now().UnixNano() % 26))
		}
		return string(b)
	}
	return hex.EncodeToString(b)[:n]
}

// isExcludedCommand 检查命令是否在排除列表中
func (ls *LocalSandbox) isExcludedCommand(cmd string) bool {
	if len(ls.excludedCommands) == 0 {
		return false
	}

	// 提取命令的第一个词（命令名）
	cmdParts := strings.Fields(cmd)
	if len(cmdParts) == 0 {
		return false
	}
	cmdName := cmdParts[0]

	for _, excluded := range ls.excludedCommands {
		if cmdName == excluded {
			return true
		}
		// 支持路径形式的命令
		if strings.HasSuffix(cmdName, "/"+excluded) {
			return true
		}
	}

	return false
}

// execDirect 直接执行命令（排除命令，仍有基本安全检查）
func (ls *LocalSandbox) execDirect(ctx context.Context, cmd string, opts *ExecOptions) (*ExecResult, error) {
	// 即使是排除命令，也要检查最危险的模式
	criticalPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\brm\s+(-[a-z]*r[a-z]*\s+)*(/|/\*)`),
		regexp.MustCompile(`:\s*\(\s*\)\s*\{\s*:\s*\|\s*:\s*&\s*\}\s*;\s*:`),
		regexp.MustCompile(`(?i)\bdd\s+.*\bof=/dev/`),
	}

	for _, pattern := range criticalPatterns {
		if pattern.MatchString(cmd) {
			return &ExecResult{
				Code:   1,
				Stdout: "",
				Stderr: "Critical dangerous command blocked even for excluded commands",
			}, nil
		}
	}

	timeout := 120 * time.Second
	if opts != nil && opts.Timeout > 0 {
		timeout = opts.Timeout
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	command := exec.CommandContext(execCtx, getShell(), "-c", cmd)

	workDir := ls.workDir
	if opts != nil && opts.WorkDir != "" {
		workDir = ls.fs.Resolve(opts.WorkDir)
	}
	command.Dir = workDir

	if opts != nil && len(opts.Env) > 0 {
		env := os.Environ()
		for k, v := range opts.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		command.Env = env
	}

	output, err := command.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return &ExecResult{
				Code:   exitErr.ExitCode(),
				Stdout: string(output),
				Stderr: string(output),
			}, nil
		}
		return &ExecResult{
			Code:   1,
			Stdout: "",
			Stderr: err.Error(),
		}, nil
	}

	return &ExecResult{
		Code:   0,
		Stdout: string(output),
		Stderr: "",
	}, nil
}

// GetSettings 获取沙箱安全设置
func (ls *LocalSandbox) GetSettings() *types.SandboxSettings {
	return ls.settings
}

// IsEnabled 检查沙箱是否启用
func (ls *LocalSandbox) IsEnabled() bool {
	if ls.settings == nil {
		return false
	}
	return ls.settings.Enabled
}

// ShouldIgnoreViolation 检查是否应忽略违规
func (ls *LocalSandbox) ShouldIgnoreViolation(violationType, path string) bool {
	if ls.ignoreViolations == nil {
		return false
	}

	var patterns []string
	switch violationType {
	case "file":
		patterns = ls.ignoreViolations.FilePatterns
	case "network":
		patterns = ls.ignoreViolations.NetworkPatterns
	default:
		return false
	}

	for _, pattern := range patterns {
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
		// 支持简单的通配符
		if strings.Contains(pattern, "*") {
			re := strings.ReplaceAll(pattern, "*", ".*")
			if matched, _ := regexp.MatchString("^"+re+"$", path); matched {
				return true
			}
		}
	}

	return false
}

// CheckNetworkAccess 检查网络访问权限
func (ls *LocalSandbox) CheckNetworkAccess(host string, port int) bool {
	if ls.networkConfig == nil {
		return true // 默认允许
	}

	// 检查本地绑定
	if port > 0 && (host == "localhost" || host == "127.0.0.1" || host == "0.0.0.0") {
		if !ls.networkConfig.AllowLocalBinding {
			return false
		}
	}

	// 检查阻止列表
	for _, blocked := range ls.networkConfig.BlockedHosts {
		if host == blocked || strings.HasSuffix(host, "."+blocked) {
			return false
		}
	}

	// 检查允许列表（如果配置了）
	if len(ls.networkConfig.AllowedHosts) > 0 {
		allowed := false
		for _, allowedHost := range ls.networkConfig.AllowedHosts {
			if host == allowedHost || strings.HasSuffix(host, "."+allowedHost) {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}

	return true
}

// CheckUnixSocketAccess 检查 Unix Socket 访问权限
func (ls *LocalSandbox) CheckUnixSocketAccess(socketPath string) bool {
	if ls.networkConfig == nil {
		return true
	}

	if ls.networkConfig.AllowAllUnixSockets {
		return true
	}

	for _, allowed := range ls.networkConfig.AllowUnixSockets {
		if socketPath == allowed {
			return true
		}
	}

	return false
}

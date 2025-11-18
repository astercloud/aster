package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// 危险命令模式
var dangerousPatterns = []*regexp.Regexp{
	regexp.MustCompile(`rm\s+-rf\s+/($|\s)`),
	regexp.MustCompile(`sudo\s+`),
	regexp.MustCompile(`shutdown`),
	regexp.MustCompile(`reboot`),
	regexp.MustCompile(`mkfs\.`),
	regexp.MustCompile(`dd\s+.*of=`),
	regexp.MustCompile(`:\(\)\{\s*:\|\:&\s*\};:`),
	regexp.MustCompile(`chmod\s+777\s+/`),
	regexp.MustCompile(`curl\s+.*\|\s*(bash|sh)`),
	regexp.MustCompile(`wget\s+.*\|\s*(bash|sh)`),
	regexp.MustCompile(`>\s*/dev/sda`),
	regexp.MustCompile(`mkswap`),
	regexp.MustCompile(`swapon`),
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

	ls := &LocalSandbox{
		workDir:         workDir,
		enforceBoundary: config.EnforceBoundary,
		allowPaths:      allowPaths,
		watchEnabled:    config.WatchFiles,
		watchers:        make(map[string]*fileWatcher),
	}

	ls.fs = &LocalFS{
		workDir:         workDir,
		enforceBoundary: config.EnforceBoundary,
		allowPaths:      allowPaths,
	}

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
	// 安全检查:阻止危险命令
	for _, pattern := range dangerousPatterns {
		if pattern.MatchString(cmd) {
			return &ExecResult{
				Code:   1,
				Stdout: "",
				Stderr: fmt.Sprintf("Dangerous command blocked for security: %s", truncate(cmd, 100)),
			}, nil
		}
	}

	// 设置超时
	timeout := 120 * time.Second
	if opts != nil && opts.Timeout > 0 {
		timeout = opts.Timeout
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 执行命令
	command := exec.CommandContext(execCtx, "sh", "-c", cmd)

	// 设置工作目录
	workDir := ls.workDir
	if opts != nil && opts.WorkDir != "" {
		workDir = ls.fs.Resolve(opts.WorkDir)
	}
	command.Dir = workDir

	// 设置环境变量
	if opts != nil && len(opts.Env) > 0 {
		env := os.Environ()
		for k, v := range opts.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		command.Env = env
	}

	// 执行并捕获输出
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

// randomString 生成随机字符串
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}

package builtin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// TaskManager 后台任务管理器接口
type TaskManager interface {
	// StartTask 启动后台任务
	StartTask(ctx context.Context, cmd string, opts *TaskOptions) (*TaskInfo, error)

	// GetTask 获取任务信息
	GetTask(taskID string) (*TaskInfo, error)

	// GetTaskOutput 获取任务输出
	GetTaskOutput(taskID string, filter string, lines int) (string, string, error)

	// KillTask 终止任务
	KillTask(taskID string, signal string, timeout int) error

	// ListTasks 列出所有任务
	ListTasks() ([]*TaskInfo, error)

	// GetTaskStatus 获取任务状态
	GetTaskStatus(taskID string) (string, error)

	// CleanupTask 清理任务相关文件
	CleanupTask(taskID string) error
}

// TaskOptions 任务启动选项
type TaskOptions struct {
	WorkDir       string            `json:"work_dir"`
	Env           map[string]string `json:"env"`
	Timeout       time.Duration     `json:"timeout"`
	Background    bool              `json:"background"`
	Shell         string            `json:"shell"`
	CaptureOutput bool              `json:"capture_output"`
	OutputDir     string            `json:"output_dir"`
}

// TaskInfo 任务信息
type TaskInfo struct {
	ID         string            `json:"id"`
	Command    string            `json:"command"`
	PID        int               `json:"pid"`
	Status     string            `json:"status"` // "running", "completed", "failed", "killed"
	ExitCode   int               `json:"exit_code"`
	StartTime  time.Time         `json:"start_time"`
	EndTime    *time.Time        `json:"end_time,omitempty"`
	Duration   time.Duration     `json:"duration"`
	WorkDir    string            `json:"work_dir"`
	Shell      string            `json:"shell"`
	Options    *TaskOptions      `json:"options"`
	Metadata   map[string]string `json:"metadata"`
	LastUpdate time.Time         `json:"last_update"`
}

// FileTaskManager 基于文件系统的任务管理器实现
type FileTaskManager struct {
	mu          sync.RWMutex
	tasks       map[string]*TaskInfo
	dataDir     string
	outputDir   string
	cleanupChan chan string
}

// NewFileTaskManager 创建基于文件的任务管理器
func NewFileTaskManager() *FileTaskManager {
	// 创建数据目录
	dataDir := filepath.Join(os.TempDir(), "agentsdk_tasks")
	outputDir := filepath.Join(dataDir, "outputs")

	// 确保目录存在
	_ = os.MkdirAll(dataDir, 0755)
	_ = os.MkdirAll(outputDir, 0755)

	tm := &FileTaskManager{
		tasks:       make(map[string]*TaskInfo),
		dataDir:     dataDir,
		outputDir:   outputDir,
		cleanupChan: make(chan string, 100),
	}

	// 加载现有任务
	_ = tm.loadTasks()

	// 启动清理协程
	go tm.cleanupRoutine()

	return tm
}

// StartTask 启动后台任务
func (tm *FileTaskManager) StartTask(ctx context.Context, cmd string, opts *TaskOptions) (*TaskInfo, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 生成任务ID
	taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())

	// 设置默认选项
	if opts == nil {
		opts = &TaskOptions{
			Timeout:       30 * time.Minute,
			Background:    true,
			Shell:         "bash",
			CaptureOutput: true,
			OutputDir:     tm.outputDir,
		}
	}

	if opts.OutputDir == "" {
		opts.OutputDir = tm.outputDir
	}

	// 创建任务信息
	taskInfo := &TaskInfo{
		ID:         taskID,
		Command:    cmd,
		Status:     "starting",
		StartTime:  time.Now(),
		WorkDir:    opts.WorkDir,
		Shell:      opts.Shell,
		Options:    opts,
		Metadata:   make(map[string]string),
		LastUpdate: time.Now(),
	}

	// 构建命令
	fullCmd := tm.buildCommand(cmd, opts)

	// 启动命令
	cmdObj := exec.CommandContext(ctx, "bash", "-c", fullCmd)
	cmdObj.Dir = opts.WorkDir

	// 设置环境变量
	if len(opts.Env) > 0 {
		env := os.Environ()
		for k, v := range opts.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmdObj.Env = env
	}

	// 创建输出文件
	outputFile := filepath.Join(opts.OutputDir, taskID+".stdout")
	errorFile := filepath.Join(opts.OutputDir, taskID+".stderr")

	if opts.CaptureOutput {
		outFile, err := os.Create(outputFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create output file: %w", err)
		}

		errFile, err := os.Create(errorFile)
		if err != nil {
			_ = outFile.Close()
			return nil, fmt.Errorf("failed to create error file: %w", err)
		}

		cmdObj.Stdout = outFile
		cmdObj.Stderr = errFile

		// 延迟关闭文件
		go func() {
			defer func() { _ = outFile.Close() }()
			defer func() { _ = errFile.Close() }()
		}()
	}

	// 启动进程
	err := cmdObj.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// 更新任务信息
	taskInfo.PID = cmdObj.Process.Pid
	taskInfo.Status = "running"

	// 保存任务
	tm.tasks[taskID] = taskInfo
	_ = tm.saveTask(taskInfo)

	// 启动监控协程
	go tm.monitorTask(ctx, taskInfo, cmdObj)

	return taskInfo, nil
}

// GetTask 获取任务信息
func (tm *FileTaskManager) GetTask(taskID string) (*TaskInfo, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	task, exists := tm.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	// 更新任务状态
	if task.Status == "running" {
		tm.updateTaskStatus(task)
	}

	return task, nil
}

// GetTaskOutput 获取任务输出
func (tm *FileTaskManager) GetTaskOutput(taskID string, filter string, lines int) (string, string, error) {
	task, err := tm.GetTask(taskID)
	if err != nil {
		return "", "", err
	}

	outputFile := filepath.Join(task.Options.OutputDir, taskID+".stdout")
	errorFile := filepath.Join(task.Options.OutputDir, taskID+".stderr")

	// 读取标准输出
	stdout, err := os.ReadFile(outputFile)
	if err != nil && !os.IsNotExist(err) {
		return "", "", fmt.Errorf("failed to read stdout: %w", err)
	}

	// 读取错误输出
	stderr, err := os.ReadFile(errorFile)
	if err != nil && !os.IsNotExist(err) {
		return "", "", fmt.Errorf("failed to read stderr: %w", err)
	}

	stdoutStr := string(stdout)
	stderrStr := string(stderr)

	// 应用过滤器
	if filter != "" {
		stdoutStr = tm.filterOutput(stdoutStr, filter)
		stderrStr = tm.filterOutput(stderrStr, filter)
	}

	// 限制行数
	if lines > 0 {
		stdoutStr = tm.limitLines(stdoutStr, lines)
		stderrStr = tm.limitLines(stderrStr, lines)
	}

	return stdoutStr, stderrStr, nil
}

// KillTask 终止任务
func (tm *FileTaskManager) KillTask(taskID string, signal string, timeout int) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if task.Status != "running" {
		return fmt.Errorf("task is not running: %s", taskID)
	}

	// 获取信号编号
	signalNum := tm.getSignalNumber(signal)

	// 发送信号
	proc, err := os.FindProcess(task.PID)
	if err != nil {
		return fmt.Errorf("failed to find process %d: %w", task.PID, err)
	}

	err = proc.Signal(syscall.Signal(signalNum))
	if err != nil {
		return fmt.Errorf("failed to send signal %s to process %d: %w", signal, task.PID, err)
	}

	// 等待进程退出
	if timeout > 0 {
		go tm.waitForProcessExit(task, timeout)
	}

	// 更新任务状态
	task.Status = "killed"
	task.LastUpdate = time.Now()
	_ = tm.saveTask(task)

	return nil
}

// ListTasks 列出所有任务
func (tm *FileTaskManager) ListTasks() ([]*TaskInfo, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tasks := make([]*TaskInfo, 0, len(tm.tasks))
	for _, task := range tm.tasks {
		// 更新运行中任务的状态
		if task.Status == "running" {
			tm.updateTaskStatus(task)
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetTaskStatus 获取任务状态
func (tm *FileTaskManager) GetTaskStatus(taskID string) (string, error) {
	task, err := tm.GetTask(taskID)
	if err != nil {
		return "", err
	}

	return task.Status, nil
}

// CleanupTask 清理任务相关文件
func (tm *FileTaskManager) CleanupTask(taskID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// 删除输出文件
	outputFile := filepath.Join(task.Options.OutputDir, taskID+".stdout")
	errorFile := filepath.Join(task.Options.OutputDir, taskID+".stderr")

	_ = os.Remove(outputFile)
	_ = os.Remove(errorFile)

	// 删除任务信息
	delete(tm.tasks, taskID)

	// 删除任务文件
	taskFile := filepath.Join(tm.dataDir, taskID+".json")
	_ = os.Remove(taskFile)

	return nil
}

// buildCommand 构建命令
func (tm *FileTaskManager) buildCommand(cmd string, opts *TaskOptions) string {
	var parts []string

	// 添加环境变量
	for k, v := range opts.Env {
		parts = append(parts, fmt.Sprintf("export %s='%s'", k, strings.ReplaceAll(v, "'", "'\"'\"'")))
	}

	// 添加命令
	parts = append(parts, cmd)

	// 根据shell类型构建
	switch opts.Shell {
	case "sh":
		return fmt.Sprintf("sh -c '%s'", strings.Join(parts, "; "))
	case "zsh":
		return fmt.Sprintf("zsh -c '%s'", strings.Join(parts, "; "))
	default: // bash
		return fmt.Sprintf("bash -c '%s'", strings.Join(parts, "; "))
	}
}

// monitorTask 监控任务执行
func (tm *FileTaskManager) monitorTask(ctx context.Context, task *TaskInfo, cmd *exec.Cmd) {
	defer func() {
		// 发送清理信号
		select {
		case tm.cleanupChan <- task.ID:
		default:
		}
	}()

	// 等待命令完成
	err := cmd.Wait()

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 更新任务状态
	now := time.Now()
	task.EndTime = &now
	task.Duration = now.Sub(task.StartTime)
	task.LastUpdate = now

	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			task.ExitCode = exitErr.ExitCode()
			task.Status = "failed"
		} else {
			task.Status = "failed"
			task.ExitCode = -1
		}
	} else {
		task.ExitCode = 0
		task.Status = "completed"
	}

	_ = tm.saveTask(task)
}

// updateTaskStatus 更新任务状态
func (tm *FileTaskManager) updateTaskStatus(task *TaskInfo) {
	if task.PID <= 0 {
		return
	}

	// 检查进程是否还存在
	proc, err := os.FindProcess(task.PID)
	if err != nil {
		return
	}

	err = proc.Signal(syscall.Signal(0))
	if err != nil {
		// 进程不存在，更新状态
		now := time.Now()
		task.EndTime = &now
		task.Duration = now.Sub(task.StartTime)
		task.Status = "completed"
		task.LastUpdate = now
		_ = tm.saveTask(task)
	}
}

// waitForProcessExit 等待进程退出
func (tm *FileTaskManager) waitForProcessExit(task *TaskInfo, timeout int) {
	time.Sleep(time.Duration(timeout) * time.Second)

	tm.mu.Lock()
	defer tm.mu.Unlock()

	if task.Status == "killed" {
		// 检查进程是否真的退出了
		proc, err := os.FindProcess(task.PID)
		if err == nil {
			err = proc.Signal(syscall.Signal(0))
			if err != nil {
				// 进程已退出
				now := time.Now()
				task.EndTime = &now
				task.Duration = now.Sub(task.StartTime)
				task.Status = "completed"
				task.LastUpdate = now
				_ = tm.saveTask(task)
			}
		}
	}
}

// saveTask 保存任务信息到文件
func (tm *FileTaskManager) saveTask(task *TaskInfo) error {
	taskFile := filepath.Join(tm.dataDir, task.ID+".json")

	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	return os.WriteFile(taskFile, data, 0644)
}

// loadTasks 从文件加载任务
func (tm *FileTaskManager) loadTasks() error {
	files, err := os.ReadDir(tm.dataDir)
	if err != nil {
		return fmt.Errorf("failed to read data directory: %w", err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		taskFile := filepath.Join(tm.dataDir, file.Name())
		data, err := os.ReadFile(taskFile)
		if err != nil {
			continue
		}

		var task TaskInfo
		if err := json.Unmarshal(data, &task); err != nil {
			continue
		}

		tm.tasks[task.ID] = &task
	}

	return nil
}

// cleanupRoutine 清理协程
func (tm *FileTaskManager) cleanupRoutine() {
	for {
		select {
		case taskID := <-tm.cleanupChan:
			_ = tm.CleanupTask(taskID)
		case <-time.After(1 * time.Hour):
			// 定期清理完成的任务
			tm.cleanupCompletedTasks()
		}
	}
}

// cleanupCompletedTasks 清理完成的任务
func (tm *FileTaskManager) cleanupCompletedTasks() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for _, task := range tm.tasks {
		if task.Status == "completed" || task.Status == "failed" {
			// 清理超过1小时的已完成任务
			if time.Since(*task.EndTime) > time.Hour {
				delete(tm.tasks, task.ID)

				taskFile := filepath.Join(tm.dataDir, task.ID+".json")
				_ = os.Remove(taskFile)
			}
		}
	}
}

// getSignalNumber 获取信号编号
func (tm *FileTaskManager) getSignalNumber(signal string) int {
	signalMap := map[string]int{
		"SIGTERM": 15,
		"SIGKILL": 9,
		"SIGINT":  2,
		"SIGHUP":  1,
		"SIGQUIT": 3,
		"SIGSTOP": 19,
		"SIGCONT": 18,
	}

	if num, exists := signalMap[signal]; exists {
		return num
	}

	// 尝试解析数字
	if num, err := strconv.Atoi(signal); err == nil {
		return num
	}

	// 默认使用SIGTERM
	return 15
}

// filterOutput 过滤输出
func (tm *FileTaskManager) filterOutput(output, filter string) string {
	if filter == "" || output == "" {
		return output
	}

	regex, err := regexp.Compile(filter)
	if err != nil {
		return output
	}

	lines := strings.Split(output, "\n")
	var filteredLines []string

	for _, line := range lines {
		if regex.MatchString(line) {
			filteredLines = append(filteredLines, line)
		}
	}

	return strings.Join(filteredLines, "\n")
}

// limitLines 限制行数
func (tm *FileTaskManager) limitLines(output string, maxLines int) string {
	if maxLines <= 0 || output == "" {
		return output
	}

	lines := strings.Split(output, "\n")
	if len(lines) <= maxLines {
		return output
	}

	return strings.Join(lines[len(lines)-maxLines:], "\n")
}

// 全局任务管理器实例
var GlobalTaskManager TaskManager

// 初始化全局任务管理器
func init() {
	GlobalTaskManager = NewFileTaskManager()
}

// GetGlobalTaskManager 获取全局任务管理器
func GetGlobalTaskManager() TaskManager {
	return GlobalTaskManager
}

// cleanupOutputFiles 清理任务的输出文件
func (tm *FileTaskManager) cleanupOutputFiles(taskID string) error {
	tm.mu.RLock()
	task, exists := tm.tasks[taskID]
	tm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	outputFile := filepath.Join(task.Options.OutputDir, taskID+".stdout")
	errorFile := filepath.Join(task.Options.OutputDir, taskID+".stderr")

	// 清空文件内容
	if err := os.WriteFile(outputFile, []byte{}, 0644); err != nil {
		return fmt.Errorf("failed to clear output file: %w", err)
	}

	if err := os.WriteFile(errorFile, []byte{}, 0644); err != nil {
		return fmt.Errorf("failed to clear error file: %w", err)
	}

	return nil
}

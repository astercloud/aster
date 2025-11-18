package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// SubagentManager 子代理管理器接口
type SubagentManager interface {
	// StartSubagent 启动子代理
	StartSubagent(ctx context.Context, config *SubagentConfig) (*SubagentInstance, error)

	// ResumeSubagent 恢复子代理
	ResumeSubagent(taskID string) (*SubagentInstance, error)

	// GetSubagent 获取子代理信息
	GetSubagent(taskID string) (*SubagentInstance, error)

	// StopSubagent 停止子代理
	StopSubagent(taskID string) error

	// ListSubagents 列出所有子代理
	ListSubagents() ([]*SubagentInstance, error)

	// GetSubagentOutput 获取子代理输出
	GetSubagentOutput(taskID string) (string, error)

	// CleanupSubagent 清理子代理资源
	CleanupSubagent(taskID string) error
}

// SubagentConfig 子代理配置
type SubagentConfig struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"` // "general-purpose", "Explore", "Plan", "statusline-setup"
	Prompt      string            `json:"prompt"`
	Model       string            `json:"model,omitempty"`
	WorkDir     string            `json:"work_dir,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Timeout     time.Duration     `json:"timeout,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// SubagentInstance 子代理实例
type SubagentInstance struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Status        string                 `json:"status"` // "starting", "running", "completed", "failed", "stopped"
	PID           int                    `json:"pid,omitempty"`
	Command       string                 `json:"command"`
	Config        *SubagentConfig        `json:"config"`
	StartTime     time.Time              `json:"start_time"`
	EndTime       *time.Time             `json:"end_time,omitempty"`
	Duration      time.Duration          `json:"duration"`
	Output        string                 `json:"output"`
	Error         string                 `json:"error,omitempty"`
	ExitCode      int                    `json:"exit_code,omitempty"`
	LastUpdate    time.Time              `json:"last_update"`
	Metadata      map[string]string     `json:"metadata,omitempty"`
	ResourceUsage *SubagentResourceUsage `json:"resource_usage,omitempty"`
}

// SubagentResourceUsage 子代理资源使用情况
type SubagentResourceUsage struct {
	MemoryMB float64 `json:"memory_mb"`
	CPUPercent float64 `json:"cpu_percent"`
	DiskMB    float64 `json:"disk_mb"`
	NetworkMB float64 `json:"network_mb"`
}

// FileSubagentManager 基于文件的子代理管理器实现
type FileSubagentManager struct {
	mu      sync.RWMutex
	agents  map[string]*SubagentInstance
	dataDir string
}

// NewFileSubagentManager 创建基于文件的子代理管理器
func NewFileSubagentManager() *FileSubagentManager {
	// 创建数据目录
	dataDir := filepath.Join(os.TempDir(), "agentsdk_subagents")
	_ = os.MkdirAll(dataDir, 0755)

	sm := &FileSubagentManager{
		agents:  make(map[string]*SubagentInstance),
		dataDir: dataDir,
	}

	// 加载现有子代理
	_ = sm.loadSubagents()

	return sm
}

// StartSubagent 启动子代理
func (sm *FileSubagentManager) StartSubagent(ctx context.Context, config *SubagentConfig) (*SubagentInstance, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 生成子代理ID
	if config.ID == "" {
		config.ID = fmt.Sprintf("subagent_%d", time.Now().UnixNano())
	}

	// 检查是否已存在
	if _, exists := sm.agents[config.ID]; exists {
		return nil, fmt.Errorf("subagent already exists: %s", config.ID)
	}

	// 创建子代理实例
	instance := &SubagentInstance{
		ID:         config.ID,
		Type:       config.Type,
		Status:     "starting",
		Config:     config,
		StartTime:  time.Now(),
		Output:     "",
		Metadata:   make(map[string]string),
		LastUpdate: time.Now(),
	}

	// 构建启动命令
	cmd, err := sm.buildSubagentCommand(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build subagent command: %v", err)
	}

	// 启动子代理进程
	cmdObj := exec.CommandContext(ctx, "bash", "-c", cmd)
	cmdObj.Dir = config.WorkDir

	// 设置环境变量
	if len(config.Env) > 0 {
		env := os.Environ()
		for k, v := range config.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmdObj.Env = env
	}

	// 创建输出文件
	outputFile := filepath.Join(sm.dataDir, fmt.Sprintf("%s.output", config.ID))
	outFile, err := os.Create(outputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %v", err)
	}

	cmdObj.Stdout = outFile
	cmdObj.Stderr = outFile

	// 启动进程
	err = cmdObj.Start()
	if err != nil {
		_ = outFile.Close()
		return nil, fmt.Errorf("failed to start subagent: %v", err)
	}

	// 更新实例信息
	instance.Status = "running"
	instance.Command = cmd
	instance.PID = cmdObj.Process.Pid
	sm.agents[config.ID] = instance
	_ = sm.saveSubagent(instance)

	// 启动监控协程
	go sm.monitorSubagent(ctx, instance, cmdObj, outFile)

	return instance, nil
}

// ResumeSubagent 恢复子代理
func (sm *FileSubagentManager) ResumeSubagent(taskID string) (*SubagentInstance, error) {
	sm.mu.RLock()
	instance, exists := sm.agents[taskID]
	sm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("subagent not found: %s", taskID)
	}

	if instance.Status != "stopped" && instance.Status != "failed" && instance.Status != "completed" {
		return nil, fmt.Errorf("subagent cannot be resumed, current status: %s", instance.Status)
	}

	// 重新启动子代理
	ctx := context.Background()
	newInstance, err := sm.StartSubagent(ctx, instance.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to resume subagent: %v", err)
	}

	// 保留原有元数据
	for k, v := range instance.Metadata {
		newInstance.Metadata[k] = v
	}

	return newInstance, nil
}

// GetSubagent 获取子代理信息
func (sm *FileSubagentManager) GetSubagent(taskID string) (*SubagentInstance, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	instance, exists := sm.agents[taskID]
	if !exists {
		return nil, fmt.Errorf("subagent not found: %s", taskID)
	}

	// 如果是运行状态，更新输出和资源信息
	if instance.Status == "running" {
		sm.updateSubagentOutput(instance)
		sm.updateResourceUsage(instance)
	}

	return instance, nil
}

// StopSubagent 停止子代理
func (sm *FileSubagentManager) StopSubagent(taskID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	instance, exists := sm.agents[taskID]
	if !exists {
		return fmt.Errorf("subagent not found: %s", taskID)
	}

	if instance.Status != "running" {
		return fmt.Errorf("subagent is not running, current status: %s", instance.Status)
	}

	// 发送终止信号
	if instance.PID > 0 {
		proc, err := os.FindProcess(instance.PID)
		if err == nil {
			_ = proc.Signal(os.Interrupt) // 发送SIGINT信号
		}
	}

	// 等待进程退出
	time.Sleep(2 * time.Second)

	// 更新状态
	now := time.Now()
	instance.Status = "stopped"
	instance.EndTime = &now
	instance.Duration = now.Sub(instance.StartTime)
	instance.LastUpdate = now
	_ = sm.saveSubagent(instance)

	return nil
}

// ListSubagents 列出所有子代理
func (sm *FileSubagentManager) ListSubagents() ([]*SubagentInstance, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	agents := make([]*SubagentInstance, 0, len(sm.agents))
	for _, instance := range sm.agents {
		// 更新运行中代理的状态
		if instance.Status == "running" {
			sm.updateSubagentOutput(instance)
			sm.updateResourceUsage(instance)
		}
		agents = append(agents, instance)
	}

	return agents, nil
}

// GetSubagentOutput 获取子代理输出
func (sm *FileSubagentManager) GetSubagentOutput(taskID string) (string, error) {
	_, err := sm.GetSubagent(taskID)
	if err != nil {
		return "", err
	}

	outputFile := filepath.Join(sm.dataDir, fmt.Sprintf("%s.output", taskID))
	data, err := ioutil.ReadFile(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to read output file: %v", err)
	}

	return string(data), nil
}

// CleanupSubagent 清理子代理资源
func (sm *FileSubagentManager) CleanupSubagent(taskID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	instance, exists := sm.agents[taskID]
	if !exists {
		return fmt.Errorf("subagent not found: %s", taskID)
	}

	// 如果还在运行，先停止
	if instance.Status == "running" {
		sm.mu.Unlock()
		_ = sm.StopSubagent(taskID)
		sm.mu.Lock()
		_, _ = sm.agents[taskID] // 重新获取实例但忽略，因为即将删除
	}

	// 删除输出文件
	outputFile := filepath.Join(sm.dataDir, fmt.Sprintf("%s.output", taskID))
	_ = os.Remove(outputFile)

	// 删除实例记录
	delete(sm.agents, taskID)

	// 删除实例文件
	instanceFile := filepath.Join(sm.dataDir, fmt.Sprintf("%s.json", taskID))
	_ = os.Remove(instanceFile)

	return nil
}

// buildSubagentCommand 构建子代理启动命令
func (sm *FileSubagentManager) buildSubagentCommand(config *SubagentConfig) (string, error) {
	// 获取当前可执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		exePath = "agentsdk" // 回退到默认值
	}

	// 构建命令
	var cmdParts []string

	// 添加环境变量
	for k, v := range config.Env {
		cmdParts = append(cmdParts, fmt.Sprintf("export %s='%s'", k, strings.ReplaceAll(v, "'", "'\"'\"'")))
	}

	// 构建主要命令
	subagentCmd := fmt.Sprintf("%s subagent --type=%s --prompt='%s'", exePath, config.Type, strings.ReplaceAll(config.Prompt, "'", "'\"'\"'"))

	if config.Model != "" {
		subagentCmd += fmt.Sprintf(" --model=%s", config.Model)
	}

	if config.Timeout > 0 {
		subagentCmd += fmt.Sprintf(" --timeout=%s", config.Timeout.String())
	}

	if config.MaxTokens > 0 {
		subagentCmd += fmt.Sprintf(" --max-tokens=%d", config.MaxTokens)
	}

	if config.Temperature > 0 {
		subagentCmd += fmt.Sprintf(" --temperature=%f", config.Temperature)
	}

	cmdParts = append(cmdParts, subagentCmd)

	return strings.Join(cmdParts, "; "), nil
}

// monitorSubagent 监控子代理执行
func (sm *FileSubagentManager) monitorSubagent(ctx context.Context, instance *SubagentInstance, cmd *exec.Cmd, outFile *os.File) {
	defer func() {
		_ = outFile.Close()
		sm.mu.Lock()
		defer sm.mu.Unlock()

		now := time.Now()
		instance.EndTime = &now
		instance.Duration = now.Sub(instance.StartTime)
		instance.LastUpdate = now
		_ = sm.saveSubagent(instance)
	}()

	// 等待命令完成
	err := cmd.Wait()

	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 更新实例状态
	now := time.Now()
	instance.EndTime = &now
	instance.Duration = now.Sub(instance.StartTime)
	instance.LastUpdate = now

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			instance.ExitCode = exitErr.ExitCode()
			instance.Status = "failed"
			instance.Error = err.Error()
		} else {
			instance.Status = "failed"
			instance.ExitCode = -1
			instance.Error = err.Error()
		}
	} else {
		instance.ExitCode = 0
		instance.Status = "completed"
	}

	// 读取最终输出
	sm.updateSubagentOutput(instance)
	_ = sm.saveSubagent(instance)
}

// updateSubagentOutput 更新子代理输出
func (sm *FileSubagentManager) updateSubagentOutput(instance *SubagentInstance) {
	outputFile := filepath.Join(sm.dataDir, fmt.Sprintf("%s.output", instance.ID))
	data, err := ioutil.ReadFile(outputFile)
	if err == nil {
		instance.Output = string(data)
	}
}

// updateResourceUsage 更新资源使用情况
func (sm *FileSubagentManager) updateResourceUsage(instance *SubagentInstance) {
	if instance.PID <= 0 {
		return
	}

	// 简化实现：使用ps命令获取进程资源信息
	cmd := exec.Command("ps", "-p", fmt.Sprintf("%d", instance.PID), "-o", "rss,pcpu", "--no-headers")
	output, err := cmd.Output()
	if err != nil {
		return
	}

	fields := strings.Fields(strings.TrimSpace(string(output)))
	if len(fields) >= 2 {
		var rss, pcpu float64
		_, _ = fmt.Sscanf(fields[0], "%f", &rss)
		_, _ = fmt.Sscanf(fields[1], "%f", &pcpu)

		instance.ResourceUsage = &SubagentResourceUsage{
			MemoryMB: rss / 1024, // 转换为MB
			CPUPercent: pcpu,
		}
	}
}

// saveSubagent 保存子代理信息到文件
func (sm *FileSubagentManager) saveSubagent(instance *SubagentInstance) error {
	instanceFile := filepath.Join(sm.dataDir, fmt.Sprintf("%s.json", instance.ID))

	data, err := json.MarshalIndent(instance, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal subagent: %v", err)
	}

	return ioutil.WriteFile(instanceFile, data, 0644)
}

// loadSubagents 从文件加载子代理
func (sm *FileSubagentManager) loadSubagents() error {
	files, err := ioutil.ReadDir(sm.dataDir)
	if err != nil {
		return fmt.Errorf("failed to read data directory: %v", err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		instanceFile := filepath.Join(sm.dataDir, file.Name())
		data, err := ioutil.ReadFile(instanceFile)
		if err != nil {
			continue
		}

		var instance SubagentInstance
		if err := json.Unmarshal(data, &instance); err != nil {
			continue
		}

		sm.agents[instance.ID] = &instance
	}

	return nil
}

// 全局子代理管理器实例
var GlobalSubagentManager SubagentManager

// 初始化全局子代理管理器
func init() {
	GlobalSubagentManager = NewFileSubagentManager()
}

// GetGlobalSubagentManager 获取全局子代理管理器
func GetGlobalSubagentManager() SubagentManager {
	return GlobalSubagentManager
}
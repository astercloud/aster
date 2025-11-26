package builtin

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// StorageManager 存储管理器接口
type StorageManager interface {
	// StoreData 存储数据
	StoreData(key string, data interface{}) error

	// LoadData 加载数据
	LoadData(key string, target interface{}) error

	// DeleteData 删除数据
	DeleteData(key string) error

	// Exists 检查数据是否存在
	Exists(key string) bool

	// ListKeys 列出所有键
	ListKeys(prefix string) ([]string, error)

	// Backup 备份数据
	Backup(backupPath string) error

	// Restore 从备份恢复数据
	Restore(backupPath string) error
}

// PlanManager 计划管理器接口
type PlanManager interface {
	// StorePlan 存储计划
	StorePlan(plan *PlanRecord) error

	// LoadPlan 加载计划
	LoadPlan(planID string) (*PlanRecord, error)

	// UpdatePlanStatus 更新计划状态
	UpdatePlanStatus(planID string, status string) error

	// GetPlansByStatus 按状态获取计划
	GetPlansByStatus(status string) ([]*PlanRecord, error)

	// ListPlans 列出所有计划
	ListPlans() ([]*PlanRecord, error)

	// DeletePlan 删除计划
	DeletePlan(planID string) error
}

// TodoManager 任务列表管理器接口
type TodoManager interface {
	// StoreTodoList 存储任务列表
	StoreTodoList(list *TodoList) error

	// LoadTodoList 加载任务列表
	LoadTodoList(listName string) (*TodoList, error)

	// ListTodoLists 列出所有任务列表
	ListTodoLists() ([]string, error)

	// DeleteTodoList 删除任务列表
	DeleteTodoList(listName string) error

	// BackupTodoLists 备份任务列表
	BackupTodoLists() (map[string]*TodoList, error)

	// RestoreTodoLists 恢复任务列表
	RestoreTodoLists(backup map[string]*TodoList) error
}

// FileStorageManager 基于文件的存储管理器实现
type FileStorageManager struct {
	mu      sync.RWMutex
	dataDir string
}

// NewFileStorageManager 创建基于文件的存储管理器
func NewFileStorageManager() *FileStorageManager {
	// 创建数据目录
	dataDir := filepath.Join(os.TempDir(), "agentsdk_storage")

	// 创建子目录
	_ = os.MkdirAll(filepath.Join(dataDir, "todos"), 0755)
	_ = os.MkdirAll(filepath.Join(dataDir, "plans"), 0755)
	_ = os.MkdirAll(filepath.Join(dataDir, "cache"), 0755)

	return &FileStorageManager{
		dataDir: dataDir,
	}
}

// StoreData 存储数据
func (fsm *FileStorageManager) StoreData(key string, data interface{}) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	// 创建目录
	dir := filepath.Dir(filepath.Join(fsm.dataDir, key))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// 序列化数据
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %v", err)
	}

	// 写入文件
	filePath := filepath.Join(fsm.dataDir, fmt.Sprintf("%s.json", key))
	return os.WriteFile(filePath, jsonData, 0644)
}

// LoadData 加载数据
func (fsm *FileStorageManager) LoadData(key string, target interface{}) error {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()

	filePath := filepath.Join(fsm.dataDir, fmt.Sprintf("%s.json", key))
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	return json.Unmarshal(data, target)
}

// DeleteData 删除数据
func (fsm *FileStorageManager) DeleteData(key string) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	filePath := filepath.Join(fsm.dataDir, fmt.Sprintf("%s.json", key))
	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %v", err)
	}

	return nil
}

// Exists 检查数据是否存在
func (fsm *FileStorageManager) Exists(key string) bool {
	filePath := filepath.Join(fsm.dataDir, fmt.Sprintf("%s.json", key))
	_, err := os.Stat(filePath)
	return err == nil
}

// ListKeys 列出所有键
func (fsm *FileStorageManager) ListKeys(prefix string) ([]string, error) {
	return filepath.Glob(filepath.Join(fsm.dataDir, prefix+"*.json"))
}

// Backup 备份数据
func (fsm *FileStorageManager) Backup(backupPath string) error {
	fsm.mu.RLock()
	defer fsm.mu.RUnlock()

	// 创建备份目录
	if err := os.MkdirAll(filepath.Dir(backupPath), 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %v", err)
	}

	// 创建备份压缩文件
	cmd := exec.Command("tar", "-czf", backupPath, "-C", fsm.dataDir, ".")
	return cmd.Run()
}

// Restore 从备份恢复数据
func (fsm *FileStorageManager) Restore(backupPath string) error {
	fsm.mu.Lock()
	defer fsm.mu.Unlock()

	// 清理现有数据
	_ = os.RemoveAll(fsm.dataDir)

	// 创建数据目录
	if err := os.MkdirAll(fsm.dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %v", err)
	}

	// 从备份恢复
	cmd := exec.Command("tar", "-xzf", backupPath, "-C", fsm.dataDir)
	return cmd.Run()
}

// FilePlanManager 计划管理器实现
type FilePlanManager struct {
	storageManager StorageManager
}

// NewFilePlanManager 创建计划管理器
func NewFilePlanManager(storageMgr StorageManager) *FilePlanManager {
	return &FilePlanManager{
		storageManager: storageMgr,
	}
}

// StorePlan 存储计划
func (fpm *FilePlanManager) StorePlan(plan *PlanRecord) error {
	key := fmt.Sprintf("plans/%s", plan.ID)
	return fpm.storageManager.StoreData(key, plan)
}

// LoadPlan 加载计划
func (fpm *FilePlanManager) LoadPlan(planID string) (*PlanRecord, error) {
	var plan PlanRecord
	key := fmt.Sprintf("plans/%s", planID)
	err := fpm.storageManager.LoadData(key, &plan)
	return &plan, err
}

// UpdatePlanStatus 更新计划状态
func (fpm *FilePlanManager) UpdatePlanStatus(planID string, status string) error {
	plan, err := fpm.LoadPlan(planID)
	if err != nil {
		return fmt.Errorf("failed to load plan: %v", err)
	}

	plan.Status = status
	if status == "approved" {
		now := time.Now()
		plan.ApprovedAt = &now
	}

	plan.UpdatedAt = time.Now()
	return fpm.StorePlan(plan)
}

// GetPlansByStatus 按状态获取计划
func (fpm *FilePlanManager) GetPlansByStatus(status string) ([]*PlanRecord, error) {
	keys, err := fpm.storageManager.ListKeys("plans/")
	if err != nil {
		return nil, err
	}

	var plans []*PlanRecord
	for _, key := range keys {
		planID := strings.TrimSuffix(filepath.Base(key), ".json")
		plan, err := fpm.LoadPlan(planID)
		if err != nil {
			continue
		}

		if plan.Status == status {
			plans = append(plans, plan)
		}
	}

	return plans, nil
}

// ListPlans 列出所有计划
func (fpm *FilePlanManager) ListPlans() ([]*PlanRecord, error) {
	keys, err := fpm.storageManager.ListKeys("plans/")
	if err != nil {
		return nil, err
	}

	var plans []*PlanRecord
	for _, key := range keys {
		planID := strings.TrimSuffix(filepath.Base(key), ".json")
		plan, err := fpm.LoadPlan(planID)
		if err != nil {
			continue
		}
		plans = append(plans, plan)
	}

	return plans, nil
}

// DeletePlan 删除计划
func (fpm *FilePlanManager) DeletePlan(planID string) error {
	key := fmt.Sprintf("plans/%s", planID)
	return fpm.storageManager.DeleteData(key)
}

// FileTodoManager 任务列表管理器实现
type FileTodoManager struct {
	storageManager StorageManager
}

// NewFileTodoManager 创建任务列表管理器
func NewFileTodoManager(storageMgr StorageManager) *FileTodoManager {
	return &FileTodoManager{
		storageManager: storageMgr,
	}
}

// StoreTodoList 存储任务列表
func (ftm *FileTodoManager) StoreTodoList(list *TodoList) error {
	key := fmt.Sprintf("todos/%s", list.Name)
	return ftm.storageManager.StoreData(key, list)
}

// LoadTodoList 加载任务列表
func (ftm *FileTodoManager) LoadTodoList(listName string) (*TodoList, error) {
	var list TodoList
	key := fmt.Sprintf("todos/%s", listName)
	err := ftm.storageManager.LoadData(key, &list)
	return &list, err
}

// ListTodoLists 列出所有任务列表
func (ftm *FileTodoManager) ListTodoLists() ([]string, error) {
	keys, err := ftm.storageManager.ListKeys("todos/")
	if err != nil {
		return nil, err
	}

	var listNames []string
	for _, key := range keys {
		listName := strings.TrimSuffix(filepath.Base(key), ".json")
		listNames = append(listNames, listName)
	}

	return listNames, nil
}

// DeleteTodoList 删除任务列表
func (ftm *FileTodoManager) DeleteTodoList(listName string) error {
	key := fmt.Sprintf("todos/%s", listName)
	return ftm.storageManager.DeleteData(key)
}

// BackupTodoLists 备份任务列表
func (ftm *FileTodoManager) BackupTodoLists() (map[string]*TodoList, error) {
	listNames, err := ftm.ListTodoLists()
	if err != nil {
		return nil, err
	}

	backup := make(map[string]*TodoList)
	for _, listName := range listNames {
		list, err := ftm.LoadTodoList(listName)
		if err != nil {
			continue
		}
		backup[listName] = list
	}

	return backup, nil
}

// RestoreTodoLists 恢复任务列表
func (ftm *FileTodoManager) RestoreTodoLists(backup map[string]*TodoList) error {
	for listName, list := range backup {
		if err := ftm.StoreTodoList(list); err != nil {
			return fmt.Errorf("failed to restore todo list %s: %v", listName, err)
		}
	}
	return nil
}

// 全局存储管理器实例
var GlobalStorageManager StorageManager
var GlobalPlanManager PlanManager
var GlobalTodoManager TodoManager

// 初始化全局存储管理器
func init() {
	GlobalStorageManager = NewFileStorageManager()
	GlobalPlanManager = NewFilePlanManager(GlobalStorageManager)
	GlobalTodoManager = NewFileTodoManager(GlobalStorageManager)
}

// GetGlobalStorageManager 获取全局存储管理器
func GetGlobalStorageManager() StorageManager {
	return GlobalStorageManager
}

// GetGlobalPlanManager 获取全局计划管理器
func GetGlobalPlanManager() PlanManager {
	return GlobalPlanManager
}

// GetGlobalTodoManager 获取全局任务列表管理器
func GetGlobalTodoManager() TodoManager {
	return GlobalTodoManager
}

// ResetGlobalManagers 重置全局存储管理器（仅用于测试）
func ResetGlobalManagers() {
	GlobalStorageManager = NewFileStorageManagerWithDir(fmt.Sprintf("%s/agentsdk_storage_%d", os.TempDir(), time.Now().UnixNano()))
	GlobalPlanManager = NewFilePlanManager(GlobalStorageManager)
	GlobalTodoManager = NewFileTodoManager(GlobalStorageManager)
}

// NewFileStorageManagerWithDir 创建使用指定目录的存储管理器
func NewFileStorageManagerWithDir(dataDir string) *FileStorageManager {
	// 创建子目录
	_ = os.MkdirAll(filepath.Join(dataDir, "todos"), 0755)
	_ = os.MkdirAll(filepath.Join(dataDir, "plans"), 0755)
	_ = os.MkdirAll(filepath.Join(dataDir, "cache"), 0755)

	return &FileStorageManager{
		dataDir: dataDir,
	}
}

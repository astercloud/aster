package project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileStore 文件存储实现（AGENTS.md 模式）
type FileStore struct {
	config *StoreConfig
}

// NewFileStore 创建文件存储
func NewFileStore(config *StoreConfig) *FileStore {
	if config == nil {
		config = DefaultStoreConfig()
	}
	return &FileStore{config: config}
}

// getFilePath 获取文件路径
func (s *FileStore) getFilePath(projectID string) string {
	return filepath.Join(s.config.BasePath, projectID, s.config.FileName)
}

// Load 加载项目记忆
func (s *FileStore) Load(ctx context.Context, projectID string) (*ProjectMemory, error) {
	filePath := s.getFilePath(projectID)

	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("project memory not found: %s", projectID)
		}
		return nil, fmt.Errorf("read file failed: %w", err)
	}

	// 解析 Markdown 内容
	return s.parseMarkdown(projectID, string(content))
}

// Save 保存项目记忆
func (s *FileStore) Save(ctx context.Context, memory *ProjectMemory) error {
	filePath := s.getFilePath(memory.ProjectID)

	// 确保目录存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory failed: %w", err)
	}

	// 生成 Markdown 内容
	content := s.generateMarkdown(memory)

	// 写入文件
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write file failed: %w", err)
	}

	return nil
}

// Delete 删除项目记忆
func (s *FileStore) Delete(ctx context.Context, projectID string) error {
	filePath := s.getFilePath(projectID)
	return os.Remove(filePath)
}

// Exists 检查项目记忆是否存在
func (s *FileStore) Exists(ctx context.Context, projectID string) (bool, error) {
	filePath := s.getFilePath(projectID)
	_, err := os.Stat(filePath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// AppendEntry 追加条目到指定章节
func (s *FileStore) AppendEntry(ctx context.Context, projectID string, section MemorySection, entry *Entry) error {
	memory, err := s.Load(ctx, projectID)
	if err != nil {
		// 如果不存在且配置了自动创建
		if s.config.AutoCreate {
			memory = NewProjectMemory(projectID, "", "")
		} else {
			return err
		}
	}

	memory.AddEntry(section, entry)
	return s.Save(ctx, memory)
}

// UpdateWorkflowStep 更新工作流步骤状态
func (s *FileStore) UpdateWorkflowStep(ctx context.Context, projectID string, step *WorkflowStep) error {
	memory, err := s.Load(ctx, projectID)
	if err != nil {
		return err
	}

	// 添加工作流更新条目
	entry := &Entry{
		ID:       fmt.Sprintf("workflow-%s-%d", step.ID, time.Now().Unix()),
		Category: "workflow",
		Content:  fmt.Sprintf("[%s] %s: %s", step.ID, step.Status, step.Note),
		Value:    step,
		Source:   "system",
	}
	memory.AddEntry(SectionWorkflow, entry)

	return s.Save(ctx, memory)
}

// RecordParams 记录生成参数
func (s *FileStore) RecordParams(ctx context.Context, projectID string, params *GenerationParams) error {
	memory, err := s.Load(ctx, projectID)
	if err != nil {
		return err
	}

	entry := &Entry{
		ID:       fmt.Sprintf("params-%d", time.Now().Unix()),
		Category: "generation",
		Content:  fmt.Sprintf("Model: %s, Temperature: %.2f", params.Model, params.Temperature),
		Value:    params,
		Source:   "system",
	}
	memory.AddEntry(SectionParams, entry)

	return s.Save(ctx, memory)
}

// GetSummary 获取摘要（用于 AI 上下文注入）
func (s *FileStore) GetSummary(ctx context.Context, projectID string) (string, error) {
	memory, err := s.Load(ctx, projectID)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("## 项目记忆摘要\n\n")

	// 用户偏好
	prefs := memory.GetEntries(SectionPreferences)
	if len(prefs) > 0 {
		sb.WriteString("### 用户偏好\n")
		for _, e := range prefs {
			sb.WriteString(fmt.Sprintf("- [%s] %s\n", e.Category, e.Content))
		}
		sb.WriteString("\n")
	}

	// 关键选择
	choices := memory.GetEntries(SectionChoices)
	if len(choices) > 0 {
		sb.WriteString("### 关键选择\n")
		for _, e := range choices {
			sb.WriteString(fmt.Sprintf("- %s\n", e.Content))
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// GetVersionHistory 获取版本历史（文件存储不支持，返回空）
func (s *FileStore) GetVersionHistory(ctx context.Context, projectID string, limit int) ([]*ProjectMemory, error) {
	// 文件存储不支持版本历史，需要配合版本控制系统使用
	return []*ProjectMemory{}, nil
}

// parseMarkdown 解析 Markdown 内容为 ProjectMemory
func (s *FileStore) parseMarkdown(projectID, content string) (*ProjectMemory, error) {
	memory := NewProjectMemory(projectID, "", "")

	lines := strings.Split(content, "\n")
	var currentSection MemorySection
	var inSection bool

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 检测章节标题
		if strings.HasPrefix(line, "## 用户偏好") {
			currentSection = SectionPreferences
			inSection = true
			continue
		} else if strings.HasPrefix(line, "## 关键选择") {
			currentSection = SectionChoices
			inSection = true
			continue
		} else if strings.HasPrefix(line, "## 工作流状态") {
			currentSection = SectionWorkflow
			inSection = true
			continue
		} else if strings.HasPrefix(line, "## 生成参数") {
			currentSection = SectionParams
			inSection = true
			continue
		} else if strings.HasPrefix(line, "## 项目历史") {
			currentSection = SectionHistory
			inSection = true
			continue
		} else if strings.HasPrefix(line, "## ") {
			inSection = false
			continue
		}

		// 解析条目（以 - 开头的行）
		if inSection && strings.HasPrefix(line, "- ") {
			entryContent := strings.TrimPrefix(line, "- ")
			if entryContent != "" && !strings.Contains(entryContent, "*暂无记录*") {
				entry := &Entry{
					ID:        fmt.Sprintf("entry-%d", time.Now().UnixNano()),
					Content:   entryContent,
					Source:    "file",
					CreatedAt: time.Now(),
				}

				// 尝试提取分类（格式: [category] content）
				if strings.HasPrefix(entryContent, "[") {
					if idx := strings.Index(entryContent, "]"); idx > 0 {
						entry.Category = entryContent[1:idx]
						entry.Content = strings.TrimSpace(entryContent[idx+1:])
					}
				}

				memory.AddEntry(currentSection, entry)
			}
		}
	}

	return memory, nil
}

// generateMarkdown 生成 Markdown 内容
func (s *FileStore) generateMarkdown(memory *ProjectMemory) string {
	var sb strings.Builder
	now := time.Now().Format("2006-01-02")

	sb.WriteString("# AGENTS.md\n\n")

	// 项目信息
	sb.WriteString("## 项目信息\n\n")
	sb.WriteString(fmt.Sprintf("- **项目 ID**: %s\n", memory.ProjectID))
	if memory.Title != "" {
		sb.WriteString(fmt.Sprintf("- **项目标题**: %s\n", memory.Title))
	}
	if memory.Description != "" {
		sb.WriteString(fmt.Sprintf("- **项目描述**: %s\n", memory.Description))
	}
	sb.WriteString(fmt.Sprintf("- **创建时间**: %s\n", memory.CreatedAt.Format("2006-01-02")))
	sb.WriteString("\n")

	// 用户偏好
	sb.WriteString("## 用户偏好 [protected]\n\n")
	sb.WriteString("<!-- 从对话中自动提取的用户偏好，AI 生成时必须遵守 -->\n\n")
	prefs := memory.GetEntries(SectionPreferences)
	if len(prefs) == 0 {
		sb.WriteString("*暂无记录*\n")
	} else {
		for _, e := range prefs {
			if e.Category != "" {
				sb.WriteString(fmt.Sprintf("- %s: [%s] %s\n", e.CreatedAt.Format("2006-01-02 15:04"), e.Category, e.Content))
			} else {
				sb.WriteString(fmt.Sprintf("- %s: %s\n", e.CreatedAt.Format("2006-01-02 15:04"), e.Content))
			}
		}
	}
	sb.WriteString("\n")

	// 关键选择
	sb.WriteString("## 关键选择 [protected]\n\n")
	sb.WriteString("<!-- 用户在工作流中的关键选择，不可自动删除 -->\n\n")
	choices := memory.GetEntries(SectionChoices)
	if len(choices) == 0 {
		sb.WriteString("*暂无记录*\n")
	} else {
		for _, e := range choices {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", e.CreatedAt.Format("2006-01-02 15:04"), e.Content))
		}
	}
	sb.WriteString("\n")

	// 工作流状态
	sb.WriteString("## 工作流状态\n\n")
	workflow := memory.GetEntries(SectionWorkflow)
	if len(workflow) == 0 {
		sb.WriteString("*暂无记录*\n")
	} else {
		for _, e := range workflow {
			sb.WriteString(fmt.Sprintf("- %s\n", e.Content))
		}
	}
	sb.WriteString("\n")

	// 生成参数
	sb.WriteString("## 生成参数\n\n")
	params := memory.GetEntries(SectionParams)
	if len(params) == 0 {
		sb.WriteString("*暂无记录*\n")
	} else {
		for _, e := range params {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", e.CreatedAt.Format("2006-01-02 15:04"), e.Content))
		}
	}
	sb.WriteString("\n")

	// 项目历史
	sb.WriteString("## 项目历史\n\n")
	history := memory.GetEntries(SectionHistory)
	if len(history) == 0 {
		sb.WriteString(fmt.Sprintf("- %s: 项目创建\n", now))
	} else {
		for _, e := range history {
			sb.WriteString(fmt.Sprintf("- %s\n", e.Content))
		}
	}
	sb.WriteString("\n")

	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("*最后更新: %s*\n", now))

	return sb.String()
}

// 确保 FileStore 实现 Store 接口
var _ Store = (*FileStore)(nil)

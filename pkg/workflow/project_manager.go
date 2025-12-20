package workflow

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ProjectManager 项目管理器接口
type ProjectManager interface {
	// 创建项目
	CreateProject(ctx context.Context, spec *ProjectSpec) (string, error)

	// 读取项目
	LoadProject(ctx context.Context, projectID string) (any, error)
	GetProjectMetadata(ctx context.Context, projectID string) (*ProjectMetadata, error)

	// 更新项目
	SaveProject(ctx context.Context, projectID string, data any) error
	UpdateProjectMetadata(ctx context.Context, projectID string, metadata *ProjectMetadata) error

	// 删除项目
	DeleteProject(ctx context.Context, projectID string) error

	// 列表操作
	ListProjects(ctx context.Context, filter *ProjectFilter) ([]*ProjectMetadata, error)

	// 归档和恢复
	ArchiveProject(ctx context.Context, projectID string) error
	UnarchiveProject(ctx context.Context, projectID string) error

	// 版本管理
	CreateSnapshot(ctx context.Context, projectID string, description string) (string, error)
	ListSnapshots(ctx context.Context, projectID string) ([]*ProjectSnapshot, error)
	RestoreSnapshot(ctx context.Context, projectID, snapshotID string) error
}

// ProjectSpec 项目规范
type ProjectSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Type        string         `json:"type"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
}

// ProjectMetadata 项目元数据
type ProjectMetadata struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Type        string         `json:"type"`
	Status      string         `json:"status"` // draft, in_progress, completed, archived
	CreatedAt   int64          `json:"created_at"`
	UpdatedAt   int64          `json:"updated_at"`
	CreatedBy   string         `json:"created_by,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	Size        int64          `json:"size,omitempty"`
}

// ProjectFilter 项目过滤器
type ProjectFilter struct {
	Type   string   `json:"type,omitempty"`
	Status string   `json:"status,omitempty"`
	Tags   []string `json:"tags,omitempty"`
	Limit  int      `json:"limit,omitempty"`
	Offset int      `json:"offset,omitempty"`
}

// ProjectSnapshot 项目快照
type ProjectSnapshot struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	Description string `json:"description"`
	CreatedAt   int64  `json:"created_at"`
	CreatedBy   string `json:"created_by,omitempty"`
	Metadata    any    `json:"metadata,omitempty"`
}

// ===== 内存实现 =====

// InMemoryProjectManager 内存项目管理器实现
type InMemoryProjectManager struct {
	projects  map[string]any
	metadata  map[string]*ProjectMetadata
	snapshots map[string][]*ProjectSnapshot
}

func NewInMemoryProjectManager() *InMemoryProjectManager {
	return &InMemoryProjectManager{
		projects:  make(map[string]any),
		metadata:  make(map[string]*ProjectMetadata),
		snapshots: make(map[string][]*ProjectSnapshot),
	}
}

func (pm *InMemoryProjectManager) CreateProject(ctx context.Context, spec *ProjectSpec) (string, error) {
	if spec == nil {
		return "", errors.New("project spec cannot be nil")
	}

	if spec.Name == "" {
		return "", errors.New("project name is required")
	}

	projectID := generateProjectID()

	metadata := &ProjectMetadata{
		ID:          projectID,
		Name:        spec.Name,
		Description: spec.Description,
		Type:        spec.Type,
		Status:      "draft",
		CreatedAt:   getTimestamp(),
		UpdatedAt:   getTimestamp(),
		Tags:        spec.Tags,
		Metadata:    spec.Metadata,
	}

	pm.metadata[projectID] = metadata
	pm.projects[projectID] = make(map[string]any)
	pm.snapshots[projectID] = make([]*ProjectSnapshot, 0)

	return projectID, nil
}

func (pm *InMemoryProjectManager) LoadProject(ctx context.Context, projectID string) (any, error) {
	data, exists := pm.projects[projectID]
	if !exists {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}
	return data, nil
}

func (pm *InMemoryProjectManager) GetProjectMetadata(ctx context.Context, projectID string) (*ProjectMetadata, error) {
	metadata, exists := pm.metadata[projectID]
	if !exists {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}
	return metadata, nil
}

func (pm *InMemoryProjectManager) SaveProject(ctx context.Context, projectID string, data any) error {
	if _, exists := pm.projects[projectID]; !exists {
		return fmt.Errorf("project not found: %s", projectID)
	}

	pm.projects[projectID] = data

	// 更新元数据的 UpdatedAt
	if metadata, exists := pm.metadata[projectID]; exists {
		metadata.UpdatedAt = getTimestamp()
	}

	return nil
}

func (pm *InMemoryProjectManager) UpdateProjectMetadata(ctx context.Context, projectID string, metadata *ProjectMetadata) error {
	if _, exists := pm.metadata[projectID]; !exists {
		return fmt.Errorf("project not found: %s", projectID)
	}

	metadata.ID = projectID
	metadata.UpdatedAt = getTimestamp()
	pm.metadata[projectID] = metadata

	return nil
}

func (pm *InMemoryProjectManager) DeleteProject(ctx context.Context, projectID string) error {
	delete(pm.projects, projectID)
	delete(pm.metadata, projectID)
	delete(pm.snapshots, projectID)
	return nil
}

func (pm *InMemoryProjectManager) ListProjects(ctx context.Context, filter *ProjectFilter) ([]*ProjectMetadata, error) {
	result := make([]*ProjectMetadata, 0)

	for _, metadata := range pm.metadata {
		if filter != nil {
			if filter.Type != "" && metadata.Type != filter.Type {
				continue
			}
			if filter.Status != "" && metadata.Status != filter.Status {
				continue
			}
			if len(filter.Tags) > 0 && !hasAnyTag(metadata.Tags, filter.Tags) {
				continue
			}
		}

		result = append(result, metadata)
	}

	return result, nil
}

func (pm *InMemoryProjectManager) ArchiveProject(ctx context.Context, projectID string) error {
	metadata, exists := pm.metadata[projectID]
	if !exists {
		return fmt.Errorf("project not found: %s", projectID)
	}

	metadata.Status = "archived"
	metadata.UpdatedAt = getTimestamp()
	return nil
}

func (pm *InMemoryProjectManager) UnarchiveProject(ctx context.Context, projectID string) error {
	metadata, exists := pm.metadata[projectID]
	if !exists {
		return fmt.Errorf("project not found: %s", projectID)
	}

	metadata.Status = "in_progress"
	metadata.UpdatedAt = getTimestamp()
	return nil
}

func (pm *InMemoryProjectManager) CreateSnapshot(ctx context.Context, projectID string, description string) (string, error) {
	data, exists := pm.projects[projectID]
	if !exists {
		return "", fmt.Errorf("project not found: %s", projectID)
	}

	snapshotID := generateSnapshotID()
	snapshot := &ProjectSnapshot{
		ID:          snapshotID,
		ProjectID:   projectID,
		Description: description,
		CreatedAt:   getTimestamp(),
		Metadata:    data,
	}

	pm.snapshots[projectID] = append(pm.snapshots[projectID], snapshot)

	return snapshotID, nil
}

func (pm *InMemoryProjectManager) ListSnapshots(ctx context.Context, projectID string) ([]*ProjectSnapshot, error) {
	snapshots, exists := pm.snapshots[projectID]
	if !exists {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}

	result := make([]*ProjectSnapshot, len(snapshots))
	copy(result, snapshots)
	return result, nil
}

func (pm *InMemoryProjectManager) RestoreSnapshot(ctx context.Context, projectID, snapshotID string) error {
	snapshots, exists := pm.snapshots[projectID]
	if !exists {
		return fmt.Errorf("project not found: %s", projectID)
	}

	for _, snapshot := range snapshots {
		if snapshot.ID == snapshotID {
			pm.projects[projectID] = snapshot.Metadata

			// 更新 metadata
			if metadata, exists := pm.metadata[projectID]; exists {
				metadata.UpdatedAt = getTimestamp()
			}

			return nil
		}
	}

	return fmt.Errorf("snapshot not found: %s", snapshotID)
}

// ===== 辅助函数 =====

func generateProjectID() string {
	return fmt.Sprintf("prj_%d", getTimestamp())
}

func generateSnapshotID() string {
	return fmt.Sprintf("snap_%d", getTimestamp())
}

func getTimestamp() int64 {
	return time.Now().Unix()
}

func hasAnyTag(tags, filterTags []string) bool {
	tagMap := make(map[string]bool)
	for _, tag := range tags {
		tagMap[tag] = true
	}

	for _, filterTag := range filterTags {
		if tagMap[filterTag] {
			return true
		}
	}

	return false
}

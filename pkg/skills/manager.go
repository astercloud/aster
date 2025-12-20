package skills

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/astercloud/aster/pkg/logging"
	"github.com/astercloud/aster/pkg/sandbox"
)

var skillMgrLog = logging.ForComponent("SkillManager")

// Info Skill 元信息（用于列表展示）
type Info struct {
	// ID Skill 逻辑标识，例如 "pdf-to-markdown" 或 "workflows/code-review"
	// 不包含版本信息（如果有版本，则从目录名中去掉 @version 后得到）。
	ID string

	// Name YAML 中的 name 字段（如果存在）
	Name string

	// Description 描述
	Description string

	// Kind "knowledge" | "executable" | ""
	Kind string

	// Version 可选版本号, 如果目录名中包含 "@version" 后缀, 则提取为版本。
	Version string

	// Path Skill 目录相对 baseDir 的路径
	Path string
}

// Manager Skill 管理器，负责在本地文件系统中安装、列出和卸载 Skills。
// 设计目标类似 Claude Skills API 的本地版本。
type Manager struct {
	baseDir string
	fs      sandbox.SandboxFS
	loader  *SkillLoader
}

// NewManager 创建一个基于本地/沙箱文件系统的 Skill 管理器。
// baseDir 为技能包根目录，例如 "skills" 或 "workspace/.claude/skills"。
func NewManager(baseDir string, fs sandbox.SandboxFS) *Manager {
	if baseDir == "" {
		baseDir = "skills"
	}
	return &Manager{
		baseDir: baseDir,
		fs:      fs,
		loader:  NewLoader(baseDir, fs),
	}
}

// List 列出当前 baseDir 下所有可用 Skill 的元信息。
func (m *Manager) List(ctx context.Context) ([]Info, error) {
	paths, err := m.loader.Discover(ctx)
	if err != nil {
		return nil, fmt.Errorf("discover skills: %w", err)
	}

	infos := make([]Info, 0, len(paths))
	for _, p := range paths {
		def, err := m.loader.Load(ctx, p)
		if err != nil {
			// 忽略单个失败，继续其他
			continue
		}

		id, version := splitIDAndVersion(p)

		infos = append(infos, Info{
			ID:          id,
			Name:        def.Name,
			Description: def.Description,
			Kind:        def.Kind,
			Version:     version,
			Path:        filepath.Join(m.baseDir, p),
		})
	}

	return infos, nil
}

// InstallFromZip 从 zip 数据安装一个 Skill。
//   - skillID: 安装后的目录名，例如 "pdf-to-markdown"
//   - r: zip 数据流
//
// zip 内部应包含单个根目录或直接包含 SKILL.md；会被展开到 baseDir/skillID 下。
func (m *Manager) InstallFromZip(ctx context.Context, skillID string, r io.ReaderAt, size int64) error {
	if strings.TrimSpace(skillID) == "" {
		return errors.New("skill id must not be empty")
	}

	zr, err := zip.NewReader(r, size)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}

	targetRoot := filepath.Join(m.baseDir, skillID)

	// 先删除旧目录（如果存在）
	if err := os.RemoveAll(targetRoot); err != nil {
		skillMgrLog.Warn(ctx, "failed to remove old skill directory", map[string]any{"path": targetRoot, "error": err})
	}

	// 解压所有文件
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}

		// 去掉可能的顶层目录前缀
		name := normalizeZipEntryName(f.Name)
		if name == "" {
			continue
		}

		targetPath := filepath.Join(skillID, name)

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open zip entry %s: %w", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return fmt.Errorf("read zip entry %s: %w", f.Name, err)
		}

		if err := m.fs.Write(ctx, targetPath, string(data)); err != nil {
			return fmt.Errorf("write skill file %s: %w", targetPath, err)
		}
	}

	return nil
}

// InstallFromDir 从本地目录复制一个 Skill（常用于开发测试）。
// srcDir 应包含 SKILL.md。
func (m *Manager) InstallFromDir(ctx context.Context, skillID string, srcDir string) error {
	if strings.TrimSpace(skillID) == "" {
		return errors.New("skill id must not be empty")
	}

	srcDir = filepath.Clean(srcDir)

	// 枚举 srcDir 下所有文件
	err := filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read src file %s: %w", path, err)
		}

		targetPath := filepath.Join(skillID, filepath.ToSlash(rel))
		if err := m.fs.Write(ctx, targetPath, string(content)); err != nil {
			return fmt.Errorf("write skill file %s: %w", targetPath, err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

// Uninstall 卸载一个 Skill，即删除 baseDir/skillID 目录。
// 注意：这里直接使用 os.RemoveAll，因为 SandboxFS 暂未提供删除接口；
// 同时假设 baseDir 对应的是本地可写路径。
func (m *Manager) Uninstall(ctx context.Context, skillID string) error {
	if strings.TrimSpace(skillID) == "" {
		return errors.New("skill id must not be empty")
	}

	// 删除主目录 skills/skillID
	fullPath := filepath.Join(m.baseDir, skillID)
	_ = os.RemoveAll(fullPath)

	// 删除所有版本目录 skills/skillID@*
	pattern := filepath.Join(m.baseDir, skillID+"@*")
	matches, err := filepath.Glob(pattern)
	if err == nil {
		for _, p := range matches {
			_ = os.RemoveAll(p)
		}
	}

	return nil
}

// normalizeZipEntryName 去掉 zip 内部可能存在的共同根目录前缀。
// 例如:
//
//	skill/SCILL.md -> SKILL.md
//	skill/scripts/pdf2md.go -> scripts/pdf2md.go
func normalizeZipEntryName(name string) string {
	name = filepath.ToSlash(name)
	parts := strings.Split(name, "/")
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	// 如果第一个组件包含 "skill" 或与第二个组件一同构成常见结构，则去掉第一个组件
	return strings.Join(parts[1:], "/")
}

// Helper: 从 []byte 安装 zip（方便上层使用）
func (m *Manager) InstallFromZipBytes(ctx context.Context, skillID string, data []byte) error {
	reader := bytes.NewReader(data)
	return m.InstallFromZip(ctx, skillID, reader, int64(len(data)))
}

// InstallFromFiles 从内存中的文件集合安装或更新 Skill。
// files 的 key 为相对路径（相对于 skillID 根目录），value 为文件内容。
func (m *Manager) InstallFromFiles(ctx context.Context, skillID string, files map[string]string) error {
	if strings.TrimSpace(skillID) == "" {
		return errors.New("skill id must not be empty")
	}
	if len(files) == 0 {
		return errors.New("files must not be empty")
	}

	for rel, content := range files {
		if rel == "" {
			continue
		}
		targetPath := filepath.Join(skillID, filepath.ToSlash(rel))
		if err := m.fs.Write(ctx, targetPath, content); err != nil {
			return fmt.Errorf("write skill file %s: %w", targetPath, err)
		}
	}

	return nil
}

// ListVersions 列出指定 Skill ID 的所有版本(包括无版本的主版本)。
func (m *Manager) ListVersions(ctx context.Context, skillID string) ([]Info, error) {
	if strings.TrimSpace(skillID) == "" {
		return nil, errors.New("skill id must not be empty")
	}

	all, err := m.List(ctx)
	if err != nil {
		return nil, err
	}

	var out []Info
	for _, info := range all {
		if info.ID == skillID {
			out = append(out, info)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("skill not found: %s", skillID)
	}
	return out, nil
}

// InstallVersionFromFiles 安装指定 Skill 的某个版本。
// 如果 version 为空, 等价于 InstallFromFiles(skillID, files)。
func (m *Manager) InstallVersionFromFiles(ctx context.Context, skillID, version string, files map[string]string) error {
	if strings.TrimSpace(skillID) == "" {
		return errors.New("skill id must not be empty")
	}
	id := skillID
	if strings.TrimSpace(version) != "" {
		id = fmt.Sprintf("%s@%s", skillID, version)
	}
	return m.InstallFromFiles(ctx, id, files)
}

// DeleteVersion 删除某个 Skill 的特定版本。
// 如果 version 为空, 等价于 Uninstall(skillID)。
func (m *Manager) DeleteVersion(ctx context.Context, skillID, version string) error {
	if strings.TrimSpace(skillID) == "" {
		return errors.New("skill id must not be empty")
	}
	if strings.TrimSpace(version) == "" {
		return m.Uninstall(ctx, skillID)
	}
	dir := filepath.Join(m.baseDir, fmt.Sprintf("%s@%s", skillID, version))
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove skill version dir %s: %w", dir, err)
	}
	return nil
}

// splitIDAndVersion 从路径推导逻辑 ID 与版本号。
// 约定: 如果目录名最后一段包含 "@", 则视为 "<id>@<version>" 结构。
// 例如:
//
//	"pdf-to-markdown"           -> ("pdf-to-markdown", "")
//	"pdf-to-markdown@20251116"  -> ("pdf-to-markdown", "20251116")
//	"workflows/code-review"     -> ("workflows/code-review", "")
//	"workflows/code-review@v2"  -> ("workflows/code-review", "v2")
func splitIDAndVersion(path string) (string, string) {
	base := filepath.Base(path)
	dir := filepath.Dir(path)

	if i := strings.LastIndex(base, "@"); i > 0 && i < len(base)-1 {
		idPart := base[:i]
		ver := base[i+1:]
		if dir == "." || dir == string(filepath.Separator) {
			return idPart, ver
		}
		return filepath.ToSlash(filepath.Join(dir, idPart)), ver
	}

	return filepath.ToSlash(path), ""
}

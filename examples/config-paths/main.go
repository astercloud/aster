// Config 演示跨平台路径管理，支持 macOS、Linux 和 Windows。
// 遵循各平台的标准路径约定 (XDG, macOS Library, Windows AppData)。
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/astercloud/aster/pkg/config"
)

func main() {
	fmt.Println("📁 Cross-Platform Paths 示例")
	fmt.Println("================================")

	// 1. 标准路径
	demonstrateStandardPaths()

	// 2. 便捷方法
	demonstrateConvenienceMethods()

	// 3. 确保目录存在
	demonstrateEnsureDir()

	fmt.Println("\n✅ Cross-Platform Paths 示例完成!")
}

func demonstrateStandardPaths() {
	fmt.Println("\n📋 标准应用路径")
	fmt.Println(repeatStr("-", 50))

	fmt.Println("  当前平台路径:")
	fmt.Printf("    配置目录: %s\n", config.ConfigDir())
	fmt.Printf("    数据目录: %s\n", config.DataDir())
	fmt.Printf("    缓存目录: %s\n", config.CacheDir())
	fmt.Printf("    日志目录: %s\n", config.LogDir())

	fmt.Println("\n  平台路径约定:")
	fmt.Println("    macOS:")
	fmt.Println("      配置: ~/Library/Application Support/Aster/")
	fmt.Println("      数据: ~/Library/Application Support/Aster/")
	fmt.Println("      缓存: ~/Library/Caches/Aster/")
	fmt.Println("      日志: ~/Library/Logs/Aster/")
	fmt.Println()
	fmt.Println("    Linux:")
	fmt.Println("      配置: ~/.config/aster/")
	fmt.Println("      数据: ~/.local/share/aster/")
	fmt.Println("      缓存: ~/.cache/aster/")
	fmt.Println("      日志: ~/.local/state/aster/logs/")
	fmt.Println()
	fmt.Println("    Windows:")
	fmt.Println("      配置: %APPDATA%\\Aster\\")
	fmt.Println("      数据: %APPDATA%\\Aster\\data\\")
	fmt.Println("      缓存: %LOCALAPPDATA%\\Aster\\cache\\")
	fmt.Println("      日志: %LOCALAPPDATA%\\Aster\\logs\\")
}

func demonstrateConvenienceMethods() {
	fmt.Println("\n📋 便捷方法")
	fmt.Println(repeatStr("-", 50))

	// 配置文件路径
	configFile := config.ConfigFile()
	fmt.Printf("  配置文件路径: %s\n", configFile)

	// 数据库文件路径
	dbFile := config.DatabaseFile()
	fmt.Printf("  数据库文件路径: %s\n", dbFile)

	// Sessions 目录
	sessionsDir := config.SessionsDir()
	fmt.Printf("  Sessions 目录: %s\n", sessionsDir)

	// Recipes 目录
	recipesDir := config.RecipesDir()
	fmt.Printf("  Recipes 目录: %s\n", recipesDir)

	// Extensions 目录
	extensionsDir := config.ExtensionsDir()
	fmt.Printf("  Extensions 目录: %s\n", extensionsDir)

	// Memories 目录
	memoriesDir := config.MemoriesDir()
	fmt.Printf("  Memories 目录: %s\n", memoriesDir)
}

func demonstrateEnsureDir() {
	fmt.Println("\n📋 确保目录存在")
	fmt.Println(repeatStr("-", 50))

	// 创建临时目录用于测试
	tmpDir, err := os.MkdirTemp("", "aster-config-demo")
	if err != nil {
		fmt.Printf("  ❌ 创建临时目录失败: %v\n", err)
		return
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// 测试目录
	testPath := filepath.Join(tmpDir, "nested", "deep", "directory")

	// 确保目录存在
	err = config.EnsureDir(testPath)
	if err != nil {
		fmt.Printf("  ❌ 创建目录失败: %v\n", err)
		return
	}

	// 验证目录已创建
	if info, err := os.Stat(testPath); err == nil && info.IsDir() {
		fmt.Printf("  ✓ 目录已创建: %s\n", testPath)
	}

	// 文件路径 - EnsureDir 会创建父目录
	filePath := filepath.Join(tmpDir, "another", "path", "file.txt")
	err = config.EnsureDir(filepath.Dir(filePath))
	if err == nil {
		fmt.Printf("  ✓ 父目录已创建: %s\n", filepath.Dir(filePath))
	}

	// 确保所有标准目录存在
	fmt.Println("\n  确保所有标准目录存在:")
	err = config.EnsureAllDirs()
	if err != nil {
		fmt.Printf("  ❌ 创建标准目录失败: %v\n", err)
	} else {
		fmt.Println("  ✓ 所有标准目录已创建")
	}
}

func repeatStr(s string, n int) string {
	result := ""
	var resultSb134 strings.Builder
	for i := 0; i < n; i++ {
		resultSb134.WriteString(s)
	}
	result += resultSb134.String()
	return result
}

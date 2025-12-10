---
title: 跨平台路径
description: 遵循各平台标准的路径管理系统
navigation:
  icon: i-lucide-folder
---

# 跨平台路径管理

Aster 提供了跨平台的路径管理系统，自动适配 macOS、Linux 和 Windows 的标准路径约定。

## 🎯 设计原则

- 📁 遵循各平台的标准路径约定
- 🔧 提供便捷的文件路径生成方法
- 🏗️ 自动创建必要的目录结构
- 🔄 支持自定义应用名称

## 📋 平台路径约定

### macOS

遵循 Apple 的应用目录标准：

```
配置: ~/Library/Application Support/aster/
数据: ~/Library/Application Support/aster/
缓存: ~/Library/Caches/aster/
日志: ~/Library/Logs/aster/
```

### Linux

遵循 XDG Base Directory 规范：

```
配置: ~/.config/aster/
数据: ~/.local/share/aster/
缓存: ~/.cache/aster/
日志: ~/.local/share/aster/logs/
```

### Windows

遵循 Windows 应用数据标准：

```
配置: %APPDATA%\aster\
数据: %LOCALAPPDATA%\aster\
缓存: %LOCALAPPDATA%\aster\cache\
日志: %LOCALAPPDATA%\aster\logs\
```

## 🚀 快速开始

### 获取标准目录

```go
import "github.com/astercloud/aster/pkg/config"

// 获取各类目录路径
configDir := config.ConfigDir()  // 配置目录
dataDir := config.DataDir()      // 数据目录
cacheDir := config.CacheDir()    // 缓存目录
logDir := config.LogDir()        // 日志目录

fmt.Printf("配置目录: %s\n", configDir)
fmt.Printf("数据目录: %s\n", dataDir)
```

### 获取文件路径

```go
// 配置文件
settingsPath := config.ConfigFile("settings.yaml")
// macOS: ~/Library/Application Support/aster/settings.yaml

// 数据文件
dataPath := config.DataFile("data.json")
// macOS: ~/Library/Application Support/aster/data.json

// 数据库文件
dbPath := config.DatabaseFile("sessions.db")
// macOS: ~/Library/Application Support/aster/sessions.db

// 日志文件
logPath := config.LogFile("app.log")
// macOS: ~/Library/Logs/aster/app.log

// 缓存文件
cachePath := config.CacheFile("temp.cache")
// macOS: ~/Library/Caches/aster/temp.cache
```

### 自定义应用名

```go
// 为不同应用创建路径管理器
paths := config.NewPaths("myapp")

configDir := paths.ConfigDir()
// macOS: ~/Library/Application Support/myapp/

dataDir := paths.DataDir()
// macOS: ~/Library/Application Support/myapp/

// 使用自定义路径的文件
dbPath := paths.DatabaseFile("sessions.db")
// macOS: ~/Library/Application Support/myapp/sessions.db
```

### 确保目录存在

```go
// 确保目录存在（递归创建）
err := config.EnsureDir("/path/to/nested/directory")
if err != nil {
    log.Fatal(err)
}

// 常见用法：确保配置目录存在
err = config.EnsureDir(config.ConfigDir())
```

## 📊 使用场景

### SQLite 数据库

```go
import (
    "github.com/astercloud/aster/pkg/config"
    "github.com/astercloud/aster/pkg/session/sqlite"
)

// 使用标准数据目录存储数据库
dbPath := config.DatabaseFile("sessions.db")

// 确保目录存在
config.EnsureDir(filepath.Dir(dbPath))

// 创建 SQLite 服务
service, err := sqlite.New(dbPath)
```

### 日志文件

```go
import (
    "github.com/astercloud/aster/pkg/config"
    "log"
    "os"
)

// 使用标准日志目录
logPath := config.LogFile("app.log")

// 确保目录存在
config.EnsureDir(config.LogDir())

// 创建日志文件
logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
if err != nil {
    log.Fatal(err)
}
log.SetOutput(logFile)
```

### 配置文件

```go
import (
    "github.com/astercloud/aster/pkg/config"
    "gopkg.in/yaml.v3"
    "os"
)

type AppConfig struct {
    APIKey string `yaml:"api_key"`
    Model  string `yaml:"model"`
}

// 加载配置
configPath := config.ConfigFile("config.yaml")
data, err := os.ReadFile(configPath)
if err != nil {
    // 使用默认配置
    return &AppConfig{Model: "claude-sonnet-4-5"}
}

var cfg AppConfig
yaml.Unmarshal(data, &cfg)
```

### 权限规则

```go
import "github.com/astercloud/aster/pkg/permission"

// 权限规则存储在配置目录
inspector, _ := permission.NewInspector(
    permission.WithPath(config.ConfigFile("permissions.json")),
)
```

## 🔧 环境变量

### Linux XDG 覆盖

支持通过环境变量覆盖默认路径：

```bash
# 自定义配置目录
export XDG_CONFIG_HOME=~/.myconfig

# 自定义数据目录
export XDG_DATA_HOME=~/.mydata

# 自定义缓存目录
export XDG_CACHE_HOME=~/.mycache

# 运行应用
./myapp
```

### 完全自定义

```go
// 完全自定义所有路径
paths := &config.CustomPaths{
    Config: "/custom/config",
    Data:   "/custom/data",
    Cache:  "/custom/cache",
    Log:    "/custom/logs",
}

configDir := paths.ConfigDir()
// 返回: /custom/config
```

## 💡 最佳实践

### 1. 使用标准路径

```go
// ✅ 推荐：使用标准路径函数
dbPath := config.DatabaseFile("sessions.db")

// ❌ 不推荐：硬编码路径
// dbPath := "/Users/me/data/sessions.db"
```

### 2. 确保目录存在

```go
// ✅ 推荐：写入前确保目录存在
config.EnsureDir(filepath.Dir(filePath))
os.WriteFile(filePath, data, 0644)
```

### 3. 处理错误

```go
// ✅ 推荐：处理路径错误
configDir := config.ConfigDir()
if configDir == "" {
    log.Fatal("无法确定配置目录")
}
```

### 4. 迁移旧数据

```go
// 检查旧位置的数据
oldPath := filepath.Join(os.Getenv("HOME"), ".aster", "sessions.db")
newPath := config.DatabaseFile("sessions.db")

if _, err := os.Stat(oldPath); err == nil {
    // 迁移到新位置
    os.Rename(oldPath, newPath)
}
```

## 📚 相关文档

- [SQLite 会话存储](/core-concepts/session-sqlite) - 数据库存储
- [桌面应用部署](/deployment/desktop) - 桌面框架集成
- [Permission 系统](/security/permission) - 权限配置存储

## 🔗 示例代码

```bash
# 运行路径示例
go run ./examples/config-paths/
```

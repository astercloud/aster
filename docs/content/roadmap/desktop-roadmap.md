# Aster 桌面端路线图

> 借鉴 Goose 项目的桌面端架构，为 Aster 提供完整的桌面应用支持。

## 一、架构设计

### 1.1 整体架构

```
┌─────────────────────────────────────────────────────┐
│                   Aster Desktop                      │
│  ┌─────────────────────────────────────────────────┐│
│  │              Tauri/Wails Shell                  ││
│  │  ┌─────────────────┐  ┌─────────────────────┐  ││
│  │  │   Web Frontend  │  │   Native Features   │  ││
│  │  │   (Vue3 + UI)   │  │  (Tray, Shortcuts)  │  ││
│  │  └────────┬────────┘  └──────────┬──────────┘  ││
│  └───────────┼──────────────────────┼─────────────┘│
│              │ HTTP/WebSocket       │              │
│  ┌───────────▼──────────────────────▼─────────────┐│
│  │              Aster Backend (sidecar)           ││
│  │  ┌─────────┐ ┌─────────┐ ┌─────────────────┐  ││
│  │  │ Agent   │ │ Session │ │   Local Tools   │  ││
│  │  │ Engine  │ │ (SQLite)│ │ (Bash, FS, MCP) │  ││
│  │  └─────────┘ └─────────┘ └─────────────────┘  ││
│  └─────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────┘
```

### 1.2 借鉴 Goose 的核心模式

| Goose 设计 | Aster 实现方案 |
|-----------|---------------|
| `goosed` (Rust binary) | `aster-desktop-server` (Go binary) |
| Electron spawn backend | Tauri/Wails sidecar 模式 |
| 随机端口 + 健康检查 | 同样实现 |
| `~/.config/goose` | `~/.config/aster` (遵循 XDG) |
| Recipe 系统 | Aster Template + Recipe 扩展 |
| Permission UI | Control Channel 事件驱动 |

## 二、需要新增的模块

### 2.1 SQLite Session Store (优先级: P0)

**文件**: `pkg/session/sqlite/store.go`

```go
package sqlite

import (
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
)

type Store struct {
    db   *sql.DB
    path string
}

func New(dbPath string) (*Store, error) {
    // 自动创建数据库文件
    // 初始化 schema (sessions, events 表)
}

func (s *Store) CreateSession(ctx context.Context, sess *session.Session) error
func (s *Store) AppendEvent(ctx context.Context, sessionID string, event *session.Event) error
func (s *Store) GetSession(ctx context.Context, id string) (*session.Session, error)
func (s *Store) ListSessions(ctx context.Context, opts ListOptions) ([]*session.Session, error)
```

### 2.2 配置路径管理 (优先级: P0)

**文件**: `pkg/config/paths.go`

```go
package config

import (
    "os"
    "path/filepath"
    "runtime"
)

// Paths 提供跨平台的配置路径
type Paths struct{}

// ConfigDir 返回配置目录
// macOS: ~/Library/Application Support/Aster
// Linux: ~/.config/aster (XDG_CONFIG_HOME)
// Windows: %APPDATA%/Aster
func ConfigDir() string {
    switch runtime.GOOS {
    case "darwin":
        return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Aster")
    case "windows":
        return filepath.Join(os.Getenv("APPDATA"), "Aster")
    default:
        if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
            return filepath.Join(xdg, "aster")
        }
        return filepath.Join(os.Getenv("HOME"), ".config", "aster")
    }
}

// DataDir 返回数据目录 (sessions, memories)
func DataDir() string { ... }

// LogDir 返回日志目录
func LogDir() string { ... }

// CacheDir 返回缓存目录
func CacheDir() string { ... }
```

### 2.3 Recipe 系统 (优先级: P1)

**文件**: `pkg/recipe/recipe.go`

借鉴 Goose 的 Recipe 格式，扩展 Aster 的 Template：

```yaml
# ~/.config/aster/recipes/code-review.yaml
version: "1.0"
title: "Code Review Assistant"
description: "AI-powered code review helper"

# 基于现有 template
template_id: "default"

# 覆盖系统提示词
instructions: |
  You are a senior code reviewer. Focus on:
  - Code quality and best practices
  - Security vulnerabilities
  - Performance issues

# 初始提示
prompt: "Please review the code in the current directory."

# 启用的工具
tools:
  - filesystem
  - bash

# MCP 扩展
extensions:
  - type: stdio
    name: git-mcp
    cmd: npx
    args: ["-y", "@anthropic/git-mcp"]

# 参数化
parameters:
  - key: language
    type: select
    options: ["go", "python", "javascript", "rust"]
    default: "go"

# 权限模式
permission_mode: smart_approve  # auto_approve | smart_approve | always_ask
```

### 2.4 Permission UI 增强 (优先级: P1)

**文件**: `pkg/permission/permission.go`

```go
package permission

type Mode string

const (
    ModeAutoApprove  Mode = "auto_approve"   // 自动批准所有操作
    ModeSmartApprove Mode = "smart_approve"  // 智能判断，只读自动批准
    ModeAlwaysAsk    Mode = "always_ask"     // 总是询问
)

// Inspector 检查工具调用是否需要用户确认
type Inspector struct {
    mode          Mode
    readOnlyTools map[string]bool
    trustedTools  map[string]bool
}

// Check 返回是否需要用户确认
func (i *Inspector) Check(toolName string, args map[string]any) (needConfirm bool, reason string)
```

通过 Control Channel 发送确认请求到前端：

```go
// 在工具执行前
if needConfirm, reason := inspector.Check(tool.Name, args); needConfirm {
    // 发送到 Control Channel
    agent.ControlChannel() <- &events.ToolApprovalRequest{
        ToolName: tool.Name,
        Args:     args,
        Reason:   reason,
    }
    // 等待用户响应
    response := <-agent.ControlChannel()
    if !response.Approved {
        return nil, ErrToolDenied
    }
}
```

## 三、桌面应用入口

### 3.1 方案选择: Wails (Go Native)

推荐使用 **Wails** 而非 Tauri/Electron：
- Go 原生，无需 Rust 编译链
- 前端复用现有 Vue3 UI
- 二进制更小 (~10MB vs Electron ~150MB)

**目录结构**:

```
cmd/
├── aster/              # CLI 工具 (已有)
├── aster-server/       # 服务端 (已有)
└── aster-desktop/      # 桌面应用 (新增)
    ├── main.go         # Wails 入口
    ├── app.go          # 应用逻辑
    ├── backend.go      # 后端服务管理
    └── frontend/       # 软链接到 ui/dist
```

### 3.2 后端服务管理

**文件**: `cmd/aster-desktop/backend.go`

```go
package main

import (
    "context"
    "fmt"
    "net"
    "os/exec"
    "time"
)

type BackendManager struct {
    cmd     *exec.Cmd
    port    int
    dataDir string
}

// FindAvailablePort 找到可用端口
func FindAvailablePort() (int, error) {
    listener, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil {
        return 0, err
    }
    defer listener.Close()
    return listener.Addr().(*net.TCPAddr).Port, nil
}

// Start 启动后端服务
func (m *BackendManager) Start(ctx context.Context) error {
    port, err := FindAvailablePort()
    if err != nil {
        return err
    }
    m.port = port

    // 启动 aster serve 作为子进程
    m.cmd = exec.CommandContext(ctx, "aster", "serve",
        "--port", fmt.Sprintf("%d", port),
        "--data-dir", m.dataDir,
        "--store", "sqlite",
    )
    
    if err := m.cmd.Start(); err != nil {
        return err
    }

    // 健康检查
    return m.waitForReady(ctx, 10*time.Second)
}

// waitForReady 等待后端就绪
func (m *BackendManager) waitForReady(ctx context.Context, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", m.port))
        if err == nil && resp.StatusCode == 200 {
            return nil
        }
        time.Sleep(100 * time.Millisecond)
    }
    return fmt.Errorf("backend failed to start within %v", timeout)
}

// Stop 停止后端服务
func (m *BackendManager) Stop() error {
    if m.cmd != nil && m.cmd.Process != nil {
        return m.cmd.Process.Kill()
    }
    return nil
}
```

## 四、CLI 增强 (借鉴 Goose)

### 4.1 新增命令

```bash
# 交互式会话 (类似 goose session)
aster session [--recipe <name>] [--dir <path>]

# 运行 recipe
aster run --recipe code-review.yaml

# 配置管理
aster configure              # 交互式配置
aster configure providers    # 配置 LLM Provider
aster configure extensions   # 配置 MCP 扩展

# Recipe 管理
aster recipe list            # 列出可用 recipes
aster recipe create          # 创建新 recipe
aster recipe validate <file> # 验证 recipe 格式
```

### 4.2 交互式 Session

**文件**: `cmd/aster/session.go`

```go
package main

import (
    "bufio"
    "fmt"
    "os"
    
    "github.com/astercloud/aster/pkg/agent"
)

func runSession(args []string) error {
    // 解析参数
    recipeFile := flagSet.String("recipe", "", "Recipe file to use")
    workDir := flagSet.String("dir", ".", "Working directory")
    
    // 加载配置
    cfg := loadConfig()
    
    // 创建 Agent
    a, err := agent.New(agent.WithConfig(cfg))
    if err != nil {
        return err
    }
    
    // REPL 循环
    scanner := bufio.NewScanner(os.Stdin)
    fmt.Print("aster> ")
    
    for scanner.Scan() {
        input := scanner.Text()
        
        // 处理特殊命令
        switch input {
        case "/exit", "/quit":
            return nil
        case "/clear":
            a.ClearHistory()
            continue
        case "/help":
            printHelp()
            continue
        }
        
        // 发送到 Agent
        for event := range a.Run(ctx, input) {
            switch e := event.(type) {
            case *events.TextChunk:
                fmt.Print(e.Text)
            case *events.ToolApprovalRequest:
                // 处理工具确认
                if confirm := askUserConfirm(e); confirm {
                    a.ApproveToolCall(e.ID)
                } else {
                    a.DenyToolCall(e.ID)
                }
            }
        }
        
        fmt.Print("\naster> ")
    }
    
    return nil
}
```

## 五、实施路线图

### Phase 1: 基础设施 (2 周)

- [ ] `pkg/session/sqlite` - SQLite 存储实现
- [ ] `pkg/config/paths.go` - 跨平台路径管理
- [ ] `cmd/aster serve --store sqlite` - 支持 SQLite 模式
- [ ] `/health` 健康检查端点

### Phase 2: CLI 增强 (2 周)

- [ ] `aster session` - 交互式会话命令
- [ ] `aster configure` - 交互式配置
- [ ] `pkg/recipe` - Recipe 系统
- [ ] `aster run --recipe` - Recipe 执行

### Phase 3: 桌面应用 (3 周)

- [ ] Wails 项目初始化
- [ ] 后端进程管理
- [ ] 前端 UI 适配 (托盘、快捷键)
- [ ] 打包脚本 (macOS, Windows, Linux)

### Phase 4: 用户体验 (2 周)

- [ ] Permission UI (工具确认对话框)
- [ ] 自动更新机制
- [ ] 系统通知集成
- [ ] 多窗口/多会话支持

## 六、配置文件示例

### 6.1 全局配置

**`~/.config/aster/config.yaml`**

```yaml
# LLM Provider 配置
providers:
  default:
    provider: anthropic
    model: claude-sonnet-4-20250514
    env_api_key: ANTHROPIC_API_KEY
  
  fast:
    provider: openai
    model: gpt-4o-mini
    env_api_key: OPENAI_API_KEY

# 默认 Provider
default_provider: default

# 权限模式
permission_mode: smart_approve

# MCP 扩展
extensions:
  - name: filesystem
    type: builtin
    enabled: true
  
  - name: git
    type: stdio
    cmd: npx
    args: ["-y", "@anthropic/git-mcp"]
    enabled: true

# 存储配置
storage:
  type: sqlite
  path: ~/.config/aster/data/aster.db

# 日志配置
logging:
  level: info
  path: ~/.config/aster/logs/
```

### 6.2 项目级配置

**`.aster/config.yaml`** (项目根目录)

```yaml
# 继承全局配置
extends: global

# 项目特定的 recipe
recipe: code-review

# 覆盖权限模式
permission_mode: always_ask

# 项目特定的 MCP 扩展
extensions:
  - name: project-docs
    type: stdio
    cmd: python
    args: ["-m", "docs_mcp", "--root", "."]
```

## 七、与 Goose 的差异化

| 特性 | Goose | Aster |
|-----|-------|-------|
| **语言** | Rust + TypeScript | Go + Vue3 |
| **工作流** | 单 Agent | Sequential/Parallel/Loop |
| **内存系统** | 简单上下文 | 三层内存 (Text/Working/Semantic) |
| **沙箱** | 本地 | 本地 + Docker + 云 |
| **Multi-Agent** | 无 | Stars 协作模式 |
| **桌面框架** | Electron (~150MB) | Wails (~15MB) |

**Aster 的优势**:
1. 更强大的工作流引擎
2. 更完善的内存管理
3. Multi-Agent 协作
4. 更轻量的桌面应用
5. Go 生态系统 (部署简单)

## 八、总结

通过借鉴 Goose 的桌面端架构，Aster 可以快速实现：

1. **即开即用的桌面体验** - 下载即运行，无需复杂配置
2. **CLI 和 Desktop 共享配置** - 用户可自由切换
3. **Recipe 系统** - 快速创建和分享 Agent 配置
4. **安全的权限控制** - 用户对工具调用有完全控制权

核心代码量估计：
- SQLite Store: ~500 行
- 配置路径管理: ~200 行
- Recipe 系统: ~800 行
- 桌面应用入口: ~600 行
- CLI Session: ~400 行

总计约 **2500 行新代码**，可在 **2-3 个月**内完成。

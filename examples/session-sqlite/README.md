# SQLite Session Store 示例

本示例演示如何使用 SQLite 作为会话存储后端，这是桌面应用的理想选择。

## 功能特点

- 🗄️ 轻量级本地存储，无需外部数据库服务
- ⚡ WAL 模式提高读写性能
- 🔄 支持与 PostgreSQL/MySQL 相同的接口
- 💾 数据持久化在单个文件中
- 🔒 支持事务和 ACID 保证

## 运行示例

```bash
go run ./examples/session-sqlite/
```

## 代码示例

### 创建 SQLite 服务

```go
import "github.com/astercloud/aster/pkg/session/sqlite"

// 创建 SQLite 会话服务
service, err := sqlite.New("./data/sessions.db")
if err != nil {
    log.Fatal(err)
}
defer service.Close()
```

### 使用跨平台路径

```go
import (
    "github.com/astercloud/aster/pkg/config"
    "github.com/astercloud/aster/pkg/session/sqlite"
)

// 使用标准应用数据目录
dbPath := config.DatabaseFile("sessions.db")
service, err := sqlite.New(dbPath)
```

### 创建和管理会话

```go
ctx := context.Background()

// 创建会话
sess, err := service.Create(ctx, &session.CreateRequest{
    AppName: "my-app",
    UserID:  "user-123",
    AgentID: "agent-001",
    Metadata: map[string]any{
        "source": "desktop",
    },
})

// 添加事件
sess.AddEvent(ctx, session.AddEventOptions{
    Author:  "user",
    Content: "Hello, AI!",
})

// 列出会话
sessions, err := service.List(ctx, &session.ListRequest{
    AppName: "my-app",
    UserID:  "user-123",
})

// 获取会话
sess, err = service.Get(ctx, &session.GetRequest{
    AppName:   "my-app",
    UserID:    "user-123",
    SessionID: "session-id",
})
```

## 数据库结构

SQLite 数据库包含以下表：

### sessions 表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | TEXT | 会话 ID (主键) |
| app_name | TEXT | 应用名称 |
| user_id | TEXT | 用户 ID |
| agent_id | TEXT | Agent ID |
| metadata | TEXT | JSON 格式的元数据 |
| created_at | DATETIME | 创建时间 |
| updated_at | DATETIME | 更新时间 |

### events 表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | TEXT | 事件 ID (主键) |
| session_id | TEXT | 关联的会话 ID |
| author | TEXT | 作者 (user/assistant/system) |
| content | TEXT | 消息内容 |
| reasoning | TEXT | 推理过程 |
| actions | TEXT | JSON 格式的工具调用 |
| metadata | TEXT | JSON 格式的元数据 |
| created_at | DATETIME | 创建时间 |

## 适用场景

- ✅ 桌面应用 (Wails, Tauri, Electron)
- ✅ CLI 工具
- ✅ 单用户应用
- ✅ 开发和测试环境
- ✅ 嵌入式系统

## 不适用场景

- ❌ 高并发多用户服务
- ❌ 分布式部署
- ❌ 需要主从复制的场景

## 相关示例

- [session](../session/) - 内存会话存储
- [session-postgres](../session-postgres/) - PostgreSQL 会话存储
- [session-mysql](../session-mysql/) - MySQL 会话存储
- [desktop](../desktop/) - 桌面应用集成

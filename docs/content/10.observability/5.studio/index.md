---
title: Aster Studio
description: 可视化监控和管理 Agent 的 Web 控制台
---

# Aster Studio

Aster Studio 是一个内置的 Web 控制台，提供实时监控、事件流查看、Agent 管理等功能。

## 功能概览

- **Overview（概览）**: 查看活跃 Agent 数量、Token 使用量、成本统计
- **Agents（Agent 列表）**: 管理本地和远程 Agent
- **Sessions（会话）**: 查看 Agent 会话历史
- **Events（事件流）**: 实时查看 Agent 事件，支持筛选和搜索
- **Traces（追踪）**: 分布式追踪和链路分析

## 快速开始

### 1. 构建 Studio 版本

```bash
# 构建前端
cd studio
npm install
npm run build

# 复制前端资源
cp -r dist ../server/studio/

# 构建带 Studio 的服务端
cd ..
go build -tags studio -o aster-server-studio ./cmd/aster-server
```

### 2. 启动服务

```bash
# 使用默认 JSON 存储
PORT=3032 ./aster-server-studio

# 使用 MySQL 存储
ASTER_STORE_TYPE=mysql \
ASTER_MYSQL_DSN="root:root@tcp(localhost:3306)/aster?charset=utf8mb4&parseTime=True&loc=Local" \
PORT=3032 ./aster-server-studio

# 使用 Redis 存储
ASTER_STORE_TYPE=redis \
ASTER_REDIS_ADDR="localhost:6379" \
PORT=3032 ./aster-server-studio
```

### 3. 访问控制台

打开浏览器访问 `http://localhost:3032/studio`

## 存储配置

Aster Studio 支持三种存储后端：

### JSON 文件存储（默认）

适合开发和测试环境，数据存储在本地文件系统。

```bash
ASTER_DATA_DIR=.data ./aster-server-studio
```

### MySQL 存储

适合生产环境，支持持久化和多实例部署。

```bash
# 环境变量
ASTER_STORE_TYPE=mysql
ASTER_MYSQL_DSN="user:password@tcp(host:port)/database?charset=utf8mb4&parseTime=True&loc=Local"

# 可选配置
ASTER_MYSQL_MAX_OPEN_CONNS=25
ASTER_MYSQL_MAX_IDLE_CONNS=10
```

自动创建的表：
- `agent_infos` - Agent 元信息
- `agent_messages` - 消息历史
- `agent_snapshots` - 快照
- `agent_todos` - Todo 列表
- `agent_tool_records` - 工具调用记录
- `aster_collections` - 通用 KV 存储

### Redis 存储

适合需要高性能缓存的场景。

```bash
# 环境变量
ASTER_STORE_TYPE=redis
ASTER_REDIS_ADDR=localhost:6379
ASTER_REDIS_PASSWORD=
ASTER_REDIS_DB=0
ASTER_REDIS_PREFIX=aster:
```

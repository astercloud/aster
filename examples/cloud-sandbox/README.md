# Cloud Sandbox Example

演示如何使用阿里云 AgentBay 和火山引擎云沙箱。

## 配置

### 阿里云 AgentBay

```bash
export CLOUD_PLATFORM=aliyun
export ALIYUN_MCP_ENDPOINT=https://your-mcp-endpoint.aliyuncs.com
export ALIYUN_ACCESS_KEY_ID=your_access_key_id
export ALIYUN_ACCESS_KEY_SECRET=your_access_key_secret
export ALIYUN_SECURITY_TOKEN=your_security_token  # 可选,用于临时凭证
```

### 火山引擎

```bash
export CLOUD_PLATFORM=volcengine
export VOLCENGINE_ENDPOINT=https://your-endpoint.volcengineapi.com
export VOLCENGINE_ACCESS_KEY=your_access_key
export VOLCENGINE_SECRET_KEY=your_secret_key
```

## 运行

```bash
cd examples/cloud-sandbox
go run main.go
```

## 测试内容

1. **执行命令** - echo 测试
2. **写入文件** - 创建 test.txt
3. **读取文件** - 读取内容
4. **列出文件** - 列出工作目录
5. **文件信息** - 获取元数据
6. **删除文件** - 清理测试文件

## 架构

```
┌─────────────────┐
│  Application    │
└────────┬────────┘
         │
┌────────▼────────┐
│  Sandbox        │
│  Interface      │
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
┌───▼───┐ ┌──▼────────┐
│Aliyun │ │Volcengine │
│Sandbox│ │ Sandbox   │
└───┬───┘ └──┬────────┘
    │        │
┌───▼────────▼───┐
│  MCP Client    │
└────────┬───────┘
         │
┌────────▼───────┐
│  Cloud API     │
└────────────────┘
```

## MCP 工具映射

### 阿里云 AgentBay

| 功能     | MCP 工具         |
| -------- | ---------------- |
| 执行命令 | `shell`          |
| 读取文件 | `read_file`      |
| 写入文件 | `write_file`     |
| 列出文件 | `list_directory` |
| 文件信息 | `get_file_info`  |
| 删除文件 | `delete_file`    |
| 搜索文件 | `search_files`   |

### 火山引擎

| 功能       | MCP 工具               |
| ---------- | ---------------------- |
| 初始化会话 | `computer_init`        |
| 执行命令   | `computer_exec`        |
| 读取文件   | `computer_read_file`   |
| 写入文件   | `computer_write_file`  |
| 列出文件   | `computer_list_files`  |
| 文件信息   | `computer_stat_file`   |
| 删除文件   | `computer_delete_file` |
| Glob 匹配  | `computer_glob`        |
| 终止会话   | `computer_terminate`   |

## 注意事项

1. **认证**: 确保 API 凭证有效且有足够权限
2. **配额**: 云平台可能有使用配额限制
3. **超时**: 默认超时 60 秒,可通过配置调整
4. **会话管理**: 火山引擎需要显式创建和销毁会话
5. **网络**: 需要能访问云平台 API 端点

## 错误处理

- **认证失败**: 检查 AccessKey 和 SecretKey
- **超时**: 增加 Timeout 配置
- **权限不足**: 检查 IAM 策略
- **网络错误**: 检查防火墙和 DNS 设置

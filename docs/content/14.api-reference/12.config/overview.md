# Config API 参考

配置 (Config) 包提供了 YAML 配置文件加载、环境变量展开、配置合并和验证功能。

## 包导入

```go
import "github.com/astercloud/aster/pkg/config"
```

## Loader

配置加载器。

### NewLoader

创建配置加载器。

```go
func NewLoader(opts ...LoaderOption) *Loader
```

**参数：**
- `opts` - 加载器选项

**返回：**
- `*Loader` - 加载器实例

**示例：**
```go
loader := config.NewLoader()

// 带选项
loader := config.NewLoader(
    config.WithEnvPrefix("APP_"),
    config.WithVariables(map[string]string{
        "VERSION": "1.0.0",
    }),
)
```

### LoadAgentConfig

加载 Agent 配置文件。

```go
func (l *Loader) LoadAgentConfig(path string) (*types.AgentConfig, error)
```

**参数：**
- `path` - 配置文件路径

**返回：**
- `*types.AgentConfig` - Agent 配置
- `error` - 错误信息

**示例：**
```go
config, err := loader.LoadAgentConfig("agent.yaml")
if err != nil {
    return err
}
```

### LoadModelConfig

加载模型配置文件。

```go
func (l *Loader) LoadModelConfig(path string) (*types.ModelConfig, error)
```

**参数：**
- `path` - 配置文件路径

**返回：**
- `*types.ModelConfig` - 模型配置
- `error` - 错误信息

**示例：**
```go
modelConfig, err := loader.LoadModelConfig("model.yaml")
if err != nil {
    return err
}
```

### LoadFromString

从字符串加载配置。

```go
func (l *Loader) LoadFromString(content string, target any) error
```

**参数：**
- `content` - YAML 内容
- `target` - 目标结构体指针

**返回：**
- `error` - 错误信息

**示例：**
```go
yamlContent := `
template_id: "test-agent"
model_config:
  provider: "anthropic"
  model: "claude-3-5-sonnet-20241022"
`

var config types.AgentConfig
err := loader.LoadFromString(yamlContent, &config)
```

## 加载器选项

### WithEnvExpansion

启用或禁用环境变量展开。

```go
func WithEnvExpansion(enabled bool) LoaderOption
```

**参数：**
- `enabled` - 是否启用（默认为 true）

**示例：**
```go
loader := config.NewLoader(
    config.WithEnvExpansion(false), // 禁用环境变量展开
)
```

### WithEnvPrefix

设置环境变量前缀。

```go
func WithEnvPrefix(prefix string) LoaderOption
```

**参数：**
- `prefix` - 环境变量前缀

**示例：**
```go
loader := config.NewLoader(
    config.WithEnvPrefix("APP_"),
)

// 配置文件中 ${API_KEY} 会查找 APP_API_KEY 环境变量
```

### WithVariables

提供自定义变量。

```go
func WithVariables(vars map[string]string) LoaderOption
```

**参数：**
- `vars` - 变量映射

**示例：**
```go
loader := config.NewLoader(
    config.WithVariables(map[string]string{
        "ENVIRONMENT": "production",
        "VERSION":     "1.0.0",
        "REGION":      "us-west-2",
    }),
)
```

## 配置合并

### MergeConfigs

合并多个配置。

```go
func MergeConfigs(base *types.AgentConfig, overlays ...*types.AgentConfig) *types.AgentConfig
```

**参数：**
- `base` - 基础配置
- `overlays` - 要合并的配置列表

**返回：**
- `*types.AgentConfig` - 合并后的配置

**说明：**
- 后面的配置会覆盖前面的配置
- 嵌套结构会递归合并
- 切片会追加而不是替换

**示例：**
```go
baseConfig, _ := loader.LoadAgentConfig("base.yaml")
prodConfig, _ := loader.LoadAgentConfig("production.yaml")

// 合并配置
finalConfig := config.MergeConfigs(baseConfig, prodConfig)

// 多层合并
finalConfig := config.MergeConfigs(
    baseConfig,
    envConfig,
    userConfig,
)
```

## 配置路径

### ConfigDir

获取配置目录。

```go
func ConfigDir() string
```

**返回：**
- `string` - 配置目录路径（~/.config/aster/）

**示例：**
```go
configDir := config.ConfigDir()
// 返回: /Users/username/.config/aster/
```

### DataDir

获取数据目录。

```go
func DataDir() string
```

**返回：**
- `string` - 数据目录路径（~/.local/share/aster/）

### LogDir

获取日志目录。

```go
func LogDir() string
```

**返回：**
- `string` - 日志目录路径（~/.local/share/aster/logs/）

### CacheDir

获取缓存目录。

```go
func CacheDir() string
```

**返回：**
- `string` - 缓存目录路径（~/.cache/aster/）

### DatabaseFile

获取数据库文件路径。

```go
func DatabaseFile() string
```

**返回：**
- `string` - 数据库文件路径

### ProjectConfigFile

获取项目配置文件路径。

```go
func ProjectConfigFile() string
```

**返回：**
- `string` - 项目配置文件路径（./.aster/config.yaml）

### ResolveConfigFile

解析配置文件路径。

```go
func ResolveConfigFile(filename string) string
```

**参数：**
- `filename` - 配置文件名

**返回：**
- `string` - 解析后的完整路径

**说明：**
按以下顺序查找文件：
1. 当前目录
2. ./.aster/ 目录
3. ~/.config/aster/ 目录

**示例：**
```go
configPath := config.ResolveConfigFile("agent.yaml")
// 查找顺序:
// 1. ./agent.yaml
// 2. ./.aster/agent.yaml
// 3. ~/.config/aster/agent.yaml
```

### EnsureDir

确保目录存在。

```go
func EnsureDir(dir string) error
```

**参数：**
- `dir` - 目录路径

**返回：**
- `error` - 错误信息

**示例：**
```go
err := config.EnsureDir(config.DataDir())
if err != nil {
    return err
}
```

## 环境变量语法

配置文件中支持以下环境变量语法：

### 基本引用

```yaml
# ${VAR} 语法
api_key: "${API_KEY}"

# $VAR 语法
api_key: "$API_KEY"
```

### 默认值

```yaml
# 如果变量不存在，使用默认值
api_key: "${API_KEY:-default-key}"
model: "${MODEL_NAME:-claude-3-5-sonnet-20241022}"
```

### 必需变量

```yaml
# 如果变量不存在，报错
api_key: "${API_KEY:?API key is required}"
database_url: "${DATABASE_URL:?Database URL must be set}"
```

### 混合使用

```yaml
# 在字符串中使用变量
base_url: "https://${API_HOST:-api.example.com}/v1"
connection_string: "postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}"
```

## 配置验证

配置加载时会自动验证以下内容：

### AgentConfig 验证

- `template_id` 必需
- 如果提供 `model_config`：
  - `provider` 必需
  - `model` 必需
- 如果启用 `multitenancy`：
  - 必须提供 `org_id` 或 `tenant_id`

### ModelConfig 验证

- `provider` 必需
- `model` 必需

## 完整示例

### 基本使用

```go
package main

import (
    "log"

    "github.com/astercloud/aster/pkg/agent"
    "github.com/astercloud/aster/pkg/config"
)

func main() {
    // 创建加载器
    loader := config.NewLoader()

    // 加载配置
    agentConfig, err := loader.LoadAgentConfig("agent.yaml")
    if err != nil {
        log.Fatalf("加载配置失败: %v", err)
    }

    // 创建 Agent
    agent, err := agent.NewAgent(agentConfig)
    if err != nil {
        log.Fatalf("创建 Agent 失败: %v", err)
    }

    // 使用 Agent
    // ...
}
```

### 多环境配置

```go
package main

import (
    "log"
    "os"

    "github.com/astercloud/aster/pkg/config"
)

func main() {
    loader := config.NewLoader()

    // 加载基础配置
    baseConfig, err := loader.LoadAgentConfig("config/base.yaml")
    if err != nil {
        log.Fatalf("加载基础配置失败: %v", err)
    }

    // 根据环境加载对应配置
    env := os.Getenv("ENVIRONMENT")
    if env == "" {
        env = "development"
    }

    envConfigPath := fmt.Sprintf("config/%s.yaml", env)
    envConfig, err := loader.LoadAgentConfig(envConfigPath)
    if err != nil {
        log.Fatalf("加载环境配置失败: %v", err)
    }

    // 合并配置
    finalConfig := config.MergeConfigs(baseConfig, envConfig)

    // 创建 Agent
    agent, err := agent.NewAgent(finalConfig)
    if err != nil {
        log.Fatalf("创建 Agent 失败: %v", err)
    }
}
```

### 自定义变量

```go
package main

import (
    "log"

    "github.com/astercloud/aster/pkg/config"
)

func main() {
    // 创建带自定义变量的加载器
    loader := config.NewLoader(
        config.WithEnvPrefix("APP_"),
        config.WithVariables(map[string]string{
            "VERSION":     "1.0.0",
            "ENVIRONMENT": "production",
            "REGION":      "us-west-2",
        }),
    )

    // 加载配置
    agentConfig, err := loader.LoadAgentConfig("agent.yaml")
    if err != nil {
        log.Fatalf("加载配置失败: %v", err)
    }

    log.Printf("加载配置成功: %s", agentConfig.TemplateID)
}
```

### 从字符串加载

```go
package main

import (
    "log"

    "github.com/astercloud/aster/pkg/config"
    "github.com/astercloud/aster/pkg/types"
)

func main() {
    loader := config.NewLoader()

    configYAML := `
template_id: "dynamic-agent"
model_config:
  provider: "anthropic"
  model: "claude-3-5-sonnet-20241022"
  api_key: "${ANTHROPIC_API_KEY}"
tools:
  - "read_file"
  - "write_file"
`

    var agentConfig types.AgentConfig
    err := loader.LoadFromString(configYAML, &agentConfig)
    if err != nil {
        log.Fatalf("加载配置失败: %v", err)
    }

    log.Printf("模板 ID: %s", agentConfig.TemplateID)
    log.Printf("工具数量: %d", len(agentConfig.Tools))
}
```

### 配置验证

```go
package main

import (
    "log"

    "github.com/astercloud/aster/pkg/config"
)

func main() {
    loader := config.NewLoader()

    // 加载配置
    agentConfig, err := loader.LoadAgentConfig("agent.yaml")
    if err != nil {
        // 检查不同类型的错误
        if os.IsNotExist(err) {
            log.Fatal("配置文件不存在")
        }

        if strings.Contains(err.Error(), "yaml") {
            log.Fatal("YAML 语法错误:", err)
        }

        if strings.Contains(err.Error(), "validate") {
            log.Fatal("配置验证失败:", err)
        }

        log.Fatalf("加载配置失败: %v", err)
    }

    log.Println("配置验证通过")
}
```

## 配置文件示例

### 基础配置

```yaml
# base.yaml
template_id: "general-assistant"
template_version: "1.0.0"

model_config:
  provider: "anthropic"
  model: "claude-3-5-sonnet-20241022"

tools:
  - "read_file"
  - "write_file"

metadata:
  version: "1.0.0"
```

### 生产环境配置

```yaml
# production.yaml
model_config:
  api_key: "${ANTHROPIC_API_KEY:?API key is required}"
  base_url: "${API_BASE_URL:-https://api.anthropic.com}"
  execution_mode: "streaming"

middlewares:
  - "logging"
  - "rate_limit"
  - "auth"

middleware_config:
  rate_limit:
    requests_per_minute: 60
  logging:
    level: "info"
    format: "json"

metadata:
  environment: "production"
  region: "${AWS_REGION:-us-west-2}"
```

### 开发环境配置

```yaml
# development.yaml
model_config:
  api_key: "${ANTHROPIC_API_KEY}"

tools:
  - "read_file"
  - "write_file"
  - "bash"
  - "debug_tool"

middlewares:
  - "logging"

middleware_config:
  logging:
    level: "debug"
    format: "text"

metadata:
  environment: "development"
  debug: true
```

## 错误处理

### 常见错误

```go
// 文件不存在
config, err := loader.LoadAgentConfig("nonexistent.yaml")
// err: open nonexistent.yaml: no such file or directory

// YAML 语法错误
// err: yaml: line 5: mapping values are not allowed in this context

// 验证错误
// err: validate config: template_id is required

// 环境变量未设置
// err: required environment variable API_KEY is not set
```

### 错误处理示例

```go
config, err := loader.LoadAgentConfig("agent.yaml")
if err != nil {
    switch {
    case os.IsNotExist(err):
        log.Fatal("配置文件不存在")
    case strings.Contains(err.Error(), "yaml"):
        log.Fatal("YAML 语法错误:", err)
    case strings.Contains(err.Error(), "validate"):
        log.Fatal("配置验证失败:", err)
    case strings.Contains(err.Error(), "required"):
        log.Fatal("缺少必需的环境变量:", err)
    default:
        log.Fatal("加载配置失败:", err)
    }
}
```

## 相关文档

- [配置系统概念](../../02.core-concepts/17.configuration.md)
- [Agent API](../1.agent/overview.md)
- [部署指南](../../09.deployment/overview.md)

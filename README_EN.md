<p align="center">
  <img src="https://raw.githubusercontent.com/astercloud/aster/main/docs/public/images/logo-banner.svg" alt="Aster · Where Stardust Converges" width="800">
</p>

<p align="center">
  <strong>Where Stardust Converges, Intelligence Emerges</strong><br>
  Empowering Every Agent to Shine in Production
</p>

<p align="center">
  Native Python, TypeScript, and Bash execution • Event-driven architecture • Enterprise-grade governance<br>
  <em>Go's performance foundation, scripting flexibility, built for production</em>
</p>

📖 **[Documentation](https://astercloud.github.io/aster/)** | 🚀 **[Quick Start](https://astercloud.github.io/aster/introduction/quickstart)** | 🏗️ **[Architecture](https://astercloud.github.io/aster/introduction/architecture)**

---

## 🌟 What is Aster?

**Aster** (星尘云枢) is a production-ready AI Agent development framework built with Go, designed to run agents safely and efficiently in enterprise environments.

Like stardust converging to form a celestial hub, Aster brings together:

- **High Performance**: Go's concurrency model supports 100+ concurrent agents
- **Native Script Execution**: Run Python, TypeScript, and Bash natively with full sandbox isolation
- **Event-Driven Architecture**: Progress/Control/Monitor tri-channel design for clear separation of concerns
- **Enterprise Security**: Cloud sandbox integration, PII auto-redaction, and comprehensive governance

## 🎯 Core Features

### 🚀 Multi-Language Script Execution

- **Python**: Execute data processing, ML workflows, and analytics scripts
- **TypeScript**: Run modern JavaScript/TypeScript for web automation and API interactions
- **Bash**: Shell commands for system operations and DevOps tasks
- All with **native performance** and **isolated sandbox environments**

### 🎪 Event-Driven Architecture

- **Progress Channel**: Real-time text streaming, tool execution progress
- **Control Channel**: Tool approval requests, human-in-the-loop interactions
- **Monitor Channel**: Governance events, error tracking, audit logs

### 🧅 Middleware Onion Model

- **Layered Processing**: Requests and responses flow through middleware layers
- **Built-in Middlewares**: Auto-summarization, PII redaction, tool interception
- **Extensible**: Create custom middleware for your specific needs

### 🔒 Enterprise-Grade Security

- **Cloud Sandbox**: Native integration with Aliyun AgentBay, Volcengine
- **PII Auto-Redaction**: Detect and redact 10+ types of sensitive data
- **Permission System**: Fine-grained tool-level access control
- **Audit Logging**: Complete tool call tracking and state management

### 🧠 Three-Layer Memory System

- **Text Memory**: File-based short-term memory for conversation context
- **Working Memory**: Persistent state across sessions with TTL and JSON schema
- **Semantic Memory**: Vector-based long-term memory with provenance tracking

### 🔄 Advanced Capabilities

- **Streaming API**: iter.Seq2-based streaming with 80%+ memory reduction
- **Long-Running Tools**: Async task management with progress tracking
- **Multi-Agent Orchestration**: Pool/Room/Workflow collaboration patterns
- **OpenTelemetry Integration**: Distributed tracing, metrics, and logging

## 📦 Installation

```bash
go get github.com/astercloud/aster
```

## 🚀 Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/astercloud/aster/pkg/agent"
    "github.com/astercloud/aster/pkg/provider"
    "github.com/astercloud/aster/pkg/sandbox"
    "github.com/astercloud/aster/pkg/store"
    "github.com/astercloud/aster/pkg/tools"
    "github.com/astercloud/aster/pkg/tools/builtin"
    "github.com/astercloud/aster/pkg/types"
)

func main() {
    // 1. Setup dependencies
    toolRegistry := tools.NewRegistry()
    builtin.RegisterAll(toolRegistry)

    jsonStore, _ := store.NewJSONStore("./.aster")
    deps := &agent.Dependencies{
        Store:            jsonStore,
        SandboxFactory:   sandbox.NewFactory(),
        ToolRegistry:     toolRegistry,
        ProviderFactory:  &provider.AnthropicFactory{},
        TemplateRegistry: agent.NewTemplateRegistry(),
    }

    // 2. Register agent template
    deps.TemplateRegistry.Register(&types.AgentTemplateDefinition{
        ID:           "assistant",
        SystemPrompt: "You are a helpful assistant with file and bash access.",
        Model:        "claude-sonnet-4-5",
        Tools:        []interface{}{"Read", "Write", "Bash"},
    })

    // 3. Create agent
    ag, err := agent.Create(context.Background(), &types.AgentConfig{
        TemplateID: "assistant",
        ModelConfig: &types.ModelConfig{
            Provider: "anthropic",
            Model:    "claude-sonnet-4-5",
            APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
        },
        Sandbox: &types.SandboxConfig{
            Kind:    types.SandboxKindLocal,
            WorkDir: "./workspace",
        },
    }, deps)
    if err != nil {
        log.Fatal(err)
    }
    defer ag.Close()

    // 4. Subscribe to events
    eventCh := ag.Subscribe([]types.AgentChannel{types.ChannelProgress}, nil)
    go func() {
        for envelope := range eventCh {
            if evt, ok := envelope.Event.(types.EventType); ok {
                switch e := evt.(type) {
                case *types.ProgressTextChunkEvent:
                    fmt.Print(e.Delta)
                case *types.ProgressToolStartEvent:
                    fmt.Printf("\n[Tool] %s\n", e.Call.Name)
                }
            }
        }
    }()

    // 5. Execute
    result, err := ag.Chat(context.Background(), "Create a hello.txt file with 'Hello World'")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("\n\nFinal Result: %s\n", result.Text)
}
```

## 🏗️ Architecture

### System Overview

![Aster Architecture](https://raw.githubusercontent.com/astercloud/aster/main/docs/public/images/architecture-overview.svg)

### Middleware Onion Model

![Middleware Onion](https://raw.githubusercontent.com/astercloud/aster/main/docs/public/images/middleware-onion.svg)

The middleware architecture processes each request/response through multiple layers:

- Higher priority middleware sits in outer layers
- Handles requests first, processes responses last
- Clean separation of concerns and easy extensibility

## 🌐 Multi-Language Execution

### Python Script Example

```python
# agent can execute this directly
import pandas as pd

data = pd.read_csv('data.csv')
result = data.groupby('category').sum()
print(result.to_json())
```

### TypeScript Example

```typescript
// Native TypeScript execution
interface User {
  name: string;
  email: string;
}

const users: User[] = await fetch("/api/users").then((r) => r.json());
console.log(users.map((u) => u.name));
```

### Bash Example

```bash
# System operations
find . -name "*.log" -mtime +7 -delete
docker ps | grep running
```

## 📊 Project Status

🚀 **Alpha Release** - Core features complete

### Completed Phases

✅ **Phase 1**: Foundation (Event system, Sandbox abstraction, Storage)
✅ **Phase 2**: Agent Runtime (Message processing, Tool system, Streaming)
✅ **Phase 3**: Cloud Integration (MCP, Aliyun, Volcengine)
✅ **Phase 4**: Multi-Agent (Pool, Room, Scheduler, Permissions)
✅ **Phase 5**: MCP Support (Protocol, Servers, Tools)
✅ **Phase 6**: Advanced Features (Commands, Skills, Middleware, Multi-provider)
✅ **Phase 7**: ADK Alignment (Streaming, OpenTelemetry, Persistence, Workflows)

**Current Stats**:

- ~18,000+ LOC
- 25+ new modules
- 80%+ test coverage
- ✅ **Production Ready**

## 🎓 Google Context Engineering Standard

Aster fully implements the **Google Context Engineering** whitepaper's 8 core capabilities:

| Capability                | Status | Description                                       |
| ------------------------- | ------ | ------------------------------------------------- |
| Sessions & Memory         | ✅     | Three-layer memory system (Text/Working/Semantic) |
| Memory Provenance         | ✅     | Source tracking with confidence scoring           |
| Memory Consolidation      | ✅     | LLM-driven intelligent memory merging             |
| PII Auto-Redaction        | ✅     | Automated privacy data protection                 |
| Event-Driven Architecture | ✅     | Progress/Control/Monitor tri-channel              |
| Streaming & Backpressure  | ✅     | iter.Seq2 streaming interface                     |
| Multi-Agent Orchestration | ✅     | Pool/Room/Workflow patterns                       |
| Observability             | ✅     | Complete OpenTelemetry integration                |

**100% Implementation** - First Go framework to fully implement the standard.

## 🙏 Acknowledgments

Aster builds upon the excellent work of the open-source community:

### Frameworks

- **[LangChain](https://github.com/langchain-ai/langchain)**: Pioneering agent framework
- **[Google ADK](https://github.com/google/genkit)**: Enterprise-grade agent toolkit
- **[Claude Agent SDK](https://github.com/anthropics/anthropic-sdk-python)**: Computer Use & MCP reference

### Research

Special thanks to the **[Google Context Engineering Whitepaper](https://cloud.google.com/blog/products/ai-machine-learning/context-engineering-for-ai-agents)** for defining agent capabilities and best practices.

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

---

<p align="center">
  <strong>Let every agent shine in production</strong> ✨
</p>

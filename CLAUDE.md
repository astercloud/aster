# AI Agent

This file provides guidance to AI Agent when working with code in this repository.

## Essential Commands

### Building

```bash
# Build both main applications
go build -v ./cmd/aster
go build -v ./cmd/aster-server

# Build with version info
go build -ldflags "-X main.version=$(git describe --tags --always)" ./cmd/aster
```

### Testing

```bash
# Run all tests
go test -v ./pkg/...

# Run tests without race detection (CI default)
go test -v -short ./pkg/...

# Run specific package tests
go test -v ./pkg/agent
go test -v ./pkg/workflow
go test -v ./pkg/session

# Run integration tests (requires Docker)
go test -v ./test/integration/
```

### Linting

```bash
# Run golangci-lint (current CI configuration)
golangci-lint run --disable=errcheck --enable=govet,ineffassign,staticcheck,unused ./pkg/...

# Run with default settings (local development)
golangci-lint run ./pkg/...
```

### Code Quality

```bash
# Format code
go fmt ./...

# Vet code
go vet ./...

# Tidy dependencies
go mod tidy

# Download dependencies
go mod download
```

## Architecture Overview

### Core Architecture

Aster is an event-driven AI Agent framework built with Go, implementing the Google Context Engineering standards with an onion-model middleware system.

### Key Components

1. **Agent System** (`pkg/agent/`)

   - Core agent implementation with event-driven architecture
   - Three event channels: Progress (real-time output), Control (human interaction), Monitor (governance)
   - Dependencies injection pattern with Registry pattern for tools and templates

2. **Workflow Engine** (`pkg/workflow/`)

   - Sequential, Parallel, and Loop workflow agents
   - 8-step types with Router for dynamic routing
   - Stream-based execution with Go 1.23 iter.Seq2

3. **Memory Management** (`pkg/memory/`)

   - Three-tier system: Text Memory, Working Memory, Semantic Memory
   - Advanced features: Provenance (source tracking), Consolidation (LLM-driven merging), PII Auto-Redaction
   - Vector store integration with confidence scoring

4. **Middleware System** (`pkg/middleware/`)

   - Onion-model architecture with priority-based layers
   - Built-in middlewares: filesystem, summarization, agent memory, working memory
   - Custom middleware support via WrapModelCall/WrapToolCall interfaces

5. **Tools System** (`pkg/tools/`)

   - Registry pattern for tool discovery and management
   - Built-in tools: filesystem operations, bash execution, HTTP requests, web search, todo management
   - MCP (Model Context Protocol) support for external tool integration
   - Long-running tools with async execution and progress tracking

6. **Session & Persistence** (`pkg/session/`)

   - PostgreSQL and MySQL support with JSONB/JSON columns
   - Event sourcing with append-only storage
   - 7-point recovery mechanism for fault tolerance

7. **Multi-Agent Collaboration** (`pkg/stars/`)

   - Stars pattern for agent collaboration
   - Scheduler for intelligent task distribution
   - Permission management system

8. **Sandbox Security** (`pkg/sandbox/`)
   - Local, Docker, Aliyun AgentBay, and Volcengine sandbox backends
   - Process isolation and resource limits
   - Security audit logging

### Event-Driven Architecture

The core is built around three distinct event channels:

- **Progress Channel**: Stream text chunks, tool execution progress (UI/Chat consumers)
- **Control Channel**: Tool approval requests, user confirmations (Human-in-the-loop systems)
- **Monitor Channel**: Governance events, errors, audit logs (Monitoring/Compliance systems)

### Provider Abstraction

Multi-provider LLM support through a unified interface:

- Anthropic Claude (primary)
- OpenAI GPT
- DeepSeek
- Other OpenAI-compatible providers

### Data Flow

1. Agent receives input → Subscribe to event channels
2. Request flows through middleware onion (pre-processing)
3. LLM provider processes with tool calling
4. Tool execution in secure sandbox
5. Response flows back through middleware (post-processing)
6. Events published to appropriate channels
7. Session persistence stores conversation history

## Development Guidelines

### Project Structure

- `cmd/`: Main applications (aster, aster-server)
- `pkg/`: Core library packages
- `examples/`: Usage examples and demos
- `docs/`: Documentation and architecture guides
- `test/`: Integration and performance tests

### Dependencies Pattern

Use dependency injection extensively:

```go
deps := &agent.Dependencies{
    Store:            store,
    SandboxFactory:   sandboxFactory,
    ToolRegistry:     toolRegistry,
    ProviderFactory:  providerFactory,
    TemplateRegistry: templateRegistry,
}
```

### Testing Strategy

- Unit tests: `*_test.go` files alongside source code
- Integration tests: `test/integration/` with Docker containers
- CI uses `-short` flag to skip long-running tests
- Race detection disabled in CI for performance

### Performance Considerations

- Streaming responses use iter.Seq2 for O(1) memory
- Long-running tools support async execution
- Session persistence optimized with JSON columns
- Resource limits enforced via GOMAXPROCS and GOMEMLIMIT

### Security Model

- All code execution in sandboxed environments
- Tool-level permissions with allow/deny/ask policies
- PII auto-redaction for sensitive data
- Complete audit trail for tool executions

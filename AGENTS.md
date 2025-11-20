# Repository Guidelines

## Project Structure & Module Organization
- `cmd/`: CLI/server entrypoints (`cmd/aster`, `cmd/aster-server`, `cmd/aster/mcp_serve.go`).
- `pkg/`: Core libraries (agent, provider, middleware, memory, workflow, tools, session, guardrails, vector, security).
- `pkg/knowledge/`: 高级知识管理器（PII/审计/谱系等）+ `core/` 轻量 RAG 管线；`tools/knowledge/` 提供可选知识工具工厂（默认不注册）。
- `server/`: HTTP/WebSocket handlers, middleware, rate limiters.
- `examples/`: Runnable demos (memory, workflow, MCP, server-http, cloud-sandbox).
- `client-sdks/client-js/`: TypeScript client and examples.
- `docs/`: Public docs assets; `aster.yaml` for config presets.

## Build, Test, and Development Commands
- `go test ./...` — run unit tests across modules.
- `go test ./... -run TestName` — focus on a specific test.
- `go vet ./...` — static checks (use before PRs).
- `go run ./cmd/aster --help` — inspect CLI options; `go run ./cmd/aster-server` to start HTTP server locally.
- `npm test` (in `client-sdks/client-js`) — JS client tests.

## Coding Style & Naming Conventions
- Go 1.21+ idioms; format with `gofmt` (or `goimports` if available).
- Package and file naming: lower_snake for files, short package names; tests end with `_test.go`.
- Exported symbols need concise doc comments; prefer clear error wrapping (`fmt.Errorf("context: %w", err)`).
- Keep middleware/tool names consistent with existing registries (`NewXxxMiddleware`, `RegisterAll`).

## Testing Guidelines
- Use Go’s `testing` package; table-driven tests preferred.
- Place tests alongside code in `_test.go`; name funcs `TestPackage_Feature`.
- Coverage check: `go test ./... -cover`.
- Integration demos live in `examples/`; keep them runnable without secret keys or guard with env checks.

## Commit & Pull Request Guidelines
- Commits: short, imperative summaries (e.g., “Add structured output middleware”); group related changes.
- Ensure `go test ./...` and `go vet ./...` are clean before pushing.
- PRs: include purpose, scope, and testing evidence; link issues when relevant; add screenshots/log snippets for middleware/router changes.
- Avoid large mixed PRs; keep feature and refactor separated. Document config/env expectations (API keys, DB DSNs) in the PR body.

## Security & Configuration Tips
- Never hardcode API keys or provider tokens; read from env (e.g., `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`).
- For sandboxed tooling, prefer `pkg/sandbox` factories and avoid writing outside configured workdirs.
- Guardrails (PII, prompt injection, moderation) live in `pkg/guardrails`/`pkg/security`; enable them in middleware stacks for new surfaces.

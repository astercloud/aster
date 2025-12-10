# Desktop Integration Guide

Aster supports three popular desktop frameworks: **Wails**, **Tauri**, and **Electron**.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Frontend (Web UI)                        │
│                 Vue/React/Svelte/etc.                       │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                   Desktop Bridge                            │
│  ┌─────────────┬─────────────┬────────────┬───────────────┐ │
│  │   Wails    │    Tauri    │  Electron  │     Web       │ │
│  │ (Go Bind)  │   (HTTP)    │   (HTTP)   │   (HTTP)      │ │
│  └─────────────┴─────────────┴────────────┴───────────────┘ │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                     Aster Core                              │
│  ┌─────────────┬─────────────┬────────────┬───────────────┐ │
│  │   Agent    │  Permission │   Tools    │   Session     │ │
│  │   System   │   System    │  Registry  │    Store      │ │
│  └─────────────┴─────────────┴────────────┴───────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## Quick Start

### 1. Wails (Recommended for Go developers)

Wails provides the most seamless integration since it uses direct Go function binding.

```go
// main.go
package main

import (
    "github.com/astercloud/aster/pkg/desktop"
    "github.com/wailsapp/wails/v2"
)

func main() {
    app, _ := desktop.NewApp(&desktop.AppConfig{
        Framework: desktop.FrameworkWails,
    })
    
    // ... create and register agent ...
    
    wails.Run(&options.App{
        Title:  "Aster Desktop",
        Width:  1024,
        Height: 768,
        Bind: []interface{}{
            app.Bridge(),
        },
    })
}
```

```javascript
// frontend/src/api.js
export async function chat(agentId, message) {
    return await window.go.desktop.WailsBridge.Chat(agentId, message);
}

export async function cancel(agentId) {
    return await window.go.desktop.WailsBridge.Cancel(agentId);
}

export async function approve(agentId, callId, decision, note) {
    return await window.go.desktop.WailsBridge.Approve(agentId, callId, decision, note);
}
```

### 2. Tauri (Recommended for Rust developers)

Tauri uses a local HTTP server for communication.

```rust
// src-tauri/main.rs
use std::process::Command;

fn main() {
    // Start Aster backend
    let _child = Command::new("./aster-desktop")
        .args(["--framework", "tauri", "--port", "9528"])
        .spawn()
        .expect("Failed to start Aster");
    
    tauri::Builder::default()
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
```

```javascript
// src/api.js
const API_URL = 'http://127.0.0.1:9528';

export async function chat(agentId, message) {
    const res = await fetch(`${API_URL}/api/chat`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ agent_id: agentId, message }),
    });
    return res.json();
}

// SSE for streaming events
export function subscribeEvents(onEvent) {
    const eventSource = new EventSource(`${API_URL}/api/events`);
    
    eventSource.addEventListener('text_chunk', (e) => {
        const data = JSON.parse(e.data);
        onEvent('text_chunk', data);
    });
    
    eventSource.addEventListener('tool_start', (e) => {
        const data = JSON.parse(e.data);
        onEvent('tool_start', data);
    });
    
    eventSource.addEventListener('tool_end', (e) => {
        const data = JSON.parse(e.data);
        onEvent('tool_end', data);
    });
    
    eventSource.addEventListener('approval_required', (e) => {
        const data = JSON.parse(e.data);
        onEvent('approval_required', data);
    });
    
    return () => eventSource.close();
}
```

### 3. Electron

Electron also uses HTTP server communication.

```javascript
// main.js
const { app, BrowserWindow } = require('electron');
const { spawn } = require('child_process');
const path = require('path');

let asterProcess;

function startAster() {
    const asterPath = path.join(__dirname, 'aster-desktop');
    asterProcess = spawn(asterPath, ['--framework', 'electron', '--port', '9527']);
    
    asterProcess.stdout.on('data', (data) => {
        console.log(`Aster: ${data}`);
    });
}

app.whenReady().then(() => {
    startAster();
    
    const win = new BrowserWindow({
        width: 1024,
        height: 768,
        webPreferences: {
            preload: path.join(__dirname, 'preload.js'),
        },
    });
    
    win.loadFile('index.html');
});

app.on('will-quit', () => {
    if (asterProcess) {
        asterProcess.kill();
    }
});
```

```javascript
// preload.js
const { contextBridge } = require('electron');

const API_URL = 'http://127.0.0.1:9527';

contextBridge.exposeInMainWorld('aster', {
    chat: async (agentId, message) => {
        const res = await fetch(`${API_URL}/api/chat`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ agent_id: agentId, message }),
        });
        return res.json();
    },
    
    cancel: async (agentId) => {
        const res = await fetch(`${API_URL}/api/cancel`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ agent_id: agentId }),
        });
        return res.json();
    },
    
    approve: async (agentId, callId, decision, note) => {
        const res = await fetch(`${API_URL}/api/approve`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                agent_id: agentId,
                call_id: callId,
                decision,
                note,
            }),
        });
        return res.json();
    },
    
    subscribeEvents: (onEvent) => {
        const eventSource = new EventSource(`${API_URL}/api/events`);
        
        ['text_chunk', 'tool_start', 'tool_end', 'approval_required', 'error', 'done']
            .forEach(type => {
                eventSource.addEventListener(type, (e) => {
                    onEvent(type, JSON.parse(e.data));
                });
            });
        
        return () => eventSource.close();
    },
});
```

## API Reference

### REST Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/chat` | Send a message to the agent |
| POST | `/api/cancel` | Cancel current operation |
| POST | `/api/approve` | Respond to permission request |
| GET | `/api/status?agent_id=xxx` | Get agent status |
| GET | `/api/history?agent_id=xxx` | Get conversation history |
| DELETE | `/api/history?agent_id=xxx` | Clear conversation history |
| GET | `/api/config` | Get configuration |
| POST | `/api/config` | Set configuration |
| GET | `/api/agents` | List all agents |
| GET | `/api/events` | SSE event stream |

### Request/Response Format

#### Chat Request
```json
{
    "agent_id": "agent-123",
    "message": "Hello, world!"
}
```

#### Approval Request
```json
{
    "agent_id": "agent-123",
    "call_id": "call-456",
    "decision": "allow",  // "allow", "deny", "allow_always", "deny_always"
    "note": "User approved"
}
```

### SSE Event Types

| Event | Description |
|-------|-------------|
| `connected` | Connection established |
| `text_chunk` | Streaming text content |
| `tool_start` | Tool execution started |
| `tool_end` | Tool execution ended |
| `tool_progress` | Tool execution progress |
| `approval_required` | Permission request |
| `error` | Error occurred |
| `done` | Response complete |
| `status_change` | Agent status changed |

## Permission Modes

The permission system supports three modes:

1. **auto_approve** - All tool executions are automatically approved
2. **smart_approve** - Low-risk tools auto-approved, high-risk requires confirmation
3. **always_ask** - All tool executions require user confirmation

```go
app, _ := desktop.NewApp(&desktop.AppConfig{
    PermissionMode: permission.ModeSmartApprove,
})
```

## Building for Distribution

### Wails
```bash
wails build -platform darwin/amd64
wails build -platform windows/amd64
wails build -platform linux/amd64
```

### Tauri
```bash
cargo tauri build
```

### Electron
```bash
npm run make
```

## Cross-Platform Paths

Aster uses platform-specific paths for configuration and data:

| Platform | Config | Data | Logs |
|----------|--------|------|------|
| macOS | `~/Library/Application Support/aster` | Same | Same |
| Linux | `~/.config/aster` | `~/.local/share/aster` | `~/.local/state/aster/logs` |
| Windows | `%APPDATA%\aster` | Same | Same |

These paths are automatically managed by `pkg/config/paths.go`.

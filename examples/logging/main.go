package main

import (
	"context"
	"fmt"
	"time"

	"github.com/astercloud/aster/pkg/logging"
)

// 本示例演示如何使用 pkg/logging 提供的 Logger 和 transports:
// - StdoutTransport: 将日志以 JSON 行写到 stdout
// - FileTransport: 将日志以 JSON 行写到指定文件
func main() {
	ctx := context.Background()

	// 1. 创建 stdout logger
	stdLogger := logging.NewLogger(logging.LevelDebug, logging.NewStdoutTransport())

	stdLogger.Info(ctx, "server.started", map[string]interface{}{
		"addr": ":8080",
		"env":  "dev",
	})

	// 2. 创建 file logger
	fileTransport, err := logging.NewFileTransport("./logs/app.log")
	if err != nil {
		panic(fmt.Sprintf("failed to create file transport: %v", err))
	}
	defer func() { _ = fileTransport.Close() }()

	fileLogger := logging.NewLogger(logging.LevelInfo, fileTransport)

	fileLogger.Info(ctx, "agent.chat.started", map[string]interface{}{
		"agent_id":   "agt-demo",
		"user_id":    "alice",
		"templateID": "assistant",
	})

	// 模拟一次工具调用
	start := time.Now()
	time.Sleep(150 * time.Millisecond)
	duration := time.Since(start)

	fileLogger.Info(ctx, "tool.call.completed", map[string]interface{}{
		"agent_id":  "agt-demo",
		"tool_name": "Read",
		"duration":  duration.Seconds(),
		"success":   true,
	})

	// 3. 使用全局 Default logger
	logging.Info(ctx, "request.completed", map[string]interface{}{
		"status":  "ok",
		"latency": 0.123,
	})

	// 刷新缓冲(如果有)
	logging.Flush(ctx)
	fileLogger.Flush(ctx)
	stdLogger.Flush(ctx)
}

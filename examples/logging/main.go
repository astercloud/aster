// Logging 演示日志系统的使用，包括 StdoutTransport 和 FileTransport
// 两种输出方式，支持 JSON 行格式日志输出。
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/astercloud/aster/pkg/logging"
)

func main() {
	ctx := context.Background()

	// 1. 创建 stdout logger
	stdLogger := logging.NewLogger(logging.LevelDebug, logging.NewStdoutTransport())

	stdLogger.Info(ctx, "server.started", map[string]any{
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

	fileLogger.Info(ctx, "agent.chat.started", map[string]any{
		"agent_id":   "agt-demo",
		"user_id":    "alice",
		"templateID": "assistant",
	})

	// 模拟一次工具调用
	start := time.Now()
	time.Sleep(150 * time.Millisecond)
	duration := time.Since(start)

	fileLogger.Info(ctx, "tool.call.completed", map[string]any{
		"agent_id":  "agt-demo",
		"tool_name": "Read",
		"duration":  duration.Seconds(),
		"success":   true,
	})

	// 3. 使用全局 Default logger
	logging.Info(ctx, "request.completed", map[string]any{
		"status":  "ok",
		"latency": 0.123,
	})

	// 刷新缓冲(如果有)
	logging.Flush(ctx)
	fileLogger.Flush(ctx)
	stdLogger.Flush(ctx)
}

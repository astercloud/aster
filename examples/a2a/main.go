// A2A 演示 Agent-to-Agent 协议实现，包括基于 JSON-RPC 的消息发送、
// 任务管理和对话历史追踪。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/astercloud/aster/pkg/a2a"
	"github.com/astercloud/aster/pkg/actor"
	pkgagent "github.com/astercloud/aster/pkg/agent"
	"github.com/astercloud/aster/pkg/types"
)

// SimpleAgent 简单的演示 Agent Actor
type SimpleAgent struct{}

func (a *SimpleAgent) Receive(ctx *actor.Context, msg actor.Message) {
	switch m := msg.(type) {
	case *pkgagent.ChatMsg:
		// 模拟处理延迟
		time.Sleep(100 * time.Millisecond)

		// 生成响应
		response := fmt.Sprintf("你好!我收到了你的消息: %s", m.Text)

		result := &pkgagent.ChatResultMsg{
			Result: &types.CompleteResult{
				Text:   response,
				Status: "ok",
			},
		}

		// 发送响应
		select {
		case m.ReplyTo <- result:
		case <-time.After(time.Second):
			fmt.Println("发送响应超时")
		}
	}
}

func main() {
	fmt.Println("=== A2A 协议示例 ===")
	fmt.Println()

	// 1. 创建 Actor System
	system := actor.NewSystem("a2a-demo")
	defer system.Shutdown()
	fmt.Println("✅ Actor System 已创建")
	fmt.Println()

	// 2. 创建并注册 Agent Actor
	agent := &SimpleAgent{}
	system.Spawn(agent, "demo-agent")
	fmt.Println("✅ Agent Actor 已注册: demo-agent")
	fmt.Println()

	// 3. 创建 A2A Server
	taskStore := a2a.NewInMemoryTaskStore()
	a2aServer := a2a.NewServer(system, taskStore)
	fmt.Println("✅ A2A Server 已创建")
	fmt.Println()

	// 4. 获取 Agent Card
	fmt.Println("--- 步骤 1: 获取 Agent Card ---")
	card, err := a2aServer.GetAgentCard("demo-agent")
	if err != nil {
		log.Fatalf("获取 Agent Card 失败: %v", err)
	}

	cardJSON, _ := json.MarshalIndent(card, "", "  ")
	fmt.Printf("Agent Card:\n%s\n\n", cardJSON)

	// 5. 发送消息 (message/send)
	fmt.Println("--- 步骤 2: 发送消息 (message/send) ---")
	params := a2a.MessageSendParams{
		Message: a2a.Message{
			MessageID: "msg-001",
			Role:      "user",
			Parts: []a2a.Part{
				{Kind: "text", Text: "你好! 这是一个 A2A 协议测试消息。"},
			},
			Kind: "message",
		},
		ContextID: "context-001",
	}
	paramsJSON, _ := json.Marshal(params)

	req := &a2a.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      "req-001",
		Method:  "message/send",
		Params:  json.RawMessage(paramsJSON),
	}

	ctx := context.Background()
	resp := a2aServer.HandleRequest(ctx, "demo-agent", req)

	if resp.Error != nil {
		log.Fatalf("消息发送失败: code=%d, message=%s", resp.Error.Code, resp.Error.Message)
	}

	var sendResult a2a.MessageSendResult
	resultBytes, _ := json.Marshal(resp.Result)
	if err := json.Unmarshal(resultBytes, &sendResult); err != nil {
		log.Fatalf("解析结果失败: %v", err)
	}
	fmt.Printf("✅ 任务已创建, TaskID: %s\n\n", sendResult.TaskID)

	// 6. 获取任务状态 (tasks/get)
	fmt.Println("--- 步骤 3: 获取任务状态 (tasks/get) ---")
	getParams := a2a.TasksGetParams(sendResult)
	getParamsJSON, _ := json.Marshal(getParams)

	getReq := &a2a.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      "req-002",
		Method:  "tasks/get",
		Params:  json.RawMessage(getParamsJSON),
	}

	getResp := a2aServer.HandleRequest(ctx, "demo-agent", getReq)

	if getResp.Error != nil {
		log.Fatalf("获取任务失败: code=%d, message=%s", getResp.Error.Code, getResp.Error.Message)
	}

	var getResult a2a.TasksGetResult
	getResultBytes, _ := json.Marshal(getResp.Result)
	if err := json.Unmarshal(getResultBytes, &getResult); err != nil {
		log.Fatalf("解析任务结果失败: %v", err)
	}

	fmt.Printf("任务状态: %s\n", getResult.Task.Status.State)
	fmt.Printf("消息数量: %d\n\n", len(getResult.Task.History))

	// 7. 显示对话历史
	fmt.Println("--- 步骤 4: 查看对话历史 ---")
	for i, msg := range getResult.Task.History {
		fmt.Printf("\n消息 #%d [%s]:\n", i+1, msg.Role)
		for _, part := range msg.Parts {
			if part.Kind == "text" {
				fmt.Printf("  %s\n", part.Text)
			}
		}
	}

	// 8. 演示任务取消
	fmt.Println("\n\n--- 步骤 5: 测试任务取消 (tasks/cancel) ---")

	// 先创建一个新任务
	params2 := a2a.MessageSendParams{
		Message: a2a.Message{
			MessageID: "msg-002",
			Role:      "user",
			Parts:     []a2a.Part{{Kind: "text", Text: "第二条消息"}},
			Kind:      "message",
		},
		ContextID: "context-001",
	}
	params2JSON, _ := json.Marshal(params2)

	req2 := &a2a.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      "req-003",
		Method:  "message/send",
		Params:  json.RawMessage(params2JSON),
	}

	resp2 := a2aServer.HandleRequest(ctx, "demo-agent", req2)
	var sendResult2 a2a.MessageSendResult
	resultBytes2, _ := json.Marshal(resp2.Result)
	if err := json.Unmarshal(resultBytes2, &sendResult2); err != nil {
		log.Fatalf("解析结果失败: %v", err)
	}

	// 取消任务
	cancelParams := a2a.TasksCancelParams(sendResult2)
	cancelParamsJSON, _ := json.Marshal(cancelParams)

	cancelReq := &a2a.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      "req-004",
		Method:  "tasks/cancel",
		Params:  json.RawMessage(cancelParamsJSON),
	}

	cancelResp := a2aServer.HandleRequest(ctx, "demo-agent", cancelReq)

	if cancelResp.Error != nil {
		log.Fatalf("取消任务失败: %s", cancelResp.Error.Message)
	}

	fmt.Println("✅ 任务已取消")

	fmt.Println("\n=== 示例完成 ===")
}

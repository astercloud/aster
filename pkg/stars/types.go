package stars

import (
	"time"
)

// Role Agent 在群星中的角色
type Role string

const (
	// RoleLeader 领导者 - 负责协调和决策
	RoleLeader Role = "leader"
	// RoleWorker 工作者 - 负责执行任务
	RoleWorker Role = "worker"
)

// Member 群星成员
type Member struct {
	AgentID string   // Agent ID
	Role    Role     // 角色
	Tags    []string // 能力标签（可选）
}

// Message 群星消息
type Message struct {
	From string    // 发送者 Agent ID
	To   string    // 接收者 Agent ID（空表示广播）
	Text string    // 消息内容
	Time time.Time // 发送时间
}

// Event 群星事件
type Event struct {
	AgentID string    // 产生事件的 Agent ID
	Type    string    // 事件类型
	Content string    // 事件内容
	Time    time.Time // 事件时间
}

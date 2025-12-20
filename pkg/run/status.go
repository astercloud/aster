package run

// Status 运行状态
type Status string

const (
	// StatusPending 待运行
	StatusPending Status = "PENDING"

	// StatusRunning 运行中
	StatusRunning Status = "RUNNING"

	// StatusCompleted 完成
	StatusCompleted Status = "COMPLETED"

	// StatusPaused 暂停
	StatusPaused Status = "PAUSED"

	// StatusCancelled 取消
	StatusCancelled Status = "CANCELED"

	// StatusError 错误
	StatusError Status = "ERROR"

	// StatusTimeout 超时
	StatusTimeout Status = "TIMEOUT"
)

// IsTerminal 判断是否为终止状态
func (s Status) IsTerminal() bool {
	return s == StatusCompleted || s == StatusCancelled || s == StatusError || s == StatusTimeout
}

// IsActive 判断是否为活跃状态
func (s Status) IsActive() bool {
	return s == StatusRunning || s == StatusPending
}

// String 返回字符串表示
func (s Status) String() string {
	return string(s)
}

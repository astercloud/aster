package strategy

import (
	"time"

	"github.com/astercloud/aster/pkg/knowledge"
)

// InMemoryAudit 简单内存审计，记录上限可配；实现 knowledge.AuditStrategy。
type InMemoryAudit struct {
	Limit int
}

func (a *InMemoryAudit) Record(action, userID, itemID, details string) {
	// 可扩展：写入日志或外部存储。当前保持无状态占位，防止 OOM。
	_ = AuditRecord{
		Timestamp: time.Now(),
		Action:    action,
		UserID:    userID,
		ItemID:    itemID,
		Details:   details,
	}
}

// AuditRecord 用于扩展持久化或排错场景。
type AuditRecord = knowledge.AuditRecord

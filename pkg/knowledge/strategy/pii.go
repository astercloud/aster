package strategy

import (
	"github.com/astercloud/aster/pkg/knowledge"
	"github.com/astercloud/aster/pkg/security"
)

// Redaction 基于正则的默认 PII 脱敏策略，兼容 ManagerConfig.PIIStrategy。
type Redaction struct {
	redactor security.ContentRedactor
}

func NewRedaction(detector security.PIIDetector) *Redaction {
	if detector == nil {
		detector = security.NewRegexPIIDetector()
	}
	return &Redaction{
		redactor: security.NewPIIRedactor(detector),
	}
}

func (r *Redaction) Sanitize(item *knowledge.KnowledgeItem) *knowledge.KnowledgeItem {
	if r == nil || r.redactor == nil || item == nil {
		return item
	}
	c := *item
	c.Content = r.redactor.Redact(item.Content)
	c.Description = r.redactor.Redact(item.Description)
	c.Title = r.redactor.Redact(item.Title)
	return &c
}

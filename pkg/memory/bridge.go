package memory

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"

	"github.com/astercloud/aster/pkg/session"
)

// LongTermBridge 提供从 Session 短期记忆保存到长期语义记忆的辅助方法。
// 这是一个轻量的工具类型，不参与核心运行时依赖注入。
type LongTermBridge struct {
	Sessions       session.Service
	SemanticMemory *SemanticMemory
}

// LongTermBridgeConfig 配置 Bridge 的行为。
type LongTermBridgeConfig struct {
	// MinTokens 用于过滤过短的内容（粗略按单词数统计），<=0 表示不做限制。
	MinTokens int
}

// SaveSessionToSemanticMemory 从指定 Session 中抽取对话内容，并写入语义记忆。
//
// 约定：
// - 将所有 user/assistant 消息拼接为一个大文本，适合用于知识性长记忆；
// - 调用方通过 scopeMeta 控制命名空间（user_id/project_id/resource_id 等）；
// - 具体要保存哪些 Session 由上层业务决定（教学会话、设置会话等）。
func (b *LongTermBridge) SaveSessionToSemanticMemory(
	ctx context.Context,
	appName string,
	userID string,
	sessionID string,
	scopeMeta map[string]any,
	cfg *LongTermBridgeConfig,
) error {
	if b == nil || b.Sessions == nil || b.SemanticMemory == nil || !b.SemanticMemory.Enabled() {
		return errors.New("long-term bridge is not properly configured")
	}

	// 加载 Session 事件
	sess, err := b.Sessions.Get(ctx, &session.GetRequest{
		AppName:   appName,
		UserID:    userID,
		SessionID: sessionID,
	})
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}
	if sess == nil {
		return errors.New("session not found")
	}

	if sess.Events() == nil || sess.Events().Len() == 0 {
		return errors.New("session has no events")
	}

	events := sess.Events()
	var lines []string
	for ev := range events.All() {
		if ev == nil {
			continue
		}
		role := string(ev.Content.Role)
		text := strings.TrimSpace(ev.Content.Content)
		if text == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: %s", role, text))
	}

	if len(lines) == 0 {
		return errors.New("no textual content to save")
	}

	joined := strings.Join(lines, "\n")

	if cfg != nil && cfg.MinTokens > 0 {
		if tokenCount(joined) < cfg.MinTokens {
			return errors.New("session content too short, skip saving")
		}
	}

	// 构造 docID：app/user/session 的组合，保证全局唯一性
	docID := fmt.Sprintf("%s/%s/%s", appName, userID, sessionID)

	meta := make(map[string]any, len(scopeMeta)+2)
	maps.Copy(meta, scopeMeta)
	meta["app_name"] = appName
	meta["session_id"] = sessionID

	return b.SemanticMemory.Index(ctx, docID, joined, meta)
}

// tokenCount 通过空格粗略估算 token 数量，用于简单过滤。
func tokenCount(text string) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}
	return len(strings.Fields(text))
}

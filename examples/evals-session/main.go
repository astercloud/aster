// EvalsSession 演示将 Session 事件与评估系统结合，通过
// BuildTextEvalInputFromEvents 转换会话事件并进行关键词覆盖率和
// 词汇相似度评估。
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/astercloud/aster/pkg/evals"
	"github.com/astercloud/aster/pkg/session"
	"github.com/astercloud/aster/pkg/types"
)

func main() {
	ctx := context.Background()

	// 1. 构造一个模拟会话事件列表
	events := []session.Event{
		{
			ID:        "evt1",
			Timestamp: time.Now(),
			Author:    "user",
			Content: types.Message{
				Role:    types.MessageRoleUser,
				Content: "What is the capital of France?",
			},
		},
		{
			ID:        "evt2",
			Timestamp: time.Now(),
			Author:    "assistant",
			Content: types.Message{
				Role:    types.MessageRoleAssistant,
				Content: "Paris is the capital of France, located in Europe.",
			},
		},
	}

	// 2. 将事件转换为 TextEvalInput
	textInput := evals.BuildTextEvalInputFromEvents(events)
	textInput.Reference = "Paris is the capital city of France, a country in Europe."

	fmt.Println("Answer:", textInput.Answer)
	fmt.Println("Context:", textInput.Context)
	fmt.Println()

	// 3. 关键词覆盖率评估
	kwScorer := evals.NewKeywordCoverageScorer(evals.KeywordCoverageConfig{
		Keywords:        []string{"paris", "capital", "france", "europe"},
		CaseInsensitive: true,
	})

	kwScore, _ := kwScorer.Score(ctx, textInput)
	fmt.Println("=== Keyword Coverage (session) ===")
	fmt.Printf("Score: %.2f\n", kwScore.Value)
	fmt.Printf("Matched: %v\n", kwScore.Details["matched"])
	fmt.Printf("Unmatched: %v\n", kwScore.Details["unmatched"])
	fmt.Println()

	// 4. 词汇相似度评估
	simScorer := evals.NewLexicalSimilarityScorer(evals.LexicalSimilarityConfig{MinTokenLength: 2})
	simScore, _ := simScorer.Score(ctx, textInput)

	fmt.Println("=== Lexical Similarity (session) ===")
	fmt.Printf("Score: %.2f\n", simScore.Value)
	fmt.Printf("Details: %+v\n", simScore.Details)
}

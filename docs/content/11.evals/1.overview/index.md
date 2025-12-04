---
title: 评估系统快速开始
description: 使用 pkg/evals、HTTP API 和 CLI 对模型输出进行本地评估
navigation: false
---

# 文本评估(Evals) 示例

本示例展示如何使用 `pkg/evals` 对模型输出进行简单的本地评估, 提供基础的 scorer 能力。

在当前版本中,你有三种使用方式:

- 直接在 Go 代码中使用 `pkg/evals`;
- 通过 HTTP 接口 `POST /v1/evals/text` 调用本地评估;
- 使用 `aster eval` CLI 在命令行中对文本进行评估。

当前提供的评估器不依赖外部 LLM,仅使用启发式算法:

- `KeywordCoverageScorer` – 关键词覆盖率
- `LexicalSimilarityScorer` – 词汇级相似度(Jaccard)

> 示例代码路径: `examples/evals/main.go`

## 1. 核心类型

```go
// pkg/evals/evals.go

// ScoreResult 评估结果
type ScoreResult struct {
    Name    string                 `json:"name"`
    Value   float64                `json:"value"`
    Details map[string]interface{} `json:"details,omitempty"`
}

// TextEvalInput 文本评估输入
type TextEvalInput struct {
    Answer    string   `json:"answer"`              // 模型输出
    Context   []string `json:"context,omitempty"`   // 可选上下文
    Reference string   `json:"reference,omitempty"` // 可选参考答案
}

// Scorer 通用评估器接口
type Scorer interface {
    Score(ctx context.Context, input *TextEvalInput) (*ScoreResult, error)
}
```

## 2. 关键词覆盖率: KeywordCoverageScorer

```go
// KeywordCoverageConfig 关键词覆盖率配置
type KeywordCoverageConfig struct {
    Keywords        []string
    CaseInsensitive bool
}

// NewKeywordCoverageScorer 创建关键词覆盖率评估器
func NewKeywordCoverageScorer(cfg KeywordCoverageConfig) *KeywordCoverageScorer
```

- 语义:
  - 给定一组关键词,检查答案中覆盖了多少。
  - 得分 = 覆盖到的关键词数量 / 总关键词数量,范围 [0,1]。

示例:

```go
input := &evals.TextEvalInput{
    Answer: "Paris is the capital of France. It is located in Europe.",
}

kwScorer := evals.NewKeywordCoverageScorer(evals.KeywordCoverageConfig{
    Keywords:        []string{"paris", "capital", "france", "europe"},
    CaseInsensitive: true,
})

kwScore, _ := kwScorer.Score(ctx, input)

fmt.Println("Score:", kwScore.Value)
fmt.Println("Matched:", kwScore.Details["matched"])
fmt.Println("Unmatched:", kwScore.Details["unmatched"])
```

## 3. 词汇相似度: LexicalSimilarityScorer

```go
// LexicalSimilarityConfig 配置
type LexicalSimilarityConfig struct {
    MinTokenLength int // 参与比较的最小 token 长度(默认 2)
}

func NewLexicalSimilarityScorer(cfg LexicalSimilarityConfig) *LexicalSimilarityScorer
```

- 语义:
  - 将 `Answer` 和 `Reference` 拆成词汇集合,计算简单的 Jaccard 相似度:
    - score = |A ∩ B| / |A ∪ B|
  - 得分范围 [0,1],1 表示完全相同(在本启发式定义下)。

示例:

```go
input := &evals.TextEvalInput{
    Answer:    "Paris is the capital of France.",
    Reference: "Paris is the capital city of France, a country in Europe.",
}

simScorer := evals.NewLexicalSimilarityScorer(evals.LexicalSimilarityConfig{
    MinTokenLength: 2,
})

simScore, _ := simScorer.Score(ctx, input)
fmt.Println("Lexical similarity:", simScore.Value)
fmt.Println("Details:", simScore.Details)
```

## 4. 完整示例: `examples/evals/main.go`

```go
func main() {
    ctx := context.Background()

    answer := "Paris is the capital of France. It is located in Europe."
    reference := "Paris is the capital city of France, a country in Europe."

    input := &evals.TextEvalInput{
        Answer:    answer,
        Reference: reference,
    }

    // 1. 关键词覆盖率
    kwScorer := evals.NewKeywordCoverageScorer(evals.KeywordCoverageConfig{
        Keywords:        []string{"paris", "capital", "france", "europe"},
        CaseInsensitive: true,
    })
    kwScore, _ := kwScorer.Score(ctx, input)

    fmt.Println("=== Keyword Coverage ===")
    fmt.Printf("Score: %.2f\n", kwScore.Value)
    fmt.Printf("Matched: %v\n", kwScore.Details["matched"])
    fmt.Printf("Unmatched: %v\n", kwScore.Details["unmatched"])

    // 2. 词汇相似度
    simScorer := evals.NewLexicalSimilarityScorer(evals.LexicalSimilarityConfig{MinTokenLength: 2})
    simScore, _ := simScorer.Score(ctx, input)

    fmt.Println("=== Lexical Similarity ===")
    fmt.Printf("Score: %.2f\n", simScore.Value)
    fmt.Printf("Details: %+v\n", simScore.Details)
}
```

运行:

```bash
cd examples
go run evals/main.go
```

## 5. 与 session 结合: 对会话进行评估

很多时候我们希望对**完整会话**中的最终回答进行评估。
`pkg/evals` 提供了一个辅助函数,可以从 `session.Event` 列表中构建 `TextEvalInput`:

```go
// pkg/evals/session.go

// BuildTextEvalInputFromEvents 根据一组 Session 事件构建 TextEvalInput。
//
// 约定:
// - 默认将最后一个 assistant 消息视为 Answer。
// - 将之前的 user / assistant 消息串联为 Context。
func BuildTextEvalInputFromEvents(events []session.Event) *TextEvalInput
```

示例: `examples/evals-session/main.go`

```go
events := []session.Event{
    {
        Author: "user",
        Content: types.Message{
            Role: types.MessageRoleUser,
            Content: []types.ContentBlock{
                &types.TextBlock{Text: "What is the capital of France?"},
            },
        },
    },
    {
        Author: "assistant",
        Content: types.Message{
            Role: types.MessageRoleAssistant,
            Content: []types.ContentBlock{
                &types.TextBlock{Text: "Paris is the capital of France, located in Europe."},
            },
        },
    },
}

txtInput := evals.BuildTextEvalInputFromEvents(events)
txtInput.Reference = "Paris is the capital city of France, a country in Europe."

kwScorer := evals.NewKeywordCoverageScorer(...)
kwScore, _ := kwScorer.Score(ctx, txtInput)

simScorer := evals.NewLexicalSimilarityScorer(...)
simScore, _ := simScorer.Score(ctx, txtInput)
```

运行:

```bash
cd examples
go run evals-session/main.go
```

你会看到针对 session 中最后一条 assistant 回复的关键词覆盖率和相似度得分。

## 6. 对比与下一步

- 当前能力:
  - 提供可组合的 scorer 接口, 作为评估 pipeline 的基础构件。
  - 不强绑定存储层, 如何落地结果交由使用方决定。
- 当前限制:
  - 目前仅提供本地启发式 scorer, 不依赖外部 LLM。
  - 后续可以在相同接口下添加:
    - 基于 Provider 的 LLM 评估器(如 faithfulness/toxicity)。
    - 与 `session.Service` 集成的批量回放评估示例。

推荐下一步:

- 在 CI 或离线脚本中,结合 `session` 或业务日志,对一批历史问答进行 evals,并把 `ScoreResult` 写入你自己的指标系统(如 Prometheus/ClickHouse/OLAP)。
- 对关键业务场景(问答正确性、安全性)建立稳定的“评估集”,用上述 scorer 做定期回归测试。

## 7. 通过 HTTP API 使用 evals

当你使用 `aster serve` 启动 HTTP Server 后,会自动暴露一个本地评估接口:

- 路径1: `POST /v1/evals/text`
- 路径2: `POST /v1/evals/session`
- 用途:
  - `text`: 对给定的 `answer` (和可选 `reference`、`keywords`) 进行本地评估。
  - `session`: 基于一段会话事件列表(`session.Event` JSON) 构建评估输入, 对最后一次 assistant 回复进行评估。

请求示例(`text`):

```bash
curl -X POST http://localhost:8080/v1/evals/text \
  -H "Content-Type: application/json" \
  -d '{
    "answer": "Paris is the capital of France. It is located in Europe.",
    "reference": "Paris is the capital city of France, a country in Europe.",
    "keywords": ["paris", "capital", "france", "europe"],
    "scorers": ["keyword_coverage", "lexical_similarity"]
  }'
```

响应示例:

```json
{
  "scores": [
    {
      "name": "keyword_coverage",
      "value": 1.0,
      "details": {
        "matched": ["paris", "capital", "france", "europe"],
        "unmatched": [],
        "total": 4
      }
    },
    {
      "name": "lexical_similarity",
      "value": 0.8,
      "details": {
        "intersection": 10,
        "union_size": 12
      }
    }
  ]
}
```

字段说明:

- `answer` (必填): 待评估的文本(通常是模型输出)。
- `reference` (可选): 参考答案,用于 `lexical_similarity`。
- `keywords` (可选): 关键词列表,用于 `keyword_coverage`。
- `scorers` (可选): 要启用的 scorer 名称列表,为空时默认启用
  `["keyword_coverage", "lexical_similarity"]`。

### Session 模式: `/v1/evals/session`

当你已经有一段会话事件(例如从 `pkg/session` 的 MySQL/Postgres 实现中读取出来), 可以使用 `/v1/evals/session`:

```bash
curl -X POST http://localhost:8080/v1/evals/session \
  -H "Content-Type: application/json" \
  -d '{
    "events": [
      {
        "id": "evt1",
        "timestamp": "2025-01-01T12:00:00Z",
        "agent_id": "demo-agent",
        "branch": "root",
        "author": "user",
        "content": {
          "role": "user",
          "content": "What is the capital of France?"
        }
      },
      {
        "id": "evt2",
        "timestamp": "2025-01-01T12:00:05Z",
        "agent_id": "demo-agent",
        "branch": "root",
        "author": "assistant",
        "content": {
          "role": "assistant",
          "content": "Paris is the capital of France, located in Europe."
        }
      }
    ],
    "reference": "Paris is the capital city of France, a country in Europe.",
    "keywords": ["paris", "capital", "france", "europe"],
    "scorers": ["keyword_coverage", "lexical_similarity"]
  }'
```

后端会:

1. 调用 `evals.BuildTextEvalInputFromEvents(events)` 生成 `TextEvalInput`, 默认:
   - 使用最后一条 assistant 消息作为 `Answer`;
   - 将之前的 user/assistant 消息拼成 `Context`。
2. 按 `scorers` 列表运行本地评估器, 返回与 `/v1/evals/text` 相同结构的 `scores` 数组。

## 8. 使用 aster CLI 进行评估

除了 HTTP API,你还可以直接在命令行使用 `aster eval` 进行本地评估,非常适合:

- 手工检查某个模型输出;
- 在 shell 脚本/CI 中对输出做简单打分。

### 基本用法

```bash
echo "Paris is the capital of France." \
  | aster eval \
      -reference "Paris is the capital city of France, a country in Europe." \
      -keywords "paris,capital,france,europe"
```

示例输出:

```text
keyword_coverage: 1.0000
  details: map[matched:[paris capital france europe] total:4 unmatched:[]]
lexical_similarity: 0.8000
  details: map[intersection:10 union_size:12]
```

### 常用参数

- `-answer string`
  直接通过参数提供待评估文本。未提供时会从 `stdin` 读取。

- `-reference string`
  参考答案,启用 `lexical_similarity` 时推荐设置。

- `-keywords string`
  逗号分隔的关键词列表,用于 `keyword_coverage` scorer。

- `-min-token-length int`
  词汇相似度中参与比较的最小 token 长度(默认 2)。

- `-no-keywords` / `-no-similarity`
  分别关闭关键词覆盖率/词汇相似度 scorer。

- `-json`
  以 JSON 格式输出评估结果,适合脚本消费:

  ```bash
  echo "text..." | aster eval -reference "ref..." -json
  ```

- `-file path`
  从 JSONL 文件中批量读取样本进行评估。文件中每行是一个 JSON 对象:

  ```jsonc
  {
    "answer": "Paris is the capital of France.",
    "reference": "Paris is the capital city of France, a country in Europe.",
    "keywords": ["paris", "capital", "france", "europe"],
  }
  ```

  使用示例:

  ```bash
  aster eval -file cases.jsonl -json
  ```

  输出中每条结果(在 JSON 模式下)包含:
  - `line` – 样本所在的行号
  - `answer` – 原始答案文本
  - `keywords` – 样本中使用的关键词
  - `scores` – 每个 scorer 的得分和详情

## 5. LLM-based Scorers

除了基于启发式算法的 Scorer，aster 还提供了 8 个基于 LLM 的 Scorer，用于更高级的评估任务。这些 Scorer 使用 LLM 作为 judge 来评估文本质量。

### 5.1 可用的 LLM-based Scorers

| Scorer              | 名称               | 评估内容                                     |
| ------------------- | ------------------ | -------------------------------------------- |
| `faithfulness`      | 忠实度评分器       | 答案是否忠实于提供的上下文，没有添加虚假信息 |
| `hallucination`     | 幻觉检测评分器     | 答案是否包含幻觉（虚假或无法验证的信息）     |
| `answer_relevancy`  | 答案相关性评分器   | 答案是否直接回答了问题                       |
| `context_relevancy` | 上下文相关性评分器 | 提供的上下文是否对回答问题有帮助             |
| `toxicity`          | 毒性检测评分器     | 文本是否包含有害或不当内容                   |
| `tone_consistency`  | 语气一致性评分器   | 文本的语气是否统一                           |
| `coherence`         | 连贯性评分器       | 文本的逻辑结构和流畅度                       |
| `completeness`      | 完整性评分器       | 答案是否全面回答了问题                       |

### 5.2 Go代码使用示例

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/astercloud/aster/pkg/evals"
    "github.com/astercloud/aster/pkg/provider"
    "github.com/astercloud/aster/pkg/types"
)

func main() {
    ctx := context.Background()

    // 1. 创建 LLM Provider（用于评分）
    cfg := &types.ModelConfig{
        Provider: "anthropic",
        Model:    "claude-sonnet-4-5",
        APIKey:   "your-api-key",
    }

    factory := provider.NewFactory()
    llmProvider, err := factory.Create(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer llmProvider.Close()

    // 2. 创建 LLM-based Scorers
    faithfulnessScorer := evals.NewFaithfulnessScorer(llmProvider)
    hallucinationScorer := evals.NewHallucinationScorer(llmProvider)
    relevancyScorer := evals.NewAnswerRelevancyScorer(llmProvider)

    // 3. 准备评估输入
    input := &evals.TextEvalInput{
        Answer: "巴黎是法国的首都，是一个美丽的城市。",
        Context: []string{"法国的首都是巴黎。"},
        Reference: "巴黎是法国的首都。",
    }

    // 4. 运行评估
    faithScore, err := faithfulnessScorer.Score(ctx, input)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("忠实度评分: %.2f\n", faithScore.Value)
    fmt.Printf("评分原因: %s\n", faithScore.Details["reason"])

    hallScore, err := hallucinationScorer.Score(ctx, input)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("幻觉检测评分: %.2f\n", hallScore.Value)
    fmt.Printf("评分原因: %s\n", hallScore.Details["reason"])
}
```

### 5.3 批量评估 API

对于需要评估多个测试用例的场景，可以使用批量评估 API。

#### HTTP API: POST /v1/evals/batch

**请求示例：**

```json
{
  "test_cases": [
    {
      "id": "case1",
      "answer": "巴黎是法国的首都。",
      "context": ["法国的首都是巴黎。"],
      "reference": "巴黎是法国的首都。"
    },
    {
      "id": "case2",
      "answer": "伦敦是英国的首都。",
      "context": ["英国的首都是伦敦。"],
      "reference": "伦敦是英国的首都。"
    }
  ],
  "scorers": [
    "keyword_coverage",
    "lexical_similarity",
    "faithfulness",
    "hallucination",
    "answer_relevancy"
  ],
  "concurrency": 5,
  "keywords": ["首都"],
  "provider_config": {
    "provider": "anthropic",
    "model": "claude-sonnet-4-5",
    "api_key": "your-api-key"
  }
}
```

**响应示例：**

```json
{
  "results": [
    {
      "test_case_id": "case1",
      "scores": [
        {
          "name": "keyword_coverage",
          "value": 1.0,
          "details": {"matched": ["首都"], "total": 1}
        },
        {
          "name": "faithfulness",
          "value": 1.0,
          "details": {"reason": "答案完全基于上下文，没有添加虚假信息"}
        }
      ],
      "duration_ms": 523
    },
    {
      "test_case_id": "case2",
      "scores": [...],
      "duration_ms": 510
    }
  ],
  "summary": {
    "total_cases": 2,
    "successful_cases": 2,
    "failed_cases": 0,
    "average_scores": {
      "keyword_coverage": 1.0,
      "faithfulness": 0.95,
      "hallucination": 0.98
    },
    "average_duration_ms": 516
  },
  "total_duration_ms": 1200
}
```

#### Go代码批量评估

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/astercloud/aster/pkg/evals"
    "github.com/astercloud/aster/pkg/provider"
    "github.com/astercloud/aster/pkg/types"
)

func main() {
    ctx := context.Background()

    // 1. 创建 Provider
    factory := provider.NewFactory()
    llmProvider, err := factory.Create(&types.ModelConfig{
        Provider: "anthropic",
        Model:    "claude-sonnet-4-5",
        APIKey:   "your-api-key",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer llmProvider.Close()

    // 2. 准备测试用例
    testCases := []*evals.BatchTestCase{
        {
            ID: "case1",
            Input: &evals.TextEvalInput{
                Answer:    "巴黎是法国的首都。",
                Context:   []string{"法国的首都是巴黎。"},
                Reference: "巴黎是法国的首都。",
            },
        },
        {
            ID: "case2",
            Input: &evals.TextEvalInput{
                Answer:    "伦敦是英国的首都。",
                Context:   []string{"英国的首都是伦敦。"},
                Reference: "伦敦是英国的首都。",
            },
        },
    }

    // 3. 创建 Scorers
    scorers := []evals.Scorer{
        evals.NewFaithfulnessScorer(llmProvider),
        evals.NewHallucinationScorer(llmProvider),
        evals.NewAnswerRelevancyScorer(llmProvider),
    }

    // 4. 运行批量评估（并发执行）
    result, err := evals.RunBatchConcurrent(ctx, testCases, scorers, 5)
    if err != nil {
        log.Fatal(err)
    }

    // 5. 输出结果
    fmt.Printf("总用例数: %d\n", result.Summary.TotalCases)
    fmt.Printf("成功: %d, 失败: %d\n",
        result.Summary.SuccessfulCases,
        result.Summary.FailedCases)

    fmt.Println("\n平均分数:")
    for name, score := range result.Summary.AverageScores {
        fmt.Printf("  %s: %.2f\n", name, score)
    }

    fmt.Printf("\n总执行时间: %v\n", result.TotalDuration)
    fmt.Printf("平均执行时间: %v\n", result.Summary.AverageDuration)
}
```

### 5.4 使用注意事项

1. **API成本**：LLM-based Scorer 会调用 LLM API，产生费用。建议在开发阶段使用采样评估。

2. **性能优化**：使用批量评估 API 并设置合理的并发数（`concurrency`）可以显著提升性能。

3. **Prompt自定义**：如需自定义评分 Prompt，可以使用 `NewLLMScorer()` 创建自定义 Scorer。

4. **模型选择**：推荐使用 Claude Sonnet 4.5 或 GPT-4 作为评分模型，获得更准确的评估结果。

5. **超时设置**：LLM-based评估可能需要较长时间，HTTP API 默认超时为5分钟。

## 6. 总结

aster 的 Evals 系统提供了完整的评估能力：

- **启发式 Scorer**：快速、免费、适合基础评估
- **LLM-based Scorer**：准确、智能、适合高级评估
- **批量评估**：高效、并发、适合大规模测试
- **多种接口**：Go API、HTTP API、CLI 三种使用方式

根据你的需求选择合适的评估方式，构建可靠的 AI 应用评估体系。

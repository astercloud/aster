package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// OpenAIEmbedder 基于 OpenAI 兼容接口的 Embedder 实现。
// 默认调用 POST {BaseURL}/v1/embeddings, 请求格式:
//
//	{ "input": [...], "model": "text-embedding-3-small" }
type OpenAIEmbedder struct {
	BaseURL string
	APIKey  string
	Model   string
	Client  *http.Client
}

// NewOpenAIEmbedder 创建 OpenAIEmbedder。
// baseURL 示例:
//   - "https://api.openai.com"
//   - "https://api.moonshot.cn" (如果兼容 OpenAI embeddings)
func NewOpenAIEmbedder(baseURL, apiKey, model string) *OpenAIEmbedder {
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	if model == "" {
		model = "text-embedding-3-small"
	}
	return &OpenAIEmbedder{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
		Client:  &http.Client{Timeout: 30 * time.Second},
	}
}

type openAIEmbeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type openAIEmbeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

// EmbedText 调用 OpenAI 兼容的 embeddings 接口。
func (e *OpenAIEmbedder) EmbedText(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}
	if e.APIKey == "" {
		return nil, errors.New("API key is required for OpenAIEmbedder")
	}

	reqBody := openAIEmbeddingRequest{
		Input: texts,
		Model: e.Model,
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.BaseURL+"/v1/embeddings", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.APIKey)

	resp, err := e.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("embeddings API error: %s", resp.Status)
	}

	var apiResp openAIEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(apiResp.Data) != len(texts) {
		return nil, fmt.Errorf("embedding response mismatch: got %d vectors, want %d", len(apiResp.Data), len(texts))
	}

	out := make([][]float32, len(apiResp.Data))
	for i, d := range apiResp.Data {
		vec := make([]float32, len(d.Embedding))
		for j, v := range d.Embedding {
			vec[j] = float32(v)
		}
		out[i] = vec
	}

	return out, nil
}

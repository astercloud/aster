package search

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

// BM25 实现 BM25 搜索算法
// BM25 (Best Matching 25) 是一种经典的文本相关性排序算法
type BM25 struct {
	// BM25 参数
	k1 float64 // 词频饱和参数，通常设为 1.2-2.0
	b  float64 // 文档长度归一化参数，通常设为 0.75

	// 文档集合
	documents     []Document
	docCount      int
	avgDocLength  float64
	docFrequency  map[string]int         // 词在多少文档中出现
	termFrequency map[int]map[string]int // 每个文档中每个词的频率
	docLengths    []int                  // 每个文档的长度（词数）
}

// Document 可搜索的文档
type Document struct {
	ID       string         // 文档唯一标识
	Content  string         // 文档内容
	Metadata map[string]any // 附加元数据
}

// SearchResult 搜索结果
type SearchResult struct {
	Document Document
	Score    float64
	Rank     int
}

// NewBM25 创建 BM25 搜索引擎
// k1: 词频饱和参数 (默认 1.5)
// b: 文档长度归一化参数 (默认 0.75)
func NewBM25(k1, b float64) *BM25 {
	if k1 <= 0 {
		k1 = 1.5
	}
	if b < 0 || b > 1 {
		b = 0.75
	}
	return &BM25{
		k1:            k1,
		b:             b,
		documents:     make([]Document, 0),
		docFrequency:  make(map[string]int),
		termFrequency: make(map[int]map[string]int),
		docLengths:    make([]int, 0),
	}
}

// NewBM25WithDefaults 使用默认参数创建 BM25 搜索引擎
func NewBM25WithDefaults() *BM25 {
	return NewBM25(1.5, 0.75)
}

// AddDocument 添加单个文档到索引
func (bm *BM25) AddDocument(doc Document) {
	docIndex := len(bm.documents)
	bm.documents = append(bm.documents, doc)
	bm.docCount++

	// 分词并计算词频
	terms := bm.tokenize(doc.Content)
	bm.docLengths = append(bm.docLengths, len(terms))

	// 初始化该文档的词频映射
	bm.termFrequency[docIndex] = make(map[string]int)

	// 记录词频和文档频率
	seenTerms := make(map[string]bool)
	for _, term := range terms {
		bm.termFrequency[docIndex][term]++
		if !seenTerms[term] {
			bm.docFrequency[term]++
			seenTerms[term] = true
		}
	}

	// 更新平均文档长度
	bm.updateAvgDocLength()
}

// AddDocuments 批量添加文档
func (bm *BM25) AddDocuments(docs []Document) {
	for _, doc := range docs {
		bm.AddDocument(doc)
	}
}

// Search 搜索文档
// 返回按相关性排序的结果
func (bm *BM25) Search(query string, topK int) []SearchResult {
	if bm.docCount == 0 {
		return nil
	}

	// 分词
	queryTerms := bm.tokenize(query)
	if len(queryTerms) == 0 {
		return nil
	}

	// 计算每个文档的 BM25 分数
	scores := make([]SearchResult, 0, bm.docCount)
	for i, doc := range bm.documents {
		score := bm.calculateScore(queryTerms, i)
		if score > 0 {
			scores = append(scores, SearchResult{
				Document: doc,
				Score:    score,
			})
		}
	}

	// 按分数降序排序
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	// 限制结果数量并添加排名
	if topK > 0 && len(scores) > topK {
		scores = scores[:topK]
	}
	for i := range scores {
		scores[i].Rank = i + 1
	}

	return scores
}

// calculateScore 计算单个文档的 BM25 分数
func (bm *BM25) calculateScore(queryTerms []string, docIndex int) float64 {
	score := 0.0
	docLength := float64(bm.docLengths[docIndex])

	for _, term := range queryTerms {
		// 获取词频
		tf := float64(bm.termFrequency[docIndex][term])
		if tf == 0 {
			continue
		}

		// 获取文档频率
		df := float64(bm.docFrequency[term])
		if df == 0 {
			continue
		}

		// 计算 IDF
		idf := bm.calculateIDF(df)

		// 计算 BM25 词频因子
		numerator := tf * (bm.k1 + 1)
		denominator := tf + bm.k1*(1-bm.b+bm.b*(docLength/bm.avgDocLength))

		score += idf * (numerator / denominator)
	}

	return score
}

// calculateIDF 计算逆文档频率
func (bm *BM25) calculateIDF(df float64) float64 {
	// 使用标准 BM25 IDF 公式
	// IDF = log((N - df + 0.5) / (df + 0.5) + 1)
	n := float64(bm.docCount)
	return math.Log((n-df+0.5)/(df+0.5) + 1)
}

// tokenize 分词函数
// 支持中英文混合分词
func (bm *BM25) tokenize(text string) []string {
	text = strings.ToLower(text)
	tokens := make([]string, 0)

	var currentToken strings.Builder
	prevIsChinese := false

	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			// 中文字符：每个字单独作为一个 token
			if currentToken.Len() > 0 {
				tokens = append(tokens, currentToken.String())
				currentToken.Reset()
			}
			tokens = append(tokens, string(r))
			prevIsChinese = true
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			// 英文字母或数字：累积到当前 token
			if prevIsChinese && currentToken.Len() > 0 {
				tokens = append(tokens, currentToken.String())
				currentToken.Reset()
			}
			currentToken.WriteRune(r)
			prevIsChinese = false
		} else if r == '_' || r == '-' {
			// 下划线和连字符：保留在 token 中（用于变量名等）
			if currentToken.Len() > 0 {
				currentToken.WriteRune(r)
			}
			prevIsChinese = false
		} else {
			// 其他字符（空格、标点等）：作为分隔符
			if currentToken.Len() > 0 {
				tokens = append(tokens, currentToken.String())
				currentToken.Reset()
			}
			prevIsChinese = false
		}
	}

	// 处理最后一个 token
	if currentToken.Len() > 0 {
		tokens = append(tokens, currentToken.String())
	}

	return tokens
}

// updateAvgDocLength 更新平均文档长度
func (bm *BM25) updateAvgDocLength() {
	if bm.docCount == 0 {
		bm.avgDocLength = 0
		return
	}

	total := 0
	for _, length := range bm.docLengths {
		total += length
	}
	bm.avgDocLength = float64(total) / float64(bm.docCount)
}

// Clear 清空索引
func (bm *BM25) Clear() {
	bm.documents = make([]Document, 0)
	bm.docCount = 0
	bm.avgDocLength = 0
	bm.docFrequency = make(map[string]int)
	bm.termFrequency = make(map[int]map[string]int)
	bm.docLengths = make([]int, 0)
}

// DocumentCount 返回索引中的文档数量
func (bm *BM25) DocumentCount() int {
	return bm.docCount
}

// GetDocument 根据 ID 获取文档
func (bm *BM25) GetDocument(id string) *Document {
	for _, doc := range bm.documents {
		if doc.ID == id {
			return &doc
		}
	}
	return nil
}

// RemoveDocument 从索引中移除文档
// 注意：这个操作需要重建索引，性能较低
func (bm *BM25) RemoveDocument(id string) bool {
	for i, doc := range bm.documents {
		if doc.ID == id {
			// 找到要删除的文档
			// 需要重建整个索引
			newDocs := make([]Document, 0, len(bm.documents)-1)
			newDocs = append(newDocs, bm.documents[:i]...)
			newDocs = append(newDocs, bm.documents[i+1:]...)

			// 清空并重建索引
			bm.Clear()
			bm.AddDocuments(newDocs)
			return true
		}
	}
	return false
}

// UpdateDocument 更新文档
func (bm *BM25) UpdateDocument(doc Document) bool {
	if bm.RemoveDocument(doc.ID) {
		bm.AddDocument(doc)
		return true
	}
	return false
}

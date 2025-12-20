package search

import (
	"testing"
)

func TestBM25_BasicSearch(t *testing.T) {
	bm := NewBM25WithDefaults()

	// 添加文档
	docs := []Document{
		{ID: "read", Content: "Read file from filesystem, supports text and binary files"},
		{ID: "write", Content: "Write content to file, creates new file or overwrites existing"},
		{ID: "grep", Content: "Search for patterns in files using regular expressions"},
		{ID: "bash", Content: "Execute shell commands in bash terminal"},
		{ID: "http", Content: "Make HTTP requests to external APIs and websites"},
	}

	for _, doc := range docs {
		bm.AddDocument(doc)
	}

	// 测试搜索
	results := bm.Search("file read", 3)

	if len(results) == 0 {
		t.Fatal("expected search results, got none")
	}

	// 验证 read 应该排在前面
	found := false
	for _, r := range results {
		if r.Document.ID == "read" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected 'read' in search results")
	}
}

func TestBM25_ChineseSearch(t *testing.T) {
	bm := NewBM25WithDefaults()

	// 添加中文文档
	docs := []Document{
		{ID: "read", Content: "读取文件内容，支持文本和二进制文件"},
		{ID: "write", Content: "写入文件内容，创建新文件或覆盖现有文件"},
		{ID: "search", Content: "搜索文件中的内容，支持正则表达式"},
	}

	for _, doc := range docs {
		bm.AddDocument(doc)
	}

	// 测试中文搜索
	results := bm.Search("读取文件", 3)

	if len(results) == 0 {
		t.Fatal("expected search results for Chinese query, got none")
	}

	// 验证 read 应该排在前面
	if results[0].Document.ID != "read" {
		t.Errorf("expected 'read' as top result, got %s", results[0].Document.ID)
	}
}

func TestBM25_EmptyQuery(t *testing.T) {
	bm := NewBM25WithDefaults()

	bm.AddDocument(Document{ID: "test", Content: "test document"})

	results := bm.Search("", 10)

	if len(results) != 0 {
		t.Errorf("expected no results for empty query, got %d", len(results))
	}
}

func TestBM25_NoMatch(t *testing.T) {
	bm := NewBM25WithDefaults()

	bm.AddDocument(Document{ID: "test", Content: "hello world"})

	results := bm.Search("xyz123", 10)

	if len(results) != 0 {
		t.Errorf("expected no results for non-matching query, got %d", len(results))
	}
}

func TestBM25_ScoreOrdering(t *testing.T) {
	bm := NewBM25WithDefaults()

	// 文档中 "file" 出现次数不同
	docs := []Document{
		{ID: "many", Content: "file "},
		{ID: "few", Content: "file operations"},
		{ID: "none", Content: "bash terminal commands"},
	}

	for _, doc := range docs {
		bm.AddDocument(doc)
	}

	results := bm.Search("file", 3)

	if len(results) < 2 {
		t.Fatal("expected at least 2 results")
	}

	// 验证分数是降序的
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("results not sorted by score: %f > %f", results[i].Score, results[i-1].Score)
		}
	}
}

func TestBM25_RemoveDocument(t *testing.T) {
	bm := NewBM25WithDefaults()

	bm.AddDocument(Document{ID: "test1", Content: "hello world"})
	bm.AddDocument(Document{ID: "test2", Content: "hello there"})

	// 搜索应该返回两个结果
	results := bm.Search("hello", 10)
	if len(results) != 2 {
		t.Errorf("expected 2 results before removal, got %d", len(results))
	}

	// 移除一个文档
	bm.RemoveDocument("test1")

	// 搜索应该只返回一个结果
	results = bm.Search("hello", 10)
	if len(results) != 1 {
		t.Errorf("expected 1 result after removal, got %d", len(results))
	}

	if results[0].Document.ID != "test2" {
		t.Errorf("expected 'test2', got %s", results[0].Document.ID)
	}
}

func TestBM25_CustomParameters(t *testing.T) {
	// 测试自定义参数
	bm := NewBM25(2.0, 0.5)

	bm.AddDocument(Document{ID: "test", Content: "test document"})

	results := bm.Search("test", 1)
	if len(results) == 0 {
		t.Error("expected results with custom parameters")
	}
}

func TestBM25_DefaultParameters(t *testing.T) {
	// 测试无效参数会使用默认值
	bm := NewBM25(-1, 2.0) // 无效参数

	bm.AddDocument(Document{ID: "test", Content: "test document"})

	results := bm.Search("test", 1)
	if len(results) == 0 {
		t.Error("expected results with default parameters")
	}
}

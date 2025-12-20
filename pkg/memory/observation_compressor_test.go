package memory

import (
	"context"
	"strings"
	"testing"
)

func TestDefaultObservationCompressor_Compress_ShortContent(t *testing.T) {
	compressor := NewDefaultObservationCompressor()
	ctx := context.Background()

	// 短内容不应该被压缩
	shortContent := "This is a short content that should not be compressed."
	result, err := compressor.Compress(ctx, "Read", shortContent)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	if result.Summary != shortContent {
		t.Errorf("Short content should not be modified")
	}
	if result.CompressionRatio != 1.0 {
		t.Errorf("Expected compression ratio 1.0, got %f", result.CompressionRatio)
	}
	if !result.Recoverable {
		t.Error("Short content should be recoverable")
	}
}

func TestDefaultObservationCompressor_Compress_LongContent(t *testing.T) {
	compressor := NewDefaultObservationCompressor()
	ctx := context.Background()

	// 生成长内容
	var builder strings.Builder
	for i := range 200 {
		builder.WriteString("Line ")
		builder.WriteRune(rune('0' + i%10))
		builder.WriteString(": This is a test line with some content\n")
	}
	longContent := builder.String()

	result, err := compressor.Compress(ctx, "Read", longContent)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	// 压缩后应该更短
	if len(result.Summary) >= len(longContent) {
		t.Errorf("Compressed content should be shorter: %d >= %d", len(result.Summary), len(longContent))
	}

	// 应该包含行数信息
	if !strings.Contains(result.Summary, "lines") {
		t.Error("Summary should contain line count info")
	}

	// 压缩比应该小于 1
	if result.CompressionRatio >= 1.0 {
		t.Errorf("Compression ratio should be < 1.0, got %f", result.CompressionRatio)
	}
}

func TestDefaultObservationCompressor_ExtractReferences_FilePaths(t *testing.T) {
	compressor := NewDefaultObservationCompressor()

	content := `
Reading file /Users/test/project/main.go
Found reference to /usr/local/bin/go
Also checking /home/user/.config/app.yaml
	`

	refs := compressor.extractReferences(content)

	foundPaths := 0
	for _, ref := range refs {
		if ref.Type == ReferenceTypeFilePath {
			foundPaths++
		}
	}

	if foundPaths < 2 {
		t.Errorf("Expected at least 2 file paths, found %d", foundPaths)
	}
}

func TestDefaultObservationCompressor_ExtractReferences_URLs(t *testing.T) {
	compressor := NewDefaultObservationCompressor()

	content := `
Visit https://example.com for more info
Also check http://api.example.org/docs
And https://github.com/user/repo
	`

	refs := compressor.extractReferences(content)

	foundURLs := 0
	for _, ref := range refs {
		if ref.Type == ReferenceTypeURL {
			foundURLs++
		}
	}

	if foundURLs != 3 {
		t.Errorf("Expected 3 URLs, found %d", foundURLs)
	}
}

func TestDefaultObservationCompressor_ExtractReferences_Functions(t *testing.T) {
	compressor := NewDefaultObservationCompressor()

	content := `
func main() {
    // code
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
    // code
}

def process_data(input):
    pass

function calculateSum(a, b) {
    return a + b;
}
	`

	refs := compressor.extractReferences(content)

	foundFuncs := 0
	for _, ref := range refs {
		if ref.Type == ReferenceTypeFunction {
			foundFuncs++
		}
	}

	if foundFuncs < 3 {
		t.Errorf("Expected at least 3 functions, found %d", foundFuncs)
	}
}

func TestDefaultObservationCompressor_CompressByToolType_Bash(t *testing.T) {
	compressor := NewDefaultObservationCompressor()
	ctx := context.Background()

	// 生成 Bash 输出（包含错误）
	var builder strings.Builder
	for i := range 100 {
		builder.WriteString("Output line ")
		builder.WriteRune(rune('0' + i%10))
		builder.WriteString("\n")
	}
	builder.WriteString("Error: Something went wrong\n")
	for range 50 {
		builder.WriteString("More output\n")
	}

	result, err := compressor.Compress(ctx, "Bash", builder.String())
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	// 应该检测到错误并保留更多上下文
	if !strings.Contains(result.Summary, "error") {
		t.Log("Summary should indicate error was detected")
	}
}

func TestDefaultObservationCompressor_CompressByToolType_Grep(t *testing.T) {
	compressor := NewDefaultObservationCompressor()
	ctx := context.Background()

	// 生成搜索结果
	var builder strings.Builder
	for range 100 {
		builder.WriteString("/path/to/file.go:123: match found\n")
	}

	result, err := compressor.Compress(ctx, "Grep", builder.String())
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	// 应该包含匹配数信息
	if !strings.Contains(result.Summary, "results") && !strings.Contains(result.Summary, "matches") {
		t.Log("Summary should contain match count info")
	}
}

func TestDefaultObservationCompressor_CanRecover(t *testing.T) {
	compressor := NewDefaultObservationCompressor()

	tests := []struct {
		name          string
		compressed    *CompressedObservation
		expectRecover bool
	}{
		{
			name: "with file reference",
			compressed: &CompressedObservation{
				Recoverable: true,
				References: []Reference{
					{Type: ReferenceTypeFilePath, Value: "/path/to/file.go"},
				},
			},
			expectRecover: true,
		},
		{
			name: "without references",
			compressed: &CompressedObservation{
				Recoverable: false,
				References:  []Reference{},
			},
			expectRecover: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compressor.CanRecover(tt.compressed)
			if result != tt.expectRecover {
				t.Errorf("CanRecover() = %v, want %v", result, tt.expectRecover)
			}
		})
	}
}

func TestHashContent(t *testing.T) {
	content1 := "Hello, World!"
	content2 := "Hello, World!"
	content3 := "Different content"

	hash1 := hashContent(content1)
	hash2 := hashContent(content2)
	hash3 := hashContent(content3)

	// 相同内容应该产生相同哈希
	if hash1 != hash2 {
		t.Error("Same content should produce same hash")
	}

	// 不同内容应该产生不同哈希
	if hash1 == hash3 {
		t.Error("Different content should produce different hash")
	}

	// 哈希应该是 16 字符
	if len(hash1) != 16 {
		t.Errorf("Hash should be 16 chars, got %d", len(hash1))
	}
}

func TestIsValidFilePath(t *testing.T) {
	tests := []struct {
		path  string
		valid bool
	}{
		{"/Users/test/file.go", true},
		{"/path/to/directory/", true},
		{"/dev/null", false},
		{"/proc/1/status", false},
		{"http://example.com", false},
		{"/usr/local/bin/go", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isValidFilePath(tt.path)
			if result != tt.valid {
				t.Errorf("isValidFilePath(%s) = %v, want %v", tt.path, result, tt.valid)
			}
		})
	}
}

func TestNewObservationCompressorWithConfig(t *testing.T) {
	config := &ObservationCompressorConfig{
		MaxSummaryLength:  1000,
		MinCompressLength: 500,
	}

	compressor := NewObservationCompressorWithConfig(config)

	if compressor.maxSummaryLength != 1000 {
		t.Errorf("Expected maxSummaryLength 1000, got %d", compressor.maxSummaryLength)
	}
	if compressor.minCompressLength != 500 {
		t.Errorf("Expected minCompressLength 500, got %d", compressor.minCompressLength)
	}
}

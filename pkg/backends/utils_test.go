package backends

import (
	"strings"
	"testing"
)

func TestSanitizeToolCallID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "正常ID",
			input:    "tool_call_12345",
			expected: "tool_call_12345",
		},
		{
			name:     "包含点号",
			input:    "call.123.456",
			expected: "call_123_456",
		},
		{
			name:     "包含斜杠",
			input:    "call/123/456",
			expected: "call_123_456",
		},
		{
			name:     "包含反斜杠",
			input:    "call\\123\\456",
			expected: "call_123_456",
		},
		{
			name:     "混合危险字符",
			input:    "../path/to./file\\name",
			expected: "___path_to__file_name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeToolCallID(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeToolCallID(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatContentWithLineNumbers(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		startLine int
		wantLines int
		checkFunc func(t *testing.T, result string)
	}{
		{
			name:      "简单内容",
			content:   "line1\nline2\nline3",
			startLine: 1,
			wantLines: 3,
			checkFunc: func(t *testing.T, result string) {
				if !strings.Contains(result, "     1\tline1") {
					t.Error("应包含格式化的第一行")
				}
				if !strings.Contains(result, "     2\tline2") {
					t.Error("应包含格式化的第二行")
				}
			},
		},
		{
			name:      "自定义起始行号",
			content:   "foo\nbar",
			startLine: 10,
			wantLines: 2,
			checkFunc: func(t *testing.T, result string) {
				if !strings.Contains(result, "    10\tfoo") {
					t.Error("应从行号 10 开始")
				}
			},
		},
		{
			name:      "超长行分块",
			content:   strings.Repeat("a", MaxLineLength+100),
			startLine: 1,
			wantLines: 2, // 会被分成 2 块
			checkFunc: func(t *testing.T, result string) {
				lines := strings.Split(result, "\n")
				if len(lines) < 2 {
					t.Error("超长行应该被分块")
				}
				// 第二行应该有延续标记 "1.1"
				if !strings.Contains(lines[1], "1.1") {
					t.Errorf("第二行应包含延续标记 '1.1', got: %s", lines[1])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatContentWithLineNumbers(tt.content, tt.startLine)
			if tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}

func TestCheckEmptyContent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantWarn bool
	}{
		{
			name:     "正常内容",
			content:  "hello world",
			wantWarn: false,
		},
		{
			name:     "空字符串",
			content:  "",
			wantWarn: true,
		},
		{
			name:     "仅空格",
			content:  "   \n  \t  ",
			wantWarn: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckEmptyContent(tt.content)
			if (result != "") != tt.wantWarn {
				t.Errorf("CheckEmptyContent(%q) returned warning=%v, want=%v", tt.content, result != "", tt.wantWarn)
			}
			if tt.wantWarn && result != EmptyContentWarning {
				t.Errorf("Expected warning message: %q, got: %q", EmptyContentWarning, result)
			}
		})
	}
}

func TestTruncateIfTooLong(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		limit        int
		wantTruncate bool
	}{
		{
			name:         "短内容不截断",
			input:        "short content",
			limit:        1000,
			wantTruncate: false,
		},
		{
			name:         "超长内容截断",
			input:        strings.Repeat("a", 100000),
			limit:        1000,
			wantTruncate: true,
		},
		{
			name:         "使用默认限制",
			input:        strings.Repeat("b", ToolResultTokenLimit*5),
			limit:        0, // 0 表示使用默认值
			wantTruncate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateIfTooLong(tt.input, tt.limit)
			isTruncated := strings.Contains(result, TruncationGuidance)
			if isTruncated != tt.wantTruncate {
				t.Errorf("TruncateIfTooLong() truncated=%v, want=%v", isTruncated, tt.wantTruncate)
			}
		})
	}
}

func TestExtractPreview(t *testing.T) {
	content := "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10\nline11\nline12"

	tests := []struct {
		name      string
		numLines  int
		wantLines int
	}{
		{
			name:      "默认10行",
			numLines:  0,
			wantLines: 10,
		},
		{
			name:      "自定义5行",
			numLines:  5,
			wantLines: 5,
		},
		{
			name:      "超过总行数",
			numLines:  100,
			wantLines: 12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractPreview(content, tt.numLines)
			lines := strings.Split(result, "\n")
			if len(lines) != tt.wantLines {
				t.Errorf("ExtractPreview() returned %d lines, want %d", len(lines), tt.wantLines)
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "正常路径",
			input:    "/foo/bar",
			expected: "/foo/bar",
		},
		{
			name:     "缺少前导斜杠",
			input:    "foo/bar",
			expected: "/foo/bar",
		},
		{
			name:     "尾部斜杠",
			input:    "/foo/bar/",
			expected: "/foo/bar",
		},
		{
			name:     "连续斜杠",
			input:    "/foo//bar///baz",
			expected: "/foo/bar/baz",
		},
		{
			name:     "根路径",
			input:    "/",
			expected: "/",
		},
		{
			name:     "带空格",
			input:    "  /foo/bar  ",
			expected: "/foo/bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizePath(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestJoinPath(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		rel      string
		expected string
	}{
		{
			name:     "基本拼接",
			base:     "/foo",
			rel:      "bar",
			expected: "/foo/bar",
		},
		{
			name:     "相对路径带斜杠",
			base:     "/foo",
			rel:      "/bar",
			expected: "/foo/bar",
		},
		{
			name:     "根路径拼接",
			base:     "/",
			rel:      "foo",
			expected: "/foo",
		},
		{
			name:     "空相对路径",
			base:     "/foo/bar",
			rel:      "",
			expected: "/foo/bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := JoinPath(tt.base, tt.rel)
			if result != tt.expected {
				t.Errorf("JoinPath(%q, %q) = %q, want %q", tt.base, tt.rel, result, tt.expected)
			}
		})
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "字节",
			bytes:    500,
			expected: "500 B",
		},
		{
			name:     "KB",
			bytes:    1536, // 1.5 KB
			expected: "1.5 KB",
		},
		{
			name:     "MB",
			bytes:    2621440, // 2.5 MB
			expected: "2.5 MB",
		},
		{
			name:     "GB",
			bytes:    3221225472, // 3 GB
			expected: "3.0 GB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatFileSize(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatFileSize(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestIsTextFile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "Go文件",
			path:     "/path/to/file.go",
			expected: true,
		},
		{
			name:     "Python文件",
			path:     "script.py",
			expected: true,
		},
		{
			name:     "二进制文件",
			path:     "binary.exe",
			expected: false,
		},
		{
			name:     "图片文件",
			path:     "image.png",
			expected: false,
		},
		{
			name:     "Markdown",
			path:     "README.md",
			expected: true,
		},
		{
			name:     "大写扩展名",
			path:     "file.GO",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTextFile(tt.path)
			if result != tt.expected {
				t.Errorf("IsTextFile(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

// BenchmarkFormatContentWithLineNumbers 性能测试
func BenchmarkFormatContentWithLineNumbers(b *testing.B) {
	content := strings.Repeat("line of text\n", 1000)

	for b.Loop() {
		FormatContentWithLineNumbers(content, 1)
	}
}

// BenchmarkSanitizeToolCallID 性能测试
func BenchmarkSanitizeToolCallID(b *testing.B) {
	id := "../../../dangerous/path/to/file.txt"

	for b.Loop() {
		SanitizeToolCallID(id)
	}
}

// TestFormatGrepResults 测试 Grep 结果格式化
func TestFormatGrepResults(t *testing.T) {
	matches := []GrepMatch{
		{Path: "/foo/bar.go", LineNumber: 10, Line: "func main() {"},
		{Path: "/foo/bar.go", LineNumber: 20, Line: "fmt.Println(\"hello\")"},
		{Path: "/foo/baz.go", LineNumber: 5, Line: "package main"},
	}

	tests := []struct {
		name     string
		mode     string
		expected string
	}{
		{
			name:     "files_with_matches模式",
			mode:     "files_with_matches",
			expected: "", // 包含 /foo/bar.go 和 /foo/baz.go (顺序不定)
		},
		{
			name:     "count模式",
			mode:     "count",
			expected: "", // 包含计数信息
		},
		{
			name:     "content模式",
			mode:     "content",
			expected: "/foo/bar.go:10:func main() {\n/foo/bar.go:20:fmt.Println(\"hello\")\n/foo/baz.go:5:package main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatGrepResults(matches, tt.mode)

			if result == "" {
				t.Error("Expected non-empty result")
			}

			// 对于 files_with_matches 和 count, 只验证包含预期内容
			switch tt.mode {
			case "files_with_matches":
				if !strings.Contains(result, "/foo/bar.go") || !strings.Contains(result, "/foo/baz.go") {
					t.Errorf("Expected both file paths, got: %s", result)
				}
			case "count":
				if !strings.Contains(result, "/foo/bar.go: 2") || !strings.Contains(result, "/foo/baz.go: 1") {
					t.Errorf("Expected count information, got: %s", result)
				}
			case "content":
				if result != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

func TestFormatGrepResults_Empty(t *testing.T) {
	result := FormatGrepResults([]GrepMatch{}, "content")
	if result != "(no matches)" {
		t.Errorf("Expected '(no matches)', got %q", result)
	}
}

func TestGroupGrepMatches(t *testing.T) {
	matches := []GrepMatch{
		{Path: "/foo/bar.go", LineNumber: 10, Line: "line1"},
		{Path: "/foo/bar.go", LineNumber: 20, Line: "line2"},
		{Path: "/foo/baz.go", LineNumber: 5, Line: "line3"},
	}

	grouped := GroupGrepMatches(matches)

	if len(grouped) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(grouped))
	}

	if len(grouped["/foo/bar.go"]) != 2 {
		t.Errorf("Expected 2 matches for /foo/bar.go, got %d", len(grouped["/foo/bar.go"]))
	}

	if len(grouped["/foo/baz.go"]) != 1 {
		t.Errorf("Expected 1 match for /foo/baz.go, got %d", len(grouped["/foo/baz.go"]))
	}
}

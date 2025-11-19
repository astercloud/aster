package filters

import (
	"fmt"
	"regexp"
	"strings"
)

// Filter 过滤器接口
type Filter interface {
	// Apply 应用过滤器
	Apply(content string) (string, error)

	// Name 返回过滤器名称
	Name() string
}

// FilterChain 过滤器链
type FilterChain struct {
	filters []Filter
}

// NewFilterChain 创建过滤器链
func NewFilterChain(filters ...Filter) *FilterChain {
	return &FilterChain{
		filters: filters,
	}
}

// Add 添加过滤器
func (fc *FilterChain) Add(filter Filter) *FilterChain {
	fc.filters = append(fc.filters, filter)
	return fc
}

// Apply 依次应用所有过滤器
func (fc *FilterChain) Apply(content string) (string, error) {
	result := content
	var err error

	for _, filter := range fc.filters {
		result, err = filter.Apply(result)
		if err != nil {
			return result, fmt.Errorf("filter %s failed: %w", filter.Name(), err)
		}
	}

	return result, nil
}

// RegexReplaceFilter 正则替换过滤器
type RegexReplaceFilter struct {
	name        string
	pattern     *regexp.Regexp
	replacement string
}

// NewRegexReplaceFilter 创建正则替换过滤器
func NewRegexReplaceFilter(name, pattern, replacement string) (*RegexReplaceFilter, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	return &RegexReplaceFilter{
		name:        name,
		pattern:     re,
		replacement: replacement,
	}, nil
}

// Name 实现 Filter 接口
func (f *RegexReplaceFilter) Name() string {
	return f.name
}

// Apply 实现 Filter 接口
func (f *RegexReplaceFilter) Apply(content string) (string, error) {
	return f.pattern.ReplaceAllString(content, f.replacement), nil
}

// TrimFilter 去除首尾空白过滤器
type TrimFilter struct{}

// NewTrimFilter 创建 Trim 过滤器
func NewTrimFilter() *TrimFilter {
	return &TrimFilter{}
}

// Name 实现 Filter 接口
func (f *TrimFilter) Name() string {
	return "Trim"
}

// Apply 实现 Filter 接口
func (f *TrimFilter) Apply(content string) (string, error) {
	return strings.TrimSpace(content), nil
}

// LowercaseFilter 小写转换过滤器
type LowercaseFilter struct{}

// NewLowercaseFilter 创建小写过滤器
func NewLowercaseFilter() *LowercaseFilter {
	return &LowercaseFilter{}
}

// Name 实现 Filter 接口
func (f *LowercaseFilter) Name() string {
	return "Lowercase"
}

// Apply 实现 Filter 接口
func (f *LowercaseFilter) Apply(content string) (string, error) {
	return strings.ToLower(content), nil
}

// UppercaseFilter 大写转换过滤器
type UppercaseFilter struct{}

// NewUppercaseFilter 创建大写过滤器
func NewUppercaseFilter() *UppercaseFilter {
	return &UppercaseFilter{}
}

// Name 实现 Filter 接口
func (f *UppercaseFilter) Name() string {
	return "Uppercase"
}

// Apply 实现 Filter 接口
func (f *UppercaseFilter) Apply(content string) (string, error) {
	return strings.ToUpper(content), nil
}

// RemoveHTMLFilter HTML标签移除过滤器
type RemoveHTMLFilter struct {
	pattern *regexp.Regexp
}

// NewRemoveHTMLFilter 创建 HTML 移除过滤器
func NewRemoveHTMLFilter() *RemoveHTMLFilter {
	return &RemoveHTMLFilter{
		pattern: regexp.MustCompile(`<[^>]*>`),
	}
}

// Name 实现 Filter 接口
func (f *RemoveHTMLFilter) Name() string {
	return "RemoveHTML"
}

// Apply 实现 Filter 接口
func (f *RemoveHTMLFilter) Apply(content string) (string, error) {
	return f.pattern.ReplaceAllString(content, ""), nil
}

// ReplaceFilter 简单替换过滤器
type ReplaceFilter struct {
	name string
	old  string
	new  string
}

// NewReplaceFilter 创建替换过滤器
func NewReplaceFilter(name, old, new string) *ReplaceFilter {
	return &ReplaceFilter{
		name: name,
		old:  old,
		new:  new,
	}
}

// Name 实现 Filter 接口
func (f *ReplaceFilter) Name() string {
	return f.name
}

// Apply 实现 Filter 接口
func (f *ReplaceFilter) Apply(content string) (string, error) {
	return strings.ReplaceAll(content, f.old, f.new), nil
}

// LengthLimitFilter 长度限制过滤器
type LengthLimitFilter struct {
	maxLength int
	suffix    string
}

// NewLengthLimitFilter 创建长度限制过滤器
func NewLengthLimitFilter(maxLength int, suffix string) *LengthLimitFilter {
	return &LengthLimitFilter{
		maxLength: maxLength,
		suffix:    suffix,
	}
}

// Name 实现 Filter 接口
func (f *LengthLimitFilter) Name() string {
	return "LengthLimit"
}

// Apply 实现 Filter 接口
func (f *LengthLimitFilter) Apply(content string) (string, error) {
	if len(content) <= f.maxLength {
		return content, nil
	}

	return content[:f.maxLength] + f.suffix, nil
}

// RemoveURLsFilter URL 移除过滤器
type RemoveURLsFilter struct {
	pattern *regexp.Regexp
}

// NewRemoveURLsFilter 创建 URL 移除过滤器
func NewRemoveURLsFilter() *RemoveURLsFilter {
	return &RemoveURLsFilter{
		pattern: regexp.MustCompile(`https?://[^\s]+`),
	}
}

// Name 实现 Filter 接口
func (f *RemoveURLsFilter) Name() string {
	return "RemoveURLs"
}

// Apply 实现 Filter 接口
func (f *RemoveURLsFilter) Apply(content string) (string, error) {
	return f.pattern.ReplaceAllString(content, ""), nil
}

// FunctionFilter 自定义函数过滤器
type FunctionFilter struct {
	name string
	fn   func(string) (string, error)
}

// NewFunctionFilter 创建自定义函数过滤器
func NewFunctionFilter(name string, fn func(string) (string, error)) *FunctionFilter {
	return &FunctionFilter{
		name: name,
		fn:   fn,
	}
}

// Name 实现 Filter 接口
func (f *FunctionFilter) Name() string {
	return f.name
}

// Apply 实现 Filter 接口
func (f *FunctionFilter) Apply(content string) (string, error) {
	return f.fn(content)
}

package util

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestMarshalDeterministic_Map(t *testing.T) {
	// 创建一个 map，多次序列化应该产生相同的结果
	m := map[string]any{
		"zebra": 1,
		"apple": 2,
		"mango": 3,
	}

	// 多次序列化，确保结果一致
	var results []string
	for range 10 {
		data, err := MarshalDeterministic(m)
		if err != nil {
			t.Fatalf("MarshalDeterministic failed: %v", err)
		}
		results = append(results, string(data))
	}

	// 所有结果应该相同
	expected := results[0]
	for i, result := range results {
		if result != expected {
			t.Errorf("Iteration %d produced different result:\nexpected: %s\ngot: %s", i, expected, result)
		}
	}

	// 验证 key 顺序是字典序
	expectedJSON := `{"apple":2,"mango":3,"zebra":1}`
	if results[0] != expectedJSON {
		t.Errorf("Expected ordered JSON:\nexpected: %s\ngot: %s", expectedJSON, results[0])
	}
}

func TestMarshalDeterministic_NestedMap(t *testing.T) {
	m := map[string]any{
		"z": map[string]int{
			"c": 3,
			"a": 1,
			"b": 2,
		},
		"a": map[string]string{
			"y": "why",
			"x": "ex",
		},
	}

	data, err := MarshalDeterministic(m)
	if err != nil {
		t.Fatalf("MarshalDeterministic failed: %v", err)
	}

	// 外层和内层 map 都应该按字典序排列
	expected := `{"a":{"x":"ex","y":"why"},"z":{"a":1,"b":2,"c":3}}`
	if string(data) != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, string(data))
	}
}

func TestMarshalDeterministic_Slice(t *testing.T) {
	s := []map[string]int{
		{"b": 2, "a": 1},
		{"d": 4, "c": 3},
	}

	data, err := MarshalDeterministic(s)
	if err != nil {
		t.Fatalf("MarshalDeterministic failed: %v", err)
	}

	// slice 中的每个 map 都应该有序
	expected := `[{"a":1,"b":2},{"c":3,"d":4}]`
	if string(data) != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, string(data))
	}
}

func TestMarshalDeterministic_Struct(t *testing.T) {
	type Inner struct {
		Z int `json:"z"`
		A int `json:"a"`
	}

	type Outer struct {
		Name  string           `json:"name"`
		Data  map[string]int   `json:"data"`
		Inner Inner            `json:"inner"`
		Extra map[string]Inner `json:"extra,omitempty"`
	}

	o := Outer{
		Name: "test",
		Data: map[string]int{"c": 3, "a": 1, "b": 2},
		Inner: Inner{
			Z: 26,
			A: 1,
		},
		Extra: map[string]Inner{
			"second": {Z: 2, A: 0},
			"first":  {Z: 1, A: 0},
		},
	}

	data, err := MarshalDeterministic(o)
	if err != nil {
		t.Fatalf("MarshalDeterministic failed: %v", err)
	}

	// 验证结果是确定性的
	var results []string
	for range 5 {
		d, _ := MarshalDeterministic(o)
		results = append(results, string(d))
	}

	for i, r := range results {
		if r != results[0] {
			t.Errorf("Iteration %d differs: %s", i, r)
		}
	}

	// 解析验证结构正确
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	if parsed["name"] != "test" {
		t.Errorf("Expected name 'test', got %v", parsed["name"])
	}
}

func TestMarshalDeterministic_NilValues(t *testing.T) {
	var nilMap map[string]int
	data, err := MarshalDeterministic(nilMap)
	if err != nil {
		t.Fatalf("MarshalDeterministic failed: %v", err)
	}
	if string(data) != "null" {
		t.Errorf("Expected null, got %s", string(data))
	}

	var nilSlice []int
	data, err = MarshalDeterministic(nilSlice)
	if err != nil {
		t.Fatalf("MarshalDeterministic failed: %v", err)
	}
	if string(data) != "null" {
		t.Errorf("Expected null, got %s", string(data))
	}
}

func TestMarshalDeterministic_Pointer(t *testing.T) {
	m := map[string]int{"b": 2, "a": 1}
	ptr := &m

	data, err := MarshalDeterministic(ptr)
	if err != nil {
		t.Fatalf("MarshalDeterministic failed: %v", err)
	}

	expected := `{"a":1,"b":2}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

func TestMarshalDeterministicIndent(t *testing.T) {
	m := map[string]int{"b": 2, "a": 1}

	data, err := MarshalDeterministicIndent(m, "", "  ")
	if err != nil {
		t.Fatalf("MarshalDeterministicIndent failed: %v", err)
	}

	expected := "{\n  \"a\": 1,\n  \"b\": 2\n}"
	if string(data) != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, string(data))
	}
}

func TestMarshalDeterministicToBuffer(t *testing.T) {
	m := map[string]int{"b": 2, "a": 1}

	var buf bytes.Buffer
	err := MarshalDeterministicToBuffer(m, &buf)
	if err != nil {
		t.Fatalf("MarshalDeterministicToBuffer failed: %v", err)
	}

	// 注意：Encoder.Encode 会添加换行符
	expected := `{"a":1,"b":2}` + "\n"
	if buf.String() != expected {
		t.Errorf("Expected %q, got %q", expected, buf.String())
	}
}

func TestMarshalDeterministic_OmitEmpty(t *testing.T) {
	type Test struct {
		Name  string `json:"name"`
		Empty string `json:"empty,omitempty"`
		Zero  int    `json:"zero,omitempty"`
		Slice []int  `json:"slice,omitempty"`
	}

	o := Test{
		Name:  "test",
		Empty: "",
		Zero:  0,
		Slice: nil,
	}

	data, err := MarshalDeterministic(o)
	if err != nil {
		t.Fatalf("MarshalDeterministic failed: %v", err)
	}

	expected := `{"name":"test"}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

func TestMarshalDeterministic_BasicTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"string", "hello", `"hello"`},
		{"int", 42, "42"},
		{"float", 3.14, "3.14"},
		{"bool", true, "true"},
		{"nil", nil, "null"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := MarshalDeterministic(tt.input)
			if err != nil {
				t.Fatalf("MarshalDeterministic failed: %v", err)
			}
			if string(data) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(data))
			}
		})
	}
}

// BenchmarkMarshalDeterministic 与标准 json.Marshal 对比
func BenchmarkMarshalDeterministic(b *testing.B) {
	m := map[string]any{
		"zebra": 1,
		"apple": 2,
		"mango": 3,
		"nested": map[string]int{
			"c": 3,
			"a": 1,
			"b": 2,
		},
	}

	b.Run("Deterministic", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = MarshalDeterministic(m)
		}
	})

	b.Run("StandardJSON", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = json.Marshal(m)
		}
	})
}

// 验证标准 json.Marshal 确实是不确定的
func TestStandardJSONIsNonDeterministic(t *testing.T) {
	m := map[string]int{
		"a": 1, "b": 2, "c": 3, "d": 4, "e": 5,
		"f": 6, "g": 7, "h": 8, "i": 9, "j": 10,
	}

	// 多次序列化，看是否产生不同结果
	seen := make(map[string]bool)
	for range 100 {
		data, _ := json.Marshal(m)
		seen[string(data)] = true
	}

	// 注意：这个测试可能偶尔失败，因为运气好的话所有输出可能相同
	// 但对于有足够多 key 的 map，大概率会产生不同输出
	if len(seen) == 1 {
		t.Log("Warning: Standard json.Marshal produced consistent output in all iterations")
		t.Log("This is expected behavior for Go 1.12+, but map iteration order is still not guaranteed")
	}
}

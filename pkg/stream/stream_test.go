package stream

import (
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

func TestPipe_BasicSendRecv(t *testing.T) {
	reader, writer := Pipe[string](3)

	go func() {
		defer writer.Close()
		writer.Send("hello", nil)
		writer.Send("world", nil)
	}()

	defer reader.Close()

	v1, err := reader.Recv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v1 != "hello" {
		t.Errorf("expected 'hello', got '%s'", v1)
	}

	v2, err := reader.Recv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v2 != "world" {
		t.Errorf("expected 'world', got '%s'", v2)
	}

	_, err = reader.Recv()
	if !errors.Is(err, io.EOF) {
		t.Errorf("expected EOF, got %v", err)
	}
}

func TestPipe_BackPressure(t *testing.T) {
	reader, writer := Pipe[int](2) // 小缓冲区

	// 填满缓冲区
	writer.Send(1, nil)
	writer.Send(2, nil)

	// 第三次发送应该阻塞，直到我们读取
	done := make(chan bool)
	go func() {
		writer.Send(3, nil)
		writer.Close()
		done <- true
	}()

	// 读取一个值来解除阻塞
	v, _ := reader.Recv()
	if v != 1 {
		t.Errorf("expected 1, got %d", v)
	}

	select {
	case <-done:
		// 好，发送完成
	case <-time.After(100 * time.Millisecond):
		t.Error("send should have completed after read")
	}

	reader.Close()
}

func TestFromSlice(t *testing.T) {
	data := []int{1, 2, 3, 4, 5}
	reader := FromSlice(data)

	result, err := reader.Collect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != len(data) {
		t.Errorf("expected %d items, got %d", len(data), len(result))
	}

	for i, v := range result {
		if v != data[i] {
			t.Errorf("at index %d: expected %d, got %d", i, data[i], v)
		}
	}
}

func TestCopy_Independent(t *testing.T) {
	reader := FromSlice([]int{1, 2, 3})
	copies := reader.Copy(3)

	if len(copies) != 3 {
		t.Fatalf("expected 3 copies, got %d", len(copies))
	}

	// 每个副本应该能独立读取所有值
	for i, cp := range copies {
		result, err := cp.Collect()
		if err != nil {
			t.Errorf("copy %d: unexpected error: %v", i, err)
		}
		if len(result) != 3 {
			t.Errorf("copy %d: expected 3 items, got %d", i, len(result))
		}
		for j, v := range result {
			if v != j+1 {
				t.Errorf("copy %d, index %d: expected %d, got %d", i, j, j+1, v)
			}
		}
	}
}

func TestCopy_Concurrent(t *testing.T) {
	reader, writer := Pipe[int](10)

	// 开始写入
	go func() {
		defer writer.Close()
		for i := range 100 {
			writer.Send(i, nil)
		}
	}()

	copies := reader.Copy(3)

	var wg sync.WaitGroup
	results := make([][]int, 3)

	for i, cp := range copies {
		wg.Add(1)
		go func(idx int, r *Reader[int]) {
			defer wg.Done()
			var result []int
			for {
				v, err := r.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					t.Errorf("copy %d: error: %v", idx, err)
					return
				}
				result = append(result, v)
			}
			results[idx] = result
			r.Close()
		}(i, cp)
	}

	wg.Wait()

	// 所有副本应该有相同的值
	for i, result := range results {
		if len(result) != 100 {
			t.Errorf("copy %d: expected 100 items, got %d", i, len(result))
		}
	}
}

func TestMerge_Basic(t *testing.T) {
	r1 := FromSlice([]int{1, 3, 5})
	r2 := FromSlice([]int{2, 4, 6})

	merged := Merge(r1, r2)
	defer merged.Close()

	result, err := merged.Collect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 6 {
		t.Errorf("expected 6 items, got %d", len(result))
	}

	// 检查所有值都存在（顺序可能不同）
	expected := map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true, 6: true}
	for _, v := range result {
		if !expected[v] {
			t.Errorf("unexpected value: %d", v)
		}
		delete(expected, v)
	}
	if len(expected) > 0 {
		t.Errorf("missing values: %v", expected)
	}
}

func TestMergeNamed_SourceEOF(t *testing.T) {
	r1, w1 := Pipe[string](2)
	r2, w2 := Pipe[string](2)

	go func() {
		w1.Send("a", nil)
		w1.Close()
	}()
	go func() {
		w2.Send("b", nil)
		w2.Send("c", nil)
		w2.Close()
	}()

	merged := MergeNamed(map[string]*Reader[string]{
		"stream1": r1,
		"stream2": r2,
	})
	defer merged.Close()

	var values []string
	var eofSources []string

	for {
		v, err := merged.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if name, ok := GetSourceName(err); ok {
			eofSources = append(eofSources, name)
			continue
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		values = append(values, v)
	}

	if len(values) != 3 {
		t.Errorf("expected 3 values, got %d: %v", len(values), values)
	}

	// 至少应该报告一个源 EOF
	if len(eofSources) == 0 {
		t.Error("expected at least one source EOF")
	}
}

func TestTransform_Basic(t *testing.T) {
	reader := FromSlice([]int{1, 2, 3, 4, 5})

	doubled := Transform(reader, func(v int) (int, error) {
		return v * 2, nil
	})
	defer doubled.Close()

	result, err := doubled.Collect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []int{2, 4, 6, 8, 10}
	if len(result) != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), len(result))
	}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("at index %d: expected %d, got %d", i, expected[i], v)
		}
	}
}

func TestTransform_WithSkip(t *testing.T) {
	reader := FromSlice([]int{1, 2, 3, 4, 5, 6})

	// 只保留偶数
	evens := Transform(reader, func(v int) (int, error) {
		if v%2 != 0 {
			return 0, ErrSkip
		}
		return v, nil
	})
	defer evens.Close()

	result, err := evens.Collect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []int{2, 4, 6}
	if len(result) != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), len(result))
	}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("at index %d: expected %d, got %d", i, expected[i], v)
		}
	}
}

func TestFilter(t *testing.T) {
	reader := FromSlice([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})

	filtered := Filter(reader, func(v int) bool {
		return v > 5
	})
	defer filtered.Close()

	result, err := filtered.Collect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []int{6, 7, 8, 9, 10}
	if len(result) != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), len(result))
	}
}

func TestMap(t *testing.T) {
	reader := FromSlice([]string{"hello", "world"})

	lengths := Map(reader, func(s string) int {
		return len(s)
	})
	defer lengths.Close()

	result, err := lengths.Collect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 || result[0] != 5 || result[1] != 5 {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestTake(t *testing.T) {
	reader := FromSlice([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})

	first3 := Take(reader, 3)
	defer first3.Close()

	result, err := first3.Collect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result))
	}
	for i, v := range result {
		if v != i+1 {
			t.Errorf("at index %d: expected %d, got %d", i, i+1, v)
		}
	}
}

func TestForEach(t *testing.T) {
	reader := FromSlice([]int{1, 2, 3})

	var sum int
	err := ForEach(reader, func(v int) error {
		sum += v
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sum != 6 {
		t.Errorf("expected sum 6, got %d", sum)
	}
}

func TestForEach_Error(t *testing.T) {
	reader := FromSlice([]int{1, 2, 3, 4, 5})

	customErr := errors.New("stop at 3")
	err := ForEach(reader, func(v int) error {
		if v == 3 {
			return customErr
		}
		return nil
	})

	if !errors.Is(err, customErr) {
		t.Errorf("expected custom error, got %v", err)
	}
}

func TestErrorPropagation(t *testing.T) {
	reader, writer := Pipe[int](2)

	customErr := errors.New("custom error")

	go func() {
		writer.Send(1, nil)
		writer.Send(0, customErr)
		writer.Close()
	}()

	defer reader.Close()

	v, err := reader.Recv()
	if err != nil {
		t.Fatalf("unexpected error on first recv: %v", err)
	}
	if v != 1 {
		t.Errorf("expected 1, got %d", v)
	}

	_, err = reader.Recv()
	if !errors.Is(err, customErr) {
		t.Errorf("expected custom error, got %v", err)
	}
}

func TestSetAutomaticClose(t *testing.T) {
	reader, writer := Pipe[int](1)

	reader.SetAutomaticClose()

	writer.Send(1, nil)
	writer.Close()

	// 读取值
	v, _ := reader.Recv()
	if v != 1 {
		t.Errorf("expected 1, got %d", v)
	}

	// 注意：GC 行为不确定，所以我们不断言关闭状态
}

func BenchmarkPipe(b *testing.B) {
	reader, writer := Pipe[int](100)

	go func() {
		for i := 0; i < b.N; i++ {
			writer.Send(i, nil)
		}
		writer.Close()
	}()

	for {
		_, err := reader.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
	}
	reader.Close()
}

func BenchmarkCopy(b *testing.B) {
	data := make([]int, 1000)
	for i := range data {
		data[i] = i
	}

	for b.Loop() {
		reader := FromSlice(data)
		copies := reader.Copy(3)
		for _, c := range copies {
			_, _ = c.Collect()
		}
	}
}

func BenchmarkTransform(b *testing.B) {
	data := make([]int, 1000)
	for i := range data {
		data[i] = i
	}

	for b.Loop() {
		reader := FromSlice(data)
		transformed := Transform(reader, func(v int) (int, error) {
			return v * 2, nil
		})
		_, _ = transformed.Collect()
	}
}

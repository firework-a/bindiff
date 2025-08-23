package core_test

import (
	"bindiff/core"
	"bindiff/pkg/config"
	"bindiff/types"
	"context"
	"fmt"
	"testing"
	"time"
)

// TestBasicDiff 测试基本差分功能
func TestBasicDiff(t *testing.T) {
	tests := []struct {
		name     string
		oldData  []byte
		newData  []byte
		expected int // 期望的补丁数量
	}{
		{
			name:     "identical_files",
			oldData:  []byte("hello world"),
			newData:  []byte("hello world"),
			expected: 1, // 应该只有一个COPY操作
		},
		{
			name:     "simple_append",
			oldData:  []byte("hello"),
			newData:  []byte("hello world"),
			expected: 2, // COPY + INSERT
		},
		{
			name:     "simple_replace",
			oldData:  []byte("hello world"),
			newData:  []byte("hello earth"),
			expected: 3, // COPY + REPLACE + COPY
		},
		{
			name:     "complete_different",
			oldData:  []byte("abc"),
			newData:  []byte("xyz"),
			expected: 1, // REPLACE
		},
		{
			name:     "empty_to_content",
			oldData:  []byte(""),
			newData:  []byte("content"),
			expected: 1, // INSERT
		},
		{
			name:     "content_to_empty",
			oldData:  []byte("content"),
			newData:  []byte(""),
			expected: 1, // DELETE
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := core.Diff(tt.oldData, tt.newData)

			if len(patches) == 0 && tt.expected > 0 {
				t.Errorf("Expected %d patches, got 0", tt.expected)
				return
			}

			// 验证补丁可以正确应用
			result := core.ApplyPatch(tt.oldData, patches)
			if string(result) != string(tt.newData) {
				t.Errorf("Patch application failed.\nExpected: %q\nGot: %q",
					string(tt.newData), string(result))
			}
		})
	}
}

// TestDiffWithOptions 测试带选项的差分
func TestDiffWithOptions(t *testing.T) {
	oldData := make([]byte, 1024)
	newData := make([]byte, 1024)

	// 填充测试数据
	for i := range oldData {
		oldData[i] = byte(i % 256)
	}

	copy(newData, oldData)
	// 在中间插入一些数据
	copy(newData[512:], []byte("inserted data"))

	options := &core.DiffOptions{
		Config: &config.Config{
			BlockSize:      64,
			MinMatchLength: 32,
			MaxWorkers:     2,
			UseParallel:    true,
			EnableFFT:      false,
		},
		ShowProgress: false,
		Context:      context.Background(),
	}

	patches := core.DiffWithOptions(oldData, newData, options)

	if len(patches) == 0 {
		t.Error("Expected patches to be generated")
		return
	}

	// 验证补丁应用
	result := core.ApplyPatch(oldData, patches)
	if len(result) != len(newData) {
		t.Errorf("Result length mismatch. Expected %d, got %d",
			len(newData), len(result))
	}
}

// TestApplyPatchWithOptions 测试带选项的补丁应用
func TestApplyPatchWithOptions(t *testing.T) {
	oldData := []byte("The quick brown fox jumps over the lazy dog")
	newData := []byte("The quick red fox jumps over the sleepy cat")

	patches := core.Diff(oldData, newData)

	options := &core.ApplyOptions{
		Config:       config.DefaultConfig(),
		ShowProgress: false,
		Context:      context.Background(),
		VerifyResult: true,
	}

	result := core.ApplyPatchWithOptions(oldData, patches, options)

	if string(result) != string(newData) {
		t.Errorf("Patch application with options failed.\nExpected: %q\nGot: %q",
			string(newData), string(result))
	}
}

// TestLargeFileDiff 测试大文件差分
func TestLargeFileDiff(t *testing.T) {
	// 创建相对较大的测试数据（1MB）
	size := 1024 * 1024
	oldData := make([]byte, size)
	newData := make([]byte, size)

	// 填充模式数据
	for i := range oldData {
		oldData[i] = byte(i % 256)
		newData[i] = byte((i + 1) % 256) // 轻微变化
	}

	start := time.Now()
	patches := core.Diff(oldData, newData)
	duration := time.Since(start)

	t.Logf("Large file diff took %v, generated %d patches", duration, len(patches))

	if len(patches) == 0 {
		t.Error("Expected patches for large file")
		return
	}

	// 验证前几个补丁的应用
	if len(patches) > 100 {
		partialPatches := patches[:100]
		// 这里不验证完整结果，只是确保不会崩溃
		_ = core.ApplyPatch(oldData, partialPatches)
	}
}

// TestStreamingDiff 测试流式差分
func TestStreamingDiff(t *testing.T) {
	oldData := make([]byte, 2*1024*1024) // 2MB
	newData := make([]byte, 2*1024*1024)

	// 创建有模式的数据
	for i := range oldData {
		oldData[i] = byte(i % 256)
		newData[i] = byte(i % 256)
	}

	// 在新数据中间插入变化
	copy(newData[1024*1024:1024*1024+1000], []byte("THIS IS A CHANGE"))

	options := &core.DiffOptions{
		Config: &config.Config{
			MaxMemoryMB:    1, // 强制使用流式处理
			BlockSize:      1024,
			MinMatchLength: 64,
		},
		ShowProgress: false,
		Context:      context.Background(),
	}

	patches := core.DiffWithOptions(oldData, newData, options)

	if len(patches) == 0 {
		t.Error("Expected patches from streaming diff")
	}
}

// TestParallelDiff 测试并行差分
func TestParallelDiff(t *testing.T) {
	size := 100 * 1024 // 100KB
	oldData := make([]byte, size)
	newData := make([]byte, size)

	// 创建测试数据
	for i := range oldData {
		oldData[i] = byte(i % 256)
		newData[i] = byte(i % 256)
	}

	// 在几个位置添加变化
	positions := []int{1000, 5000, 10000, 15000}
	for _, pos := range positions {
		if pos < len(newData)-10 {
			copy(newData[pos:pos+10], []byte("CHANGE####"))
		}
	}

	options := &core.DiffOptions{
		Config: &config.Config{
			MaxWorkers:     4,
			UseParallel:    true,
			BlockSize:      1024,
			MinMatchLength: 32,
			MaxMemoryMB:    512, // 确保不触发流式处理
		},
		ShowProgress: false,
		Context:      context.Background(),
	}

	patches := core.DiffWithOptions(oldData, newData, options)

	if len(patches) == 0 {
		t.Error("Expected patches from parallel diff")
		return
	}

	// 验证结果
	result := core.ApplyPatch(oldData, patches)
	if len(result) != len(newData) {
		t.Errorf("Parallel diff result size mismatch. Expected %d, got %d",
			len(newData), len(result))
		return
	}

	// 验证内容是否正确
	for i := 0; i < len(newData); i++ {
		if result[i] != newData[i] {
			t.Errorf("Content mismatch at position %d: expected %d, got %d", i, newData[i], result[i])
			break
		}
	}
}

// TestOptimizePatches 测试补丁优化
func TestOptimizePatches(t *testing.T) {
	patches := []types.Patch{
		{Op: types.OP_INSERT, Offset: 0, Length: 5, Data: []byte("hello")},
		{Op: types.OP_INSERT, Offset: 5, Length: 6, Data: []byte(" world")},
		{Op: types.OP_COPY, Offset: 20, Length: 10},
		{Op: types.OP_COPY, Offset: 30, Length: 5},
	}

	optimized := core.OptimizePatches(patches)

	// 检查相邻的INSERT操作是否被合并
	insertCount := 0
	for _, p := range optimized {
		if p.Op == types.OP_INSERT {
			insertCount++
		}
	}

	if insertCount != 1 {
		t.Errorf("Expected 1 INSERT after optimization, got %d", insertCount)
	}

	// 验证合并后的数据
	if len(optimized) > 0 && optimized[0].Op == types.OP_INSERT {
		expectedData := "hello world"
		if string(optimized[0].Data) != expectedData {
			t.Errorf("Merged data incorrect. Expected %q, got %q",
				expectedData, string(optimized[0].Data))
		}
	}
}

// TestContextCancellation 测试上下文取消
func TestContextCancellation(t *testing.T) {
	oldData := make([]byte, 1024*1024) // 1MB
	newData := make([]byte, 1024*1024)

	// 填充数据
	for i := range oldData {
		oldData[i] = byte(i % 256)
		newData[i] = byte((i + 1) % 256)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	options := &core.DiffOptions{
		Config:       config.DefaultConfig(),
		ShowProgress: false,
		Context:      ctx,
	}

	// 这个测试可能会或不会被取消，取决于执行速度
	patches := core.DiffWithOptions(oldData, newData, options)
	t.Logf("Generated %d patches before cancellation (if any)", len(patches))
}

// BenchmarkDiff 基准测试差分性能
func BenchmarkDiff(b *testing.B) {
	sizes := []int{1024, 10240, 102400} // 1KB, 10KB, 100KB

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			oldData := make([]byte, size)
			newData := make([]byte, size)

			for i := range oldData {
				oldData[i] = byte(i % 256)
				newData[i] = byte((i + 10) % 256)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				core.Diff(oldData, newData)
			}
		})
	}
}

// BenchmarkApplyPatch 基准测试补丁应用性能
func BenchmarkApplyPatch(b *testing.B) {
	oldData := make([]byte, 10240) // 10KB
	newData := make([]byte, 10240)

	for i := range oldData {
		oldData[i] = byte(i % 256)
		newData[i] = byte((i + 5) % 256)
	}

	patches := core.Diff(oldData, newData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		core.ApplyPatch(oldData, patches)
	}
}

// TestErrorHandling 测试错误处理
func TestErrorHandling(t *testing.T) {
	t.Run("invalid_patch_offset", func(t *testing.T) {
		oldData := []byte("hello")
		patches := []types.Patch{
			{Op: types.OP_COPY, Offset: 1000, Length: 5}, // 无效偏移
		}

		result := core.ApplyPatch(oldData, patches)
		// 应该不会崩溃，返回原始数据或部分数据
		if len(result) == 0 {
			t.Error("Expected some result even with invalid patches")
		}
	})

	t.Run("empty_data", func(t *testing.T) {
		patches := core.Diff([]byte{}, []byte{})
		if len(patches) != 0 {
			t.Errorf("Expected no patches for empty data, got %d", len(patches))
		}
	})
}

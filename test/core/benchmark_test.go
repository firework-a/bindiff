package core_test

import (
	"bindiff/core"
	"bindiff/pkg/config"
	"bindiff/types"
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"testing"
	"time"
)

// BenchmarkSuite 基准测试套件
type BenchmarkSuite struct {
	oldData []byte
	newData []byte
	config  *config.Config
}

// NewBenchmarkSuite 创建基准测试套件
func NewBenchmarkSuite(size int, changeRatio float64) *BenchmarkSuite {
	oldData := make([]byte, size)
	newData := make([]byte, size)

	// 生成随机数据
	rand.Seed(time.Now().UnixNano())
	for i := range oldData {
		oldData[i] = byte(rand.Intn(256))
	}

	// 复制数据并应用变化
	copy(newData, oldData)
	changeCount := int(float64(size) * changeRatio)
	for i := 0; i < changeCount; i++ {
		pos := rand.Intn(size)
		newData[pos] = byte(rand.Intn(256))
	}

	return &BenchmarkSuite{
		oldData: oldData,
		newData: newData,
		config:  config.DefaultConfig(),
	}
}

// BenchmarkDiffAlgorithms 比较不同差分算法的性能
func BenchmarkDiffAlgorithms(b *testing.B) {
	sizes := []int{1024, 10 * 1024, 100 * 1024}
	changeRatios := []float64{0.01, 0.1, 0.5}

	for _, size := range sizes {
		for _, ratio := range changeRatios {
			b.Run(fmt.Sprintf("size_%d_change_%.0f%%", size, ratio*100), func(b *testing.B) {
				suite := NewBenchmarkSuite(size, ratio)

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					patches := core.Diff(suite.oldData, suite.newData)
					_ = patches
				}
			})
		}
	}
}

// BenchmarkParallelVsSequential 比较并行和串行差分性能
func BenchmarkParallelVsSequential(b *testing.B) {
	suite := NewBenchmarkSuite(1024*1024, 0.1) // 1MB, 10% 变化

	b.Run("sequential", func(b *testing.B) {
		options := &core.DiffOptions{
			Config: &config.Config{
				BlockSize:      1024,
				MinMatchLength: 64,
				MaxWorkers:     1,
				UseParallel:    false,
				EnableFFT:      false,
			},
			ShowProgress: false,
			Context:      context.Background(),
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			patches := core.DiffWithOptions(suite.oldData, suite.newData, options)
			_ = patches
		}
	})

	b.Run("parallel", func(b *testing.B) {
		options := &core.DiffOptions{
			Config: &config.Config{
				BlockSize:      1024,
				MinMatchLength: 64,
				MaxWorkers:     runtime.NumCPU(),
				UseParallel:    true,
				EnableFFT:      false,
			},
			ShowProgress: false,
			Context:      context.Background(),
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			patches := core.DiffWithOptions(suite.oldData, suite.newData, options)
			_ = patches
		}
	})
}

// BenchmarkFFTAlignment 测试 FFT 对齐性能
func BenchmarkFFTAlignment(b *testing.B) {
	sizes := []int{1024, 4096, 16384}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			suite := NewBenchmarkSuite(size, 0.1)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				offset := core.ComputeOffset(suite.oldData, suite.newData)
				_ = offset
			}
		})
	}
}

// BenchmarkDifferentBlockSizes 测试不同块大小的性能影响
func BenchmarkDifferentBlockSizes(b *testing.B) {
	suite := NewBenchmarkSuite(100*1024, 0.1) // 100KB
	blockSizes := []int{256, 512, 1024, 2048, 4096}

	for _, blockSize := range blockSizes {
		b.Run(fmt.Sprintf("block_%d", blockSize), func(b *testing.B) {
			options := &core.DiffOptions{
				Config: &config.Config{
					BlockSize:      blockSize,
					MinMatchLength: blockSize / 16,
					MaxWorkers:     4,
					UseParallel:    true,
					EnableFFT:      false,
				},
				ShowProgress: false,
				Context:      context.Background(),
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				patches := core.DiffWithOptions(suite.oldData, suite.newData, options)
				_ = patches
			}
		})
	}
}

// BenchmarkMemoryUsage 内存使用基准测试
func BenchmarkMemoryUsage(b *testing.B) {
	suite := NewBenchmarkSuite(5*1024*1024, 0.05) // 5MB, 5% 变化

	b.Run("memory_efficient", func(b *testing.B) {
		options := &core.DiffOptions{
			Config: &config.Config{
				BlockSize:      1024,
				MinMatchLength: 64,
				MaxMemoryMB:    16, // 限制内存使用
				MaxWorkers:     2,
				UseParallel:    true,
				EnableFFT:      false,
			},
			ShowProgress: false,
			Context:      context.Background(),
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			patches := core.DiffWithOptions(suite.oldData, suite.newData, options)
			_ = patches
		}
	})
}

// BenchmarkApplyPatchPerformance 补丁应用性能测试
func BenchmarkApplyPatchPerformance(b *testing.B) {
	sizes := []int{10 * 1024, 100 * 1024, 1024 * 1024}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			suite := NewBenchmarkSuite(size, 0.1)
			patches := core.Diff(suite.oldData, suite.newData)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result := core.ApplyPatch(suite.oldData, patches)
				_ = result
			}
		})
	}
}

// calculatePatchSize 计算补丁大小
func calculatePatchSize(patches []types.Patch) int {
	size := 0
	for _, patch := range patches {
		size += len(patch.Data) + 24 // 数据 + 头信息
	}
	return size
}

// BenchmarkCompressionRatio 压缩率基准测试
func BenchmarkCompressionRatio(b *testing.B) {
	suite := NewBenchmarkSuite(1024*1024, 0.1) // 1MB, 10% 变化

	b.Run("compression_analysis", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			patches := core.Diff(suite.oldData, suite.newData)
			patchSize := calculatePatchSize(patches)
			compressionRatio := float64(patchSize) / float64(len(suite.newData))
			_ = compressionRatio
		}
	})
}

// BenchmarkMultipleFiles 多文件处理性能测试
func BenchmarkMultipleFiles(b *testing.B) {
	fileCount := 10
	fileSize := 10 * 1024 // 10KB per file

	b.Run("concurrent_files", func(b *testing.B) {
		suites := make([]*BenchmarkSuite, fileCount)
		for i := 0; i < fileCount; i++ {
			suites[i] = NewBenchmarkSuite(fileSize, 0.05)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, suite := range suites {
				patches := core.Diff(suite.oldData, suite.newData)
				_ = patches
			}
		}
	})
}

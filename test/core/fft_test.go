package core_test

import (
	"bindiff/core"
	"fmt"
	"math"
	"math/cmplx"
	"testing"
)

// TestBasicFFT 测试基本 FFT 功能
func TestBasicFFT(t *testing.T) {
	// 测试小尺寸 FFT
	sizes := []int{2, 4, 8, 16, 32, 64}

	for _, n := range sizes {
		t.Run(fmt.Sprintf("size_%d", n), func(t *testing.T) {
			fft := core.NewFFT(n)

			// 创建测试输入（简单的脉冲信号）
			input := make([]complex128, n)
			input[0] = complex(1, 0)

			// 执行正向 FFT
			output := make([]complex128, n)
			fft.Transform(input, output, false)

			// 执行反向 FFT
			recovered := make([]complex128, n)
			fft.Transform(output, recovered, true)

			// 验证往返转换的准确性
			for i := 0; i < n; i++ {
				diff := cmplx.Abs(recovered[i] - input[i])
				if diff > 1e-10 {
					t.Errorf("Round-trip error at index %d: %v vs %v (diff: %e)",
						i, input[i], recovered[i], diff)
				}
			}
		})
	}
}

// TestFFTWithRealData 测试使用真实数据的 FFT
func TestFFTWithRealData(t *testing.T) {
	n := 16
	fft := core.NewFFT(n)

	// 创建正弦波信号
	input := make([]complex128, n)
	for i := 0; i < n; i++ {
		freq := 2.0 * math.Pi * 3.0 / float64(n) // 3 cycles
		input[i] = complex(math.Sin(freq*float64(i)), 0)
	}

	// 正向变换
	output := make([]complex128, n)
	fft.Transform(input, output, false)

	// 应该在频率 3 和 n-3 处有峰值
	expectedPeaks := []int{3, n - 3}
	for _, peak := range expectedPeaks {
		magnitude := cmplx.Abs(output[peak])
		if magnitude < 5.0 { // 期望的峰值幅度
			t.Errorf("Expected peak at frequency %d, got magnitude %f", peak, magnitude)
		}
	}
}

// TestIterativeFFT 测试迭代式 FFT 实现
func TestIterativeFFT(t *testing.T) {
	n := 32
	fft := core.NewFFT(n)

	// 创建随机测试数据
	input := make([]complex128, n)
	for i := 0; i < n; i++ {
		input[i] = complex(float64(i%7), float64(i%5))
	}

	// 使用标准 FFT 变换（内部使用迭代式实现）
	output1 := make([]complex128, n)
	fft.Transform(input, output1, false)

	// 验证结果的合理性（检查直流分量）
	dcComponent := output1[0]
	expectedDC := complex(0, 0)
	for _, val := range input {
		expectedDC += val
	}

	diff := cmplx.Abs(dcComponent - expectedDC)
	if diff > 1e-10 {
		t.Errorf("DC component mismatch: expected %v, got %v", expectedDC, dcComponent)
	}
}

// TestParallelFFT 测试并行 FFT
func TestParallelFFT(t *testing.T) {
	n := 1024
	fft := core.NewFFT(n)

	// 创建测试数据
	input := make([]complex128, n)
	for i := 0; i < n; i++ {
		input[i] = complex(math.Sin(2*math.Pi*float64(i)/64), 0)
	}

	// 串行版本
	serialOutput := make([]complex128, n)
	fft.Transform(input, serialOutput, false)

	// 并行版本
	parallelOutput := make([]complex128, n)
	fft.ParallelTransform(input, parallelOutput, false, 4)

	// 比较结果
	for i := 0; i < n; i++ {
		diff := cmplx.Abs(serialOutput[i] - parallelOutput[i])
		if diff > 1e-10 {
			t.Errorf("Parallel FFT mismatch at index %d: %e", i, diff)
		}
	}
}

// TestConvolutionFFT 测试 FFT 卷积
func TestConvolutionFFT(t *testing.T) {
	// 简单的卷积测试
	a := []complex128{complex(1, 0), complex(2, 0), complex(3, 0)}
	b := []complex128{complex(0.5, 0), complex(0.25, 0)}

	result := core.ConvolutionFFT(a, b)

	// 期望的卷积结果长度
	expectedLen := len(a) + len(b) - 1
	if len(result) != expectedLen {
		t.Errorf("Convolution result length: expected %d, got %d", expectedLen, len(result))
	}

	// 验证第一个元素（应该是 1 * 0.5 = 0.5）
	expected := complex(0.5, 0)
	if cmplx.Abs(result[0]-expected) > 1e-10 {
		t.Errorf("Convolution first element: expected %v, got %v", expected, result[0])
	}
}

// TestRealFFT 测试实数 FFT
func TestRealFFT(t *testing.T) {
	n := 16
	rfft := core.NewRealFFT(n)

	// 创建实数输入
	input := make([]float64, n)
	for i := 0; i < n; i++ {
		input[i] = math.Sin(2 * math.Pi * float64(i) / float64(n))
	}

	// 执行实数 FFT
	output := make([]complex128, n)
	rfft.Transform(input, output, false)

	// 验证厄米特对称性（实数 FFT 的特性）
	for i := 1; i < n/2; i++ {
		conjugate := cmplx.Conj(output[n-i])
		diff := cmplx.Abs(output[i] - conjugate)
		if diff > 1e-10 {
			t.Errorf("Hermitian symmetry violation at index %d: %e", i, diff)
		}
	}
}

// TestBitReverse 测试位反转
func TestBitReverse(t *testing.T) {
	testCases := []struct {
		num, bits, expected int
	}{
		{0, 3, 0}, // 000 -> 000
		{1, 3, 4}, // 001 -> 100
		{2, 3, 2}, // 010 -> 010
		{3, 3, 6}, // 011 -> 110
		{4, 3, 1}, // 100 -> 001
		{5, 3, 5}, // 101 -> 101
		{6, 3, 3}, // 110 -> 011
		{7, 3, 7}, // 111 -> 111
	}

	for _, tc := range testCases {
		result := core.ReverseBits(tc.num, tc.bits)
		if result != tc.expected {
			t.Errorf("reverseBits(%d, %d) = %d, expected %d",
				tc.num, tc.bits, result, tc.expected)
		}
	}
}

// TestFFTErrorHandling 测试 FFT 错误处理
func TestFFTErrorHandling(t *testing.T) {
	fft := core.NewFFT(8)

	// 测试输入长度不匹配
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for mismatched input length")
		}
	}()

	input := make([]complex128, 4) // 错误的长度
	output := make([]complex128, 8)
	fft.Transform(input, output, false)
}

// TestNextPowerOfTwo 测试下一个2的幂函数
func TestNextPowerOfTwo(t *testing.T) {
	testCases := []struct {
		input, expected int
	}{
		{0, 1},
		{1, 1},
		{2, 2},
		{3, 4},
		{4, 4},
		{5, 8},
		{8, 8},
		{9, 16},
		{1000, 1024},
		{1024, 1024},
		{1025, 2048},
	}

	for _, tc := range testCases {
		result := core.NextPowerOfTwo(tc.input)
		if result != tc.expected {
			t.Errorf("NextPowerOfTwo(%d) = %d, expected %d",
				tc.input, result, tc.expected)
		}
	}
}

// BenchmarkFFT 基准测试 FFT 性能
func BenchmarkFFT(b *testing.B) {
	sizes := []int{64, 256, 1024, 4096}

	for _, n := range sizes {
		b.Run(fmt.Sprintf("size_%d", n), func(b *testing.B) {
			fft := core.NewFFT(n)
			input := make([]complex128, n)
			output := make([]complex128, n)

			// 填充随机数据
			for i := 0; i < n; i++ {
				input[i] = complex(float64(i%100), float64(i%50))
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				fft.Transform(input, output, false)
			}
		})
	}
}

// BenchmarkParallelFFT 基准测试并行 FFT 性能
func BenchmarkParallelFFT(b *testing.B) {
	n := 4096
	fft := core.NewFFT(n)
	input := make([]complex128, n)
	output := make([]complex128, n)

	for i := 0; i < n; i++ {
		input[i] = complex(float64(i%100), float64(i%50))
	}

	workers := []int{1, 2, 4, 8}

	for _, numWorkers := range workers {
		b.Run(fmt.Sprintf("workers_%d", numWorkers), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				fft.ParallelTransform(input, output, false, numWorkers)
			}
		})
	}
}

// BenchmarkConvolutionFFT 基准测试 FFT 卷积性能
func BenchmarkConvolutionFFT(b *testing.B) {
	sizes := []int{64, 256, 1024}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			a := make([]complex128, size)
			b_data := make([]complex128, size/2)

			for i := range a {
				a[i] = complex(float64(i%100), 0)
			}
			for i := range b_data {
				b_data[i] = complex(float64(i%50), 0)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				core.ConvolutionFFT(a, b_data)
			}
		})
	}
}

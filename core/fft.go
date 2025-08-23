package core

import (
	"bindiff/pkg/logger"
	"fmt"
	"math"
	"math/cmplx"
	"sync"
)

// FFT 实现（优化版本）
type FFT struct {
	n          int
	roots      []complex128
	bitReverse []int
}

// FFTOptions FFT 配置选项
type FFTOptions struct {
	EnableCache bool
	Parallel    bool
	Threshold   int // 并行阈值
}

// DefaultFFTOptions 默认 FFT 配置
func DefaultFFTOptions() *FFTOptions {
	return &FFTOptions{
		EnableCache: true,
		Parallel:    true,
		Threshold:   1024,
	}
}

// NewFFT 创建 FFT 实例（优化版本）
func NewFFT(n int) *FFT {
	return NewFFTWithOptions(n, DefaultFFTOptions())
}

// NewFFTWithOptions 使用选项创建 FFT 实例
func NewFFTWithOptions(n int, options *FFTOptions) *FFT {
	if n <= 0 || (n&(n-1)) != 0 {
		logger.Warnf("FFT size %d is not a power of 2, performance may be suboptimal", n)
	}

	fft := &FFT{
		n:          n,
		roots:      make([]complex128, n),
		bitReverse: make([]int, n),
	}

	// 预计算旋转因子
	angle := 2 * math.Pi / float64(n)
	for i := 0; i < n; i++ {
		fft.roots[i] = cmplx.Rect(1, float64(i)*angle)
	}

	// 预计算位反转索引
	fft.precomputeBitReverse()

	return fft
}

// Transform FFT 变换（优化版本）
func (fft *FFT) Transform(input, output []complex128, inverse bool) {
	if len(input) != fft.n || len(output) != fft.n {
		panic(fmt.Sprintf("input/output length (%d, %d) must match FFT size (%d)",
			len(input), len(output), fft.n))
	}

	if fft.n == 1 {
		output[0] = input[0]
		return
	}

	// 使用迭代式 FFT 算法（更高效）
	fft.iterativeFFT(input, output, inverse)
}

// iterativeFFT 迭代式 FFT 实现
func (fft *FFT) iterativeFFT(input, output []complex128, inverse bool) {
	n := fft.n

	// 位反转重排
	for i := 0; i < n; i++ {
		output[i] = input[fft.bitReverse[i]]
	}

	// 迭代计算
	for length := 2; length <= n; length <<= 1 {
		half := length >> 1
		step := n / length

		// 选择合适的旋转因子
		var wlen complex128
		if inverse {
			wlen = fft.roots[n-step]
		} else {
			wlen = fft.roots[step]
		}

		for start := 0; start < n; start += length {
			w := complex(1, 0)
			for j := 0; j < half; j++ {
				u := output[start+j]
				v := output[start+j+half] * w
				output[start+j] = u + v
				output[start+j+half] = u - v
				w *= wlen
			}
		}
	}

	// 逆变换需要除以 n
	if inverse {
		scale := complex(1.0/float64(n), 0)
		for i := range output {
			output[i] *= scale
		}
	}
}

// precomputeBitReverse 计算位反转索引
func (fft *FFT) precomputeBitReverse() {
	n := fft.n
	logN := int(math.Log2(float64(n)))

	for i := 0; i < n; i++ {
		fft.bitReverse[i] = reverseBits(i, logN)
	}
}

// ReverseBits 导出的位反转函数
func ReverseBits(num, bits int) int {
	return reverseBits(num, bits)
}

// reverseBits 反转位
func reverseBits(num, bits int) int {
	result := 0
	for i := 0; i < bits; i++ {
		if (num>>i)&1 == 1 {
			result |= 1 << (bits - 1 - i)
		}
	}
	return result
}

// ParallelTransform 并行 FFT 变换
func (fft *FFT) ParallelTransform(input, output []complex128, inverse bool, numWorkers int) {
	if numWorkers <= 1 || fft.n < 1024 {
		fft.Transform(input, output, inverse)
		return
	}

	// 大数据集使用并行处理
	fft.parallelIterativeFFT(input, output, inverse, numWorkers)
}

// parallelIterativeFFT 并行迭代式 FFT
func (fft *FFT) parallelIterativeFFT(input, output []complex128, inverse bool, numWorkers int) {
	n := fft.n

	// 位反转重排
	for i := 0; i < n; i++ {
		output[i] = input[fft.bitReverse[i]]
	}

	var wg sync.WaitGroup

	// 迭代计算
	for length := 2; length <= n; length <<= 1 {
		half := length >> 1
		step := n / length

		var wlen complex128
		if inverse {
			wlen = fft.roots[n-step]
		} else {
			wlen = fft.roots[step]
		}

		// 并行处理不同的段
		chunkSize := (n / length) / numWorkers
		if chunkSize == 0 {
			chunkSize = 1
		}

		for workerStart := 0; workerStart < n; workerStart += chunkSize * length {
			wg.Add(1)
			go func(start int) {
				defer wg.Done()
				end := start + chunkSize*length
				if end > n {
					end = n
				}

				for chunkStart := start; chunkStart < end; chunkStart += length {
					w := complex(1, 0)
					for j := 0; j < half; j++ {
						u := output[chunkStart+j]
						v := output[chunkStart+j+half] * w
						output[chunkStart+j] = u + v
						output[chunkStart+j+half] = u - v
						w *= wlen
					}
				}
			}(workerStart)
		}

		wg.Wait()
	}

	// 逆变换归一化
	if inverse {
		scale := complex(1.0/float64(n), 0)
		for i := range output {
			output[i] *= scale
		}
	}
}

// ConvolutionFFT FFT 卷积
func ConvolutionFFT(a, b []complex128) []complex128 {
	lenA, lenB := len(a), len(b)
	n := NextPowerOfTwo(lenA + lenB - 1)

	// 扩展到合适的大小
	paddedA := make([]complex128, n)
	paddedB := make([]complex128, n)
	copy(paddedA, a)
	copy(paddedB, b)

	// 创建 FFT 实例
	fft := NewFFT(n)

	// 正向 FFT
	fftA := make([]complex128, n)
	fftB := make([]complex128, n)
	fft.Transform(paddedA, fftA, false)
	fft.Transform(paddedB, fftB, false)

	// 逐点相乘
	product := make([]complex128, n)
	for i := 0; i < n; i++ {
		product[i] = fftA[i] * fftB[i]
	}

	// 逆向 FFT
	result := make([]complex128, n)
	fft.Transform(product, result, true)

	// 返回有效长度的结果
	return result[:lenA+lenB-1]
}

// RealFFT 实数 FFT（更高效）
type RealFFT struct {
	n    int
	fft  *FFT
	temp []complex128
}

// NewRealFFT 创建实数 FFT
func NewRealFFT(n int) *RealFFT {
	return &RealFFT{
		n:    n,
		fft:  NewFFT(n),
		temp: make([]complex128, n),
	}
}

// Transform 实数变换
func (rfft *RealFFT) Transform(input []float64, output []complex128, inverse bool) {
	if len(input) != rfft.n {
		panic("input length must match RealFFT size")
	}

	// 将实数转换为复数
	for i, val := range input {
		rfft.temp[i] = complex(val, 0)
	}

	// 执行复数 FFT
	rfft.fft.Transform(rfft.temp, output, inverse)
}

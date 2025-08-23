package core


// 计算两个二进制数据的最佳对齐偏移量
func ComputeOffset(oldData, newData []byte) int {
	lenA := len(oldData)
	lenB := len(newData)
	
	// 确定FFT大小
	n := NextPowerOfTwo(lenA + lenB - 1)
	
	// 准备FFT输入
	fft := NewFFT(n)
	a := make([]complex128, n)
	b := make([]complex128, n)
	
	for i := 0; i < lenA; i++ {
		a[i] = complex(float64(oldData[i]), 0)
	}
	
	// 翻转新数据
	for i := 0; i < lenB; i++ {
		b[i] = complex(float64(newData[lenB-1-i]), 0)
	}
	
	// 计算FFT
	aFFT := make([]complex128, n)
	bFFT := make([]complex128, n)
	fft.Transform(a, aFFT, false)
	fft.Transform(b, bFFT, false)
	
	// 点乘
	product := make([]complex128, n)
	for i := range aFFT {
		product[i] = aFFT[i] * bFFT[i]
	}
	
	// 逆FFT
	corr := make([]complex128, n)
	fft.Transform(product, corr, true)
	
	// 找到最大相关值的位置
	maxVal := real(corr[0])
	maxIdx := 0
	for i := 1; i < n; i++ {
		val := real(corr[i])
		if val > maxVal {
			maxVal = val
			maxIdx = i
		}
	}
	
	// 计算偏移量
	offset := maxIdx - lenB + 1
	if offset < -lenB+1 {
		offset += n
	}
	
	return offset
}
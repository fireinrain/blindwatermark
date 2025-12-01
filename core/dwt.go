package core

import (
	_ "math"
)

// Sqrt2 用于归一化
const Sqrt2 = 1.41421356237

// DWT2D 对矩阵进行一次二维 Haar 小波变换
// 返回转换后的矩阵，其中：
// 左上: LL (低频近似)
// 右上: HL (水平细节 - 适合嵌入)
// 左下: LH (垂直细节 - 适合嵌入)
// 右下: HH (对角细节 - 噪点多，不适合)
func DWT2D(matrix [][]float64) [][]float64 {
	h := len(matrix)
	w := len(matrix[0])

	// 1. 行变换 (Row Transform)
	temp := make([][]float64, h)
	for i := 0; i < h; i++ {
		temp[i] = dwt1D(matrix[i])
	}

	// 2. 列变换 (Col Transform)
	// 需要先转置或者按列读，这里为了简单，分配结果矩阵直接填入
	output := make([][]float64, h)
	for i := range output {
		output[i] = make([]float64, w)
	}

	//halfH := h / 2
	for j := 0; j < w; j++ {
		// 提取这一列
		col := make([]float64, h)
		for i := 0; i < h; i++ {
			col[i] = temp[i][j]
		}
		// 做 1D 变换
		transCol := dwt1D(col)
		// 填回 output
		for i := 0; i < h; i++ {
			output[i][j] = transCol[i]
		}
	}

	// 此时 output 的四个象限已经是 LL, HL, LH, HH
	// 行的上半部分是 L，下半部分是 H (因为 dwt1D 把 L 放前，H 放后)
	// 但通常图像处理习惯将 LL 放在左上角。
	// 我们的 dwt1D 逻辑是: [L..., H...]，所以经过行列变换后：
	// 行(L, H) -> 列(L, H) -> 结果自然就是:
	// LL HL
	// LH HH
	return output
}

// IDWT2D 二维离散小波逆变换
func IDWT2D(matrix [][]float64) [][]float64 {
	h := len(matrix)
	w := len(matrix[0])

	// 1. 列逆变换
	temp := make([][]float64, h)
	for i := range temp {
		temp[i] = make([]float64, w)
	}

	for j := 0; j < w; j++ {
		col := make([]float64, h)
		for i := 0; i < h; i++ {
			col[i] = matrix[i][j]
		}
		origCol := idwt1D(col)
		for i := 0; i < h; i++ {
			temp[i][j] = origCol[i]
		}
	}

	// 2. 行逆变换
	output := make([][]float64, h)
	for i := 0; i < h; i++ {
		output[i] = idwt1D(temp[i])
	}

	return output
}

// dwt1D 一维 Haar 变换
// 输入长度必须是偶数
func dwt1D(data []float64) []float64 {
	n := len(data)
	half := n / 2
	output := make([]float64, n)

	for i := 0; i < half; i++ {
		// Haar 公式:
		// L = (a + b) / sqrt(2)
		// H = (a - b) / sqrt(2)
		output[i] = (data[2*i] + data[2*i+1]) / Sqrt2      // Low freq 部分放在前半段
		output[half+i] = (data[2*i] - data[2*i+1]) / Sqrt2 // High freq 部分放在后半段
	}
	return output
}

// idwt1D 一维 Haar 逆变换
func idwt1D(data []float64) []float64 {
	n := len(data)
	half := n / 2
	output := make([]float64, n)

	for i := 0; i < half; i++ {
		// 逆公式:
		// a = (L + H) / sqrt(2)  <-- 因为之前除了 sqrt(2)，逆变换要乘回去吗？
		// Haar 正交变换通常正逆都是除以 sqrt(2)，或者一个除2一个不除。
		// 这里采用正交归一化：
		// a = (L + H) / sqrt(2)
		// b = (L - H) / sqrt(2)
		L := data[i]
		H := data[half+i]

		output[2*i] = (L + H) / Sqrt2
		output[2*i+1] = (L - H) / Sqrt2
	}
	return output
}

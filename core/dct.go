package core

import (
	"math"
)

const N = 8 // 块大小 8x8

// SimpleDCT 简单的二维离散余弦变换
// 输入 8x8 空间域矩阵，输出 8x8 频域矩阵
func SimpleDCT(block [][]float64) [][]float64 {
	result := make([][]float64, N)
	for i := range result {
		result[i] = make([]float64, N)
	}

	for u := 0; u < N; u++ {
		for v := 0; v < N; v++ {
			sum := 0.0
			for x := 0; x < N; x++ {
				for y := 0; y < N; y++ {
					sum += block[x][y] *
						math.Cos((2*float64(x)+1)*float64(u)*math.Pi/(2*N)) *
						math.Cos((2*float64(y)+1)*float64(v)*math.Pi/(2*N))
				}
			}
			result[u][v] = c(u) * c(v) * sum / 4.0 // 4.0 = sqrt(2/N)^2 * something... standard normalization
		}
	}
	return result
}

// SimpleIDCT 简单的二维逆离散余弦变换
func SimpleIDCT(coeff [][]float64) [][]float64 {
	result := make([][]float64, N)
	for i := range result {
		result[i] = make([]float64, N)
	}

	for x := 0; x < N; x++ {
		for y := 0; y < N; y++ {
			sum := 0.0
			for u := 0; u < N; u++ {
				for v := 0; v < N; v++ {
					sum += c(u) * c(v) * coeff[u][v] *
						math.Cos((2*float64(x)+1)*float64(u)*math.Pi/(2*N)) *
						math.Cos((2*float64(y)+1)*float64(v)*math.Pi/(2*N))
				}
			}
			result[x][y] = sum / 4.0
		}
	}
	return result
}

func c(k int) float64 {
	if k == 0 {
		return 1.0 / math.Sqrt(2)
	}
	return 1.0
}

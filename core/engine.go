package core

import (
	"image"
	"image/color"
)

// Engine 负责具体的嵌入和提取逻辑
type Engine struct {
	Strength float64 // 水印强度 (Alpha)
}

// Embed 将 bits 嵌入到 img 中
// Embed 将 bits 嵌入到 img 中 (DWT + DCT 版)
func (e *Engine) Embed(img image.Image, bits []bool) image.Image {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// DWT 要求宽高必须是偶数，且最好是 2 的幂。为了简单，我们裁剪到偶数
	w := width - (width % 2)
	h := height - (height % 2)

	// 1. 提取 Y 通道 (整张图)
	yMatrix := make([][]float64, h)
	for i := 0; i < h; i++ {
		yMatrix[i] = make([]float64, w)
		for j := 0; j < w; j++ {
			r, g, b, _ := img.At(j, i).RGBA()
			yMatrix[i][j] = 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
		}
	}

	// 2. 全局 DWT 变换
	dwtMatrix := DWT2D(yMatrix)

	// 3. 选择 HL 区域 (右上角) 进行嵌入
	// HL 的区域范围：行 [0, h/2), 列 [w/2, w)
	halfH := h / 2
	halfW := w / 2

	// 我们把 HL 区域当做一个新的“图像”，在里面做 8x8 DCT
	// 遍历 HL 区域内的 8x8 块
	bitIdx := 0
	maxBits := len(bits)

	for i := 0; i <= halfH-N; i += N {
		for j := halfW; j <= w-N; j += N {
			if bitIdx >= maxBits {
				break
			}

			// 3.1 从 DWT 矩阵中取出 8x8 块
			block := make([][]float64, N)
			for bi := 0; bi < N; bi++ {
				block[bi] = make([]float64, N)
				for bj := 0; bj < N; bj++ {
					block[bi][bj] = dwtMatrix[i+bi][j+bj]
				}
			}

			// 3.2 DCT 变换
			dctBlock := SimpleDCT(block)

			// 3.3 修改系数嵌入 (同之前逻辑)
			v1 := dctBlock[4][3]
			v2 := dctBlock[3][4]
			bit := bits[bitIdx]

			if bit {
				if v1-v2 < e.Strength {
					diff := (e.Strength - (v1 - v2)) / 2.0
					v1 += diff
					v2 -= diff
				}
			} else {
				if v2-v1 < e.Strength {
					diff := (e.Strength - (v2 - v1)) / 2.0
					v2 += diff
					v1 -= diff
				}
			}
			dctBlock[4][3] = v1
			dctBlock[3][4] = v2

			// 3.4 IDCT
			idctBlock := SimpleIDCT(dctBlock)

			// 3.5 填回 DWT 矩阵 (注意：填回 HL 区域)
			for bi := 0; bi < N; bi++ {
				for bj := 0; bj < N; bj++ {
					dwtMatrix[i+bi][j+bj] = idctBlock[bi][bj]
				}
			}

			bitIdx++
		}
	}

	// 4. 全局 IDWT 反变换
	reconstructedY := IDWT2D(dwtMatrix)

	// 5. 合成最终图片
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < h; i++ {
		for j := 0; j < w; j++ {
			origR, origG, origB, _ := img.At(j, i).RGBA()

			oldY := yMatrix[i][j]
			newY := reconstructedY[i][j]
			diff := newY - oldY // 算出 Y 的变化量

			// 将变化量加回 RGB
			r := clamp(float64(origR>>8) + diff)
			g := clamp(float64(origG>>8) + diff)
			b := clamp(float64(origB>>8) + diff)

			out.Set(j, i, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}

	return out
}

// Extract 从图片中提取 bits
func (e *Engine) Extract(img image.Image) []bool {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	w := width - (width % 2)
	h := height - (height % 2)

	// 1. 提取 Y
	yMatrix := make([][]float64, h)
	for i := 0; i < h; i++ {
		yMatrix[i] = make([]float64, w)
		for j := 0; j < w; j++ {
			r, g, b, _ := img.At(j, i).RGBA()
			yMatrix[i][j] = 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
		}
	}

	// 2. DWT
	dwtMatrix := DWT2D(yMatrix)

	// 3. 遍历 HL 区域 (右上)
	halfH := h / 2
	halfW := w / 2
	var bits []bool

	for i := 0; i <= halfH-N; i += N {
		for j := halfW; j <= w-N; j += N {
			// 提取 8x8
			block := make([][]float64, N)
			for bi := 0; bi < N; bi++ {
				block[bi] = make([]float64, N)
				for bj := 0; bj < N; bj++ {
					block[bi][bj] = dwtMatrix[i+bi][j+bj]
				}
			}

			// DCT
			dctBlock := SimpleDCT(block)

			// 比较
			v1 := dctBlock[4][3]
			v2 := dctBlock[3][4]

			if v1 >= v2 {
				bits = append(bits, true)
			} else {
				bits = append(bits, false)
			}
		}
	}
	return bits
}

func clamp(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

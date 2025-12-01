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
func (e *Engine) Embed(img image.Image, bits []bool) image.Image {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// 创建一个新的 RGBA 图片用于输出
	out := image.NewRGBA(bounds)

	// 简单复制原图
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			out.Set(x, y, img.At(x, y))
		}
	}

	bitIdx := 0
	maxBits := len(bits)

	// 遍历 8x8 的块
	for y := 0; y <= height-N; y += N {
		for x := 0; x <= width-N; x += N {
			if bitIdx >= maxBits {
				break
			}

			// 1. 提取 Y 通道 (亮度)
			block := make([][]float64, N)
			for i := 0; i < N; i++ {
				block[i] = make([]float64, N)
				for j := 0; j < N; j++ {
					r, g, b, _ := img.At(x+j, y+i).RGBA()
					// RGB 转 YUV 的 Y 分量
					yy := 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
					block[i][j] = yy
				}
			}

			// 2. DCT 变换
			dctBlock := SimpleDCT(block)

			// 3. 修改中频系数 (位置 [4][3] 和 [3][4]) 嵌入 bit
			// 简单的嵌入逻辑：
			// 如果 bit=1，确保 A > B + Strength
			// 如果 bit=0，确保 B > A + Strength
			v1 := dctBlock[4][3]
			v2 := dctBlock[3][4]
			bit := bits[bitIdx]

			if bit {
				// 需要 v1 > v2
				if v1-v2 < e.Strength {
					diff := (e.Strength - (v1 - v2)) / 2.0
					v1 += diff
					v2 -= diff
				}
			} else {
				// 需要 v2 > v1
				if v2-v1 < e.Strength {
					diff := (e.Strength - (v2 - v1)) / 2.0
					v2 += diff
					v1 -= diff
				}
			}
			dctBlock[4][3] = v1
			dctBlock[3][4] = v2

			// 4. IDCT 反变换
			idctBlock := SimpleIDCT(dctBlock)

			// 5. 写回图片 (还需要处理溢出和颜色转换)
			for i := 0; i < N; i++ {
				for j := 0; j < N; j++ {
					origR, origG, origB, _ := img.At(x+j, y+i).RGBA()
					// 仅使用简单的 Y 替换是不够完美的，这里简化处理：
					// 算出新的 Y 和旧的 Y 的差值，应用到 RGB 上
					oldY := block[i][j]
					newY := idctBlock[i][j]
					diff := newY - oldY

					r := clamp(float64(origR>>8) + diff)
					g := clamp(float64(origG>>8) + diff)
					b := clamp(float64(origB>>8) + diff)

					out.Set(x+j, y+i, color.RGBA{R: r, G: g, B: b, A: 255})
				}
			}

			bitIdx++
		}
	}

	return out
}

// Extract 从图片中提取 bits
func (e *Engine) Extract(img image.Image) []bool {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	var bits []bool

	for y := 0; y <= height-N; y += N {
		for x := 0; x <= width-N; x += N {
			// 1. 提取 Y
			block := make([][]float64, N)
			for i := 0; i < N; i++ {
				block[i] = make([]float64, N)
				for j := 0; j < N; j++ {
					r, g, b, _ := img.At(x+j, y+i).RGBA()
					block[i][j] = 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
				}
			}

			// 2. DCT
			dctBlock := SimpleDCT(block)

			// 3. 比较系数
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

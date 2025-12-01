package blindwatermark

import (
	"blindwatermark/converter"
	"blindwatermark/core"
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"

	"golang.org/x/image/draw"

	"github.com/skip2/go-qrcode"
)

type BlindWatermarker struct {
	engine *core.Engine
}

func NewBlindWatermarker() *BlindWatermarker {
	return &BlindWatermarker{
		engine: &core.Engine{Strength: 20.0}, // 强度越大越抗干扰，但画质损失越大
	}
}

// Result 提取结果
type Result struct {
	Type        converter.WatermarkType
	TextContent string
	ImageBytes  []byte // 如果是图片或二维码，存储原始字节
}

// 1. 嵌入字符串
func (b *BlindWatermarker) EmbedText(src image.Image, text string) (image.Image, error) {
	// Pack: [Type:Text] [Len] [TextData]
	bits := converter.Pack(converter.TypeText, []byte(text))
	return b.embed(src, bits)
}

// EmbedImage 3. 嵌入图片水印
// watermark.go

// 3. 嵌入图片水印 (优化版：转为 1-bit 二值化数据存储)
// 3. 嵌入图片水印 (支持动态尺寸)
func (b *BlindWatermarker) EmbedImage(src image.Image, wmImage image.Image) (image.Image, error) {
	wmImage = ConvertToGray(wmImage)
	// --- 新增逻辑：检查容量并自动缩放 ---
	// 1. 计算底图的最大容量
	subWidth := src.Bounds().Dx() / 2
	subHeight := src.Bounds().Dy() / 2
	maxCapacityBits := (subWidth / 8) * (subHeight / 8)
	// 预留一些 header 空间 (比如 64 bits)
	maxPixels := maxCapacityBits - 64

	// 2. 获取当前水印尺寸
	w := wmImage.Bounds().Dx()
	h := wmImage.Bounds().Dy()

	// 3. 如果水印太大，进行缩放
	if w*h > maxPixels {
		fmt.Printf("⚠️ 水印过大 (%dx%d, %d pixels)，底图容量仅为 %d pixels。正在自动缩小...\n", w, h, w*h, maxPixels)

		// 计算缩放比例
		ratio := math.Sqrt(float64(maxPixels) / float64(w*h))
		newW := int(float64(w) * ratio)
		newH := int(float64(h) * ratio)

		// 缩放图片
		dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
		draw.CatmullRom.Scale(dst, dst.Rect, wmImage, wmImage.Bounds(), draw.Over, nil)

		// 更新 w, h 和 wmImage
		wmImage = dst
		w = newW
		h = newH
		fmt.Printf("✅ 水印已缩小为: %dx%d\n", w, h)
	}

	// 限制一下最大尺寸，防止溢出 uint16 (65535)
	if w > 65535 || h > 65535 {
		return nil, fmt.Errorf("watermark image too large")
	}

	// 计算像素数据长度
	// 向上取整：(w*h + 7) / 8
	pixelLen := (w*h + 7) / 8

	// 创建 Payload
	// 2 bytes 宽 + 2 bytes 高 + 像素数据
	payload := make([]byte, 2+2+pixelLen)

	// 1. 写入宽和高 (使用 BigEndian)
	binary.BigEndian.PutUint16(payload[0:2], uint16(w))
	binary.BigEndian.PutUint16(payload[2:4], uint16(h))

	// 2. 写入二值化像素数据
	// 注意：payload 的像素部分从第 4 个字节开始 (索引 4)
	pixelData := payload[4:]

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bb, _ := wmImage.At(x, y).RGBA()
			lum := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(bb)

			if lum/256 > 128 { // 判定为白色
				globalIdx := y*w + x // 注意这里乘的是 w (动态宽度)
				byteIdx := globalIdx / 8
				bitIdx := globalIdx % 8

				pixelData[byteIdx] |= 1 << (7 - bitIdx)
			}
		}
	}

	fmt.Printf("嵌入动态尺寸图片: %dx%d, 总数据量: %d bytes\n", w, h, len(payload))

	// 3. 打包并嵌入
	bits := converter.Pack(converter.TypeImage, payload)
	return b.embed(src, bits)
}

// 2. 嵌入二维码 (优化版：只存文本，不存图片文件)
func (b *BlindWatermarker) EmbedQRCode(src image.Image, content string) (image.Image, error) {
	// 关键修改：我们不再生成 PNG 图片存进去，而是直接存字符串
	// 但是我们要用 converter.TypeQRCode 标记它，这样提取时我们就知道把它还原成图片

	// Pack: [Type:QRCode] [Len] [ContentString]
	bits := converter.Pack(converter.TypeQRCode, []byte(content))

	return b.embed(src, bits)
}

// 内部嵌入逻辑，检查容量
func (b *BlindWatermarker) embed(src image.Image, bits []bool) (image.Image, error) {
	// DWT 版本容量计算修正：
	// 我们只在 HL 频带嵌入，它是原图宽高的 1/2，所以面积是 1/4。
	// 每个 8x8 的块存 1 bit。

	// 计算 DWT 后子带的大小
	subWidth := src.Bounds().Dx() / 2
	subHeight := src.Bounds().Dy() / 2

	// 计算子带中有多少个 8x8 的块
	capacity := (subWidth / 8) * (subHeight / 8)

	fmt.Printf("当前图片水印容量: %d bits, 待写入数据: %d bits\n", capacity, len(bits)) // 方便调试

	if len(bits) > capacity {
		return nil, fmt.Errorf("image is too small to hold this watermark. Capacity: %d bits, Need: %d bits", capacity, len(bits))
	}

	return b.engine.Embed(src, bits), nil
}

// watermark.go

// 3. 提取并自动识别
func (b *BlindWatermarker) Extract(watermarkedImg image.Image) (*Result, error) {
	rawBits := b.engine.Extract(watermarkedImg)

	wmType, data, err := converter.Unpack(rawBits)
	if err != nil {
		return nil, err
	}

	res := &Result{
		Type: wmType,
	}

	switch wmType {
	case converter.TypeText:
		res.TextContent = string(data)

	case converter.TypeImage:
		// 至少要有 4 个字节存宽高
		if len(data) < 4 {
			res.ImageBytes = data
			fmt.Println("Error: Data too short to contain dimensions")
			break
		}

		// 1. 读取宽和高
		w := int(binary.BigEndian.Uint16(data[0:2]))
		h := int(binary.BigEndian.Uint16(data[2:4]))

		fmt.Printf("提取到图片尺寸信息: %dx%d\n", w, h)

		// 2. 校验数据长度是否匹配
		expectedPixelLen := (w*h + 7) / 8
		actualPixelLen := len(data) - 4

		if actualPixelLen != expectedPixelLen {
			// 允许最后多一点点 padding bit，但不能少
			if actualPixelLen < expectedPixelLen {
				fmt.Printf("Warning: Data incomplete. Expected %d pixels, got %d bytes\n", expectedPixelLen, actualPixelLen)
			}
		}

		// 3. 重建图片
		img := image.NewRGBA(image.Rect(0, 0, w, h)) // 使用提取出的 w, h
		black := color.RGBA{0, 0, 0, 255}
		white := color.RGBA{255, 255, 255, 255}

		pixelData := data[4:] // 像素数据从第 4 字节开始

		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				globalIdx := y*w + x
				byteIdx := globalIdx / 8
				bitIdx := globalIdx % 8

				// 防止数组越界（虽然前面校验过）
				if byteIdx < len(pixelData) {
					isWhite := (pixelData[byteIdx]>>(7-bitIdx))&1 == 1
					if isWhite {
						img.Set(x, y, white)
					} else {
						img.Set(x, y, black)
					}
				}
			}
		}

		var buf bytes.Buffer
		png.Encode(&buf, img)
		res.ImageBytes = buf.Bytes()
	case converter.TypeQRCode:
		// 关键修改：提取到的是文本数据
		content := string(data)
		res.TextContent = content

		// 核心逻辑：检测到是 QRCode 类型，帮用户把图片“画”出来
		// 这样用户依然得到了一张二维码图片，但我们只占用了文本的空间
		qrPng, err := qrcode.Encode(content, qrcode.Medium, 256)
		if err != nil {
			// 如果生成失败，至少返回文本
			fmt.Printf("Warning: Failed to regenerate QR image: %v\n", err)
		} else {
			res.ImageBytes = qrPng
		}
	}

	return res, nil
}

// 将生成的图片字节保存为图片
func (b *BlindWatermarker) SaveImgFile(name string, img image.Image) {
	f, _ := os.Create(name)
	defer f.Close()
	// 使用 jpeg 保存，设置质量为 100 以尽量减少压缩带来的水印损失
	// 注意：实际盲水印需要抵抗压缩，这里为了演示简单使用了基本 DCT
	jpeg.Encode(f, img, &jpeg.Options{Quality: 100})
}

// ConvertToGray 将任意图片转换为 8位灰度图
func ConvertToGray(src image.Image) *image.Gray {
	// 获取源图片的尺寸
	bounds := src.Bounds()

	// 创建一个新的灰度图对象
	grayImg := image.NewGray(bounds)

	// 使用 draw.Draw 自动将源图片绘制到灰度图上
	// Go 的 draw 包会自动处理颜色模型转换 (RGB -> Gray)
	draw.Draw(grayImg, bounds, src, bounds.Min, draw.Src)

	return grayImg
}

// ConvertToBinary 将图片转换为二值图 (只有纯黑和纯白)
// threshold: 阈值 (0-255)，通常设为 128。大于此值为白，小于此值为黑。
func ConvertToBinary(src image.Image, threshold uint8) *image.Gray {
	bounds := src.Bounds()
	grayImg := image.NewGray(bounds)

	// 纯黑与纯白
	black := color.Gray{Y: 0}
	white := color.Gray{Y: 255}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// 1. 获取原始像素
			oldColor := src.At(x, y)

			// 2. 转为 RGBA
			r, g, b, _ := oldColor.RGBA()

			// 3. 计算亮度 (Luminance 公式: 0.299R + 0.587G + 0.114B)
			// 注意：RGBA() 返回的是 16bit (0-65535)，所以需要除以 256 归一化到 8bit
			lum := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
			pixelGray := uint8(lum / 256)

			// 4. 根据阈值二值化
			if pixelGray > threshold {
				grayImg.SetGray(x, y, white)
			} else {
				grayImg.SetGray(x, y, black)
			}
		}
	}
	return grayImg
}

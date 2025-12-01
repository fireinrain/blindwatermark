package blindwatermark

import (
	"blindwatermark/converter"
	"blindwatermark/core"
	"errors"
	"image"
	"image/jpeg"
	"os"

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

// 2. 嵌入二维码
func (b *BlindWatermarker) EmbedQRCode(src image.Image, content string) (image.Image, error) {
	// 生成二维码图片字节流
	qrPng, err := qrcode.Encode(content, qrcode.Medium, 256)
	if err != nil {
		return nil, err
	}

	// Pack: [Type:QRCode] [Len] [PNGData]
	bits := converter.Pack(converter.TypeQRCode, qrPng)
	return b.embed(src, bits)
}

// 内部嵌入逻辑，检查容量
func (b *BlindWatermarker) embed(src image.Image, bits []bool) (image.Image, error) {
	// 计算容量: 每个 8x8 块存 1 bit
	capacity := (src.Bounds().Dx() / 8) * (src.Bounds().Dy() / 8)
	if len(bits) > capacity {
		return nil, errors.New("image is too small to hold this watermark")
	}

	return b.engine.Embed(src, bits), nil
}

// 3. 提取并自动识别
func (b *BlindWatermarker) Extract(watermarkedImg image.Image) (*Result, error) {
	// 提取所有 bits
	rawBits := b.engine.Extract(watermarkedImg)

	// 解析协议头
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
	case converter.TypeQRCode:
		// 这里返回二维码的图片文件数据
		// 调用者可以将 res.ImageBytes 保存为 .png 文件，然后扫码
		res.ImageBytes = data
		res.TextContent = "Contains QRCode Image"
	case converter.TypeImage:
		res.ImageBytes = data
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

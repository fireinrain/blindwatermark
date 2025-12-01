package main

import (
	"blindwatermark"
	"blindwatermark/converter"
	"fmt"
	"image"
	_ "image/png"
	"os"
)

func main() {
	// 1. 打开原始图片
	file, _ := os.Open("dist/source.png") // 确保你有一张 source.jpg，最好大于 800x600
	defer file.Close()
	srcImg, _, _ := image.Decode(file)

	bw := blindwatermark.NewBlindWatermarker()

	// 2. 嵌入字符串
	fmt.Println("正在嵌入字符串水印...")
	resImg, err := bw.EmbedText(srcImg, "Hello Golang Watermark!")
	if err != nil {
		panic(err)
	}

	// 保存加了水印的图
	bw.SaveImgFile("dist/output_text.jpg", resImg)

	// 3. 提取水印
	fmt.Println("正在提取水印...")
	// 重新打开（模拟从网络下载）
	outFile, _ := os.Open("dist/output_text.jpg")
	watermarkedImg, _, _ := image.Decode(outFile)

	result, err := bw.Extract(watermarkedImg)
	if err != nil {
		fmt.Println("提取失败:", err)
		return
	}

	// 4. 自动识别结果
	fmt.Printf("检测到水印类型: %d (1=Text, 2=Img, 3=QR)\n", result.Type)
	if result.Type == converter.TypeText {
		fmt.Printf("提取内容: %s\n", result.TextContent)
	}

	// --- 演示二维码 ---
	fmt.Println("\n正在嵌入二维码水印...")
	resQrImg, err := bw.EmbedQRCode(srcImg, "https://github.com/golang")
	if err != nil {
		panic(err)
	}
	bw.SaveImgFile("dist/output_qr.jpg", resQrImg)

	fmt.Println("正在提取二维码水印...")
	outQrFile, _ := os.Open("dist/output_qr.jpg")
	wmQrImg, _, _ := image.Decode(outQrFile)
	qrResult, _ := bw.Extract(wmQrImg)

	if qrResult.Type == converter.TypeQRCode {
		fmt.Printf("提取到二维码，大小: %d bytes. 已保存为 extracted_qr.png\n", len(qrResult.ImageBytes))
		os.WriteFile("dist/extracted_qr.png", qrResult.ImageBytes, 0644)
	}
}

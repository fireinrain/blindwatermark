package main

import (
	"blindwatermark"
	"blindwatermark/converter"
	"fmt"
	"image"
	_ "image/jpeg"
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
	outQrFile, err := os.Open("dist/output_qr.jpg") // 检查文件是否打开成功
	if err != nil {
		fmt.Printf("❌ 打开文件失败: %v\n", err)
		return
	}
	defer outQrFile.Close()

	// ✅ 修复点：严查 Decode 错误
	wmQrImg, _, err := image.Decode(outQrFile)
	if err != nil {
		fmt.Printf("❌ 图片解码失败 (可能文件损坏或格式未支持): %v\n", err)
		return // 停止执行，防止 panic
	}

	qrResult, err := bw.Extract(wmQrImg)
	if err != nil {
		fmt.Printf("❌ 提取过程出错: %v\n", err)
		return
	}

	// 只有 err 为 nil 时，qrResult 才有值
	if qrResult.Type == converter.TypeQRCode {
		fmt.Printf("提取到二维码，大小: %d bytes. 已保存为 extracted_qr.png\n", len(qrResult.ImageBytes))
		os.WriteFile("dist/extracted_qr.png", qrResult.ImageBytes, 0644)
	}

	//嵌入图片
	fmt.Println("正在嵌入图片水印...")
	//读取水印图片
	// 打开水印图片
	wmImgFile, err := os.Open("dist/watermark.png")
	if err != nil {
		panic(err)
	}
	defer wmImgFile.Close()
	wmImg, _, err := image.Decode(wmImgFile)
	if err != nil {
		panic(err)
	}
	resImg, err = bw.EmbedImage(srcImg, wmImg)
	if err != nil {
		panic(err)
	}
	bw.SaveImgFile("dist/output_img.jpg", resImg)

	fmt.Println("\n--- 正在提取图片水印 ---")

	// 1. 读取带水印的图片
	encodedFile, _ := os.Open("dist/output_img.jpg")
	encodedImg, _, err := image.Decode(encodedFile)
	if err != nil {
		panic(err)
	}

	// 2. 执行提取
	result, err = bw.Extract(encodedImg)
	if err != nil {
		fmt.Printf("提取失败: %v\n", err)
		return
	}

	// 3. 处理结果
	if result.Type == converter.TypeImage {
		fmt.Printf("✅ 识别成功！发现嵌入了图片水印。\n")
		// 4. 保存文件
		// result.ImageBytes 里已经是我们刚刚重建好的 PNG 数据了
		outputName := "dist/extracted_secret.png"
		err := os.WriteFile(outputName, result.ImageBytes, 0644)
		if err != nil {
			panic(err)
		}

		fmt.Printf("🎉 提取出的图片已保存为: %s\n", outputName)
		fmt.Println("请打开该文件查看，它应该是一个 32x32 的黑白像素图。")
	} else {
		fmt.Println("未检测到图片水印，检测到的类型是:", result.Type)
	}
}

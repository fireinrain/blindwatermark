

# Go-BlindWatermark

[](https://golang.org)
[](https://www.google.com/search?q=LICENSE)

**Go-BlindWatermark** 是一个用 Golang 编写的频域盲水印库。它允许你将信息（文本、图片或二维码）隐写到图片中，这些信息在视觉上是不可见的，并且可以在不需要原始图片的情况下提取出来。

该项目深受 Python 项目 [BlindWatermark](https://github.com/fire-keeper/BlindWatermark) 的启发，并针对 Golang 特性进行了移植和优化，特别引入了 **自定义协议头**，实现了提取时**自动识别水印类型**的功能。

## ✨ 功能特性

* **隐蔽性强**：基于频域（Block-DCT）算法，水印对人眼不可见。
* **多类型支持**：
    * 📝 **字符串水印**：直接嵌入文本信息。
    * 🖼️ **图片水印**：嵌入一张微缩图片。
    * 📱 **二维码水印**：自动将文本转换为二维码嵌入，提取后可扫码。
* **智能识别**：提取时无需指定类型，算法自动解析协议头（Header），判断是文本、图片还是二维码。
* **纯 Go 实现**：核心算法依赖 `gonum` 进行矩阵运算，无复杂的 CGo 依赖。

## 📦 安装

```bash
go get github.com/your_username/blindwatermark
```

## 🚀 快速开始

### 1\. 初始化

```go
import (
    "os"
    "image"
    _ "image/jpeg"
    _ "image/png"
    "github.com/your_username/blindwatermark"
)

// 加载图片
file, _ := os.Open("source.jpg")
srcImg, _, _ := image.Decode(file)
defer file.Close()

// 初始化水印引擎 (默认强度 20.0)
bw := blindwatermark.NewBlindWatermarker()
```

### 2\. 嵌入水印 (Encode)

**嵌入文本：**

```go
// 嵌入字符串
watermarkedImg, err := bw.EmbedText(srcImg, "Hello World 2025")
if err != nil {
    panic(err)
}

// 保存结果
saveFile("output_text.jpg", watermarkedImg)
```

**嵌入二维码 (推荐)：**

```go
// 嵌入二维码 (库会自动生成二维码图片并嵌入)
watermarkedImg, err := bw.EmbedQRCode(srcImg, "https://github.com/golang")
if err != nil {
    panic(err)
}
saveFile("output_qr.jpg", watermarkedImg)
```

### 3\. 提取水印 (Decode)

最酷的部分来了！你不需要告诉库你之前嵌入了什么，它会自己告诉你。

```go
import "github.com/your_username/blindwatermark/converter"

// 加载带水印的图片
file, _ := os.Open("output_qr.jpg")
wmImg, _, _ := image.Decode(file)

// 提取
result, err := bw.Extract(wmImg)
if err != nil {
    panic(err)
}

// 自动识别结果类型
switch result.Type {
case converter.TypeText:
    fmt.Printf("提取到文本: %s\n", result.TextContent)

case converter.TypeQRCode:
    fmt.Println("提取到二维码图片")
    // 保存提取出来的二维码图片文件
    os.WriteFile("extracted_qr.png", result.ImageBytes, 0644)
    fmt.Println("请扫描 extracted_qr.png 查看内容")

case converter.TypeImage:
    fmt.Println("提取到图片")
    os.WriteFile("extracted_img.png", result.ImageBytes, 0644)
}
```

## 🧠 核心原理

本库使用了 **Block-DCT (分块离散余弦变换)** 算法：

1.  **预处理**：将图像转换为 YUV 颜色空间，仅对 **Y 通道** (亮度) 进行操作。
2.  **分块**：将图像分割为 `8x8` 的像素块。
3.  **变换**：对每个块进行 DCT 变换，转换到频域。
4.  **嵌入**：
    * 将数据打包为协议格式：`[Type 1byte] + [Length 4bytes] + [Data]`。
    * 通过修改 DCT 变换后的**中频系数**（不易被察觉且抗压缩能力尚可）来嵌入二进制位。
    * 如果 bit 为 1，强化系数差值；如果 bit 为 0，反转系数差值。
5.  **逆变换**：执行 IDCT 还原图像。

## ⚠️ 局限性与注意事项

* **图片尺寸限制**：水印容量受限于图片分辨率。每个 `8x8` 的像素块只能存储 1 bit 信息。
    * 例如：`800x600` 的图片最大容量约为 900 字节。如果二维码过于复杂，可能会超出容量。
* **抗攻击性**：
    * ✅ **支持**：高质量的 JPEG 压缩、轻微的噪点。
    * ❌ **不支持**：图片裁剪（Crop）、旋转（Rotation）、缩放（Resize）。这些操作会破坏 8x8 的分块结构，导致无法提取。
* **性能**：由于涉及大量矩阵运算，处理超大分辨率图片（如 4K）时可能会有几秒钟的延迟。

## 🤝 贡献

欢迎提交 Issue 或 Pull Request！

如果你想改进算法（例如引入 DWT 小波变换或 Arnold 置乱算法以提高鲁棒性），请随时 Fork。

## 📄 许可证

MIT License
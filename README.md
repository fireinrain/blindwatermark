
# Go-BlindWatermark

[](https://golang.org)
[](https://www.google.com/search?q=LICENSE)

**Go-BlindWatermark** 是一个高性能的 Golang 频域盲水印库。它利用 **DWT (离散小波变换)** 和 **DCT (离散余弦变换)** 算法，将信息隐写到图片的频域中。

✨ **核心特性**：水印对人眼不可见，提取时不需要原图（盲提取），且支持自动识别水印类型。

## 🌟 功能特性

* **隐蔽性强**：基于 DWT+DCT 混合算法，只在图像的高频/中频子带（HL）嵌入数据，视觉无损。
* **多类型支持**：
    * 📝 **字符串水印**：直接嵌入文本信息。
    * 🖼️ **图片水印**：支持嵌入 Logo 或图标。
        * *优化*：自动二值化（1-bit）压缩存储，极大节省空间。
        * *智能*：自动根据底图容量缩小水印尺寸，防止溢出。
    * 📱 **二维码水印**：只存储二维码文本内容，提取时自动重绘二维码图片，空间利用率极高。
* **智能识别**：自定义二进制协议头（Header），提取时自动判断是文本、图片还是二维码。
* **纯 Go 实现**：核心矩阵运算依赖 `gonum`，图像处理依赖标准库及扩展库。

## 📦 安装

```bash
go get github.com/your_username/blindwatermark
```

*注意：本项目依赖 `gonum` 进行矩阵运算，以及 `golang.org/x/image` 进行图像缩放。*

## 🚀 快速开始

### 1\. 初始化

```go
import (
    "os"
    "image"
    _ "image/jpeg" // 必须引入解码器
    _ "image/png"
    "github.com/your_username/blindwatermark"
)

// 加载底图 (建议使用 1080P 或更高分辨率的图片以获得足够容量)
file, _ := os.Open("source.jpg")
srcImg, _, _ := image.Decode(file)
defer file.Close()

// 初始化引擎
bw := blindwatermark.NewBlindWatermarker()
```

### 2\. 嵌入水印 (Embedding)

#### 📝 嵌入字符串

```go
resImg, err := bw.EmbedText(srcImg, "Hello World 2025")
if err != nil {
    panic(err)
}
saveFile("output_text.jpg", resImg)
```

#### 🖼️ 嵌入图片 (Logo)

库会自动将 Logo 转为黑白二值图，并根据底图容量自动缩放。

```go
// 加载水印图
wmFile, _ := os.Open("logo.png")
wmImg, _, _ := image.Decode(wmFile)

// 嵌入
resImg, err := bw.EmbedImage(srcImg, wmImg)
if err != nil {
    fmt.Println("嵌入失败:", err) // 可能是底图太小
} else {
    saveFile("output_logo.jpg", resImg)
}
```

#### 📱 嵌入二维码

库只存储 URL 字符串，比直接存二维码图片节省 20 倍空间。

```go
resImg, err := bw.EmbedQRCode(srcImg, "https://github.com/golang")
saveFile("output_qr.jpg", resImg)
```

### 3\. 提取水印 (Extraction)

提取时无需知道水印类型，库会通过协议头自动解析。

```go
import "github.com/your_username/blindwatermark/converter"

// 加载带水印图片
file, _ := os.Open("output_logo.jpg")
watermarkedImg, _, _ := image.Decode(file)

// 提取
result, err := bw.Extract(watermarkedImg)
if err != nil {
    panic(err)
}

// 处理结果
switch result.Type {
case converter.TypeText:
    fmt.Printf("📝 提取到文本: %s\n", result.TextContent)

case converter.TypeQRCode:
    fmt.Printf("📱 提取到二维码内容: %s\n", result.TextContent)
    // 库会自动重建二维码图片到 ImageBytes
    os.WriteFile("extracted_qr.png", result.ImageBytes, 0644)

case converter.TypeImage:
    fmt.Printf("🖼️ 提取到图片，数据大小: %d bytes\n", len(result.ImageBytes))
    // 提取出的是还原后的黑白 PNG 图片
    os.WriteFile("extracted_logo.png", result.ImageBytes, 0644)
}
```

## 🧠 核心算法原理

1.  **颜色空间转换**：RGB -\> YUV，仅对 **Y 通道** (亮度) 进行操作。
2.  **DWT (离散小波变换)**：将图像分解为 LL, LH, HL, HH 四个频带。
    * *策略*：我们选择 **HL (右上)** 频带进行嵌入，兼顾了隐蔽性和鲁棒性。
3.  **分块 DCT**：在 HL 频带上进行 `8x8` 分块 DCT 变换。
4.  **量化嵌入**：修改 DCT 中频系数的相对大小来编码 bit (0 或 1)。
5.  **逆变换**：IDCT -\> IDWT -\> YUV转RGB -\> 生成图片。

### 关于容量 (Capacity)

由于使用了 DWT 变换到 HL 子带，可用容量约为原图像素数的 **1/64** 到 **1/100** (取决于具体参数)。

* **计算公式**：`MaxBits ≈ (Width / 16) * (Height / 16)`
* **示例**：
    * `800x600` 图片 ≈ 1800 bits (约 220 字节) -\> *只能存短文本*
    * `1920x1080` 图片 ≈ 8100 bits (约 1000 字节) -\> *可存二维码或小 Logo*

**如果遇到 `image is too small` 错误，请更换更高分辨率的底图。**

## 📂 目录结构

```text
blindwatermark/
├── cmd/
│   └── main.go           # 示例入口
├── converter/
│   ├── protocol.go       # 协议打包与解包 (Header处理)
│   └── converter.go      # 类型转换工具
├── core/
│   ├── dwt.go            # DWT / IDWT 算法实现
│   ├── dct.go            # DCT / IDCT 算法实现
│   └── engine.go         # 核心嵌入提取引擎
├── watermark.go          # 对外高级接口 (Embed/Extract)
├── go.mod
└── README.md
```

## ⚠️ 局限性

1.  **有损压缩**：提取出的图片水印是 **黑白二值化** 的，不包含彩色信息（为了最大化容量）。
2.  **抗攻击性**：
    * ✅ 支持：JPEG 压缩、轻微噪声、涂抹。
    * ❌ 不支持：截图（裁剪）、旋转、缩放。这些几何变换会破坏频域同步。

## 📄 License

MIT License
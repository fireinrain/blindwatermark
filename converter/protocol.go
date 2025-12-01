package converter

import (
	"encoding/binary"
	"errors"
)

// WatermarkType 定义水印类型
type WatermarkType byte

const (
	TypeText   WatermarkType = 0x01
	TypeImage  WatermarkType = 0x02
	TypeQRCode WatermarkType = 0x03
)

// Pack 将原始数据加上头部信息，并转换为 bool 数组（用于嵌入）
// 协议结构: [Type(1 byte)] + [Length(4 bytes)] + [Data]
func Pack(wmType WatermarkType, data []byte) []bool {
	// 1. 构建二进制 Buffer
	length := uint32(len(data))
	buf := make([]byte, 1+4+len(data))

	buf[0] = byte(wmType)
	binary.BigEndian.PutUint32(buf[1:5], length)
	copy(buf[5:], data)

	// 2. 将 []byte 转换为 []bool (bits)
	return bytesToBits(buf)
}

// Unpack 从提取出的 bool 数组中还原数据，并解析类型
func Unpack(bits []bool) (WatermarkType, []byte, error) {
	bytesData := bitsToBytes(bits)

	if len(bytesData) < 5 {
		return 0, nil, errors.New("extracted data too short")
	}

	// 1. 解析头部
	wmType := WatermarkType(bytesData[0])
	length := binary.BigEndian.Uint32(bytesData[1:5])

	// 2. 校验长度
	totalLen := 5 + int(length)
	if len(bytesData) < totalLen {
		return 0, nil, errors.New("data corrupted or incomplete")
	}

	return wmType, bytesData[5:totalLen], nil
}

// 辅助：byte 转 bit
func bytesToBits(data []byte) []bool {
	bits := make([]bool, len(data)*8)
	for i, b := range data {
		for j := 0; j < 8; j++ {
			// 取出第 j 位
			bits[i*8+j] = (b>>(7-j))&1 == 1
		}
	}
	return bits
}

// 辅助：bit 转 byte
func bitsToBytes(bits []bool) []byte {
	numBytes := len(bits) / 8
	data := make([]byte, numBytes)
	for i := 0; i < numBytes; i++ {
		var b byte
		for j := 0; j < 8; j++ {
			if bits[i*8+j] {
				b |= 1 << (7 - j)
			}
		}
		data[i] = b
	}
	return data
}

package imagepolicy

import "bytes"

// isoBMFFUnsupportedBrands 是移动端常见、但本仓库没有解码器的 ISO-BMFF
// （ftyp）主品牌。设计边界见 issue #408：不新增解码依赖，只把这类字节归入
// 「已知但不支持」，新增格式往集合里加一行即可。
var isoBMFFUnsupportedBrands = map[string]bool{
	"heic": true, "heix": true, "hevc": true, "hevx": true,
	"heim": true, "heis": true, "hevm": true, "hevs": true,
	"mif1": true, "msf1": true, "avif": true, "avis": true,
}

var (
	tiffLittleEndian = []byte{'I', 'I', 0x2A, 0x00}
	tiffBigEndian    = []byte{'M', 'M', 0x00, 0x2A}
)

// sniffUnsupportedImageFormat 报告 data 是否以本仓库刻意不解码的已知图片格式
// 开头。只嗅探固定头部，不做完整 box 解析；漏检会退化为
// ErrInvalidImageContent（仍是拒绝），需要更细分类时再升级。
//
// ponytail: 固定头部匹配（ISO-BMFF 主品牌 + TIFF 魔数），新增格式往表里加一行。
func sniffUnsupportedImageFormat(data []byte) bool {
	if len(data) >= 12 && string(data[4:8]) == "ftyp" && isoBMFFUnsupportedBrands[string(data[8:12])] {
		return true
	}
	return bytes.HasPrefix(data, tiffLittleEndian) || bytes.HasPrefix(data, tiffBigEndian)
}

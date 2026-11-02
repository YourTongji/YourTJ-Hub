package imagepolicy

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strings"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
)

var (
	// ErrInvalidImageContent 表示字节不是可用图片（非图片字节或图片头无法解码），
	// 与扩展名无关：扩展名冲突由调用方规范化，不再归入本错误。
	ErrInvalidImageContent = errors.New("invalid image content")
	// ErrEmptyImageContent 表示上传内容为空（0 字节）。
	ErrEmptyImageContent = errors.New("empty image content")
	// ErrUnsupportedImageFormat 表示字节是已知、但本仓库刻意不解码的图片格式
	// （HEIC/HEIF/AVIF/TIFF 等）。它与 ErrInvalidImageContent 分开，是为了让
	// 用户/维护者直接从稳定码知道「需要转 JPG/PNG」而不是「文件损坏了」。
	ErrUnsupportedImageFormat = errors.New("unsupported image format")
)

// ImageHeaderSniffBytes bounds how much of the upload DetectContentType reads.
const ImageHeaderSniffBytes = 512 * 1024

// DetectContentType returns the canonical image MIME type of the uploaded bytes.
// The bytes are authoritative: the client-declared filename and MIME type never
// decide the type, so a correctly-decoded image under any allowed extension is
// accepted and stored under its real type (issue #1393). Only a bounded header
// is read; the object's full size is enforced by the upload path (the presigned
// exact-size pin plus VerifyUpload's StatObject check), so an image larger than
// the sniff bound is still valid as long as its header decodes.
func DetectContentType(reader io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(reader, ImageHeaderSniffBytes))
	if err != nil {
		return "", fmt.Errorf("read image header: %w", err)
	}
	if len(data) == 0 {
		return "", fmt.Errorf("%w: image header is empty", ErrEmptyImageContent)
	}
	// 判定顺序是契约的一部分：HEIC/AVIF 的 DetectContentType 是
	// application/octet-stream，先归入「已知但不支持」才能给出可操作提示；
	// 两者同样拒绝，扩展名 allowlist 与解码校验未放宽（issue #408）。
	if sniffUnsupportedImageFormat(data) {
		return "", fmt.Errorf("%w: image header uses a known but unsupported format", ErrUnsupportedImageFormat)
	}
	_, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("%w: decode image header: %w", ErrInvalidImageContent, err)
	}
	decodedContentType, ok := DecodedFormatContentType(format)
	if !ok {
		return "", fmt.Errorf("%w: decoded image format %q is not supported", ErrInvalidImageContent, format)
	}
	// 魔数嗅探必须与解码格式一致：不一致说明头部被拼接过，按不可用图片拒绝。
	if detected := http.DetectContentType(data); !strings.EqualFold(detected, decodedContentType) {
		return "", fmt.Errorf("%w: sniffed content type %q does not match decoded format %q", ErrInvalidImageContent, detected, decodedContentType)
	}
	return decodedContentType, nil
}

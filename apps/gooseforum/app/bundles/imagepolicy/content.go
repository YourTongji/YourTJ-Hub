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
	// ErrInvalidImageContent 表示字节不是有效图片，或内容与扩展名推出的类型不符。
	ErrInvalidImageContent = errors.New("invalid image content")
	// ErrEmptyImageContent 表示上传内容为空（0 字节）。
	ErrEmptyImageContent = errors.New("empty image content")
	// ErrUnsupportedImageFormat 表示字节是已知、但本仓库刻意不解码的图片格式
	// （HEIC/HEIF/AVIF/TIFF 等）。它与 ErrInvalidImageContent 分开，是为了让
	// 用户/维护者直接从稳定码知道「需要转 JPG/PNG」而不是「文件损坏了」。
	ErrUnsupportedImageFormat = errors.New("unsupported image format")
)

// ValidateContent decodes the image header and verifies both the sniffed
// and the decoded format match the expected content type. It is used to reject
// forged MIME uploads whose bytes are not actually an image. Only a bounded
// header is read: the object's full size is enforced by the upload path (the
// presigned exact-size pin plus VerifyUpload's StatObject check), so an image
// larger than the sniff bound is still valid as long as its header decodes.
func ValidateContent(reader io.Reader, expectedContentType string) error {
	data, err := io.ReadAll(io.LimitReader(reader, 512*1024))
	if err != nil {
		return fmt.Errorf("read image header: %w", err)
	}
	if len(data) == 0 {
		return fmt.Errorf("%w: image header is empty", ErrEmptyImageContent)
	}
	// 判定顺序是契约的一部分：HEIC/AVIF 的 DetectContentType 是
	// application/octet-stream，先归入「已知但不支持」才能给出可操作提示；
	// 两者同样拒绝，扩展名 allowlist 与解码校验未放宽（issue #408）。
	if sniffUnsupportedImageFormat(data) {
		return fmt.Errorf("%w: image header uses a known but unsupported format", ErrUnsupportedImageFormat)
	}
	detected := http.DetectContentType(data)
	if !strings.EqualFold(detected, expectedContentType) {
		return fmt.Errorf("%w: detected content type %q does not match %q", ErrInvalidImageContent, detected, expectedContentType)
	}
	_, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("%w: decode image header: %w", ErrInvalidImageContent, err)
	}
	decodedContentType, ok := DecodedFormatContentType(format)
	if !ok || !strings.EqualFold(decodedContentType, expectedContentType) {
		return fmt.Errorf("%w: decoded image format %q does not match %q", ErrInvalidImageContent, format, expectedContentType)
	}
	return nil
}

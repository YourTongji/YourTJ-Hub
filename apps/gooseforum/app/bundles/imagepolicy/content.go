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

var ErrInvalidImageContent = errors.New("invalid image content")

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
		return fmt.Errorf("%w: image header is empty", ErrInvalidImageContent)
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

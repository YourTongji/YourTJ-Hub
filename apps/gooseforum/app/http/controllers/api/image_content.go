package api

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

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/storageservice"
)

var errInvalidImageContent = errors.New("invalid image content")

var imageFormatContentTypes = map[string]string{
	"jpeg": "image/jpeg",
	"png":  "image/png",
	"gif":  "image/gif",
	"webp": "image/webp",
	"bmp":  "image/bmp",
}

// validateUploadedImage decodes the image header and verifies both the sniffed
// and the decoded format match the expected content type. It is used to reject
// forged MIME uploads whose bytes are not actually an image. Only a bounded
// header is read: the object's full size is enforced by the upload path (the
// presigned exact-size pin plus VerifyUpload's StatObject check), so an image
// larger than the sniff bound is still valid as long as its header decodes.
func validateUploadedImage(reader io.Reader, expectedContentType string) error {
	data, err := io.ReadAll(io.LimitReader(reader, storageservice.ImageHeaderSniffBytes))
	if err != nil {
		return fmt.Errorf("read image header: %w", err)
	}
	if len(data) == 0 {
		return fmt.Errorf("%w: image header is empty", errInvalidImageContent)
	}
	detected := http.DetectContentType(data)
	if !strings.EqualFold(detected, expectedContentType) {
		return fmt.Errorf("%w: detected content type %q does not match %q", errInvalidImageContent, detected, expectedContentType)
	}
	_, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("%w: decode image header: %w", errInvalidImageContent, err)
	}
	decodedContentType, ok := imageFormatContentTypes[format]
	if !ok || !strings.EqualFold(decodedContentType, expectedContentType) {
		return fmt.Errorf("%w: decoded image format %q does not match %q", errInvalidImageContent, format, expectedContentType)
	}
	return nil
}

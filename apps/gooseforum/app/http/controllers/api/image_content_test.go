package api

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/imagepolicy"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/storageservice"
)

// tinyPNG is a 1x1 pixel PNG.
var tinyPNG = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
	0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
	0x89, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x44, 0x41,
	0x54, 0x78, 0x9C, 0x62, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
	0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
	0x42, 0x60, 0x82,
}

func encodeTestJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode test jpeg: %v", err)
	}
	return buf.Bytes()
}

// encodeLargeTestJPEG returns a valid JPEG larger than the header sniff bound
// (and larger than the old 1MB cap) to prove that a legal image whose body
// exceeds the header sniff bound is not rejected: only the bounded header is
// read, the full size is enforced by the upload path.
func encodeLargeTestJPEG(t *testing.T) []byte {
	t.Helper()
	const side = 1024
	img := image.NewRGBA(image.Rect(0, 0, side, side))
	for y := 0; y < side; y++ {
		for x := 0; x < side; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * y), G: uint8(x*7 + y*13), B: uint8(x*31 + y*17), A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("encode large test jpeg: %v", err)
	}
	if buf.Len() <= imagepolicy.ImageHeaderSniffBytes {
		t.Fatalf("large test jpeg is %d bytes, want > %d", buf.Len(), imagepolicy.ImageHeaderSniffBytes)
	}
	return buf.Bytes()
}

// TestDetectUploadedImageContent pins the issue #1393 contract change: the
// bytes alone decide the type, so PNG bytes under a `.jpg` claim still detect as
// image/png; only unusable bytes fail.
func TestDetectUploadedImageContent(t *testing.T) {
	jpegData := encodeTestJPEG(t)
	heic := []byte{0x00, 0x00, 0x00, 0x18, 'f', 't', 'y', 'p', 'h', 'e', 'i', 'c', 0x00, 0x00, 0x00, 0x00}
	tests := []struct {
		name    string
		data    []byte
		want    string
		wantErr error
	}{
		{name: "valid jpeg", data: jpegData, want: "image/jpeg"},
		{name: "valid png", data: tinyPNG, want: "image/png"},
		{name: "large legal jpeg over the sniff bound", data: encodeLargeTestJPEG(t), want: "image/jpeg"},
		{name: "empty bytes report empty", data: nil, wantErr: imagepolicy.ErrEmptyImageContent},
		{name: "text bytes report invalid", data: []byte("this is definitely not an image"), wantErr: imagepolicy.ErrInvalidImageContent},
		{name: "oversized non-image reports invalid", data: make([]byte, imagepolicy.ImageHeaderSniffBytes+1), wantErr: imagepolicy.ErrInvalidImageContent},
		{name: "heic bytes report unsupported", data: heic, wantErr: imagepolicy.ErrUnsupportedImageFormat},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := detectUploadedImageContent(bytes.NewReader(tt.data))
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("detectUploadedImageContent() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("detectUploadedImageContent() = %q, %v; want %q, nil", got, err, tt.want)
			}
		})
	}
}

// TestImageContentFailureCode pins the stable code per sentinel.
func TestImageContentFailureCode(t *testing.T) {
	tests := []struct {
		err  error
		want component.MessageCode
	}{
		{err: imagepolicy.ErrEmptyImageContent, want: component.MessageUploadEmptyImage},
		{err: imagepolicy.ErrUnsupportedImageFormat, want: component.MessageUploadUnsupportedImage},
		{err: imagepolicy.ErrInvalidImageContent, want: component.MessageUploadInvalidImage},
		{err: errors.New("unexpected read failure"), want: component.MessageUploadInvalidImage},
	}
	for _, tt := range tests {
		if got := imageContentFailureCode(tt.err); got != tt.want {
			t.Fatalf("imageContentFailureCode(%v) = %q, want %q", tt.err, got, tt.want)
		}
	}
}

// TestValidateDetectedContentTypeAllowsConflict is the issue #1393 acceptance
// rule for the allowlist: a detected type that is allowed is accepted even when
// it conflicts with the declared extension.
func TestValidateDetectedContentTypeAllowsConflict(t *testing.T) {
	policy := imageUploadPolicy{AllowedExts: imagepolicy.DefaultExtensions()}
	for _, contentType := range []string{"image/jpeg", "image/png", "image/gif", "image/webp", "image/bmp"} {
		if failure := policy.validateDetectedContentType(contentType); failure != nil {
			t.Fatalf("validateDetectedContentType(%q) = %+v, want nil", contentType, failure)
		}
	}
	// A format outside the configured allowlist is rejected with the same code
	// as a declared extension outside it.
	narrow := imageUploadPolicy{AllowedExts: []string{".png"}}
	failure := narrow.validateDetectedContentType("image/jpeg")
	if failure == nil || failure.Data.MessageCode != component.MessageUploadUnsupportedExt {
		t.Fatalf("validateDetectedContentType(image/jpeg) = %+v, want upload.extension.unsupported", failure)
	}
}

// TestUploadFailureErrorRollsBackDirectUpload pins the bridge that keeps the
// direct-upload complete on the same code path: the wrapped failure still
// unwraps to the storageservice invalid-object sentinel (so the object and the
// pending row are rolled back) and stays extractable for the response.
func TestUploadFailureErrorRollsBackDirectUpload(t *testing.T) {
	policy := imageUploadPolicy{AllowedExts: []string{".png"}}
	failure := policy.validateDetectedContentType("image/jpeg")
	if failure == nil {
		t.Fatal("validateDetectedContentType() = nil, want a failure")
	}
	err := error(&uploadFailureError{failure: failure})
	if !errors.Is(err, storageservice.ErrDirectUploadInvalidObject) {
		t.Fatalf("error = %v, want it to unwrap to ErrDirectUploadInvalidObject", err)
	}
	var failureErr *uploadFailureError
	if !errors.As(err, &failureErr) || failureErr.failure != failure {
		t.Fatalf("errors.As(*uploadFailureError) = %+v, want the original failure", failureErr)
	}
}

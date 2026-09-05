package api

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"testing"

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
// exceeds the header sniff bound is not rejected during validation. 1024x1024
// with a noise pattern lands ~1.1MB: above the sniff bound yet below the 4MB
// upload cap, so the route-level upload gate stays satisfied.
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
	if buf.Len() <= storageservice.ImageHeaderSniffBytes {
		t.Fatalf("large test jpeg is %d bytes, want > %d", buf.Len(), storageservice.ImageHeaderSniffBytes)
	}
	return buf.Bytes()
}

func TestValidateUploadedImage(t *testing.T) {
	jpegData := encodeTestJPEG(t)
	tests := []struct {
		name        string
		data        []byte
		contentType string
		wantErr     bool
	}{
		{name: "valid jpeg", data: jpegData, contentType: "image/jpeg"},
		{name: "png bytes with wrong content type", data: tinyPNG, contentType: "image/jpeg", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUploadedImage(bytes.NewReader(tt.data), tt.contentType)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("validateUploadedImage() error = nil, want error")
				}
				if !errors.Is(err, errInvalidImageContent) {
					t.Fatalf("validateUploadedImage() error = %v, want errInvalidImageContent", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("validateUploadedImage() error = %v, want nil", err)
			}
		})
	}
}

func TestValidateUploadedImageAcceptsLargeLegalImage(t *testing.T) {
	// A valid image larger than the header sniff bound (and the old 1MB cap)
	// must validate: size consistency is enforced by the upload path, not by
	// the header validator, which only reads the bounded header.
	data := encodeLargeTestJPEG(t)
	err := validateUploadedImage(bytes.NewReader(data), "image/jpeg")
	if err != nil {
		t.Fatalf("validateUploadedImage() error = %v, want nil for a legal >512KB image", err)
	}
}

func TestValidateUploadedImageRejectsOversizedHeader(t *testing.T) {
	// A blob larger than the sniff bound whose header is not a valid image
	// (all zero bytes) must still be rejected by the sniff/decode check.
	data := make([]byte, storageservice.ImageHeaderSniffBytes+1)
	err := validateUploadedImage(bytes.NewReader(data), "image/png")
	if !errors.Is(err, errInvalidImageContent) {
		t.Fatalf("validateUploadedImage() error = %v, want errInvalidImageContent", err)
	}
}

// tinyWebPLossless is a 1x1 pixel lossless VP8L WebP (exercises the vp8l
// decoder path that CVE-2026-46603 / GO-2026-6222 covers).
var tinyWebPLossless = []byte{
	0x52, 0x49, 0x46, 0x46, 0x1c, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50,
	0x56, 0x50, 0x38, 0x4c, 0x0f, 0x00, 0x00, 0x00, 0x2f, 0x00, 0x00, 0x00,
	0x00, 0x07, 0x10, 0xfd, 0x8f, 0xfe, 0x07, 0x22, 0xa2, 0xff, 0x01, 0x00,
}

// tinyBMP is a 1x1 pixel 24-bit BMP.
var tinyBMP = []byte{
	0x42, 0x4d, 0x3a, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x36, 0x00,
	0x00, 0x00, 0x28, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00,
	0x00, 0x00, 0x01, 0x00, 0x18, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04, 0x00,
	0x00, 0x00, 0xc4, 0x0e, 0x00, 0x00, 0xc4, 0x0e, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0x00,
}

// truncatedWebP is a WebP/VP8L header cut off mid-chunk, so the decoder must
// report a stable error instead of panicking or hanging.
var truncatedWebP = []byte{
	0x52, 0x49, 0x46, 0x46, 0x1c, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50,
	0x56, 0x50, 0x38, 0x4c, 0x0f, 0x00, 0x00, 0x00,
}

// TestValidateUploadedImageWebPAndBMP guards the x/image upgrade (issue #405,
// CVE-2026-46603): lossless VP8L and BMP inputs must keep validating, and
// truncated/corrupt WebP input must fail with the stable invalid-image error.
func TestValidateUploadedImageWebPAndBMP(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		contentType string
		wantErr     bool
	}{
		{name: "valid lossless webp", data: tinyWebPLossless, contentType: "image/webp"},
		{name: "valid bmp", data: tinyBMP, contentType: "image/bmp"},
		{name: "webp bytes with wrong content type", data: tinyWebPLossless, contentType: "image/png", wantErr: true},
		{name: "truncated webp", data: truncatedWebP, contentType: "image/webp", wantErr: true},
		{name: "bmp bytes with wrong content type", data: tinyBMP, contentType: "image/webp", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUploadedImage(bytes.NewReader(tt.data), tt.contentType)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("validateUploadedImage() error = nil, want error")
				}
				if !errors.Is(err, errInvalidImageContent) {
					t.Fatalf("validateUploadedImage() error = %v, want errInvalidImageContent", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("validateUploadedImage() error = %v, want nil", err)
			}
		})
	}
}

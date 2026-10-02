package imagepolicy

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

// tinyPNG is a valid 1x1 pixel PNG (decodable by image/png).
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

func testJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode test jpeg: %v", err)
	}
	return buf.Bytes()
}

// isoBMFFHeader builds a 16-byte ISO-BMFF header with the given major brand.
func isoBMFFHeader(brand string) []byte {
	data := []byte{
		0x00, 0x00, 0x00, 0x18, // box size
		'f', 't', 'y', 'p', // major box type
	}
	data = append(data, []byte(brand)...)
	data = append(data, 0x00, 0x00, 0x00, 0x00)
	return data
}

// TestValidateContentSentinelOrder pins the contract order: empty first, then
// known-but-unsupported formats, then sniff/decode mismatch. HEIC bytes paired
// with a PNG expectation must report "unsupported", not "invalid", so users
// learn they need to convert the file instead of retrying.
func TestValidateContentSentinelOrder(t *testing.T) {
	jpegData := testJPEG(t)
	heic := isoBMFFHeader("heic")
	avif := isoBMFFHeader("avif")
	tiffLE := []byte{'I', 'I', 0x2A, 0x00, 0x08, 0x00, 0x00, 0x00}
	tiffBE := []byte{'M', 'M', 0x00, 0x2A, 0x00, 0x00, 0x00, 0x08}

	tests := []struct {
		name        string
		data        []byte
		contentType string
		wantErr     error
	}{
		{name: "empty bytes report empty", data: nil, contentType: "image/png", wantErr: ErrEmptyImageContent},
		{name: "zero-length PNG reports empty", data: []byte{}, contentType: "image/png", wantErr: ErrEmptyImageContent},
		{name: "heic reports unsupported", data: heic, contentType: "image/png", wantErr: ErrUnsupportedImageFormat},
		{name: "avif reports unsupported", data: avif, contentType: "image/png", wantErr: ErrUnsupportedImageFormat},
		{name: "heif brand reports unsupported", data: isoBMFFHeader("mif1"), contentType: "image/png", wantErr: ErrUnsupportedImageFormat},
		{name: "tiff little-endian reports unsupported", data: tiffLE, contentType: "image/png", wantErr: ErrUnsupportedImageFormat},
		{name: "tiff big-endian reports unsupported", data: tiffBE, contentType: "image/png", wantErr: ErrUnsupportedImageFormat},
		{name: "png bytes with jpeg expectation report invalid", data: tinyPNG, contentType: "image/jpeg", wantErr: ErrInvalidImageContent},
		{name: "jpeg bytes with png expectation report invalid", data: jpegData, contentType: "image/png", wantErr: ErrInvalidImageContent},
		{name: "truncated png reports invalid", data: tinyPNG[:12], contentType: "image/png", wantErr: ErrInvalidImageContent},
		{name: "valid png passes", data: tinyPNG, contentType: "image/png"},
		{name: "valid jpeg passes", data: jpegData, contentType: "image/jpeg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateContent(bytes.NewReader(tt.data), tt.contentType)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("ValidateContent() error = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateContent() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateContentCaseInsensitiveExpectation keeps the historic tolerance
// for callers that pass a differently-cased MIME type.
func TestValidateContentCaseInsensitiveExpectation(t *testing.T) {
	if err := ValidateContent(bytes.NewReader(tinyPNG), "IMAGE/PNG"); err != nil {
		t.Fatalf("ValidateContent() with upper-case expectation error = %v, want nil", err)
	}
}

package api

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
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

func TestValidateUploadedImageRejectsOversizedHeader(t *testing.T) {
	data := make([]byte, maximumImageHeaderSize+1)
	err := validateUploadedImage(bytes.NewReader(data), "image/png")
	if !errors.Is(err, errInvalidImageContent) {
		t.Fatalf("validateUploadedImage() error = %v, want errInvalidImageContent", err)
	}
}

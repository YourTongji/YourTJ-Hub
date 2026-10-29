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

// tinyGIF is a 1x1 transparent GIF89a.
var tinyGIF = []byte("GIF89a\x01\x00\x01\x00\x80\x00\x00\x00\x00\x00\xFF\xFF\xFF\x21\xF9\x04\x01\x00\x00\x00\x00\x2C\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x02\x44\x01\x00\x3B")

// tinyWebP is a 1x1 lossless VP8L WebP.
var tinyWebP = []byte{
	0x52, 0x49, 0x46, 0x46, 0x1c, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50,
	0x56, 0x50, 0x38, 0x4c, 0x0f, 0x00, 0x00, 0x00, 0x2f, 0x00, 0x00, 0x00,
	0x00, 0x07, 0x10, 0xfd, 0x8f, 0xfe, 0x07, 0x22, 0xa2, 0xff, 0x01, 0x00,
}

// truncatedWebP is a WebP header cut off mid-chunk: the decoder must report a
// stable error instead of panicking or hanging.
var truncatedWebP = tinyWebP[:20]

// tinyBMP is a 1x1 24-bit BMP.
var tinyBMP = []byte{
	0x42, 0x4d, 0x3a, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x36, 0x00,
	0x00, 0x00, 0x28, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00,
	0x00, 0x00, 0x01, 0x00, 0x18, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04, 0x00,
	0x00, 0x00, 0xc4, 0x0e, 0x00, 0x00, 0xc4, 0x0e, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0x00,
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

// TestDetectContentTypeSentinelBranches pins the byte-authoritative contract
// (issue #1393): empty bytes first, then known-but-unsupported formats, then
// unusable images, and the canonical MIME for every decodable format.
func TestDetectContentTypeSentinelBranches(t *testing.T) {
	jpegData := testJPEG(t)
	tiffLE := []byte{'I', 'I', 0x2A, 0x00, 0x08, 0x00, 0x00, 0x00}
	tiffBE := []byte{'M', 'M', 0x00, 0x2A, 0x00, 0x00, 0x00, 0x08}

	tests := []struct {
		name    string
		data    []byte
		want    string
		wantErr error
	}{
		{name: "nil bytes report empty", data: nil, wantErr: ErrEmptyImageContent},
		{name: "zero-length bytes report empty", data: []byte{}, wantErr: ErrEmptyImageContent},
		{name: "heic reports unsupported", data: isoBMFFHeader("heic"), wantErr: ErrUnsupportedImageFormat},
		{name: "avif reports unsupported", data: isoBMFFHeader("avif"), wantErr: ErrUnsupportedImageFormat},
		{name: "heif brand reports unsupported", data: isoBMFFHeader("mif1"), wantErr: ErrUnsupportedImageFormat},
		{name: "tiff little-endian reports unsupported", data: tiffLE, wantErr: ErrUnsupportedImageFormat},
		{name: "tiff big-endian reports unsupported", data: tiffBE, wantErr: ErrUnsupportedImageFormat},
		{name: "text bytes report invalid", data: []byte("this is definitely not an image"), wantErr: ErrInvalidImageContent},
		{name: "truncated png reports invalid", data: tinyPNG[:12], wantErr: ErrInvalidImageContent},
		{name: "truncated webp reports invalid", data: truncatedWebP, wantErr: ErrInvalidImageContent},
		{name: "oversized non-image report invalid", data: make([]byte, ImageHeaderSniffBytes+1), wantErr: ErrInvalidImageContent},
		{name: "png", data: tinyPNG, want: "image/png"},
		{name: "jpeg", data: jpegData, want: "image/jpeg"},
		{name: "gif", data: tinyGIF, want: "image/gif"},
		{name: "webp", data: tinyWebP, want: "image/webp"},
		{name: "bmp", data: tinyBMP, want: "image/bmp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DetectContentType(bytes.NewReader(tt.data))
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("DetectContentType() error = %v, want %v", err, tt.wantErr)
				}
				if got != "" {
					t.Fatalf("DetectContentType() type = %q, want empty on error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("DetectContentType() error = %v, want nil", err)
			}
			if got != tt.want {
				t.Fatalf("DetectContentType() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestDetectContentTypeIgnoresDeclaredName is the issue #1393 contract change:
// the bytes decide the type, so the same PNG bytes report image/png no matter
// what extension or MIME the client claimed.
func TestDetectContentTypeIgnoresDeclaredName(t *testing.T) {
	got, err := DetectContentType(bytes.NewReader(tinyPNG))
	if err != nil || got != "image/png" {
		t.Fatalf("DetectContentType() = %q, %v; want image/png, nil", got, err)
	}
}

// TestExtForContentType pins the canonical storage extension per detected MIME.
func TestExtForContentType(t *testing.T) {
	tests := []struct {
		contentType string
		want        string
	}{
		{contentType: "image/jpeg", want: ".jpg"},
		{contentType: "IMAGE/PNG", want: ".png"},
		{contentType: "image/gif", want: ".gif"},
		{contentType: "image/webp", want: ".webp"},
		{contentType: "image/bmp", want: ".bmp"},
		{contentType: "image/svg+xml"},
		{contentType: ""},
	}
	for _, tt := range tests {
		got, ok := ExtForContentType(tt.contentType)
		if got != tt.want || ok != (tt.want != "") {
			t.Fatalf("ExtForContentType(%q) = %q, %v; want %q, %v", tt.contentType, got, ok, tt.want, tt.want != "")
		}
	}
	// Every default extension round-trips to a MIME whose canonical extension is
	// again in the supported set (jpeg collapses to .jpg).
	for _, ext := range DefaultExtensions() {
		contentType, ok := ContentTypeForExt(ext)
		if !ok {
			t.Fatalf("DefaultExtensions() entry %q has no content type", ext)
		}
		canonical, ok := ExtForContentType(contentType)
		if !ok || !IsAllowedExt(canonical, DefaultExtensions()) {
			t.Fatalf("ExtForContentType(%q) = %q, %v; want a supported extension", contentType, canonical, ok)
		}
	}
}

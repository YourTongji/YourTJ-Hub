package filedata

import (
	"bytes"
	"image"
	"image/color/palette"
	"image/gif"
	"image/jpeg"
	"testing"
)

func TestProcessUploadedImageStoresDimensionsAndVariants(t *testing.T) {
	setupFileDataTestDB(t)

	var source bytes.Buffer
	if err := jpeg.Encode(&source, image.NewRGBA(image.Rect(0, 0, 2048, 1024)), nil); err != nil {
		t.Fatalf("encode source image: %v", err)
	}
	const name = "2026/09/24/feed.jpg"
	if _, err := SaveFile(7, name, "image/jpeg", source.Bytes()); err != nil {
		t.Fatalf("save source image: %v", err)
	}

	metadata, err := ProcessUploadedImage(name, source.Bytes())
	if err != nil {
		t.Fatalf("ProcessUploadedImage() error = %v", err)
	}
	if metadata.Width != 2048 || metadata.Height != 1024 {
		t.Fatalf("dimensions = %dx%d, want 2048x1024", metadata.Width, metadata.Height)
	}
	wantWidths := []int{320, 640, 1280}
	wantHeights := []int{160, 320, 640}
	if len(metadata.Variants) != len(wantWidths) {
		t.Fatalf("variant count = %d, want %d", len(metadata.Variants), len(wantWidths))
	}
	var variantRows []Entity
	if err := builder().Where("parent_name = ?", name).Order("image_width").Find(&variantRows).Error; err != nil {
		t.Fatalf("read variant rows: %v", err)
	}
	for i, variant := range metadata.Variants {
		if variant.Width != wantWidths[i] || variant.Height != wantHeights[i] {
			t.Errorf("variant[%d] = %dx%d, want %dx%d", i, variant.Width, variant.Height, wantWidths[i], wantHeights[i])
		}
		variantName := variantRows[i].Name
		if variant.URL != accessPath(variantName) {
			t.Errorf("variant URL = %q, want %q", variant.URL, accessPath(variantName))
		}
		if got := ReferenceName(variantName); got != name {
			t.Errorf("ReferenceName(%q) = %q, want original %q", variantName, got, name)
		}
		if _, err := GetFileByName(variantName); err != nil {
			t.Errorf("GetFileByName(%q) error = %v", variantName, err)
		}
	}

	stored, err := ImageMetadataByNames([]string{name, name, "missing.png"})
	if err != nil {
		t.Fatalf("ImageMetadataByNames() error = %v", err)
	}
	if got := stored[name]; got.Width != 2048 || len(got.Variants) != len(wantWidths) {
		t.Fatalf("stored metadata = %+v, want dimensions and %d variants", got, len(wantWidths))
	}

	if err := DeleteByName(name); err != nil {
		t.Fatalf("DeleteByName() error = %v", err)
	}
	var count int64
	if err := builder().Model(&Entity{}).Where("parent_name = ?", name).Count(&count).Error; err != nil {
		t.Fatalf("count deleted variants: %v", err)
	}
	if count != 0 || GetByName(name).Id != 0 {
		t.Fatalf("delete left %d variants or original row", count)
	}
}

func TestProcessUploadedImageKeepsLegacyAndAnimatedFallbacks(t *testing.T) {
	setupFileDataTestDB(t)

	legacy, err := SaveFile(1, "2026/09/24/legacy.jpg", "image/jpeg", []byte("legacy"))
	if err != nil {
		t.Fatalf("save legacy file: %v", err)
	}
	metadata, err := ImageMetadataByNames([]string{legacy.Name})
	if err != nil {
		t.Fatalf("ImageMetadataByNames() error = %v", err)
	}
	if _, ok := metadata[legacy.Name]; ok {
		t.Fatal("legacy row without intrinsic dimensions should be omitted")
	}

	var animated bytes.Buffer
	if err := gif.Encode(&animated, image.NewPaletted(image.Rect(0, 0, 64, 32), palette.Plan9), nil); err != nil {
		t.Fatalf("encode GIF fallback: %v", err)
	}
	const name = "2026/09/24/animated.gif"
	if _, err := SaveFile(2, name, "image/gif", animated.Bytes()); err != nil {
		t.Fatalf("save GIF: %v", err)
	}
	animatedMetadata, err := ProcessUploadedImage(name, animated.Bytes())
	if err != nil {
		t.Fatalf("ProcessUploadedImage(GIF) error = %v", err)
	}
	if animatedMetadata.Width != 64 || animatedMetadata.Height != 32 || len(animatedMetadata.Variants) != 0 {
		t.Fatalf("GIF metadata = %+v, want 64x32 and no derivative", animatedMetadata)
	}
}

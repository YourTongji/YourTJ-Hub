package filedata

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"math"
	"path"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/storageservice"
	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
	"gorm.io/gorm"
)

// ponytail: skip thumbnail decode above 25MP to cap upload memory; use a
// bounded worker if larger uploads later need server-side derivatives.
const maxVariantDecodePixels int64 = 25_000_000

var feedVariantWidths = [...]int{320, 640, 1280}

type storedImageVariant struct {
	Name   string `json:"name"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type ImageVariant struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type ImageMetadata struct {
	URL      string         `json:"url"`
	Width    int            `json:"width"`
	Height   int            `json:"height"`
	Variants []ImageVariant `json:"variants,omitempty"`
}

// ProcessUploadedImage persists intrinsic dimensions and stores smaller
// derivatives for static JPEG, PNG and BMP uploads. Animated/unsupported
// formats retain their original bytes and dimensions without a derivative.
func ProcessUploadedImage(name string, data []byte) (ImageMetadata, error) {
	var media ImageMetadata
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return media, fmt.Errorf("decode image dimensions: %w", err)
	}
	if config.Width < 1 || config.Height < 1 {
		return media, errors.New("decoded image has invalid dimensions")
	}
	media = ImageMetadata{URL: accessPath(name), Width: config.Width, Height: config.Height}
	if err := updateImageMetadata(name, config.Width, config.Height, nil); err != nil {
		return media, err
	}
	if format != "jpeg" && format != "png" && format != "bmp" || int64(config.Width)*int64(config.Height) > maxVariantDecodePixels {
		return media, nil
	}

	source, decodedFormat, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return media, fmt.Errorf("decode image for thumbnails: %w", err)
	}
	if decodedFormat != format {
		return media, errors.New("image format changed while decoding")
	}
	variantExt, variantType := ".png", "image/png"
	if format == "jpeg" {
		variantExt, variantType = ".jpg", "image/jpeg"
	}
	base := strings.TrimSuffix(name, path.Ext(name))
	variants := make([]storedImageVariant, 0, len(feedVariantWidths))
	for _, width := range feedVariantWidths {
		if width >= config.Width {
			continue
		}
		height := max(1, int(math.Round(float64(config.Height)*float64(width)/float64(config.Width))))
		variantName := fmt.Sprintf("%s__w%d%s", base, width, variantExt)
		encoded, err := encodeImageVariant(source, width, height, format)
		if err != nil {
			cleanupErr := deleteDerivedFiles(name)
			_ = updateImageMetadata(name, config.Width, config.Height, nil)
			return media, errors.Join(fmt.Errorf("encode image variant %d: %w", width, err), cleanupErr)
		}
		if err := saveImageVariant(name, variantName, variantType, width, height, encoded); err != nil {
			cleanupErr := deleteDerivedFiles(name)
			_ = updateImageMetadata(name, config.Width, config.Height, nil)
			return media, errors.Join(fmt.Errorf("save image variant %d: %w", width, err), cleanupErr)
		}
		variants = append(variants, storedImageVariant{Name: variantName, Width: width, Height: height})
	}
	if err := updateImageMetadata(name, config.Width, config.Height, variants); err != nil {
		_ = deleteDerivedFiles(name)
		return media, err
	}
	media.Variants = make([]ImageVariant, 0, len(variants))
	for _, variant := range variants {
		media.Variants = append(media.Variants, ImageVariant{
			URL: accessPath(variant.Name), Width: variant.Width, Height: variant.Height,
		})
	}
	return media, nil
}

func encodeImageVariant(source image.Image, width, height int, format string) ([]byte, error) {
	resized := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.ApproxBiLinear.Scale(resized, resized.Bounds(), source, source.Bounds(), draw.Over, nil)
	var encoded bytes.Buffer
	if format == "jpeg" {
		if err := jpeg.Encode(&encoded, resized, &jpeg.Options{Quality: 84}); err != nil {
			return nil, err
		}
	} else if err := png.Encode(&encoded, resized); err != nil {
		return nil, err
	}
	return encoded.Bytes(), nil
}

func saveImageVariant(parentName, name, contentType string, width, height int, data []byte) error {
	var existing Entity
	result := builder().Select("id", "parent_name").Where("name = ?", name).First(&existing)
	if result.Error == nil {
		if existing.ParentName == parentName {
			return nil
		}
		return fmt.Errorf("image variant name collision: %s", name)
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return result.Error
	}
	entity := &Entity{
		Name: name, Type: contentType, Data: data, Size: int64(len(data)),
		Width: width, Height: height, ParentName: parentName,
		StorageStatus: StorageStatusReady,
	}
	if create(entity) == 0 {
		return errors.New("failed to save image variant")
	}
	if !storageservice.IsLocalProvider() {
		if err := storageservice.Current().Save(context.Background(), name, data, contentType); err != nil {
			_ = deleteStoredFile(name)
			return fmt.Errorf("save image variant to storage provider: %w", err)
		}
	}
	return nil
}

func updateImageMetadata(name string, width, height int, variants []storedImageVariant) error {
	encoded, err := json.Marshal(variants)
	if err != nil {
		return err
	}
	return builder().Model(&Entity{}).Where("name = ? AND storage_status = ?", name, StorageStatusReady).Updates(map[string]any{
		"image_width": width, "image_height": height, "image_variants": string(encoded),
	}).Error
}

// ImageMetadataByNames returns only ready original rows; legacy files with no
// dimensions are omitted so clients keep their URL-based fallback.
func ImageMetadataByNames(names []string) (map[string]ImageMetadata, error) {
	unique := make([]string, 0, len(names))
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		if name != "" && !seen[name] {
			seen[name] = true
			unique = append(unique, name)
		}
	}
	result := make(map[string]ImageMetadata, len(unique))
	if len(unique) == 0 {
		return result, nil
	}
	var entities []Entity
	if err := builder().Select("name", "image_width", "image_height", "image_variants").
		Where("name IN ? AND parent_name = '' AND storage_status = ?", unique, StorageStatusReady).
		Find(&entities).Error; err != nil {
		return nil, err
	}
	for _, entity := range entities {
		if entity.Width < 1 || entity.Height < 1 {
			continue
		}
		media := ImageMetadata{URL: accessPath(entity.Name), Width: entity.Width, Height: entity.Height}
		for _, variant := range entity.Variants {
			media.Variants = append(media.Variants, ImageVariant{
				URL: accessPath(variant.Name), Width: variant.Width, Height: variant.Height,
			})
		}
		result[entity.Name] = media
	}
	return result, nil
}

func ReferenceName(name string) string {
	var row struct{ ParentName string }
	if err := builder().Select("parent_name").Where("name = ?", name).Take(&row).Error; err == nil && row.ParentName != "" {
		return row.ParentName
	}
	return name
}

func deleteDerivedFiles(parentName string) error {
	var variants []Entity
	if err := builder().Select("name").Where("parent_name = ?", parentName).Find(&variants).Error; err != nil {
		return err
	}
	for _, variant := range variants {
		if err := deleteStoredFile(variant.Name); err != nil {
			return err
		}
	}
	return nil
}

func deleteStoredFile(name string) error {
	if !storageservice.IsLocalProvider() {
		if err := storageservice.Current().Delete(context.Background(), name); err != nil && !errors.Is(err, storageservice.ErrNotFound) {
			return err
		}
	}
	return builder().Where("name = ?", name).Delete(&Entity{}).Error
}

package filedata

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"testing"
)

// Each quadrant has a distinct color so rotations and mirrors are observable
// after JPEG decoding, resizing and re-encoding, without relying on EXIF.
func orientationJPEG(t *testing.T, width, height int) []byte {
	t.Helper()
	source := image.NewRGBA(image.Rect(0, 0, width, height))
	colors := []color.Color{color.RGBA{255, 0, 0, 255}, color.RGBA{0, 255, 0, 255}, color.RGBA{0, 0, 255, 255}, color.RGBA{255, 255, 0, 255}}
	for i, c := range colors {
		x, y := (i%2)*width/2, (i/2)*height/2
		draw.Draw(source, image.Rect(x, y, x+width/2, y+height/2), image.NewUniform(c), image.Point{}, draw.Src)
	}
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, source, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	return encoded.Bytes()
}

func jpegWithOrientation(data []byte, orientation uint16, order binary.ByteOrder) []byte {
	// TIFF IFD0 with one inline SHORT Orientation (0x0112) entry.
	exif := make([]byte, 32)
	copy(exif, "Exif\x00\x00MM")
	if order == binary.LittleEndian {
		copy(exif[6:], "II")
	}
	order.PutUint16(exif[8:], 42)
	order.PutUint32(exif[10:], 8)
	order.PutUint16(exif[14:], 1)
	order.PutUint16(exif[16:], 0x0112)
	order.PutUint16(exif[18:], 3)
	order.PutUint32(exif[20:], 1)
	order.PutUint16(exif[24:], orientation)
	return jpegWithAPP1(data, exif)
}

func jpegWithAPP1(data, payload []byte) []byte {
	result := append([]byte{}, data[:2]...)
	result = append(result, 0xff, 0xe1, byte((len(payload)+2)>>8), byte(len(payload)+2))
	result = append(result, payload...)
	return append(result, data[2:]...)
}

func TestProcessUploadedImageNormalizesEXIFOrientation(t *testing.T) {
	setupFileDataTestDB(t)
	source := orientationJPEG(t, 3200, 1600)
	// Expected quadrant order: red=A, green=B, blue=C, yellow=D.
	orders := []string{"ABCD", "BADC", "DCBA", "CDAB", "ACBD", "CADB", "DBCA", "BDAC"}
	for i, quadrants := range orders {
		t.Run(fmt.Sprint(i+1), func(t *testing.T) {
			var order binary.ByteOrder = binary.BigEndian
			if i%2 == 1 {
				order = binary.LittleEndian
			}
			data := jpegWithOrientation(source, uint16(i+1), order)
			name := fmt.Sprintf("orientation-%d.jpg", i+1)
			if _, err := SaveFile(7, name, "image/jpeg", data); err != nil {
				t.Fatal(err)
			}
			media, err := ProcessUploadedImage(name, data)
			if err != nil {
				t.Fatal(err)
			}
			width, height := 3200, 1600
			if i+1 >= 5 {
				width, height = height, width
			}
			if media.Width != width || media.Height != height {
				t.Fatalf("dimensions = %dx%d, want %dx%d", media.Width, media.Height, width, height)
			}
			stored, err := ImageMetadataByNames([]string{name})
			if err != nil || stored[name].Width != width || stored[name].Height != height || len(stored[name].Variants) != 3 {
				t.Fatalf("persisted metadata = %+v, error = %v", stored[name], err)
			}
			if len(media.Variants) != 3 {
				t.Fatalf("variant count = %d, want 3", len(media.Variants))
			}
			for j, variant := range media.Variants {
				wantWidth := feedVariantWidths[j]
				wantHeight := height * wantWidth / width
				if variant.Width != wantWidth || variant.Height != wantHeight {
					t.Fatalf("variant = %+v, want %dx%d", variant, wantWidth, wantHeight)
				}
				row, err := GetFileByName(fmt.Sprintf("orientation-%d__w%d.jpg", i+1, wantWidth))
				if err != nil {
					t.Fatal(err)
				}
				decoded, err := jpeg.Decode(bytes.NewReader(row.Data))
				if err != nil || decoded.Bounds().Dx() != wantWidth || decoded.Bounds().Dy() != wantHeight {
					t.Fatalf("invalid derivative dimensions, error = %v", err)
				}
				for corner, want := range quadrants {
					x, y := wantWidth/4+(corner%2)*wantWidth/2, wantHeight/4+(corner/2)*wantHeight/2
					r, g, b, _ := decoded.At(x, y).RGBA()
					got := 'A'
					switch {
					case r > 40000 && g > 40000:
						got = 'D'
					case g > r && g > b:
						got = 'B'
					case b > r && b > g:
						got = 'C'
					}
					if got != want {
						t.Errorf("derivative %d quadrant %d = %c, want %c", wantWidth, corner, got, want)
					}
				}
			}
			original, err := GetFileByName(name)
			if err != nil || !bytes.Equal(original.Data, data) {
				t.Fatal("original upload bytes or EXIF changed")
			}
		})
	}
}

func TestProcessUploadedImageOrientationFallbacksAndLimits(t *testing.T) {
	setupFileDataTestDB(t)
	source := orientationJPEG(t, 400, 200)
	invalidOffset := jpegWithOrientation(source, 6, binary.BigEndian)
	binary.BigEndian.PutUint32(invalidOffset[16:], ^uint32(0))
	invalidType := jpegWithOrientation(source, 6, binary.BigEndian)
	binary.BigEndian.PutUint16(invalidType[24:], 4)
	invalidCount := jpegWithOrientation(source, 6, binary.BigEndian)
	binary.BigEndian.PutUint16(invalidCount[20:], 0xffff)
	oversized := jpegWithOrientation(source, 6, binary.BigEndian)
	// Change only SOF dimensions: DecodeConfig is valid, but full decoding would
	// fail. Success proves the 25MP guard runs before pixel decoding/rotation.
	sof := bytes.Index(oversized, []byte{0xff, 0xc0})
	if sof < 0 {
		t.Fatal("fixture missing JPEG SOF0")
	}
	binary.BigEndian.PutUint16(oversized[sof+5:], 4500)
	binary.BigEndian.PutUint16(oversized[sof+7:], 6000)
	for _, tc := range []struct {
		name          string
		data          []byte
		width, height int
		variants      int
	}{
		{"no-exif", source, 400, 200, 1},
		{"xmp-app1", jpegWithAPP1(source, []byte("http://ns.adobe.com/xap/1.0/\x00")), 400, 200, 1},
		{"truncated-exif", jpegWithAPP1(source, []byte("Exif\x00\x00MM")), 400, 200, 1},
		{"invalid-value", jpegWithOrientation(source, 9, binary.BigEndian), 400, 200, 1},
		{"invalid-offset", invalidOffset, 400, 200, 1},
		{"invalid-type", invalidType, 400, 200, 1},
		{"invalid-count", invalidCount, 400, 200, 1},
		{"small-portrait", jpegWithOrientation(source, 6, binary.BigEndian), 200, 400, 0},
		{"over-decode-limit", oversized, 4500, 6000, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name := tc.name + ".jpg"
			if _, err := SaveFile(7, name, "image/jpeg", tc.data); err != nil {
				t.Fatal(err)
			}
			media, err := ProcessUploadedImage(name, tc.data)
			if err != nil || media.Width != tc.width || media.Height != tc.height || len(media.Variants) != tc.variants {
				t.Fatalf("metadata = %+v, error = %v; want %dx%d with %d variants", media, err, tc.width, tc.height, tc.variants)
			}
		})
	}
}

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

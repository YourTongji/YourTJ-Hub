package filedata

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
)

// jpegOrientation reads only the JPEG APP1 EXIF IFD0 Orientation tag.
// Missing or malformed optional metadata falls back to the stored pixel order.
// Tag semantics: https://exiftool.org/TagNames/EXIF.html (Orientation, 0x0112).
func jpegOrientation(data []byte) int {
	if len(data) < 2 || data[0] != 0xff || data[1] != 0xd8 {
		return 1
	}
	for pos := 2; pos < len(data); {
		if data[pos] != 0xff {
			break
		}
		for pos < len(data) && data[pos] == 0xff {
			pos++
		}
		if pos >= len(data) {
			break
		}
		marker := data[pos]
		pos++
		if marker == 0xda || marker == 0xd9 { // SOS / EOI: do not scan compressed pixels.
			break
		}
		if marker == 0x01 || marker >= 0xd0 && marker <= 0xd8 {
			continue // Standalone markers have no length field.
		}
		if len(data)-pos < 2 {
			break
		}
		length := int(binary.BigEndian.Uint16(data[pos:]))
		if length < 2 || length > len(data)-pos {
			break
		}
		payload := data[pos+2 : pos+length]
		if marker == 0xe1 && bytes.HasPrefix(payload, []byte("Exif\x00\x00")) {
			if orientation := exifOrientation(payload[6:]); orientation != 1 {
				return orientation
			}
		}
		pos += length
	}
	return 1
}

func exifOrientation(tiff []byte) int {
	if len(tiff) < 8 {
		return 1
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 1
	}
	if order.Uint16(tiff[2:]) != 42 {
		return 1
	}
	offset := uint64(order.Uint32(tiff[4:]))
	if offset < 8 || offset+2 > uint64(len(tiff)) {
		return 1
	}
	entries := tiff[offset+2:]
	count := int(order.Uint16(tiff[offset:]))
	if count > len(entries)/12 {
		return 1
	}
	for i := 0; i < count; i++ {
		entry := entries[i*12 : (i+1)*12]
		if order.Uint16(entry) != 0x0112 {
			continue
		}
		if order.Uint16(entry[2:]) != 3 || order.Uint32(entry[4:]) != 1 {
			return 1 // Orientation must be one inline SHORT.
		}
		value := int(order.Uint16(entry[8:]))
		if value >= 1 && value <= 8 {
			return value
		}
		return 1
	}
	return 1
}

// orientedImage presents normalized pixels to the existing scaler without a
// second full-size image allocation. The original bytes and EXIF stay intact.
type orientedImage struct {
	image.Image
	orientation int
}

func (im orientedImage) Bounds() image.Rectangle {
	w, h := im.Image.Bounds().Dx(), im.Image.Bounds().Dy()
	if im.orientation >= 5 {
		w, h = h, w
	}
	return image.Rect(0, 0, w, h)
}

func (im orientedImage) At(x, y int) color.Color {
	if !image.Pt(x, y).In(im.Bounds()) {
		return color.RGBA{}
	}
	bounds := im.Image.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	switch im.orientation {
	case 2:
		x = w - 1 - x
	case 3:
		x, y = w-1-x, h-1-y
	case 4:
		y = h - 1 - y
	case 5:
		x, y = y, x
	case 6:
		x, y = y, h-1-x
	case 7:
		x, y = w-1-y, h-1-x
	case 8:
		x, y = w-1-y, x
	}
	return im.Image.At(bounds.Min.X+x, bounds.Min.Y+y)
}

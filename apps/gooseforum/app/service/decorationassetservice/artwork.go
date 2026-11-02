package decorationassetservice

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"io"
	"math"
	"path"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/imagepolicy"
	resvg "github.com/kanrichan/resvg-go"
	"github.com/tetratelabs/wazero"
)

var svgCompilationCache = wazero.NewCompilationCache()
var svgRenderSlots = make(chan struct{}, 2)

func ContentType(filename string) (string, bool) {
	switch strings.ToLower(path.Ext(filename)) {
	case ".svg":
		return "image/svg+xml", true
	case ".png", ".webp", ".gif":
		return imagepolicy.ContentTypeForFilename(filename)
	default:
		return "", false
	}
}

func prepareArtwork(ctx context.Context, data []byte, filename string, frame bool) ([]byte, string, error) {
	contentType, ok := ContentType(filename)
	if !ok || len(data) == 0 || len(data) > MaxArtworkBytes {
		return nil, "", ErrInvalid
	}
	if contentType == "image/svg+xml" {
		rendered, err := renderSVG(ctx, data, frame)
		if err != nil {
			return nil, "", err
		}
		if len(rendered) > MaxArtworkBytes {
			return nil, "", ErrInvalid
		}
		data = rendered
		filename = strings.TrimSuffix(filename, path.Ext(filename)) + ".png"
		contentType = "image/png"
	}
	if err := validateRaster(data, contentType, frame); err != nil {
		return nil, "", err
	}
	if !frame {
		normalized, name, err := normalizeBadge(ctx, data, filename, contentType)
		if err != nil {
			return nil, "", err
		}
		if len(normalized) > MaxArtworkBytes {
			return nil, "", ErrInvalid
		}
		return normalized, name, nil
	}
	return data, filename, nil
}

// Both clients contain artwork in a square, so mildly rectangular exports need no stretching.
func nearSquare(width, height float64) bool {
	return min(width, height) >= max(width, height)*0.85
}

// validateRaster 与上传路径同口径，以文件头判定的真实类型为准（issue #1393）；
// 装饰资源仍要求真实类型与声明格式一致（管理端资源不做改名）。
func validateRaster(data []byte, contentType string, frame bool) error {
	if contentType != "image/png" && contentType != "image/webp" && contentType != "image/gif" {
		return ErrInvalid
	}
	if detected, err := imagepolicy.DetectContentType(bytes.NewReader(data)); err != nil || detected != contentType {
		return ErrInvalid
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width < 1 || config.Height < 1 || config.Width > 4096 || config.Height > 4096 || (frame && !nearSquare(float64(config.Width), float64(config.Height))) {
		return ErrInvalid
	}
	if contentType == "image/png" && frame {
		for pos := 8; pos+12 <= len(data); {
			size := int(binary.BigEndian.Uint32(data[pos:]))
			if size > len(data)-pos-12 {
				return ErrInvalid
			}
			if string(data[pos+4:pos+8]) == "acTL" {
				return ErrInvalid
			}
			pos += 12 + size
		}
	}
	if contentType == "image/gif" {
		return validateGIF(data, frame)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return ErrInvalid
	}
	if frame && !usableOpening(img) {
		return ErrInvalid
	}
	return nil
}

// Check the whole circular opening: directional samples can miss opaque pixels between rays.
func usableOpening(img image.Image) bool {
	b := img.Bounds()
	radius := float64(min(b.Dx(), b.Dy())) * minArtworkOpeningRatio / 2
	cx, cy := float64(b.Min.X+b.Dx()/2), float64(b.Min.Y+b.Dy()/2)
	for y := max(b.Min.Y, int(math.Ceil(cy-radius))); y <= min(b.Max.Y-1, int(math.Floor(cy+radius))); y++ {
		for x := max(b.Min.X, int(math.Ceil(cx-radius))); x <= min(b.Max.X-1, int(math.Floor(cx+radius))); x++ {
			dx, dy := float64(x)-cx, float64(y)-cy
			if dx*dx+dy*dy >= radius*radius {
				continue
			}
			if _, _, _, a := img.At(x, y).RGBA(); a > 257 {
				return false
			}
		}
	}
	return true
}

// Bound frame allocations before gif.DecodeAll expands attacker-controlled LZW data.
func gifBudget(data []byte) bool {
	if len(data) < 13 {
		return false
	}
	pos := 13
	if data[10]&128 != 0 {
		pos += 3 * (1 << ((data[10] & 7) + 1))
	}
	frames, pixels := 0, 0
	canvasPixels := int(binary.LittleEndian.Uint16(data[6:])) * int(binary.LittleEndian.Uint16(data[8:]))
	skipBlocks := func() bool {
		for pos < len(data) {
			n := int(data[pos])
			pos++
			if n == 0 {
				return true
			}
			pos += n
		}
		return false
	}
	for pos < len(data) {
		tag := data[pos]
		pos++
		switch tag {
		case 0x3b:
			return frames > 0 && pos == len(data)
		case 0x21:
			pos++
			if !skipBlocks() {
				return false
			}
		case 0x2c:
			if pos+9 > len(data) {
				return false
			}
			w := int(binary.LittleEndian.Uint16(data[pos+4:]))
			h := int(binary.LittleEndian.Uint16(data[pos+6:]))
			packed := data[pos+8]
			pos += 9
			frames++
			pixels += w * h
			if frames > animatedArtworkMaxFrames || pixels > animatedArtworkMaxPixels || frames*canvasPixels > animatedArtworkMaxPixels {
				return false
			}
			if packed&128 != 0 {
				pos += 3 * (1 << ((packed & 7) + 1))
			}
			pos++
			if !skipBlocks() {
				return false
			}
		default:
			return false
		}
	}
	return false
}

func validateGIF(data []byte, frame bool) error {
	if !gifBudget(data) {
		return ErrInvalid
	}
	animation, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return ErrInvalid
	}
	return visitGIF(animation, func(img image.Image) error {
		if frame && !usableOpening(img) {
			return ErrInvalid
		}
		return nil
	})
}

func visitGIF(animation *gif.GIF, visit func(image.Image) error) error {
	canvas := image.NewNRGBA(image.Rect(0, 0, animation.Config.Width, animation.Config.Height))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(gifBackground(animation, animation.Image[0])), image.Point{}, draw.Src)
	for i, img := range animation.Image {
		if !img.Bounds().In(canvas.Bounds()) {
			return ErrInvalid
		}
		var previous *image.NRGBA
		disposal := byte(0)
		if i < len(animation.Disposal) {
			disposal = animation.Disposal[i]
		}
		if disposal == gif.DisposalPrevious {
			previous = image.NewNRGBA(canvas.Bounds())
			copy(previous.Pix, canvas.Pix)
		}
		draw.Draw(canvas, img.Bounds(), img, img.Bounds().Min, draw.Over)
		if err := visit(canvas); err != nil {
			return err
		}
		switch disposal {
		case gif.DisposalBackground:
			draw.Draw(canvas, img.Bounds(), image.NewUniform(gifBackground(animation, img)), image.Point{}, draw.Src)
		case gif.DisposalPrevious:
			copy(canvas.Pix, previous.Pix)
		}
	}
	return nil
}

func gifBackground(animation *gif.GIF, img *image.Paletted) color.Color {
	for _, c := range img.Palette {
		if _, _, _, a := c.RGBA(); a == 0 {
			return color.Transparent
		}
	}
	if palette, ok := animation.Config.ColorModel.(color.Palette); ok && int(animation.BackgroundIndex) < len(palette) {
		return palette[animation.BackgroundIndex]
	}
	return color.Transparent
}

func renderSVG(ctx context.Context, data []byte, frame bool) ([]byte, error) {
	// Never give the renderer a path, embedded image, script or external-resource reference.
	decoder := xml.NewDecoder(bytes.NewReader(data))
	depth, nodes, styleDepth := 0, 0, 0
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, ErrInvalid
		}
		switch item := token.(type) {
		case xml.Directive:
			return nil, ErrInvalid
		case xml.StartElement:
			depth++
			nodes++
			if depth > 64 || nodes > 10000 || (nodes == 1 && item.Name.Local != "svg") {
				return nil, ErrInvalid
			}
			if item.Name.Space != "" && item.Name.Space != "http://www.w3.org/2000/svg" {
				return nil, ErrInvalid
			}
			switch strings.ToLower(item.Name.Local) {
			case "script", "foreignobject", "image", "feimage", "text", "tspan", "animate", "animatetransform", "animatemotion", "set":
				return nil, ErrInvalid
			}
			if item.Name.Local == "style" {
				styleDepth = depth
			}
			for _, attr := range item.Attr {
				name, value := strings.ToLower(attr.Name.Local), strings.TrimSpace(attr.Value)
				if strings.HasPrefix(name, "on") || !safeSVGPaint(value) {
					return nil, ErrInvalid
				}
				if name == "href" && !strings.HasPrefix(value, "#") {
					return nil, ErrInvalid
				}
				if attr.Name.Space == "http://www.w3.org/XML/1998/namespace" && name == "base" {
					return nil, ErrInvalid
				}
			}
		case xml.CharData:
			if styleDepth != 0 && !safeSVGPaint(string(item)) {
				return nil, ErrInvalid
			}
		case xml.ProcInst:
			if item.Target != "xml" {
				return nil, ErrInvalid
			}
		case xml.EndElement:
			if styleDepth == depth {
				styleDepth = 0
			}
			depth--
		}
	}
	if nodes == 0 || depth != 0 {
		return nil, ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	select {
	case svgRenderSlots <- struct{}{}:
		defer func() { <-svgRenderSlots }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	worker, err := resvg.NewWorker(ctx, wazero.NewRuntimeConfig().WithMemoryLimitPages(4096).WithCloseOnContextDone(true).WithCompilationCache(svgCompilationCache))
	if err != nil {
		return nil, err
	}
	defer func() { _ = worker.Close() }()
	tree, err := worker.NewTreeFromData(data, &resvg.Options{})
	if err != nil {
		return nil, ErrInvalid
	}
	defer func() { _ = tree.Close() }()
	w, h, err := tree.GetSize()
	if err != nil || math.IsNaN(float64(w)) || math.IsNaN(float64(h)) || w < 1 || h < 1 || w > 4096 || h > 4096 || (frame && !nearSquare(float64(w), float64(h))) {
		return nil, ErrInvalid
	}
	edge := float32(512)
	if frame {
		edge = 1024
	}
	scale := edge / max(w, h)
	pixmap, err := worker.NewPixmap(uint32(math.Ceil(float64(w*scale))), uint32(math.Ceil(float64(h*scale))))
	if err != nil {
		return nil, ErrInvalid
	}
	defer func() { _ = pixmap.Close() }()
	if err := tree.Render(resvg.TransformFromScale(scale, scale), pixmap); err != nil {
		return nil, ErrInvalid
	}
	png, err := pixmap.EncodePNG()
	if err != nil {
		return nil, ErrInvalid
	}
	return png, nil
}

// CSS paint servers may refer only to local fragments; external imports and escapes are not accepted.
func safeSVGPaint(value string) bool {
	if strings.ContainsAny(value, "\\@") {
		return false
	}
	for rest := strings.ToLower(value); strings.Contains(rest, "url("); {
		_, rest, _ = strings.Cut(rest, "url(")
		ref, tail, ok := strings.Cut(rest, ")")
		if !ok || !strings.HasPrefix(strings.Trim(ref, " \t\r\n'\""), "#") {
			return false
		}
		rest = tail
	}
	return true
}

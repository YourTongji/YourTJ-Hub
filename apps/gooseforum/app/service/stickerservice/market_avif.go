package stickerservice

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"math"
	"sort"
	"sync"

	"github.com/gen2brain/avif"
	xdraw "golang.org/x/image/draw"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
)

// Market packs often ship AVIF, which neither the upload policy nor Flutter
// accepts. The import transcodes it once: stills become PNG (full alpha),
// animations become GIF, the format the market already serves for animated
// packs. The decoder is libavif/dav1d compiled to WASM; release builds use
// the nodynamic tag so it never loads a system libavif.
const (
	// maxMarketAVIFFrames and marketAVIFPixelBudget bound decoded memory before
	// any frame is decoded: 512x512 x 120 frames is about 126 MB as NRGBA.
	maxMarketAVIFFrames   = 240
	marketAVIFPixelBudget = 512 * 512 * 120
	// marketGIFTarget is the preferred size of a transcoded animation; larger
	// results retry at the next smaller side, up to the storage limit.
	marketGIFTarget = 2 << 20
)

// marketGIFSides are the long-side caps tried in order for animations.
var marketGIFSides = []int{512, 384, 256}

// One transcode at a time keeps the decoded frames of a single sticker as the
// memory peak; other items keep downloading meanwhile.
var marketTranscodeMu sync.Mutex

// avifBrand reports whether data starts with an AVIF ftyp box. Image sequences
// carry the avis brand.
func avifBrand(data []byte) (sequence, ok bool) {
	if len(data) < 16 || string(data[4:8]) != "ftyp" {
		return false, false
	}
	size := int(binary.BigEndian.Uint32(data))
	if size < 16 || size > len(data) {
		return false, false
	}
	brands := [][]byte{data[8:12]}
	for offset := 16; offset+4 <= size; offset += 4 {
		brands = append(brands, data[offset:offset+4])
	}
	for _, brand := range brands {
		switch string(brand) {
		case "avis":
			sequence, ok = true, true
		case "avif":
			ok = true
		}
	}
	return sequence, ok
}

// avifFrameCount returns the largest sample count of any track (color and
// alpha tracks match), or 0 when the file has no movie box.
func avifFrameCount(data []byte) (int, error) {
	moov, found, err := isoChild(data, "moov")
	if err != nil || !found {
		return 0, err
	}
	frames := 0
	err = isoEach(moov, "trak", func(trak []byte) error {
		stsz, err := isoPath(trak, "mdia", "minf", "stbl", "stsz")
		if err != nil {
			return err
		}
		// FullBox header (4 bytes), sample_size (4), sample_count (4).
		if len(stsz) < 12 {
			return errMarketInvalidImage
		}
		frames = max(frames, int(binary.BigEndian.Uint32(stsz[8:12])))
		return nil
	})
	if err == nil && frames == 0 {
		err = errMarketInvalidImage
	}
	return frames, err
}

// isoEach calls visit with the payload of every direct child box of kind.
func isoEach(data []byte, kind string, visit func([]byte) error) error {
	for offset := 0; offset < len(data); {
		if len(data)-offset < 8 {
			return errMarketInvalidImage
		}
		size, header := uint64(binary.BigEndian.Uint32(data[offset:])), 8
		switch size {
		case 0:
			size = uint64(len(data) - offset)
		case 1:
			if len(data)-offset < 16 {
				return errMarketInvalidImage
			}
			size, header = binary.BigEndian.Uint64(data[offset+8:]), 16
		}
		if size < uint64(header) || size > uint64(len(data)-offset) {
			return errMarketInvalidImage
		}
		if string(data[offset+4:offset+8]) == kind {
			if err := visit(data[offset+header : offset+int(size)]); err != nil {
				return err
			}
		}
		offset += int(size)
	}
	return nil
}

var errISOFound = errors.New("found")

func isoChild(data []byte, kind string) ([]byte, bool, error) {
	var child []byte
	err := isoEach(data, kind, func(payload []byte) error {
		child = payload
		return errISOFound
	})
	if errors.Is(err, errISOFound) {
		return child, true, nil
	}
	return nil, false, err
}

func isoPath(data []byte, kinds ...string) ([]byte, error) {
	for _, kind := range kinds {
		child, found, err := isoChild(data, kind)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, errMarketInvalidImage
		}
		data = child
	}
	return data, nil
}

// transcodeMarketAVIF converts AVIF bytes to PNG or GIF; callers re-detect the
// output type. Other formats the upload policy rejects (HEIC, TIFF) stay unsupported.
func transcodeMarketAVIF(data []byte) ([]byte, error) {
	if _, ok := avifBrand(data); !ok {
		return nil, errMarketUnsupportedImage
	}
	config, err := avif.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width < 1 || config.Height < 1 {
		return nil, errMarketInvalidImage
	}
	if config.Width > maxMarketImageSide || config.Height > maxMarketImageSide {
		return nil, errMarketImageTooLarge
	}
	frames, err := avifFrameCount(data)
	if err != nil {
		return nil, errMarketInvalidImage
	}
	frames = max(frames, 1)
	if frames > maxMarketAVIFFrames || (frames > 1 && config.Width*config.Height*frames > marketAVIFPixelBudget) {
		return nil, errMarketImageTooLarge
	}

	marketTranscodeMu.Lock()
	defer marketTranscodeMu.Unlock()
	anim, err := decodeMarketAVIF(data)
	if err != nil {
		return nil, errMarketInvalidImage
	}
	if len(anim.Image) == 0 || len(anim.Image) > frames {
		return nil, errMarketInvalidImage
	}
	for i, frame := range anim.Image {
		anim.Image[i] = toNRGBA(frame)
	}
	if len(anim.Image) == 1 {
		var out bytes.Buffer
		if err := png.Encode(&out, anim.Image[0]); err != nil {
			return nil, errMarketInvalidImage
		}
		if out.Len() > filedata.MaxFileSize {
			return nil, errMarketImageTooLarge
		}
		return out.Bytes(), nil
	}
	var best []byte
	for _, side := range marketGIFSides {
		if best != nil && side >= max(config.Width, config.Height) {
			continue
		}
		out, err := encodeMarketGIF(anim, side)
		if err != nil {
			return nil, errMarketInvalidImage
		}
		if best == nil || len(out) < len(best) {
			best = out
		}
		if len(out) <= marketGIFTarget {
			break
		}
	}
	if len(best) > filedata.MaxFileSize {
		return nil, errMarketImageTooLarge
	}
	return best, nil
}

// decodeMarketAVIF turns a decoder panic on hostile input into an error.
func decodeMarketAVIF(data []byte) (anim *avif.AVIF, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			anim, err = nil, fmt.Errorf("avif decode panic: %v", recovered)
		}
	}()
	return avif.DecodeAll(bytes.NewReader(data))
}

func toNRGBA(img image.Image) *image.NRGBA {
	bounds := img.Bounds()
	if nrgba, ok := img.(*image.NRGBA); ok && bounds.Min == (image.Point{}) {
		return nrgba
	}
	nrgba := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(nrgba, nrgba.Bounds(), img, bounds.Min, draw.Src)
	return nrgba
}

// encodeMarketGIF scales frames to fit side, quantizes them to one shared
// palette (stable colors across frames) and merges identical neighbours.
// GIF transparency is binary: alpha below one half becomes transparent.
func encodeMarketGIF(anim *avif.AVIF, side int) ([]byte, error) {
	frames := make([]*image.NRGBA, len(anim.Image))
	for i, img := range anim.Image {
		frames[i] = scaleToFit(img.(*image.NRGBA), side)
	}
	palette := marketPalette(frames)
	nearest := paletteMatcher(palette)
	bounds := frames[0].Bounds()
	out := &gif.GIF{LoopCount: gifLoopCount(anim.LoopCount)}
	elapsed, emitted := 0.0, 0
	for i, frame := range frames {
		if !frame.Bounds().Eq(bounds) {
			return nil, errMarketInvalidImage
		}
		paletted := image.NewPaletted(bounds, palette)
		for y := range bounds.Dy() {
			row := frame.Pix[y*frame.Stride:]
			for x := range bounds.Dx() {
				if pixel := row[x*4 : x*4+4]; pixel[3] >= 128 {
					paletted.Pix[y*paletted.Stride+x] = nearest(rgbKey(pixel))
				}
			}
		}
		delay := 0.1
		if i < len(anim.Delay) && anim.Delay[i] > 0 {
			delay = anim.Delay[i]
		}
		// Round the running total so many short frames keep the source timing.
		elapsed += delay
		centiseconds := max(int(math.Round(elapsed*100))-emitted, 2)
		emitted += centiseconds
		if last := len(out.Image) - 1; last >= 0 && bytes.Equal(out.Image[last].Pix, paletted.Pix) {
			out.Delay[last] += centiseconds
			continue
		}
		out.Image = append(out.Image, paletted)
		out.Delay = append(out.Delay, centiseconds)
		// Clear each frame so transparent areas never show the previous one.
		out.Disposal = append(out.Disposal, gif.DisposalBackground)
	}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, out); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// gifLoopCount maps AVIF play counts (0 = forever) to GIF repeats (0 = forever, -1 = once).
func gifLoopCount(plays int) int {
	switch {
	case plays <= 0:
		return 0
	case plays == 1:
		return -1
	default:
		return plays - 1
	}
}

func scaleToFit(img *image.NRGBA, side int) *image.NRGBA {
	bounds := img.Bounds()
	long := max(bounds.Dx(), bounds.Dy())
	if long <= side {
		return img
	}
	width := max(1, bounds.Dx()*side/long)
	height := max(1, bounds.Dy()*side/long)
	scaled := image.NewNRGBA(image.Rect(0, 0, width, height))
	xdraw.CatmullRom.Scale(scaled, scaled.Bounds(), img, bounds, xdraw.Src, nil)
	return scaled
}

func rgbKey(pixel []byte) uint32 {
	return uint32(pixel[0])<<16 | uint32(pixel[1])<<8 | uint32(pixel[2])
}

// marketPalette median-cuts a sample of opaque pixels from every frame into
// 255 colors; index 0 is the transparent color.
func marketPalette(frames []*image.NRGBA) color.Palette {
	const maxSamples = 1 << 18
	total := 0
	for _, frame := range frames {
		total += len(frame.Pix) / 4
	}
	step := max(1, total/maxSamples)
	samples := make([]uint32, 0, min(total, maxSamples))
	index := 0
	for _, frame := range frames {
		for offset := 0; offset+4 <= len(frame.Pix); offset += 4 {
			index++
			if index%step == 0 && frame.Pix[offset+3] >= 128 {
				samples = append(samples, rgbKey(frame.Pix[offset:offset+4]))
			}
		}
	}
	palette := color.Palette{color.NRGBA{}}
	if len(samples) == 0 {
		return append(palette, color.NRGBA{A: 255})
	}
	boxes := [][]uint32{samples}
	for len(boxes) < 255 {
		widest, channel, span := -1, 0, uint32(0)
		for i, box := range boxes {
			if len(box) < 2 {
				continue
			}
			if c, s := widestChannel(box); widest < 0 || s > span {
				widest, channel, span = i, c, s
			}
		}
		if widest < 0 || span == 0 {
			break
		}
		box := boxes[widest]
		shift := 16 - 8*channel
		sort.Slice(box, func(i, j int) bool { return box[i]>>shift&0xff < box[j]>>shift&0xff })
		boxes[widest] = box[:len(box)/2]
		boxes = append(boxes, box[len(box)/2:])
	}
	for _, box := range boxes {
		var r, g, b int
		for _, c := range box {
			r += int(c >> 16 & 0xff)
			g += int(c >> 8 & 0xff)
			b += int(c & 0xff)
		}
		n := len(box)
		palette = append(palette, color.NRGBA{R: uint8(r / n), G: uint8(g / n), B: uint8(b / n), A: 255})
	}
	return palette
}

func widestChannel(box []uint32) (int, uint32) {
	channel, span := 0, uint32(0)
	for c := range 3 {
		shift := 16 - 8*c
		lo, hi := uint32(255), uint32(0)
		for _, v := range box {
			lo, hi = min(lo, v>>shift&0xff), max(hi, v>>shift&0xff)
		}
		if hi-lo > span {
			channel, span = c, hi-lo
		}
	}
	return channel, span
}

// paletteMatcher finds the closest opaque palette entry, weighting green
// highest as the eye does, and memoizes repeated colors.
func paletteMatcher(palette color.Palette) func(uint32) uint8 {
	cache := make(map[uint32]uint8)
	return func(key uint32) uint8 {
		if index, ok := cache[key]; ok {
			return index
		}
		r, g, b := int(key>>16&0xff), int(key>>8&0xff), int(key&0xff)
		best, distance := 1, math.MaxInt
		for i := 1; i < len(palette); i++ {
			entry := palette[i].(color.NRGBA)
			dr, dg, db := r-int(entry.R), g-int(entry.G), b-int(entry.B)
			if d := 2*dr*dr + 4*dg*dg + 3*db*db; d < distance {
				best, distance = i, d
			}
		}
		cache[key] = uint8(best)
		return uint8(best)
	}
}

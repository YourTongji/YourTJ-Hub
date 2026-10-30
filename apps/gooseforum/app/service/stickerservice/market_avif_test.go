package stickerservice

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/png"
	"os"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
)

func readMarketFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestAVIFContainerProbe(t *testing.T) {
	animated, still := readMarketFixture(t, "market-animated.avif"), readMarketFixture(t, "market-still.avif")
	if sequence, ok := avifBrand(animated); !ok || !sequence {
		t.Fatalf("animated brand = (%v, %v), want an AVIF sequence", sequence, ok)
	}
	if sequence, ok := avifBrand(still); !ok || sequence {
		t.Fatalf("still brand = (%v, %v), want a still AVIF", sequence, ok)
	}
	if _, ok := avifBrand([]byte("\x00\x00\x00\x18ftypheic\x00\x00\x00\x00mif1heic")); ok {
		t.Fatal("HEIC must not be treated as AVIF")
	}
	if frames, err := avifFrameCount(animated); err != nil || frames != 3 {
		t.Fatalf("animated frame count = %d, %v; want 3", frames, err)
	}
	if frames, err := avifFrameCount(still); err != nil || frames != 0 {
		t.Fatalf("still frame count = %d, %v; want 0 without a movie box", frames, err)
	}
	truncated := append([]byte{}, animated[:bytes.Index(animated, []byte("moov"))+64]...)
	if _, err := avifFrameCount(truncated); err == nil {
		t.Fatal("a truncated movie box must be rejected")
	}
}

// A small file may declare enormous frame counts; the budget applies before decoding.
func TestTranscodeMarketAVIFRejectsFrameBombsBeforeDecoding(t *testing.T) {
	data := readMarketFixture(t, "market-animated.avif")
	for offset := 0; ; {
		next := bytes.Index(data[offset:], []byte("stsz"))
		if next < 0 {
			break
		}
		offset += next
		binary.BigEndian.PutUint32(data[offset+12:], 100000)
		offset += 4
	}
	if _, err := transcodeMarketAVIF(data); !errors.Is(err, errMarketImageTooLarge) {
		t.Fatalf("frame bomb error = %v, want tooLarge", err)
	}
}

func TestTranscodeMarketAVIFAnimationToGIF(t *testing.T) {
	out, err := transcodeMarketAVIF(readMarketFixture(t, "market-animated.avif"))
	if err != nil {
		t.Fatal(err)
	}
	anim, err := gif.DecodeAll(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	// Frames 2 and 3 are identical and merge; 0.05 s + 0.05 s keeps the source timing.
	if len(anim.Image) != 2 || fmt.Sprint(anim.Delay) != "[10 10]" || anim.LoopCount != 0 {
		t.Fatalf("gif frames=%d delays=%v loop=%d, want 2 frames [10 10] looping", len(anim.Image), anim.Delay, anim.LoopCount)
	}
	for i, disposal := range anim.Disposal {
		if disposal != gif.DisposalBackground {
			t.Fatalf("frame %d disposal = %d, want background", i, disposal)
		}
	}
	first, second := anim.Image[0], anim.Image[1]
	if _, _, _, a := first.At(0, 0).RGBA(); a != 0 {
		t.Fatal("transparent corner became opaque")
	}
	if r, g, _, a := first.At(5, 8).RGBA(); a == 0 || r>>8 < 150 || g>>8 > 110 {
		t.Fatalf("first frame square = %v, want opaque red", first.At(5, 8))
	}
	if r, g, _, a := second.At(10, 8).RGBA(); a == 0 || g>>8 < 140 || r>>8 > 110 {
		t.Fatalf("second frame square = %v, want opaque green", second.At(10, 8))
	}
	if _, _, _, a := second.At(3, 8).RGBA(); a != 0 {
		t.Fatal("the second frame shows the first frame's square through its transparency")
	}
}

func TestTranscodeMarketAVIFStillToPNG(t *testing.T) {
	out, err := transcodeMarketAVIF(readMarketFixture(t, "market-still.avif"))
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds() != image.Rect(0, 0, 16, 16) {
		t.Fatalf("png bounds = %v", img.Bounds())
	}
	if _, _, _, a := img.At(0, 0).RGBA(); a != 0 {
		t.Fatal("still PNG lost its transparency")
	}
	if _, _, _, a := img.At(8, 8).RGBA(); a>>8 < 250 {
		t.Fatal("still PNG square is not opaque")
	}
}

func TestTranscodeMarketAVIFKeepsOtherFormatsUnsupported(t *testing.T) {
	if _, err := transcodeMarketAVIF([]byte("II*\x00rest-of-a-tiff")); !errors.Is(err, errMarketUnsupportedImage) {
		t.Fatalf("TIFF error = %v, want unsupported", err)
	}
}

func TestMarketImportTranscodesAVIF(t *testing.T) {
	fake := setupMarketTest(t)
	fake.catalog(marketGroup("group-8", "AVIF Pack", 2))
	if _, err := SyncMarketCatalog(context.Background()); err != nil {
		t.Fatal(err)
	}
	fake.json[testMarketBase+"group-8.json"] = `{"id":"group-8","emojis":[
		{"id":"moving","url":"https://cdn3.ldstatic.com/moving.avif"},
		{"id":"still","url":"https://cdn3.ldstatic.com/still.avif"}]}`
	fake.images["https://cdn3.ldstatic.com/moving.avif"] = readMarketFixture(t, "market-animated.avif")
	fake.images["https://cdn3.ldstatic.com/still.avif"] = readMarketFixture(t, "market-still.avif")
	pack := runQueuedMarketImport(t, packByUpstream(t, "group-8").Id)
	if pack.Status != sticker.MarketStatusReady || pack.ItemCount != 2 {
		t.Fatalf("AVIF import = %#v, want both items imported", pack)
	}
	items, _ := sticker.MarketItems(pack.Id)
	extensions := map[string]string{}
	for _, item := range items {
		row, err := sticker.GetById(item.StickerID)
		if err != nil {
			t.Fatal(err)
		}
		extensions[item.UpstreamID] = row.FileName[strings.LastIndex(row.FileName, "."):]
	}
	if extensions["moving"] != ".gif" || extensions["still"] != ".png" {
		t.Fatalf("stored extensions = %v, want GIF for the animation and PNG for the still", extensions)
	}
}

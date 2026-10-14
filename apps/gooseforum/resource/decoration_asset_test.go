package resource

import (
	"bytes"
	"image/png"
	"io/fs"
	"testing"
)

func TestBuiltInAvatarFrameHasTransparentOpening(t *testing.T) {
	static, err := GetStaticFS()
	if err != nil {
		t.Fatal(err)
	}
	data, err := fs.ReadFile(static, "decorations/tea-garden-v1.png")
	if err != nil {
		t.Fatal(err)
	}
	frame, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	b := frame.Bounds()
	if b.Dx() != b.Dy() {
		t.Fatalf("non-square frame: %v", b)
	}
	for y := b.Dy() * 35 / 100; y < b.Dy()*65/100; y++ {
		for x := b.Dx() * 35 / 100; x < b.Dx()*65/100; x++ {
			_, _, _, alpha := frame.At(x, y).RGBA()
			// ImageGen antialias noise is at most 1/255; no portrait is obscured.
			if alpha > 257 {
				t.Fatalf("opaque centre pixel (%d,%d), alpha=%d", x, y, alpha)
			}
		}
	}
}

package anonymousnames

import (
	"encoding/xml"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"
)

func TestAvatarStableSeedAndNoIdentityText(t *testing.T) {
	seed := strings.Repeat("a", 32)
	got := Avatar(seed)
	if got != Avatar(seed) || got == Avatar(strings.Repeat("c", 32)) {
		t.Fatal("unstable or constant avatar")
	}
	var element struct{ XMLName xml.Name }
	if err := xml.Unmarshal([]byte(got), &element); err != nil || element.XMLName.Local != "svg" {
		t.Fatal(element, err)
	}
	if strings.Contains(got, seed) || strings.Contains(got, "script") {
		t.Fatal("identity or active script in SVG")
	}
}

// SVG clamps oversized rect radii in browsers, but mobile vector renderers can
// produce a self-crossing path. Emit the equivalent bounded radius explicitly.
func TestAvatarRoundedRectsFitBoxForMobileRendering(t *testing.T) {
	for _, seed := range []string{"preview-persona-seed", strings.Repeat("a", 32)} {
		decoder := xml.NewDecoder(strings.NewReader(Avatar(seed)))
		for {
			token, err := decoder.Token()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			element, ok := token.(xml.StartElement)
			if !ok || element.Name.Local != "rect" {
				continue
			}
			values := map[string]float64{}
			for _, attr := range element.Attr {
				if attr.Name.Local == "width" || attr.Name.Local == "rx" {
					value, err := strconv.ParseFloat(attr.Value, 64)
					if err != nil {
						t.Fatal(err)
					}
					values[attr.Name.Local] = value
				}
			}
			if values["rx"] > values["width"]/2 {
				t.Fatalf("radius %g exceeds half-width %g for seed %q", values["rx"], values["width"]/2, seed)
			}
		}
	}
}

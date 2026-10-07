package anonymousnames

import (
	"encoding/xml"
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

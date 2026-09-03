package imagepolicy

import (
	"reflect"
	"testing"
)

func TestContentTypeForExt(t *testing.T) {
	cases := map[string]string{
		".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png",
		".gif": "image/gif", ".webp": "image/webp", ".bmp": "image/bmp",
		".JPG": "image/jpeg", ".PNG": "image/png", "png": "image/png",
	}
	for raw, want := range cases {
		got, ok := ContentTypeForExt(raw)
		if !ok || got != want {
			t.Errorf("ContentTypeForExt(%q) = %q, %v; want %q, true", raw, got, ok, want)
		}
	}
	for _, raw := range []string{".svg", ".html", ".htm", ".xml", ".js", ".css", ".json", ".pdf", "avatar.png.exe", ""} {
		if got, ok := ContentTypeForExt(raw); ok {
			t.Errorf("ContentTypeForExt(%q) = %q, true; want unsupported", raw, got)
		}
	}
}

func TestContentTypeForFilename(t *testing.T) {
	if got, ok := ContentTypeForFilename("2026/09/01/uuid.PNG"); !ok || got != "image/png" {
		t.Fatalf("ContentTypeForFilename(uuid.PNG) = %q, %v; want image/png, true", got, ok)
	}
	if _, ok := ContentTypeForFilename("avatar.png.exe"); ok {
		t.Fatal("ContentTypeForFilename(avatar.png.exe) allowed a double extension")
	}
}

func TestDecodedFormatContentType(t *testing.T) {
	cases := map[string]string{
		"jpeg": "image/jpeg", "png": "image/png", "gif": "image/gif",
		"webp": "image/webp", "bmp": "image/bmp",
	}
	for format, want := range cases {
		if got, ok := DecodedFormatContentType(format); !ok || got != want {
			t.Errorf("DecodedFormatContentType(%q) = %q, %v; want %q, true", format, got, ok, want)
		}
	}
	if _, ok := DecodedFormatContentType("svg"); ok {
		t.Fatal("DecodedFormatContentType(svg) allowed an undecodable format")
	}
}

func TestCanonicalizeList(t *testing.T) {
	valid, dropped := CanonicalizeList([]string{".PNG", "png", ".jpg", ".svg", ".html", ".bmp"})
	if want := []string{".png", ".jpg", ".bmp"}; !reflect.DeepEqual(valid, want) {
		t.Errorf("CanonicalizeList valid = %#v, want %#v", valid, want)
	}
	if want := []string{".svg", ".html"}; !reflect.DeepEqual(dropped, want) {
		t.Errorf("CanonicalizeList dropped = %#v, want %#v", dropped, want)
	}

	valid, dropped = CanonicalizeList([]string{"avatar.png.exe", "x.jpg", "jpeg"})
	if want := []string{".jpeg"}; !reflect.DeepEqual(valid, want) {
		t.Errorf("CanonicalizeList valid = %#v, want %#v", valid, want)
	}
	if want := []string{"avatar.png.exe", "x.jpg"}; !reflect.DeepEqual(dropped, want) {
		t.Errorf("CanonicalizeList dropped = %#v, want the impure extension tokens only", dropped)
	}

	valid, dropped = CanonicalizeList([]string{".svg", ".html", "js"})
	if len(valid) != 0 {
		t.Errorf("CanonicalizeList valid = %#v, want empty", valid)
	}
	if len(dropped) != 3 {
		t.Errorf("CanonicalizeList dropped = %#v, want all three dangerous tokens", dropped)
	}
}

func TestCanonicalizeListDeduplicates(t *testing.T) {
	valid, dropped := CanonicalizeList([]string{"png", ".png", "JPG", ".jpg"})
	if want := []string{".png", ".jpg"}; !reflect.DeepEqual(valid, want) {
		t.Errorf("CanonicalizeList valid = %#v, want %#v", valid, want)
	}
	if len(dropped) != 0 {
		t.Errorf("CanonicalizeList dropped = %#v, want none", dropped)
	}
}

func TestEffectiveAllowedExtensions(t *testing.T) {
	if got := EffectiveAllowedExtensions(nil); !reflect.DeepEqual(got, DefaultExtensions()) {
		t.Errorf("empty config effective = %#v, want defaults %#v", got, DefaultExtensions())
	}
	valid := EffectiveAllowedExtensions([]string{".webp"})
	if want := []string{".webp"}; !reflect.DeepEqual(valid, want) {
		t.Errorf("subset config effective = %#v, want %#v", valid, want)
	}
	// 集合外条目不会扩大生效列表；只剩危险条目时回退默认全集，危险条目仍被排除。
	if got := EffectiveAllowedExtensions([]string{".svg", ".html"}); !reflect.DeepEqual(got, DefaultExtensions()) {
		t.Errorf("dangerous-only config effective = %#v, want defaults %#v", got, DefaultExtensions())
	}
}

func TestIsAllowedExt(t *testing.T) {
	allowed := []string{".jpg", ".png"}
	if !IsAllowedExt(".PNG", allowed) || !IsAllowedExt("png", allowed) {
		t.Error("IsAllowedExt accepted canonical jpg/png variants")
	}
	if IsAllowedExt(".webp", allowed) || IsAllowedExt(".svg", allowed) {
		t.Error("IsAllowedExt rejected out-of-list extensions")
	}
}

func TestFilterConfiguredList(t *testing.T) {
	kept, dropped := FilterConfiguredList([]string{"png", "jpg", ".svg", ".html", "avatar.png.exe"})
	if want := []string{"png", "jpg"}; !reflect.DeepEqual(kept, want) {
		t.Errorf("FilterConfiguredList kept = %#v, want %#v", kept, want)
	}
	if want := []string{".svg", ".html", "avatar.png.exe"}; !reflect.DeepEqual(dropped, want) {
		t.Errorf("FilterConfiguredList dropped = %#v, want %#v", dropped, want)
	}

	if kept, dropped := FilterConfiguredList(nil); len(kept) != 0 || len(dropped) != 0 {
		t.Errorf("FilterConfiguredList(nil) = %#v, %#v; want empty", kept, dropped)
	}
}

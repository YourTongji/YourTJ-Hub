package topics

import (
	"fmt"
	"reflect"
	"testing"
)

func TestPublicGallery(t *testing.T) {
	long := make([]string, 0, 11)
	for i := 0; i < 11; i++ {
		long = append(long, fmt.Sprintf("/file/img/%d.webp", i))
	}
	for _, tc := range []struct {
		name string
		in   Entity
		want []string
	}{
		{"gallery wins", Entity{ImageUrls: []string{"/a", "/b"}, FirstImageURL: "/c"}, []string{"/a", "/b"}},
		{"fallback", Entity{FirstImageURL: "/c"}, []string{"/c"}},
		{"empty", Entity{}, []string{}},
		{"capped", Entity{ImageUrls: long}, long[:MaxPublicGalleryImages]},
	} {
		got := tc.in.PublicGallery()
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: PublicGallery() = %#v, want %#v", tc.name, got, tc.want)
		}
	}
	src := Entity{ImageUrls: []string{"/a"}}
	src.PublicGallery()[0] = "/mutated"
	if src.ImageUrls[0] != "/a" {
		t.Fatal("PublicGallery must return a copy")
	}
}

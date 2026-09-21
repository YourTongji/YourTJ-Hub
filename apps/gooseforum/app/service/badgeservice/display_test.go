package badgeservice

import (
	"reflect"
	"testing"
)

func TestDisplayBadgeSelection(t *testing.T) {
	owned := []UserBadge{}
	for _, code := range []string{"a", "b", "c", "d", "e", "f"} {
		owned = append(owned, UserBadge{Badge: Badge{Code: code, IsEnabled: true}})
	}
	for _, tc := range []struct {
		preference string
		want       []string
	}{
		{"", []string{"a", "b", "c", "d", "e"}}, {"[]", []string{}},
		{`["c","a"]`, []string{"c", "a"}}, {`["missing","b","b"]`, []string{"b"}},
		{`broken`, []string{}},
	} {
		t.Run(tc.preference, func(t *testing.T) {
			got := []string{}
			for _, badge := range DisplayBadgesFromList(owned, tc.preference) {
				got = append(got, badge.Code)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
	owned[0].IsEnabled = false
	if len(DisplayBadgesFromList(owned, `["a"]`)) != 0 {
		t.Fatal("disabled badge displayed")
	}
	for _, codes := range [][]string{{"a"}, {"b", "b"}, {"unknown"}, {""}, {"a", "b", "c", "d", "e", "f"}} {
		if ValidDisplayBadges(owned, codes) {
			t.Fatalf("accepted %v", codes)
		}
	}
	if !ValidDisplayBadges(owned, []string{}) || !ValidDisplayBadges(owned, []string{"c", "b"}) {
		t.Fatal("valid selection rejected")
	}
}

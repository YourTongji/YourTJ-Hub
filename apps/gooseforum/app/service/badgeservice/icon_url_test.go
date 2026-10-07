package badgeservice

import "testing"

// 系统徽章图形改动后 URL 必须变化，否则 /static 的长公共缓存会让老用户一直看到旧图形。
func TestVersionedBadgeIconURL(t *testing.T) {
	want := "/static/badges/robot.svg?v=" + badgeAssetVersion
	cases := []struct {
		input string
		want  string
	}{
		{input: "/static/badges/robot.svg", want: want},
		// 管理端编辑系统徽章时会把带旧版本号的 URL 回存进覆盖记录
		{input: "/static/badges/robot.svg?v=1", want: want},
		{input: want, want: want},
		// 已有其它查询参数时只替换 v，fragment 保留
		{input: "/static/badges/robot.svg?foo=1", want: "/static/badges/robot.svg?foo=1&v=" + badgeAssetVersion},
		{input: "/static/badges/robot.svg?v=1&foo=1", want: "/static/badges/robot.svg?foo=1&v=" + badgeAssetVersion},
		{input: "/static/badges/robot.svg#detail", want: "/static/badges/robot.svg?v=" + badgeAssetVersion + "#detail"},
		{input: "", want: ""},
		{input: "/file/img/2026/10/custom.png", want: "/file/img/2026/10/custom.png"},
		{input: "https://cdn.example.com/static/badges/a.svg", want: "https://cdn.example.com/static/badges/a.svg"},
	}
	for _, tc := range cases {
		if got := versionedBadgeIconURL(tc.input); got != tc.want {
			t.Errorf("versionedBadgeIconURL(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

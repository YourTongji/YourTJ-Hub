package aiservice

import "testing"

// issue #1284 #35：全局生成配额的默认上限只在 aiservice 定义一次；
// 课程摘要与校历解析都必须经 ResolveGlobalPerMinute 取值，防止两处默认值漂移。
func TestResolveGlobalPerMinute(t *testing.T) {
	cases := []struct {
		name       string
		configured int
		want       int
	}{
		{"unset uses default", 0, defaultGlobalPerMinute},
		{"negative uses default", -3, defaultGlobalPerMinute},
		{"configured value wins", 12, 12},
		{"configured one wins", 1, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveGlobalPerMinute(tc.configured); got != tc.want {
				t.Errorf("ResolveGlobalPerMinute(%d) = %d, want %d", tc.configured, got, tc.want)
			}
		})
	}
}

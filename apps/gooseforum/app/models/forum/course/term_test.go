package course

import "testing"

func TestNormalizeTermLabel(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "标准码原样返回", in: "2025-2026-1", want: "2025-2026-1"},
		{name: "标准码第三学期", in: "2024-2025-3", want: "2024-2025-3"},
		{name: "中文学期名归一", in: "2026-2027学年第1学期", want: "2026-2027-1"},
		{name: "中文数字序数归一", in: "2025-2026 第二学期", want: "2025-2026-2"},
		{name: "中英混合学期名", in: "2025-2026学年第2学期", want: "2025-2026-2"},
		{name: "无法识别保持原值", in: "其他", want: "其他"},
		{name: "带空白 trim", in: "  2026-2027-1  ", want: "2026-2027-1"},
		{name: "空串", in: "", want: ""},
		{name: "纯空白", in: "   ", want: ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := NormalizeTermLabel(c.in); got != c.want {
				t.Errorf("NormalizeTermLabel(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

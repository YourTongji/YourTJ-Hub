package course

import (
	"reflect"
	"testing"
)

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
		{name: "十字开头序数", in: "2024-2025 第十一学期", want: "2024-2025-11"},
		{name: "无法识别保持原值", in: "其他", want: "其他"},
		{name: "短学期标记保持原值", in: "2024-2025学年短学期", want: "2024-2025学年短学期"},
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

func TestIsCanonicalTermCode(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{name: "标准码", in: "2025-2026-1", want: true},
		{name: "两位数序数", in: "2024-2025-11", want: true},
		{name: "中文学期名", in: "2026-2027学年第1学期", want: false},
		{name: "短学期", in: "2024-2025学年短学期", want: false},
		{name: "无序数", in: "2024-2025", want: false},
		{name: "其他", in: "其他", want: false},
		{name: "空串", in: "", want: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsCanonicalTermCode(c.in); got != c.want {
				t.Errorf("IsCanonicalTermCode(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestTermLabelCandidates(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{name: "标准码仅自身", in: "2025-2026-1", want: []string{"2025-2026-1"}},
		{name: "中文学期名追加标准码", in: "2026-2027学年第1学期", want: []string{"2026-2027学年第1学期", "2026-2027-1"}},
		{name: "中文数字序数追加标准码", in: "2025-2026 第二学期", want: []string{"2025-2026 第二学期", "2025-2026-2"}},
		{name: "无法识别仅自身", in: "其他", want: []string{"其他"}},
		{name: "空串 nil", in: "", want: nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := TermLabelCandidates(c.in)
			if c.want == nil {
				if got != nil {
					t.Errorf("TermLabelCandidates(%q) = %v, want nil", c.in, got)
				}
				return
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("TermLabelCandidates(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

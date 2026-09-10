package defaultconfig

import (
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
)

func TestPageConfigDefaultsLoad(t *testing.T) {
	defaults, err := loadPageConfigDefaults()
	if err != nil {
		t.Fatalf("load page config defaults: %v", err)
	}
	if defaults.Site.SiteName != "YourTJHub" {
		t.Fatalf("site name = %q, want YourTJHub", defaults.Site.SiteName)
	}
	if len(defaults.FriendLinks) == 0 {
		t.Fatal("friend links defaults should not be empty")
	}
	if len(defaults.Sponsors.Rules) == 0 {
		t.Fatal("sponsor rules defaults should not be empty")
	}
	if defaults.Posting.UploadControl.MaxAttachmentSizeKb == 0 {
		t.Fatal("posting max attachment size should not be zero")
	}
	if !defaults.Terms.Enabled || defaults.Terms.Content == "" {
		t.Fatal("terms defaults should be enabled with content")
	}
	for _, needle := range []string{"30", "恢复", "治理"} {
		if !strings.Contains(defaults.Terms.Content, needle) {
			t.Fatalf("terms content missing %q: %q", needle, defaults.Terms.Content)
		}
	}
	if !defaults.Privacy.Enabled || defaults.Privacy.Content == "" {
		t.Fatal("privacy defaults should be enabled with content")
	}
	for _, needle := range []string{"30", "恢复", "治理", "6 个月", "网络"} {
		if !strings.Contains(defaults.Privacy.Content, needle) {
			t.Fatalf("privacy content missing %q: %q", needle, defaults.Privacy.Content)
		}
	}
}

func TestPageConfigDefaultGettersReturnCopies(t *testing.T) {
	links := GetDefaultFriendLinksConfig()
	links[0].Links[0].Name = "changed"
	if got := GetDefaultFriendLinksConfig()[0].Links[0].Name; got == "changed" {
		t.Fatal("friend links getter returned shared mutable data")
	}

	site := GetDefaultSiteSettingsConfig()
	site.SiteName = "changed"
	if got := GetDefaultSiteSettingsConfig().SiteName; got == "changed" {
		t.Fatal("site settings getter returned shared mutable data")
	}

	chrome := GetDefaultSiteChromeConfig()
	chrome.FooterInfo.List[0].Name = "changed"
	if got := GetDefaultSiteChromeConfig().FooterInfo.List[0].Name; got == "changed" {
		t.Fatal("site chrome getter returned shared mutable footer data")
	}

	sponsors := GetDefaultSponsorsConfig()
	sponsors.Rules[0].Content = "changed"
	if got := GetDefaultSponsorsConfig().Rules[0].Content; got == "changed" {
		t.Fatal("sponsors getter returned shared mutable rules")
	}
}

func TestNormalizeStoredScheduleSettings(t *testing.T) {
	rows := func(pairs ...[3]string) []pageConfig.ScheduleSectionTime {
		times := make([]pageConfig.ScheduleSectionTime, 0, len(pairs))
		for i, p := range pairs {
			times = append(times, pageConfig.ScheduleSectionTime{Section: i + 1, Start: p[0], End: p[1]})
		}
		return times
	}
	sections := func(times []pageConfig.ScheduleSectionTime) map[int]pageConfig.ScheduleSectionTime {
		out := make(map[int]pageConfig.ScheduleSectionTime, len(times))
		for _, item := range times {
			out[item.Section] = item
		}
		return out
	}

	t.Run("存量旧 12 节默认表（第 9 节 17:10）按旧编号重映射", func(t *testing.T) {
		legacy := pageConfig.ScheduleSettingsConfig{SectionTimes: rows(
			[3]string{"08:00", "08:45"}, [3]string{"08:50", "09:35"}, [3]string{"10:00", "10:45"},
			[3]string{"10:50", "11:35"}, [3]string{"13:30", "14:15"}, [3]string{"14:20", "15:05"},
			[3]string{"15:30", "16:15"}, [3]string{"16:20", "17:05"}, [3]string{"17:10", "17:55"},
			[3]string{"18:30", "19:15"}, [3]string{"19:20", "20:05"}, [3]string{"20:10", "20:55"},
		)}
		got := sections(NormalizeStoredScheduleSettings(legacy).SectionTimes)
		if len(got) != 11 {
			t.Fatalf("normalized rows = %d, want 11", len(got))
		}
		want := map[int][2]string{9: {"18:30", "19:15"}, 10: {"19:20", "20:05"}, 11: {"20:10", "20:55"}}
		for section, times := range want {
			item, ok := got[section]
			if !ok || item.Start != times[0] || item.End != times[1] {
				t.Fatalf("normalized section %d = %#v, want %v", section, item, times)
			}
		}
		for _, item := range got {
			if item.Start == "17:10" {
				t.Fatalf("legacy 17:10 row survived normalization: %#v", got)
			}
		}
		if normalized := NormalizeStoredScheduleSettings(legacy); normalized.Numbering != pageConfig.ScheduleNumberingCurrent {
			t.Fatalf("normalized numbering = %q, want current stamp", normalized.Numbering)
		}
	})

	t.Run("旧 12 节配置第 9 节被自定义（非默认锚点）仍按旧编号重映射", func(t *testing.T) {
		// review P1 路径 (a)：自定义过第 9 节的存量旧配置不能因时间值不匹配默认锚点而漏判。
		legacy := pageConfig.ScheduleSettingsConfig{SectionTimes: rows(
			[3]string{"08:00", "08:45"}, [3]string{"08:50", "09:35"}, [3]string{"10:00", "10:45"},
			[3]string{"10:50", "11:35"}, [3]string{"13:30", "14:15"}, [3]string{"14:20", "15:05"},
			[3]string{"15:30", "16:15"}, [3]string{"16:20", "17:05"}, [3]string{"17:00", "17:45"},
			[3]string{"18:30", "19:15"}, [3]string{"19:20", "20:05"}, [3]string{"20:10", "20:55"},
		)}
		got := sections(NormalizeStoredScheduleSettings(legacy).SectionTimes)
		if item := got[1]; item.Start != "08:00" {
			t.Fatalf("daytime row lost: %#v", item)
		}
		// 新 9/10/11 节 ← 旧 10/11/12 节；自定义的旧第 9 节（17:00）随旧编号体系丢弃。
		want := map[int][2]string{9: {"18:30", "19:15"}, 10: {"19:20", "20:05"}, 11: {"20:10", "20:55"}}
		for section, times := range want {
			item, ok := got[section]
			if !ok || item.Start != times[0] || item.End != times[1] {
				t.Fatalf("normalized section %d = %#v, want %v", section, item, times)
			}
		}
		for _, item := range got {
			if item.Start == "17:00" {
				t.Fatalf("legacy custom section 9 survived normalization: %#v", got)
			}
		}
	})

	t.Run("现行编号盖章的配置完全透传（第 9 节合法设为 17:10 也不重映射）", func(t *testing.T) {
		// review P1 路径 (b)：现行语义下第 9 节允许任意取值，不得基于时间值误判为旧编号。
		stamped := pageConfig.ScheduleSettingsConfig{
			Numbering: pageConfig.ScheduleNumberingCurrent,
			SectionTimes: rows(
				[3]string{"08:00", "08:45"}, [3]string{"08:50", "09:35"}, [3]string{"10:00", "10:45"},
				[3]string{"10:50", "11:35"}, [3]string{"13:30", "14:15"}, [3]string{"14:20", "15:05"},
				[3]string{"15:30", "16:15"}, [3]string{"16:20", "17:05"}, [3]string{"17:10", "17:55"},
				[3]string{"18:30", "19:15"}, [3]string{"19:20", "20:05"},
			),
		}
		gotConfig := NormalizeStoredScheduleSettings(stamped)
		if gotConfig.Numbering != pageConfig.ScheduleNumberingCurrent {
			t.Fatalf("numbering = %q, want current", gotConfig.Numbering)
		}
		got := sections(gotConfig.SectionTimes)
		if item := got[9]; item.Section != 9 || item.Start != "17:10" || item.End != "17:55" {
			t.Fatalf("current-numbered section 9 was remapped: %#v", item)
		}
		if len(gotConfig.SectionTimes) != 11 {
			t.Fatalf("rows = %d, want 11", len(gotConfig.SectionTimes))
		}
	})

	t.Run("现行编号盖章的 12 行配置原样透传", func(t *testing.T) {
		stamped := pageConfig.ScheduleSettingsConfig{
			Numbering: pageConfig.ScheduleNumberingCurrent,
			SectionTimes: rows(
				[3]string{"08:00", "08:45"}, [3]string{"08:50", "09:35"}, [3]string{"10:00", "10:45"},
				[3]string{"10:50", "11:35"}, [3]string{"13:30", "14:15"}, [3]string{"14:20", "15:05"},
				[3]string{"15:30", "16:15"}, [3]string{"16:20", "17:05"}, [3]string{"18:30", "19:15"},
				[3]string{"19:20", "20:05"}, [3]string{"20:10", "20:55"}, [3]string{"20:10", "20:55"},
			),
		}
		got := NormalizeStoredScheduleSettings(stamped)
		if len(got.SectionTimes) != 12 || got.Numbering != pageConfig.ScheduleNumberingCurrent {
			t.Fatalf("versioned config must pass through verbatim: numbering=%q rows=%d", got.Numbering, len(got.SectionTimes))
		}
	})

	t.Run("空配置原样返回", func(t *testing.T) {
		empty := pageConfig.ScheduleSettingsConfig{}
		if got := NormalizeStoredScheduleSettings(empty); len(got.SectionTimes) != 0 {
			t.Fatalf("empty config changed: %#v", got)
		}
	})
}

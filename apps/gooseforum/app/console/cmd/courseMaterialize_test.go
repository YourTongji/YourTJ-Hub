package cmd

import (
	"strings"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
)

// seedMaterializeCalendars 建 pk_calendar 测试学期：121 标准码 i18n、122 中文学期名 i18n。
func seedMaterializeCalendars(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&pk.CalendarEntity{}); err != nil {
		t.Fatalf("migrate pk_calendar: %v", err)
	}
	conn.Unscoped().Where("1 = 1").Delete(&pk.CalendarEntity{})
	if err := conn.Create(&pk.CalendarEntity{CalendarId: 121, CalendarIdI18n: "2025-2026-1"}).Error; err != nil {
		t.Fatalf("seed calendar 121: %v", err)
	}
	if err := conn.Create(&pk.CalendarEntity{CalendarId: 122, CalendarIdI18n: "2026-2027学年第1学期"}).Error; err != nil {
		t.Fatalf("seed calendar 122: %v", err)
	}
	t.Cleanup(func() { conn.Unscoped().Where("1 = 1").Delete(&pk.CalendarEntity{}) })
}

// TestResolveMaterializeCalendarsDedupAndValidate 回归 review：
//   - 重复参数去重（122 122 → [122]，报告计数不虚高）；
//   - 学期参数支持标准码反查中文学期名（"2026-2027-1" → 122，review P2）；
//   - 未同步/错拼的学期 ID 显式报错，不得空成功（review P2）。
func TestResolveMaterializeCalendarsDedupAndValidate(t *testing.T) {
	seedMaterializeCalendars(t)

	deduped, err := resolveMaterializeCalendars([]string{"122", "122"})
	if err != nil {
		t.Fatalf("resolve dedup: %v", err)
	}
	if len(deduped) != 1 || deduped[0] != 122 {
		t.Fatalf("resolve([122 122]) = %v, want [122]（重复参数去重）", deduped)
	}

	byStandardCode, err := resolveMaterializeCalendars([]string{"2026-2027-1"})
	if err != nil {
		t.Fatalf("resolve standard code against chinese label: %v", err)
	}
	if len(byStandardCode) != 1 || byStandardCode[0] != 122 {
		t.Fatalf("resolve(2026-2027-1) = %v, want [122]（归一化反查中文学期名）", byStandardCode)
	}

	byChineseLabel, err := resolveMaterializeCalendars([]string{"2026-2027学年第1学期"})
	if err != nil {
		t.Fatalf("resolve chinese label: %v", err)
	}
	if len(byChineseLabel) != 1 || byChineseLabel[0] != 122 {
		t.Fatalf("resolve(chinese label) = %v, want [122]", byChineseLabel)
	}

	if _, err := resolveMaterializeCalendars([]string{"999"}); err == nil {
		t.Fatal("expected error for unsynced calendar 999（只物化已同步学期）")
	} else if !strings.Contains(err.Error(), "未同步") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := resolveMaterializeCalendars([]string{"2026-2027-2"}); err == nil {
		t.Fatal("expected error for unmatched term")
	}
}

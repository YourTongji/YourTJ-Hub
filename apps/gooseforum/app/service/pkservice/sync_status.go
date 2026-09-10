package pkservice

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
)

// ResolveSyncTerm 解析管理端提交的学期参数为数字 calendarId。
// term 接受两种形式（与 CLI course-pk-sync 对齐）：
//   - 一系统数字 calendarId（如 121）
//   - 学期名（如 2025-2026-1），经 pk_calendar.calendar_id_i18n 反查
//
// 返回 (calendarId, 归一化后的 term)。首次同步时 pk_calendar 尚无记录，学期名无法反查，
// 需直接传数字 calendarId。
func ResolveSyncTerm(term string) (uint64, string, error) {
	return ResolveSyncTermForAudience(pk.AudienceUndergraduate, term)
}

func ResolveSyncTermForAudience(audience pk.Audience, term string) (uint64, string, error) {
	t := strings.TrimSpace(term)
	if t == "" {
		return 0, "", errors.New("缺少学期参数：请输入一系统数字 calendarId（如 121）或学期名（如 2025-2026-1）")
	}
	if id, err := strconv.ParseUint(t, 10, 64); err == nil && pk.ValidExternalID(id) {
		return id, t, nil
	}
	if id, ok := pk.GetCalendarIdByAudienceI18n(audience, t); ok {
		return id, t, nil
	}
	calendars, err := pk.ListAllCalendarsForAudience(audience)
	if err != nil {
		return 0, "", err
	}
	for _, calendar := range calendars {
		if course.NormalizeTermLabel(calendar.CalendarIdI18n) == course.NormalizeTermLabel(t) {
			return pk.ExternalID(audience, calendar.CalendarId), t, nil
		}
	}
	return 0, "", fmt.Errorf("无法解析学期 %q：请输入一系统数字 calendarId（如 121），或先以该学期名同步一次使其进入 pk_calendar", t)
}

// SyncStatusItem 单个学期的同步状态（管理端排课数据同步入口展示用）。
type SyncStatusItem struct {
	Audience          string     `json:"audience"`
	CalendarId        uint64     `json:"calendarId"`
	CalendarName      string     `json:"calendarName"`
	Status            string     `json:"status"`
	RowsWritten       int        `json:"rowsWritten"`
	TotalPages        int        `json:"totalPages"`
	LastCommittedPage int        `json:"lastCommittedPage"`
	ErrorMsg          string     `json:"errorMsg"`
	StartedAt         *time.Time `json:"startedAt"`
	FinishedAt        *time.Time `json:"finishedAt"`
}

// SyncStatusOverview 汇总各学期同步状态：以 pk_calendar 现有学期为骨架，对每个学期附加
// 最近一次 fetchlog 状态；另补充「有 fetchlog 但 calendar 尚未写入」的学期（如首次同步失败），
// 便于管理端看到失败的同步尝试及原因。按 calendarId 倒序（最近学期在前）。
func SyncStatusOverview() ([]SyncStatusItem, error) {
	calendars, err := pk.ListAllCalendarsAllAudiences(200)
	if err != nil {
		return nil, err
	}
	var logs []pk.FetchLogEntity
	latestLogIDs := db.Connect().Model(&pk.FetchLogEntity{}).
		Select("MAX(id)").Group("audience, calendar_id")
	if err := db.Connect().Model(&pk.FetchLogEntity{}).
		Where("id IN (?)", latestLogIDs).Order("id DESC").Find(&logs).Error; err != nil {
		return nil, err
	}
	// 子查询在数据库端每个 calendar 只保留最新一条，避免管理端轮询搬运完整历史。
	type calendarKey struct {
		audience   string
		calendarId uint64
	}
	latestByCalendar := make(map[calendarKey]pk.FetchLogEntity, len(logs))
	for _, log := range logs {
		key := calendarKey{audience: string(pk.DefaultAudience(log.Audience)), calendarId: log.CalendarId}
		if _, ok := latestByCalendar[key]; !ok {
			latestByCalendar[key] = log
		}
	}

	seen := make(map[calendarKey]bool, len(calendars))
	items := make([]SyncStatusItem, 0, len(calendars)+len(latestByCalendar))
	for _, c := range calendars {
		audience := pk.DefaultAudience(c.Audience)
		key := calendarKey{audience: string(audience), calendarId: c.CalendarId}
		seen[key] = true
		item := SyncStatusItem{Audience: string(audience), CalendarId: pk.ExternalID(audience, c.CalendarId), CalendarName: c.CalendarIdI18n}
		if log, ok := latestByCalendar[key]; ok {
			attachFetchLog(&item, log)
		}
		items = append(items, item)
	}
	for key, log := range latestByCalendar {
		if seen[key] {
			continue
		}
		audience := pk.DefaultAudience(key.audience)
		item := SyncStatusItem{Audience: string(audience), CalendarId: pk.ExternalID(audience, key.calendarId)}
		attachFetchLog(&item, log)
		items = append(items, item)
	}
	slices.SortFunc(items, func(a, b SyncStatusItem) int {
		if result := cmp.Compare(b.CalendarId, a.CalendarId); result != 0 {
			return result
		}
		return cmp.Compare(a.Audience, b.Audience)
	})
	return items, nil
}

// attachFetchLog 将最近一次同步日志的状态字段填入状态项。
// 超过断点续跑时间窗的 running 日志视为中断（进程崩溃/goroutine 被杀），
// 标记为 failed 并附加说明，避免管理端状态列表永远显示「同步中」。
func attachFetchLog(item *SyncStatusItem, log pk.FetchLogEntity) {
	item.Status = log.Status
	item.RowsWritten = log.RowsWritten
	item.TotalPages = log.TotalPages
	item.LastCommittedPage = log.LastCommittedPage
	item.ErrorMsg = log.ErrorMsg
	item.StartedAt = log.StartedAt
	item.FinishedAt = log.FinishedAt
	if staleRunning(log) {
		item.Status = pk.FetchStatusFailed
		if item.ErrorMsg == "" {
			item.ErrorMsg = "上次同步中断（running 超过 1 小时，视为失败）"
		}
	}
}

// staleRunning 判断 running 日志是否已超过 fetchLogStaleWindow（1 小时）断点续跑窗口。
func staleRunning(log pk.FetchLogEntity) bool {
	return log.Status == pk.FetchStatusRunning &&
		log.StartedAt != nil &&
		time.Since(*log.StartedAt) > fetchLogStaleWindow
}

package pkservice

import (
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
)

// ErrInvalidParams 请求参数缺失或非法（controller 映射为 HTTP 400 + code≠0）。
var ErrInvalidParams = errors.New("invalid pk request parameters")

// PK_AUX_SCHEMA_VERSION teacher_timeslots 辅助表构建版本。
// 解析/落库逻辑变更时递增该值，触发存量表重建（上游同款语义）。
const PK_AUX_SCHEMA_VERSION = "20260812-hub-pk-timeslots-v1"

// MAX_SQL_VARS SQLite/MySQL 单查询变量上限；IN 条件按此分块（跨方言安全）。
const MAX_SQL_VARS = 80

// OPTIONAL_LABEL_NAMES 通识/选修课程标签：courses-by-time 查询只返回这些性质
// 的课程（上游 `findCourseByTime` 的 OPTIONAL_LABEL_NAMES 过滤语义）。
var OPTIONAL_LABEL_NAMES = []string{
	"通识选修课",
	"人文经典与审美素养",
	"工程能力与创新思维",
	"社会发展与国际视野",
	"科学探索与生命关怀",
}

// CROSS_DISCIPLINE_LABEL_NAMES 判定"跨学科选修"的课程标签（courses-by-nature）。
var CROSS_DISCIPLINE_LABEL_NAMES = []string{
	"个性化课程",
	"个性课程",
	"任选课程",
	"专业选修课",
	"专业课选修",
	"专业特色模块",
	"领域基础课",
}

// normalizeText 去除首尾空白。
func normalizeText(value string) string {
	return strings.TrimSpace(value)
}

// uniqueText 保序去重非空文本。
func uniqueText(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		v := normalizeText(value)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func isCrossDisciplineLabel(labelName string) bool {
	for _, name := range CROSS_DISCIPLINE_LABEL_NAMES {
		if normalizeText(labelName) == name {
			return true
		}
	}
	return false
}

// ---- teacher_timeslots 懒构建状态（对齐上游 pkAuxInitPromise / ready 缓存） ----

var (
	pkAuxStateMu      sync.Mutex
	pkAuxBuildRunning bool
	pkAuxReadyValue   bool
	pkAuxReadyExpires time.Time // isPkAuxiliaryReady 的就绪状态缓存（正/负均缓存 30s）。
	pkAuxRetryAfter   time.Time // 构建失败后的指数退避截止；此时间前 TriggerPkAuxiliaryBuild 不再启动重建。
	pkAuxFailCount    int
	pkAuxBuildWG      sync.WaitGroup
)

// ResetPkAuxiliaryStateForTest 重置辅助表就绪缓存与构建状态（契约/单元测试隔离用）。
func ResetPkAuxiliaryStateForTest() {
	pkAuxStateMu.Lock()
	defer pkAuxStateMu.Unlock()
	pkAuxBuildRunning = false
	pkAuxReadyValue = false
	pkAuxReadyExpires = time.Time{}
	pkAuxRetryAfter = time.Time{}
	pkAuxFailCount = 0
}

// WaitPkAuxiliaryBuildForTest 等待后台 timeslots 构建完成（测试在清理前调用，避免写回竞态）。
func WaitPkAuxiliaryBuildForTest() {
	pkAuxBuildWG.Wait()
}

// isPkAuxiliaryReady 判断 teacher_timeslots 辅助表是否就绪（30s 缓存）。
func isPkAuxiliaryReady() bool {
	pkAuxStateMu.Lock()
	defer pkAuxStateMu.Unlock()
	now := time.Now()
	if !pkAuxReadyExpires.IsZero() && pkAuxReadyExpires.After(now) {
		return pkAuxReadyValue
	}
	ready := pkAuxSchemaVersionMatches()
	pkAuxReadyValue = ready
	pkAuxReadyExpires = now.Add(30 * time.Second)
	return ready
}

// pkAuxSchemaVersionMatches 检查 pk_setting 中记录的辅助表版本。
func pkAuxSchemaVersionMatches() bool {
	entity, err := pk.GetSetting(pkSettingAuxSchemaVersion)
	if err != nil {
		return false
	}
	return entity.Value == PK_AUX_SCHEMA_VERSION
}

const pkSettingAuxSchemaVersion = "pk_aux_schema_version"

// pkAuxBackoffSeconds 构建失败后的退避秒数（指数增长，上限 5 分钟），
// 避免持续失败时每 10s 全量清表+重建空转，把应用长时间拖在降级 LIKE 路径。
func pkAuxBackoffSeconds(failCount int) time.Duration {
	if failCount <= 0 {
		return 10 * time.Second
	}
	if failCount > 4 {
		return 5 * time.Minute
	}
	return time.Duration(10<<uint(failCount)) * time.Second // 10,20,40,80
}

// TriggerPkAuxiliaryBuild 触发 teacher_timeslots 后台重建（非阻塞）。
// 已在构建或已就绪时不重复启动；构建失败按指数退避重置 ready 缓存，
// 且退避窗口内（pkAuxRetryAfter 之前）不再次触发，避免持续失败时
// 每 10s 全量清表+重建空转（见 pkAuxBackoffSeconds）。
func TriggerPkAuxiliaryBuild() {
	pkAuxStateMu.Lock()
	if pkAuxBuildRunning || pkAuxSchemaVersionMatches() {
		pkAuxStateMu.Unlock()
		return
	}
	// 失败退避窗口内不重新触发（锁内判定，避免与失败写回竞态）。
	if time.Now().Before(pkAuxRetryAfter) {
		pkAuxStateMu.Unlock()
		return
	}
	pkAuxBuildRunning = true
	pkAuxBuildWG.Add(1)
	pkAuxStateMu.Unlock()

	go func() {
		defer pkAuxBuildWG.Done()
		defer func() {
			pkAuxStateMu.Lock()
			pkAuxBuildRunning = false
			pkAuxStateMu.Unlock()
		}()
		if err := rebuildTeacherTimeslots(); err != nil {
			slog.Error("pk auxiliary timeslot build failed", "error", err)
			pkAuxStateMu.Lock()
			pkAuxFailCount++
			backoff := pkAuxBackoffSeconds(pkAuxFailCount)
			pkAuxReadyValue = false
			pkAuxReadyExpires = time.Now().Add(backoff)
			pkAuxRetryAfter = pkAuxReadyExpires
			pkAuxStateMu.Unlock()
			return
		}
		pkAuxStateMu.Lock()
		pkAuxFailCount = 0
		pkAuxReadyValue = true
		pkAuxReadyExpires = time.Now().Add(30 * time.Second)
		pkAuxRetryAfter = time.Time{}
		pkAuxStateMu.Unlock()
	}()
}

// rebuildTeacherTimeslots 全量重建 teacher_timeslots（先在内存解析生成 pending，
// 解析/读取成功后再清空旧表 → 批量 upsert → 写版本，避免中途失败留下空表）。
func rebuildTeacherTimeslots() error {
	rows, err := pk.ListTeacherArrangeRows()
	if err != nil {
		return err
	}
	pending := map[string]pk.TeacherTimeslotEntity{}
	for _, row := range rows {
		if row.TeachingClassId == 0 || row.CalendarId <= 0 {
			continue
		}
		teacherCode := normalizeText(row.TeacherCode)
		teacherName := normalizeText(row.TeacherName)
		for _, line := range splitEndline(row.ArrangeInfoText) {
			info := arrangementTextToObj(line)
			if info.OccupyDay == nil || *info.OccupyDay <= 0 || len(info.OccupyTime) == 0 {
				continue
			}
			for _, section := range info.OccupyTime {
				if section <= 0 {
					continue
				}
				entity := pk.TeacherTimeslotEntity{
					CalendarId:      row.CalendarId,
					TeachingClassId: row.TeachingClassId,
					OccupyDay:       *info.OccupyDay,
					OccupySection:   section,
					TeacherCode:     teacherCode,
					TeacherName:     teacherName,
				}
				pending[timeslotKey(entity)] = entity
			}
		}
	}
	if err := pk.ClearTeacherTimeslots(); err != nil {
		return err
	}
	if err := pk.UpsertTeacherTimeslots(collectTimeslots(pending)); err != nil {
		return err
	}
	return pk.SetSetting(pkSettingAuxSchemaVersion, PK_AUX_SCHEMA_VERSION)
}

func timeslotKey(e pk.TeacherTimeslotEntity) string {
	return strings.Join([]string{
		strconv.Itoa(e.CalendarId),
		strconv.FormatUint(e.TeachingClassId, 10),
		strconv.Itoa(e.OccupyDay),
		strconv.Itoa(e.OccupySection),
		e.TeacherCode,
		e.TeacherName,
	}, "|")
}

func collectTimeslots(m map[string]pk.TeacherTimeslotEntity) []pk.TeacherTimeslotEntity {
	out := make([]pk.TeacherTimeslotEntity, 0, len(m))
	for _, e := range m {
		out = append(out, e)
	}
	return out
}

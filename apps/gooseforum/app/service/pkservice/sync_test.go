package pkservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"gorm.io/gorm"
)

// migratePkTables 迁移并清空 PK 域表（测试用内存 sqlite）。
func migratePkTables(t *testing.T) {
	t.Helper()
	models := []any{
		&pk.CalendarEntity{},
		&pk.LanguageEntity{},
		&pk.CourseNatureEntity{},
		&pk.CourseNatureByCalendarEntity{},
		&pk.AssessmentEntity{},
		&pk.CampusEntity{},
		&pk.FacultyEntity{},
		&pk.MajorEntity{},
		&pk.MajorCourseEntity{},
		&pk.CourseDetailEntity{},
		&pk.TeacherEntity{},
		&pk.TeacherTimeslotEntity{},
		&pk.FetchLogEntity{},
	}
	conn := db.Connect()
	if err := conn.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate pk tables: %v", err)
	}
	for _, m := range models {
		if err := conn.Unscoped().Where("1 = 1").Delete(m).Error; err != nil {
			t.Fatalf("clean pk table: %v", err)
		}
	}
}

// fakeOneSystem 模拟一系统 manualArrange 分页服务，可注入按页失败、全局 401 与 HTTP 200 业务失败。
type fakeOneSystem struct {
	mu           sync.Mutex
	courses      []CourseRaw
	failPages    map[int]int // pageNum -> HTTP status
	cookieOK     bool
	businessCode int   // 非 0 时返回 HTTP 200 + code!=0（模拟一系统业务/鉴权失败信封）
	calls        []int // 按顺序记录请求的 pageNum
}

func (f *fakeOneSystem) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			PageNum_  int `json:"pageNum_"`
			Condition struct {
				Calendar int `json:"calendar"`
			} `json:"condition"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)

		f.mu.Lock()
		f.calls = append(f.calls, req.PageNum_)
		status := f.failPages[req.PageNum_]
		cookieOK := f.cookieOK
		businessCode := f.businessCode
		f.mu.Unlock()

		if !cookieOK {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message":"未登录或会话失效"}`))
			return
		}
		if businessCode != 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(fmt.Sprintf(`{"code":%d,"msg":"未登录或会话失效","data":null}`, businessCode)))
			return
		}
		if status != 0 {
			w.WriteHeader(status)
			_, _ = w.Write([]byte("injected failure"))
			return
		}
		start := (req.PageNum_ - 1) * onesystemPageSize
		end := start + onesystemPageSize
		if end > len(f.courses) {
			end = len(f.courses)
		}
		if start > len(f.courses) {
			start = len(f.courses)
		}
		// 真实一系统 teachingClassId 全局唯一；按学期偏移 id，避免跨学期测试数据互相覆盖主键。
		offset := uint64(req.Condition.Calendar) * 100000
		out := make([]CourseRaw, 0, end-start)
		for _, c := range f.courses[start:end] {
			copyC := c
			if copyC.Id != nil {
				id := *copyC.Id + offset
				copyC.Id = &id
			}
			for j := range copyC.TeacherList {
				if copyC.TeacherList[j].Id != nil {
					id := *copyC.TeacherList[j].Id + offset
					copyC.TeacherList[j].Id = &id
				}
			}
			out = append(out, copyC)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(pageResponse(len(f.courses), out)))
	})
}

func (f *fakeOneSystem) pageCallsSince(offset int) []int {
	f.mu.Lock()
	defer f.mu.Unlock()
	if offset >= len(f.calls) {
		return nil
	}
	out := make([]int, len(f.calls)-offset)
	copy(out, f.calls[offset:])
	return out
}

// genCourses 生成 n 个教学班（每行含教师与排课文本，便于 timeslots 重建断言）。
func genCourses(n int) []CourseRaw {
	out := make([]CourseRaw, 0, n)
	for i := 1; i <= n; i++ {
		uid := uint64(i)
		credit := 5.0
		code := fmt.Sprintf("%06d", 100000+i)
		courseCode := code[:4]
		teacherID := uint64(i)
		c := CourseRaw{
			Id:             &uid,
			Code:           code,
			Name:           "课程" + code,
			CourseCode:     courseCode,
			CourseName:     "课程" + code,
			Credits:        &credit,
			CalendarIdI18n: "2025-2026-1",
			TeacherList: []TeacherRaw{{
				Id:          &teacherID,
				TeacherCode: fmt.Sprintf("T%04d", i),
				TeacherName: "教师" + fmt.Sprintf("%02d", i%50),
			}},
			ArrangeInfo: "教师" + fmt.Sprintf("%02d", i%50) + fmt.Sprintf("(T%04d)", i) + " 星期一1-2节[1-16周] 四平路校区\n",
		}
		out = append(out, c)
	}
	return out
}

// countCourseDetails 统计某学期教学班数。
func countCourseDetails(t *testing.T, calendarId uint64) int64 {
	t.Helper()
	var n int64
	if err := db.Connect().Model(&pk.CourseDetailEntity{}).Where("calendar_id = ?", calendarId).Count(&n).Error; err != nil {
		t.Fatalf("count course details: %v", err)
	}
	return n
}

func newSyncClient(t *testing.T, fake *fakeOneSystem) *onesystemClient {
	t.Helper()
	srv := httptest.NewServer(fake.handler())
	t.Cleanup(srv.Close)
	c := newOnesystemClient()
	c.baseURL = srv.URL
	c.maxAttempts = 3
	c.sleep = func(time.Duration) {}
	c.backoff = func(int) time.Duration { return 0 }
	return c
}

func TestSyncAC1IdempotentRerun(t *testing.T) {
	migratePkTables(t)
	fake := &fakeOneSystem{courses: genCourses(1200), cookieOK: true}
	client := newSyncClient(t, fake)

	const calendarID = uint64(121)

	report, err := syncWith(context.Background(), client, "valid-cookie", calendarID, 1, false)
	if err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if report.TeachingClassInserted != 1200 {
		t.Errorf("first inserted = %d, want 1200", report.TeachingClassInserted)
	}
	if report.TimeslotsRebuilt == 0 {
		t.Error("expected timeslots rebuilt > 0")
	}
	if got := countCourseDetails(t, calendarID); got != 1200 {
		t.Fatalf("course details after first sync = %d, want 1200", got)
	}
	firstLog, _ := pk.LatestFetchLogByCalendar(calendarID)
	if firstLog.Status != pk.FetchStatusCompleted {
		t.Errorf("first fetchlog status = %q, want completed", firstLog.Status)
	}

	// 幂等重跑：清空后全量重写，数量不变、无重复。
	report2, err := syncWith(context.Background(), client, "valid-cookie", calendarID, 1, false)
	if err != nil {
		t.Fatalf("second sync: %v", err)
	}
	if report2.TeachingClassInserted != 1200 {
		t.Errorf("second inserted = %d, want 1200", report2.TeachingClassInserted)
	}
	if got := countCourseDetails(t, calendarID); got != 1200 {
		t.Errorf("course details after second sync = %d, want 1200 (no duplication)", got)
	}
	// 两次完整运行产生两条 fetchlog。
	var logCount int64
	if err := db.Connect().Model(&pk.FetchLogEntity{}).Where("calendar_id = ?", calendarID).Count(&logCount).Error; err != nil {
		t.Fatalf("count fetchlogs: %v", err)
	}
	if logCount != 2 {
		t.Errorf("fetchlog count = %d, want 2", logCount)
	}
}

func TestSyncAC2CookieFailureKeepsCommittedBatches(t *testing.T) {
	migratePkTables(t)
	fake := &fakeOneSystem{courses: genCourses(1200), cookieOK: true, failPages: map[int]int{4: http.StatusUnauthorized}}
	client := newSyncClient(t, fake)

	const calendarID = uint64(121)
	_, err := syncWith(context.Background(), client, "expired-cookie", calendarID, 1, false)
	if err == nil {
		t.Fatal("expected error on cookie failure")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error should mention HTTP 401, got: %v", err)
	}
	// 前 3 页（600 行）已提交，不因失败回滚。
	if got := countCourseDetails(t, calendarID); got != 600 {
		t.Errorf("committed rows after failure = %d, want 600", got)
	}
	log, _ := pk.LatestFetchLogByCalendar(calendarID)
	if log.Status != pk.FetchStatusFailed {
		t.Errorf("fetchlog status = %q, want failed", log.Status)
	}
	if log.LastCommittedPage != 3 {
		t.Errorf("lastCommittedPage = %d, want 3", log.LastCommittedPage)
	}
}

func TestSyncAC3ResumeFromFailedBatch(t *testing.T) {
	migratePkTables(t)
	fake := &fakeOneSystem{courses: genCourses(1200), cookieOK: true, failPages: map[int]int{4: http.StatusUnauthorized}}
	client := newSyncClient(t, fake)

	const calendarID = uint64(121)
	if _, err := syncWith(context.Background(), client, "cookie", calendarID, 1, false); err == nil {
		t.Fatal("expected first run to fail")
	}
	callsBefore := len(fake.calls)

	// 恢复服务后重跑：应从第 4 页续跑，不回滚、不重抓已提交页。
	fake.mu.Lock()
	delete(fake.failPages, 4)
	fake.mu.Unlock()

	report, err := syncWith(context.Background(), client, "cookie", calendarID, 1, false)
	if err != nil {
		t.Fatalf("resume sync: %v", err)
	}
	if got := countCourseDetails(t, calendarID); got != 1200 {
		t.Errorf("final course details = %d, want 1200", got)
	}
	if report.ResumedFromPage != 4 {
		t.Errorf("resumedFromPage = %d, want 4", report.ResumedFromPage)
	}
	// 重跑首次请求的页是 4（未重抓 1-3）。
	resumeCalls := fake.pageCallsSince(callsBefore)
	if len(resumeCalls) == 0 || resumeCalls[0] != 4 {
		t.Errorf("resume should start fetching from page 4, got calls %v", resumeCalls)
	}
	// 同一 fetchlog 被续用并标记 completed。
	log, _ := pk.LatestFetchLogByCalendar(calendarID)
	if log.Status != pk.FetchStatusCompleted {
		t.Errorf("resumed fetchlog status = %q, want completed", log.Status)
	}
	if log.LastCommittedPage != 6 {
		t.Errorf("lastCommittedPage = %d, want 6", log.LastCommittedPage)
	}
	if log.RowsWritten != 1200 {
		t.Errorf("rowsWritten = %d, want 1200", log.RowsWritten)
	}
}

// TestSyncExpiredCookiePreservesPriorData 回归（review HIGH）：fresh 同步不得在验证 cookie 前
// 删除存量数据。已完成的学期用无效 cookie 重跑时，存量教学班/时间片必须保留，fetchlog 记 failed。
func TestSyncExpiredCookiePreservesPriorData(t *testing.T) {
	migratePkTables(t)
	fake := &fakeOneSystem{courses: genCourses(1200), cookieOK: true}
	client := newSyncClient(t, fake)

	const calendarID = uint64(121)
	if _, err := syncWith(context.Background(), client, "valid-cookie", calendarID, 1, false); err != nil {
		t.Fatalf("prior successful sync: %v", err)
	}
	if got := countCourseDetails(t, calendarID); got != 1200 {
		t.Fatalf("prior course details = %d, want 1200", got)
	}

	// 用无效 cookie 重跑：必须失败，且不得摧毁存量数据。
	fake.mu.Lock()
	fake.cookieOK = false
	fake.mu.Unlock()
	if _, err := syncWith(context.Background(), client, "expired-cookie", calendarID, 1, false); err == nil {
		t.Fatal("expected expired-cookie re-run to fail")
	}
	if got := countCourseDetails(t, calendarID); got != 1200 {
		t.Errorf("course details after failed re-run = %d, want 1200 (prior data must be preserved)", got)
	}
	var timeslots int64
	if err := db.Connect().Model(&pk.TeacherTimeslotEntity{}).Where("calendar_id = ?", calendarID).Count(&timeslots).Error; err != nil {
		t.Fatalf("count timeslots: %v", err)
	}
	if timeslots == 0 {
		t.Error("teacher_timeslots wiped by failed re-run; prior data must be preserved")
	}
	log, _ := pk.LatestFetchLogByCalendar(calendarID)
	if log.Status != pk.FetchStatusFailed {
		t.Errorf("fetchlog status = %q, want failed", log.Status)
	}
}

// TestSyncBusinessCodeFailurePreservesPriorData 回归（review HIGH）：一系统 HTTP 200 + code!=0
// 的鉴权失败同样不得当成功页，不得在验证前删除存量数据。
func TestSyncBusinessCodeFailurePreservesPriorData(t *testing.T) {
	migratePkTables(t)
	fake := &fakeOneSystem{courses: genCourses(1200), cookieOK: true}
	client := newSyncClient(t, fake)

	const calendarID = uint64(121)
	if _, err := syncWith(context.Background(), client, "valid-cookie", calendarID, 1, false); err != nil {
		t.Fatalf("prior successful sync: %v", err)
	}
	if got := countCourseDetails(t, calendarID); got != 1200 {
		t.Fatalf("prior course details = %d, want 1200", got)
	}

	// 一系统返回 HTTP 200 + code=1 的业务失败：必须失败，且不得摧毁存量数据。
	fake.mu.Lock()
	fake.businessCode = 1
	fake.mu.Unlock()
	_, bizErr := syncWith(context.Background(), client, "expired-cookie", calendarID, 1, false)
	if bizErr == nil {
		t.Fatal("expected business-code re-run to fail")
	}
	if !strings.Contains(bizErr.Error(), "code=1") {
		t.Errorf("error should mention code=1, got: %v", bizErr)
	}
	if got := countCourseDetails(t, calendarID); got != 1200 {
		t.Errorf("course details after failed re-run = %d, want 1200 (prior data must be preserved)", got)
	}
	log, _ := pk.LatestFetchLogByCalendar(calendarID)
	if log.Status != pk.FetchStatusFailed {
		t.Errorf("fetchlog status = %q, want failed", log.Status)
	}
}

func TestSyncMissingCookie(t *testing.T) {
	migratePkTables(t)
	fake := &fakeOneSystem{courses: genCourses(10), cookieOK: true}
	client := newSyncClient(t, fake)
	_, err := syncWith(context.Background(), client, "   ", 121, 1, false)
	if err == nil || !strings.Contains(err.Error(), "ONESYSTEM_COOKIE") {
		t.Fatalf("expected missing-cookie error, got %v", err)
	}
}

func TestSyncDepthSyncsConsecutiveCalendars(t *testing.T) {
	migratePkTables(t)
	fake := &fakeOneSystem{courses: genCourses(5), cookieOK: true}
	client := newSyncClient(t, fake)

	report, err := syncWith(context.Background(), client, "cookie", 125, 3, false)
	if err != nil {
		t.Fatalf("depth sync: %v", err)
	}
	if len(report.CalendarIDs) != 3 {
		t.Errorf("calendars = %v, want 3 entries", report.CalendarIDs)
	}
	want := []uint64{123, 124, 125}
	for i, id := range want {
		if report.CalendarIDs[i] != id {
			t.Errorf("calendar[%d] = %d, want %d", i, report.CalendarIDs[i], id)
		}
	}
	for _, id := range want {
		if got := countCourseDetails(t, id); got != 5 {
			t.Errorf("calendar %d course details = %d, want 5", id, got)
		}
	}
}

func TestFetchLogRunningUniqueConstraint(t *testing.T) {
	// 并发防护（review HIGH）：同一学期只允许一条 running 日志。实现是可空 running_key
	// （running 时=calendar_id，其余=NULL）+ 普通 UNIQUE 索引：同一学期第二条 running 冲突失败；
	// completed/failed（NULL）行之间永不冲突。跨方言，不依赖 PG 专属 partial unique index。
	migratePkTables(t)
	const calendarID = uint64(121)

	first, err := pk.CreateFetchLog(calendarID)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	if first.RunningKey == nil || *first.RunningKey != calendarID {
		t.Errorf("fresh running running_key = %v, want %d", first.RunningKey, calendarID)
	}
	// 不同学期可各自拥有一条 running（唯一索引按 running_key 值区分，而非全局唯一）。
	if _, err := pk.CreateFetchLog(122); err != nil {
		t.Fatalf("different calendar running should coexist: %v", err)
	}
	// 同一学期第二条 running 冲突失败（两行 running_key 同为 calendarID）。
	if _, err := pk.CreateFetchLog(calendarID); !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("second running create error = %v, want gorm.ErrDuplicatedKey", err)
	}

	// 标记 completed 后 running_key 置 NULL，唯一约束释放，可再建新 running（幂等重跑路径）。
	var log pk.FetchLogEntity
	if err := db.Connect().Where("calendar_id = ?", calendarID).First(&log).Error; err != nil {
		t.Fatalf("load log: %v", err)
	}
	log.Status = pk.FetchStatusCompleted
	if err := pk.SaveFetchLog(&log); err != nil {
		t.Fatalf("mark completed: %v", err)
	}
	if log.RunningKey != nil {
		t.Errorf("completed running_key = %v, want nil", log.RunningKey)
	}
	if _, err := pk.CreateFetchLog(calendarID); err != nil {
		t.Fatalf("create after completed should succeed: %v", err)
	}
}

func TestClaimFetchLogCAS(t *testing.T) {
	// 续跑原子认领（review HIGH 租约 + P1）：ClaimFetchLog 用 lease_version 精确 CAS，
	// 防止两进程同时读到同一条 stale-running/failed 日志并都续跑造成 double-delete。
	// 不用 started_at 时间戳做 token——方言时间精度不一致，两次写入可能取同一值，旧实现第二次 claim 仍成功。
	migratePkTables(t)

	running, err := pk.CreateFetchLog(121)
	if err != nil {
		t.Fatalf("create running: %v", err)
	}
	if running.LeaseVersion != 0 {
		t.Fatalf("fresh log lease_version = %d, want 0", running.LeaseVersion)
	}
	oldVersion := running.LeaseVersion

	// stale running 认领：用当前 lease_version 认领，第一次成功（version 0→1）。
	claimed, err := pk.ClaimFetchLog(running.Id, pk.FetchStatusRunning, oldVersion)
	if err != nil || !claimed {
		t.Fatalf("claim stale running = (%v, %v), want (true, nil)", claimed, err)
	}
	// 第二个并发者仍用同一个旧 version 认领：赢家已把 lease_version 置为 1，
	// WHERE lease_version=0 不再匹配，应失败（RowsAffected==0）。
	claimed, err = pk.ClaimFetchLog(running.Id, pk.FetchStatusRunning, oldVersion)
	if err != nil || claimed {
		t.Fatalf("second claim with stale version = (%v, %v), want (false, nil)", claimed, err)
	}
	// 重新读取：lease_version 已递增，用新版本可再次认领（模拟陈旧窗口后的下一轮续跑）。
	gotRunning, _ := pk.LatestFetchLogByCalendar(121)
	if gotRunning.LeaseVersion != oldVersion+1 {
		t.Fatalf("after claim lease_version = %d, want %d", gotRunning.LeaseVersion, oldVersion+1)
	}
	claimed, err = pk.ClaimFetchLog(gotRunning.Id, pk.FetchStatusRunning, gotRunning.LeaseVersion)
	if err != nil || !claimed {
		t.Fatalf("reclaim with fresh version = (%v, %v), want (true, nil)", claimed, err)
	}

	// failed 状态可认领（状态转换 + version CAS 双重串行化），认领后置回 running。
	failed, err := pk.CreateFetchLog(122)
	if err != nil {
		t.Fatalf("create 122: %v", err)
	}
	failed.Status = pk.FetchStatusFailed
	if err := pk.SaveFetchLog(failed); err != nil {
		t.Fatalf("mark failed via SaveFetchLog: %v", err)
	}
	if failed.RunningKey != nil {
		t.Errorf("failed running_key = %v, want nil", failed.RunningKey)
	}
	failedVersion := failed.LeaseVersion // SaveFetchLog 不改 lease_version
	if claimed, err = pk.ClaimFetchLog(failed.Id, pk.FetchStatusFailed, failedVersion); err != nil || !claimed {
		t.Fatalf("claim failed = (%v, %v), want (true, nil)", claimed, err)
	}
	var got pk.FetchLogEntity
	if err := db.Connect().Where("id = ?", failed.Id).First(&got).Error; err != nil {
		t.Fatalf("load claimed: %v", err)
	}
	if got.Status != pk.FetchStatusRunning {
		t.Errorf("claimed status = %q, want running", got.Status)
	}
	if got.LeaseVersion != failedVersion+1 {
		t.Errorf("claimed lease_version = %d, want %d", got.LeaseVersion, failedVersion+1)
	}
	if got.RunningKey == nil || *got.RunningKey != 122 {
		t.Errorf("claimed running_key = %v, want 122", got.RunningKey)
	}
	// failed 认领后状态已转 running 且 version 已递增：再按 failed 用旧 version 认领应失败。
	if claimed, err = pk.ClaimFetchLog(failed.Id, pk.FetchStatusFailed, failedVersion); err != nil || claimed {
		t.Fatalf("second claim failed with stale version = (%v, %v), want (false, nil)", claimed, err)
	}
}

// TestFetchLogNewRunningAfterCompleted 回归（review P1 跨方言场景）：某 calendar 的 fetchlog 标记
// completed 后，必须能为同一 calendar 新建 running log。旧实现用 (calendar_id) WHERE status='running'
// partial unique index——部分方言的 migrator 不生成 WHERE 子句，退化为普通 UNIQUE(calendar_id)，completed
// 行仍占住 calendar_id，重跑新建必失败；新实现用可空 running_key + 普通唯一索引，completed 行
// running_key=NULL，两库均允许多个 NULL，重跑新建成功。
func TestFetchLogNewRunningAfterCompleted(t *testing.T) {
	migratePkTables(t)
	const calendarID = uint64(121)

	if _, err := pk.CreateFetchLog(calendarID); err != nil {
		t.Fatalf("first create: %v", err)
	}
	// 完整跑完后标记 completed（经 SaveFetchLog，running_key 被置 NULL）。
	log, _ := pk.LatestFetchLogByCalendar(calendarID)
	log.Status = pk.FetchStatusCompleted
	if err := pk.SaveFetchLog(&log); err != nil {
		t.Fatalf("mark completed: %v", err)
	}
	// 同 calendar 新建 running：必须成功（completed 行 running_key 为 NULL，不占唯一索引）。
	re, err := pk.CreateFetchLog(calendarID)
	if err != nil {
		t.Fatalf("create running after completed should succeed: %v", err)
	}
	if re.Status != pk.FetchStatusRunning {
		t.Errorf("new log status = %q, want running", re.Status)
	}
	if re.RunningKey == nil || *re.RunningKey != calendarID {
		t.Errorf("new log running_key = %v, want %d", re.RunningKey, calendarID)
	}
}

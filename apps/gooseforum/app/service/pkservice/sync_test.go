package pkservice

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
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

// fakeOneSystem 模拟一系统 manualArrange 分页服务，可注入按页失败与全局 401。
type fakeOneSystem struct {
	mu        sync.Mutex
	courses   []CourseRaw
	failPages map[int]int // pageNum -> HTTP status
	cookieOK  bool
	calls     []int // 按顺序记录请求的 pageNum
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
		f.mu.Unlock()

		if !cookieOK {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message":"未登录或会话失效"}`))
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

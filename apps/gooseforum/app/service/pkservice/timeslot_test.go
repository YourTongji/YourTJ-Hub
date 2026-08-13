package pkservice

import (
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"gorm.io/gorm"
)

var pkServiceModels = []any{
	&pk.CalendarEntity{},
	&pk.CampusEntity{},
	&pk.FacultyEntity{},
	&pk.LanguageEntity{},
	&pk.AssessmentEntity{},
	&pk.CourseNatureEntity{},
	&pk.MajorEntity{},
	&pk.MajorCourseEntity{},
	&pk.CourseDetailEntity{},
	&pk.TeacherEntity{},
	&pk.TeacherTimeslotEntity{},
	&pk.FetchLogEntity{},
	&pk.SettingEntity{},
}

func setupPkServiceTest(t *testing.T) *gorm.DB {
	t.Helper()
	ResetPkAuxiliaryStateForTest()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(pkServiceModels...); err != nil {
		t.Fatalf("migrate pk service tables: %v", err)
	}
	cleanupPkServiceTables(t, conn)
	t.Cleanup(func() {
		WaitPkAuxiliaryBuildForTest()
		ResetPkAuxiliaryStateForTest()
		cleanupPkServiceTables(t, conn)
	})
	return conn
}

func cleanupPkServiceTables(t *testing.T, conn *gorm.DB) {
	t.Helper()
	for _, m := range pkServiceModels {
		if err := conn.Unscoped().Where("1 = 1").Delete(m).Error; err != nil {
			t.Fatalf("clean pk table: %v", err)
		}
	}
}

// TestRebuildTeacherTimeslots 验证 arrangeInfoText 解析 → timeslots 投影 → 版本标记 → 就绪。
func TestRebuildTeacherTimeslots(t *testing.T) {
	conn := setupPkServiceTest(t)
	seedTimeslotFixture(t, conn)

	if err := rebuildTeacherTimeslots(); err != nil {
		t.Fatalf("rebuildTeacherTimeslots: %v", err)
	}
	if !isPkAuxiliaryReady() {
		t.Fatal("timeslots should be ready after rebuild")
	}
	var count int64
	if err := conn.Model(&pk.TeacherTimeslotEntity{}).Count(&count).Error; err != nil {
		t.Fatalf("count timeslots: %v", err)
	}
	// 900001 两行 → 4 个（1-2 与 3-4 各展开为两个节次）；900003 一行 "星期五1-2节" → 2 个。
	if count != 6 {
		t.Fatalf("timeslot count = %d, want 6", count)
	}
}

// TestFindCoursesByTimeReady 就绪后 courses-by-time 返回 auxiliaryReady:true 与精确结果。
func TestFindCoursesByTimeReady(t *testing.T) {
	conn := setupPkServiceTest(t)
	seedTimeslotFixture(t, conn)
	if err := rebuildTeacherTimeslots(); err != nil {
		t.Fatalf("rebuildTeacherTimeslots: %v", err)
	}

	result, err := FindCoursesByTime(99999, 5, 1)
	if err != nil {
		t.Fatalf("FindCoursesByTime: %v", err)
	}
	if !result.AuxiliaryReady {
		t.Fatal("auxiliaryReady = false, want true after rebuild")
	}
	if len(result.Courses) != 1 {
		t.Fatalf("courses = %d, want 1: %+v", len(result.Courses), result.Courses)
	}
	if result.Courses[0].CourseCode != "TJCS201" {
		t.Fatalf("courseCode = %q, want TJCS201", result.Courses[0].CourseCode)
	}
	if len(result.Courses[0].CourseNature) != 1 || result.Courses[0].CourseNature[0] != "通识选修课" {
		t.Fatalf("courseNature = %v, want [通识选修课]", result.Courses[0].CourseNature)
	}
}

// TestFindCoursesByTimeDegraded 未就绪时返回 auxiliaryReady:false 的降级 LIKE 结果。
func TestFindCoursesByTimeDegraded(t *testing.T) {
	conn := setupPkServiceTest(t)
	seedTimeslotFixture(t, conn)

	result, err := FindCoursesByTime(99999, 5, 1)
	if err != nil {
		t.Fatalf("FindCoursesByTime: %v", err)
	}
	if result.AuxiliaryReady {
		t.Fatal("auxiliaryReady = true, want false (no timeslots seeded)")
	}
	if len(result.Courses) != 1 || result.Courses[0].CourseCode != "TJCS201" {
		t.Fatalf("degraded courses = %+v, want [TJCS201]", result.Courses)
	}
}

// TestFindCoursesByTimeSection6Degraded 回归：section=6 降级路径生成两个 LIKE pattern
// （10-11 / 10-12），必须变参展开为多个占位符，否则 SQL 报错 500。
func TestFindCoursesByTimeSection6Degraded(t *testing.T) {
	conn := setupPkServiceTest(t)
	seedTimeslotFixture(t, conn)
	conn.Create(&pk.CourseDetailEntity{Id: 900004, Code: "TJCS30101", CourseCode: "TJCS301", CourseName: "艺术鉴赏", CourseLabelId: 2, Credit: 2, Campus: "JD", Faculty: "CS", TeachingLanguage: "ZH", CalendarId: 99999})
	conn.Create(&pk.TeacherEntity{Id: 4, TeachingClassId: 900004, TeacherCode: "T009", TeacherName: "王芳", ArrangeInfoText: "王芳(T009) 星期五10-11节[1-16周] 嘉定校区 D404"})

	result, err := FindCoursesByTime(99999, 5, 6)
	if err != nil {
		t.Fatalf("FindCoursesByTime(section 6): %v", err)
	}
	if result.AuxiliaryReady {
		t.Fatal("auxiliaryReady = true, want false (degraded)")
	}
	if len(result.Courses) != 1 || result.Courses[0].CourseCode != "TJCS301" {
		t.Fatalf("degraded courses = %+v, want [TJCS301]", result.Courses)
	}
}

func seedTimeslotFixture(t *testing.T, conn *gorm.DB) {
	t.Helper()
	models := []any{
		&pk.CourseDetailEntity{Id: 900003, Code: "TJCS20101", CourseCode: "TJCS201", CourseName: "数据结构与算法", CourseLabelId: 2, Credit: 4, Campus: "JD", Faculty: "CS", TeachingLanguage: "ZH", CalendarId: 99999},
		&pk.CourseDetailEntity{Id: 900001, Code: "TJCS10101", CourseCode: "TJCS101", CourseName: "计算机程序设计", CourseLabelId: 1, Credit: 3, Campus: "SP", Faculty: "CS", TeachingLanguage: "ZH", CalendarId: 99999},
		&pk.CourseNatureEntity{CalendarId: 99999, CourseLabelId: 1, CourseLabelName: "专业必修"},
		&pk.CourseNatureEntity{CalendarId: 99999, CourseLabelId: 2, CourseLabelName: "通识选修课"},
		&pk.FacultyEntity{Faculty: "CS", FacultyI18n: "计算机科学与技术系", CalendarId: 99999},
		&pk.CampusEntity{Campus: "SP", CampusI18n: "四平路校区", CalendarId: 99999},
		&pk.CampusEntity{Campus: "JD", CampusI18n: "嘉定校区", CalendarId: 99999},
		&pk.TeacherEntity{Id: 1, TeachingClassId: 900001, TeacherCode: "T001", TeacherName: "张伟", ArrangeInfoText: "张伟(T001) 星期一1-2节[1-16周] 四平路校区 A101\n张伟(T001) 星期三3-4节[1-16周] 四平路校区 A101"},
		&pk.TeacherEntity{Id: 3, TeachingClassId: 900003, TeacherCode: "T001", TeacherName: "张伟", ArrangeInfoText: "张伟(T001) 星期五1-2节[1-16周] 嘉定校区 C303"},
	}
	for _, m := range models {
		if err := conn.Create(m).Error; err != nil {
			t.Fatalf("create %T: %v", m, err)
		}
	}
}

// TestTriggerPkAuxiliaryBuildBackoffThrottle 验证构建失败后的指数退避窗口内
// TriggerPkAuxiliaryBuild 不再启动重建（回归 synergy-agent 评审 P1）。
func TestTriggerPkAuxiliaryBuildBackoffThrottle(t *testing.T) {
	setupPkServiceTest(t)

	// 手工构造"刚失败"状态：ready=false，退避截止在 10s 后。
	pkAuxStateMu.Lock()
	pkAuxBuildRunning = false
	pkAuxReadyValue = false
	pkAuxReadyExpires = time.Now().Add(10 * time.Second)
	pkAuxRetryAfter = time.Now().Add(10 * time.Second)
	pkAuxFailCount = 1
	pkAuxStateMu.Unlock()

	// 退避窗口内：不应启动新构建（pkAuxBuildRunning 保持 false，failCount 不变）。
	TriggerPkAuxiliaryBuild()
	pkAuxStateMu.Lock()
	running := pkAuxBuildRunning
	ready := pkAuxReadyValue
	failCount := pkAuxFailCount
	pkAuxStateMu.Unlock()
	if running {
		t.Fatal("退避窗口内 TriggerPkAuxiliaryBuild 不应启动重建")
	}
	if ready {
		t.Fatal("退避窗口内就绪状态应保持 false")
	}
	if failCount != 1 {
		t.Fatalf("退避窗口内 failCount 不应变化，got %d", failCount)
	}

	// 退避结束后：应允许重建并正常完成。
	pkAuxStateMu.Lock()
	pkAuxRetryAfter = time.Time{}
	pkAuxStateMu.Unlock()
	TriggerPkAuxiliaryBuild()
	WaitPkAuxiliaryBuildForTest()
	pkAuxStateMu.Lock()
	running = pkAuxBuildRunning
	pkAuxStateMu.Unlock()
	if running {
		t.Fatal("重建完成后 pkAuxBuildRunning 应为 false")
	}
}

// TestRebuildTeacherTimeslotsDoesNotClearOnReadFailure 验证重建先在内存解析、
// 读取成功后才清空旧表：ListTeacherArrangeRows 失败（pk_course_detail 缺失）时
// teacher_timeslots 不被清空（回归 WALKERKILLER 评审 P2：clear-first 失败窗口）。
func TestRebuildTeacherTimeslotsDoesNotClearOnReadFailure(t *testing.T) {
	conn := setupPkServiceTest(t)
	seedTimeslotFixture(t, conn)

	if err := rebuildTeacherTimeslots(); err != nil {
		t.Fatalf("initial rebuild: %v", err)
	}
	countBefore := countTeacherTimeslots(t, conn)
	if countBefore != 6 {
		t.Fatalf("seed timeslots = %d, want 6", countBefore)
	}

	// 删除 pk_course_detail 表，使 ListTeacherArrangeRows（JOIN 该表）必然失败。
	if err := conn.Migrator().DropTable(&pk.CourseDetailEntity{}); err != nil {
		t.Fatalf("drop pk_course_detail: %v", err)
	}
	t.Cleanup(func() {
		if err := conn.AutoMigrate(&pk.CourseDetailEntity{}); err != nil {
			t.Errorf("restore pk_course_detail: %v", err)
		}
	})

	if err := rebuildTeacherTimeslots(); err == nil {
		t.Fatal("rebuild 应在 pk_course_detail 缺失时失败")
	}
	if got := countTeacherTimeslots(t, conn); got != countBefore {
		t.Fatalf("重建失败后 timeslots = %d, want %d（clear 必须发生在解析成功后）", got, countBefore)
	}
}

func countTeacherTimeslots(t *testing.T, conn *gorm.DB) int64 {
	t.Helper()
	var count int64
	if err := conn.Model(&pk.TeacherTimeslotEntity{}).Count(&count).Error; err != nil {
		t.Fatalf("count timeslots: %v", err)
	}
	return count
}

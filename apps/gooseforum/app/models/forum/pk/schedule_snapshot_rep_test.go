package pk

import (
	"errors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"os"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
)

// setupScheduleSnapshotTest 迁移快照表并注册清理。
func setupScheduleSnapshotTest(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&ScheduleSnapshotEntity{}); err != nil {
		t.Fatalf("migrate schedule snapshot: %v", err)
	}
	t.Cleanup(func() {
		if err := conn.Where("1 = 1").Delete(&ScheduleSnapshotEntity{}).Error; err != nil {
			t.Errorf("cleanup schedule snapshot: %v", err)
		}
	})
}

func samplePlans() PlanList {
	return PlanList{{
		Id:              "plan_r1",
		Name:            "方案 1",
		CreatedAt:       1725000000000,
		StagedCourses:   []StagedCoursePayload{},
		SelectedCourses: []string{},
		CustomEvents:    []CustomEventPayload{},
	}}
}

// review: UpsertScheduleSnapshot 首写建行，updated_at 为服务端时钟；
// GET 在无行时返回 ErrScheduleSnapshotNotFound（控制器转为 data:null）。
func TestScheduleSnapshotGetMissingReturnsNotFound(t *testing.T) {
	setupScheduleSnapshotTest(t)
	if _, err := GetScheduleSnapshotByUser(5301); err == nil || !errors.Is(err, ErrScheduleSnapshotNotFound) {
		t.Fatalf("missing user error = %v, want ErrScheduleSnapshotNotFound", err)
	}
}

// review: 快照 JSON 列（plans/majorSelected/weekView）经 gorm serializer 往返
// 必须无损——这是「云端快照与客户端 schema 逐字段互携」不变量的存储侧证明。
func TestScheduleSnapshotRoundTripPreservesPayload(t *testing.T) {
	setupScheduleSnapshotTest(t)
	week := 5
	major := "080601"
	plans := samplePlans()
	plans[0].StagedCourses = []StagedCoursePayload{{
		CourseCode:   "122004",
		CourseName:   "数据结构",
		Credit:       4,
		CourseType:   "必",
		CourseNature: []string{"专业必修"},
		Teacher:      []TeacherPayload{{TeacherName: "张伟", TeacherCode: "T001"}},
		Status:       2,
		CourseDetail: []CourseDetailPayload{{
			ArrangementInfo: []ArrangementPayload{{
				ArrangementText: "[1-8周]周一第3-4节",
				OccupyDay:       1,
				OccupyTime:      []int{3, 4},
				OccupyWeek:      []int{1, 2, 3, 4, 5, 6, 7, 8},
				OccupyRoom:      "同德楼A201",
				TeacherAndCode:  "张伟(T001)",
			}},
			Campus:           "四平路",
			Code:             "122004.01",
			TeachingClassId:  ptrUint64(445566),
			IsExclusive:      ptrBool(true),
			Status:           ptrInt(2),
			Teachers:         []TeacherPayload{{TeacherName: "张伟", TeacherCode: "T001"}},
			TeachingLanguage: "中文",
		}},
	}}
	plans[0].SelectedCourses = []string{"122004.01"}
	plans[0].CustomEvents = []CustomEventPayload{{
		Id: "evt_1", Label: "有事", Day: 6, Sections: []int{1, 2}, Weeks: []int{1, 3, 5},
	}}
	entity := &ScheduleSnapshotEntity{
		UserId:        5302,
		Plans:         plans,
		ActivePlanId:  "plan_r1",
		MajorSelected: MajorSelectionPayload{CalendarId: ptrInt(121), Grade: ptrInt(2024), Major: &major, MajorName: ptrString("计算机科学与技术")},
		WeekView:      WeekViewPayload{Week: &week, UseCurrent: true},
	}
	if err := UpsertScheduleSnapshot(entity); err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	got, err := GetScheduleSnapshotByUser(5302)
	if err != nil {
		t.Fatalf("get snapshot: %v", err)
	}
	if got.UserId != 5302 || got.ActivePlanId != "plan_r1" || len(got.Plans) != 1 {
		t.Fatalf("scalar fields lost: %+v", got)
	}
	course := got.Plans[0].StagedCourses[0]
	if course.CourseName != "数据结构" || course.Status != 2 || len(course.CourseDetail) != 1 {
		t.Fatalf("staged course lost: %+v", course)
	}
	detail := course.CourseDetail[0]
	if detail.Code != "122004.01" || detail.TeachingClassId == nil || *detail.TeachingClassId != 445566 ||
		detail.IsExclusive == nil || !*detail.IsExclusive || detail.Status == nil || *detail.Status != 2 {
		t.Fatalf("course detail optional fields lost: %+v", detail)
	}
	arr := detail.ArrangementInfo[0]
	if arr.OccupyDay != 1 || len(arr.OccupyTime) != 2 || len(arr.OccupyWeek) != 8 || arr.OccupyRoom != "同德楼A201" {
		t.Fatalf("arrangement lost: %+v", arr)
	}
	event := got.Plans[0].CustomEvents[0]
	if event.Label != "有事" || event.Day != 6 || len(event.Sections) != 2 || len(event.Weeks) != 3 {
		t.Fatalf("custom event lost: %+v", event)
	}
	if got.MajorSelected.Major == nil || *got.MajorSelected.Major != major ||
		got.MajorSelected.CalendarId == nil || *got.MajorSelected.CalendarId != 121 ||
		got.MajorSelected.MajorName == nil || *got.MajorSelected.MajorName != "计算机科学与技术" {
		t.Fatalf("major selection lost: %+v", got.MajorSelected)
	}
	if got.WeekView.Week == nil || *got.WeekView.Week != 5 || !got.WeekView.UseCurrent {
		t.Fatalf("week view lost: %+v", got.WeekView)
	}
}

// review: 二次 upsert 是整体替换语义——四字段全部更新、created_at 保持首建、
// updated_at 服务端时钟单调刷新（客户端 pk.syncedAt 比对依赖该语义）。
func TestScheduleSnapshotUpsertReplacesAndBumpsClock(t *testing.T) {
	setupScheduleSnapshotTest(t)
	first := &ScheduleSnapshotEntity{
		UserId:       5303,
		Plans:        samplePlans(),
		ActivePlanId: "plan_r1",
	}
	if err := UpsertScheduleSnapshot(first); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	original, err := GetScheduleSnapshotByUser(5303)
	if err != nil {
		t.Fatalf("get after first upsert: %v", err)
	}

	second := &ScheduleSnapshotEntity{
		UserId:       5303,
		Plans:        samplePlans(),
		ActivePlanId: "plan_r1",
	}
	if err := UpsertScheduleSnapshot(second); err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	updated, err := GetScheduleSnapshotByUser(5303)
	if err != nil {
		t.Fatalf("get after second upsert: %v", err)
	}
	if updated.Id != original.Id {
		t.Fatalf("upsert created a second row: id %d -> %d", original.Id, updated.Id)
	}
	if !updated.CreatedAt.Equal(original.CreatedAt) {
		t.Fatalf("created_at drifted on upsert: %v -> %v", original.CreatedAt, updated.CreatedAt)
	}
	if !updated.UpdatedAt.After(original.UpdatedAt) {
		t.Fatalf("updated_at not refreshed on upsert: %v -> %v", original.UpdatedAt, updated.UpdatedAt)
	}
}

// review: 替换语义逐字段生效——新 activePlanId 与周次视图覆盖旧值。
func TestScheduleSnapshotUpsertReplacesAllFields(t *testing.T) {
	setupScheduleSnapshotTest(t)
	week := 3
	first := &ScheduleSnapshotEntity{
		UserId:       5304,
		Plans:        samplePlans(),
		ActivePlanId: "plan_r1",
		WeekView:     WeekViewPayload{Week: &week, UseCurrent: false},
	}
	if err := UpsertScheduleSnapshot(first); err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	second := &ScheduleSnapshotEntity{
		UserId:       5304,
		Plans:        samplePlans(),
		ActivePlanId: "plan_r1",
		WeekView:     WeekViewPayload{Week: nil, UseCurrent: true},
	}
	if err := UpsertScheduleSnapshot(second); err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	got, err := GetScheduleSnapshotByUser(5304)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.WeekView.Week != nil || !got.WeekView.UseCurrent {
		t.Fatalf("week view not replaced (zero-value fields must overwrite): %+v", got.WeekView)
	}
}

// review: DeleteScheduleSnapshotByUser 幂等；删除后回到 not-found；
// 多用户互不影响（user_id 锚定，仅本人数据可删）。
func TestScheduleSnapshotDeleteIsIdempotentAndScoped(t *testing.T) {
	setupScheduleSnapshotTest(t)
	for _, userId := range []uint64{5305, 5306} {
		if err := UpsertScheduleSnapshot(&ScheduleSnapshotEntity{UserId: userId, Plans: samplePlans(), ActivePlanId: "plan_r1"}); err != nil {
			t.Fatalf("upsert user %d: %v", userId, err)
		}
	}
	if err := DeleteScheduleSnapshotByUser(5305); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := DeleteScheduleSnapshotByUser(5305); err != nil {
		t.Fatalf("idempotent delete: %v", err)
	}
	if _, err := GetScheduleSnapshotByUser(5305); !errors.Is(err, ErrScheduleSnapshotNotFound) {
		t.Fatalf("deleted user error = %v, want not found", err)
	}
	if _, err := GetScheduleSnapshotByUser(5306); err != nil {
		t.Fatalf("other user snapshot must survive: %v", err)
	}
}

// review: user_id 唯一索引兜底并发首写——绕过 rep 事务直接 Create 第二行必须
// 被唯一约束拒绝（gorm.ErrDuplicatedKey），保证「每用户至多一行」不变量。
func TestScheduleSnapshotUserIdUnique(t *testing.T) {
	setupScheduleSnapshotTest(t)
	conn := db.Connect()
	if err := conn.Create(&ScheduleSnapshotEntity{UserId: 5307, Plans: samplePlans(), ActivePlanId: "plan_r1"}).Error; err != nil {
		t.Fatalf("seed first row: %v", err)
	}
	dup := &ScheduleSnapshotEntity{UserId: 5307, Plans: samplePlans(), ActivePlanId: "plan_r1"}
	if err := conn.Create(dup).Error; err == nil {
		t.Fatal("duplicate user_id insert succeeded, want unique constraint error")
	}
}

func ptrInt(v int) *int          { return &v }
func ptrUint64(v uint64) *uint64 { return &v }
func ptrBool(v bool) *bool       { return &v }
func ptrString(v string) *string { return &v }

func TestScheduleSnapshotPostgresCompareAndSwap(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set")
	}
	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.AutoMigrate(&ScheduleSnapshotEntity{}); err != nil {
		t.Fatal(err)
	}
	const uid = 5570001
	defer conn.Where("user_id = ?", uid).Delete(&ScheduleSnapshotEntity{})
	entity := &ScheduleSnapshotEntity{UserId: uid, Plans: samplePlans(), ActivePlanId: "plan_r1"}
	if err := compareAndSwapScheduleSnapshot(conn, entity, ""); err != nil {
		t.Fatal(err)
	}
	var read ScheduleSnapshotEntity
	if err := conn.Where("user_id = ?", uid).First(&read).Error; err != nil {
		t.Fatal(err)
	}
	if !read.UpdatedAt.Equal(entity.UpdatedAt) {
		t.Fatal("PUT clock differs from PG read")
	}
	base := read.UpdatedAt.UTC().Format(time.RFC3339Nano)
	entity.Plans[0].Name = "winner"
	if err := compareAndSwapScheduleSnapshot(conn, entity, base); err != nil {
		t.Fatal(err)
	}
	entity.Plans[0].Name = "stale"
	if err := compareAndSwapScheduleSnapshot(conn, entity, base); !errors.Is(err, ErrScheduleSnapshotConflict) {
		t.Fatalf("stale error: %v", err)
	}
	if err := conn.Where("user_id = ?", uid).First(&read).Error; err != nil {
		t.Fatal(err)
	}
	if read.Plans[0].Name != "winner" {
		t.Fatal("stale writer changed snapshot")
	}
}

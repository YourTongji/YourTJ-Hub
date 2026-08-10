package searchservice

import (
	"errors"
	"testing"

	"github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/course"
	"github.com/leancodebox/GooseForum/app/models/forum/taskQueue"
	"gorm.io/gorm"
)

func setupCourseSearchTestDB(t *testing.T) {
	t.Helper()
	conn := dbconnect.Connect()
	models := []any{
		&course.Entity{},
		&course.AliasEntity{},
		&course.TermEntity{},
		&course.OfferingEntity{},
		&course.InstructorEntity{},
		&course.OfferingInstructorEntity{},
		&taskQueue.Entity{},
	}
	if err := conn.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate course search tables: %v", err)
	}
	for _, model := range models {
		conn.Unscoped().Where("1 = 1").Delete(model)
	}
}

func TestShouldIndexCourse(t *testing.T) {
	if !shouldIndexCourse(course.Entity{Id: 1, Status: course.StatusVisible}) {
		t.Fatal("visible course should be indexed")
	}
	if shouldIndexCourse(course.Entity{Id: 0, Status: course.StatusVisible}) {
		t.Fatal("zero-id course should not be indexed")
	}
	if shouldIndexCourse(course.Entity{Id: 1, Status: course.StatusHidden}) {
		t.Fatal("hidden course should not be indexed")
	}
}

func TestConvertCourseToSearchDocument(t *testing.T) {
	setupCourseSearchTestDB(t)
	conn := dbconnect.Connect()

	entity := &course.Entity{
		Id:             42,
		PrimaryCode:    "100001",
		Name:           "高等数学(A)上",
		NormalizedName: "gaodengshuxue",
		NamePinyin:     "gaodengshuxueashang",
		NameInitials:   "gdsxas",
		Department:     "数学科学学院",
		CreditX10:      50,
		Status:         course.StatusVisible,
	}
	if err := conn.Create(entity).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}
	if err := conn.Create(&course.AliasEntity{
		CourseId:        entity.Id,
		Kind:            course.AliasKindName,
		Value:           "高数",
		NormalizedValue: "高数",
		Source:          "test",
	}).Error; err != nil {
		t.Fatalf("create alias: %v", err)
	}
	term := &course.TermEntity{Code: "2025-2026-1", Name: "2025-2026 第一学期"}
	if err := conn.Create(term).Error; err != nil {
		t.Fatalf("create term: %v", err)
	}
	ins := &course.InstructorEntity{Name: "张三", NormalizedName: "张三", Department: "数学科学学院"}
	if err := conn.Create(ins).Error; err != nil {
		t.Fatalf("create instructor: %v", err)
	}
	offering := &course.OfferingEntity{
		CourseId: entity.Id,
		TermId:   term.Id,
		Campus:   "四平路校区",
		Status:   course.OfferingStatusVisible,
	}
	if err := conn.Create(offering).Error; err != nil {
		t.Fatalf("create offering: %v", err)
	}
	if err := conn.Create(&course.OfferingInstructorEntity{
		OfferingId:   offering.Id,
		InstructorId: ins.Id,
		Role:         "lecturer",
	}).Error; err != nil {
		t.Fatalf("create offering instructor: %v", err)
	}

	doc, err := convertCourseToSearchDocument(*entity)
	if err != nil {
		t.Fatalf("convert course doc: %v", err)
	}
	if doc.ID != 42 || doc.PrimaryCode != "100001" {
		t.Fatalf("doc identity wrong: %+v", doc)
	}
	if len(doc.Aliases) != 1 || doc.Aliases[0] != "高数" {
		t.Fatalf("doc aliases wrong: %+v", doc.Aliases)
	}
	if len(doc.Instructors) != 1 || doc.Instructors[0] != "张三" {
		t.Fatalf("doc instructors wrong: %+v", doc.Instructors)
	}
	if len(doc.Terms) != 1 || doc.Terms[0] != "2025-2026-1" {
		t.Fatalf("doc terms wrong: %+v", doc.Terms)
	}
	if len(doc.Campus) != 1 || doc.Campus[0] != "四平路校区" {
		t.Fatalf("doc campus wrong: %+v", doc.Campus)
	}
}

func TestEnqueueCourseSearchTaskTxBound(t *testing.T) {
	conn := dbconnect.Connect()

	countTasks := func() int64 {
		var n int64
		conn.Model(&taskQueue.Entity{}).Where("type LIKE ?", TaskTypeCourseSearch+"%").Count(&n)
		return n
	}

	// 事务内 enqueue 并提交 → 任务可见。
	if err := conn.Transaction(func(tx *gorm.DB) error {
		return EnqueueCourseSearchTask(tx, 42)
	}); err != nil {
		t.Fatalf("enqueue in tx: %v", err)
	}
	if n := countTasks(); n != 1 {
		t.Fatalf("committed tx should leave 1 task, got %d", n)
	}

	// 回滚事务 → 任务不存在（outbox 事务性）。
	rollbackErr := conn.Transaction(func(tx *gorm.DB) error {
		if err := EnqueueCourseSearchTask(tx, 43); err != nil {
			return err
		}
		return errors.New("rollback")
	})
	if rollbackErr == nil || rollbackErr.Error() != "rollback" {
		t.Fatalf("expected rollback error, got %v", rollbackErr)
	}
	if n := countTasks(); n != 1 {
		t.Fatalf("rolled-back tx must not leave task, got %d", n)
	}
}

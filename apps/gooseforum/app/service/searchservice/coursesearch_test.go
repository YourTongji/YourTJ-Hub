package searchservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
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

func TestEnqueueTopicSearchTaskTxBoundAndDeduplicated(t *testing.T) {
	setupCourseSearchTestDB(t)
	conn := dbconnect.Connect()
	countTasks := func() int64 {
		var n int64
		if err := conn.Model(&taskQueue.Entity{}).Where("type LIKE ?", TaskTypeTopicSearch+"%").Count(&n).Error; err != nil {
			t.Fatalf("count topic tasks: %v", err)
		}
		return n
	}

	if err := conn.Transaction(func(tx *gorm.DB) error {
		return EnqueueTopicSearchTask(tx, 42)
	}); err != nil {
		t.Fatalf("enqueue topic task: %v", err)
	}
	if err := conn.Transaction(func(tx *gorm.DB) error {
		return EnqueueTopicSearchTask(tx, 42)
	}); err != nil {
		t.Fatalf("deduplicate topic task: %v", err)
	}
	if got := countTasks(); got != 1 {
		t.Fatalf("duplicate topic task count = %d, want 1", got)
	}

	rollbackErr := conn.Transaction(func(tx *gorm.DB) error {
		if err := EnqueueTopicSearchTask(tx, 43); err != nil {
			return err
		}
		return errors.New("rollback")
	})
	if rollbackErr == nil || rollbackErr.Error() != "rollback" {
		t.Fatalf("expected rollback error, got %v", rollbackErr)
	}
	if got := countTasks(); got != 1 {
		t.Fatalf("rolled-back topic task count = %d, want 1", got)
	}
}

// TestBuildCourseIndexPagesConvertFailure 任一页转换失败必须整体返回错误：
// 索引已在此前清空，若只累计 FailedCount 继续，该批课程会永久丢失且 CLI 仍报成功。
func TestBuildCourseIndexPagesConvertFailure(t *testing.T) {
	listCalls := 0
	_, err := buildCourseIndexPages(context.Background(),
		func(limit, offset int) ([]course.Entity, error) {
			listCalls++
			if listCalls == 1 {
				return []course.Entity{{Id: 1, Status: course.StatusVisible}}, nil
			}
			return nil, nil
		},
		func(entities []course.Entity) ([]CourseSearchDocument, error) {
			return nil, errors.New("boom: conversion failed")
		},
		func(docs []CourseSearchDocument) error {
			t.Fatal("addDocs must not be called when conversion fails")
			return nil
		})
	if err == nil {
		t.Fatal("expected conversion failure to abort rebuild")
	}
	if !strings.Contains(err.Error(), "convert course search docs batch 0") {
		t.Fatalf("expected batch-context error, got %v", err)
	}
}

// TestBuildCourseIndexPagesAddDocsFailure 写入失败同样中止 rebuild 并返回错误。
func TestBuildCourseIndexPagesAddDocsFailure(t *testing.T) {
	_, err := buildCourseIndexPages(context.Background(),
		func(limit, offset int) ([]course.Entity, error) {
			return []course.Entity{{Id: 1, Status: course.StatusVisible}}, nil
		},
		func(entities []course.Entity) ([]CourseSearchDocument, error) {
			return []CourseSearchDocument{{ID: 1}}, nil
		},
		func(docs []CourseSearchDocument) error {
			return errors.New("boom: add documents failed")
		})
	if err == nil || !strings.Contains(err.Error(), "add documents failed") {
		t.Fatalf("expected addDocs failure to abort rebuild, got %v", err)
	}
}

// TestBuildCourseIndexPagesSuccess 分页成功路径：hidden 课程过滤、多批追加、计数正确。
func TestBuildCourseIndexPagesSuccess(t *testing.T) {
	pages := [][]course.Entity{
		{
			{Id: 1, Status: course.StatusVisible, Name: "高数"},
			{Id: 2, Status: course.StatusHidden, Name: "隐藏课"},
			{Id: 3, Status: course.StatusVisible, Name: "线代"},
		},
		nil, // 第二页为空 → 结束
	}
	pageIdx := 0
	var added [][]CourseSearchDocument
	result, err := buildCourseIndexPages(context.Background(),
		func(limit, offset int) ([]course.Entity, error) {
			if pageIdx < len(pages) {
				p := pages[pageIdx]
				pageIdx++
				return p, nil
			}
			return nil, nil
		},
		func(entities []course.Entity) ([]CourseSearchDocument, error) {
			docs := make([]CourseSearchDocument, 0, len(entities))
			for _, e := range entities {
				docs = append(docs, CourseSearchDocument{ID: e.Id, Name: e.Name})
			}
			return docs, nil
		},
		func(docs []CourseSearchDocument) error {
			added = append(added, docs)
			return nil
		})
	if err != nil {
		t.Fatalf("buildCourseIndexPages: %v", err)
	}
	if len(added) != 1 || len(added[0]) != 2 {
		t.Fatalf("expected 1 batch of 2 docs (hidden filtered out), got %+v", added)
	}
	if result.ProcessedCount != 3 || result.TotalBatches != 1 {
		t.Fatalf("expected processed=3 batches=1, got %+v", result)
	}
}

// TestBuildCourseIndexPagesContextCancel 上下文取消时中止并返回 ctx.Err()。
func TestBuildCourseIndexPagesContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := buildCourseIndexPages(ctx,
		func(limit, offset int) ([]course.Entity, error) {
			return []course.Entity{{Id: 1, Status: course.StatusVisible}}, nil
		},
		func(entities []course.Entity) ([]CourseSearchDocument, error) {
			return nil, nil
		},
		func(docs []CourseSearchDocument) error { return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

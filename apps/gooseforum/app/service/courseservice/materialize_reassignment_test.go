package courseservice

import (
	"context"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
)

// A teacher change keeps the offering identity and moves its class lookup and statistics.
func TestMaterializeFromPkTeacherReassignment(t *testing.T) {
	migrateMaterializeTables(t)
	seedPkForMaterialize(t)
	conn := db.Connect()
	check := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	check(conn.AutoMigrate(&course.ReviewEntity{}, &course.CourseStatsEntity{}, &course.OfferingStatsEntity{}))
	_, err := MaterializeFromPk(context.Background(), []uint64{1})
	check(err)
	var before course.OfferingEntity
	check(conn.Where("teaching_class_id = ?", 1).First(&before).Error)
	rating := 5
	check(conn.Create(&course.ReviewEntity{OfferingId: before.Id, Rating: &rating, Content: "synthetic review"}).Error)
	check(course.RebuildAllCourseStats())
	check(conn.Unscoped().Where("1=1").Delete(&taskQueue.Entity{}).Error)
	check(conn.Model(&pk.TeacherEntity{}).Where("id = ?", 100).Updates(map[string]any{"teacher_code": "T002", "teacher_name": "李四"}).Error)
	report, err := MaterializeFromPk(context.Background(), []uint64{1})
	check(err)
	var after course.OfferingEntity
	check(conn.Where("teaching_class_id = ?", 1).First(&after).Error)
	var alias course.AliasEntity
	check(conn.Where("normalized_value = ?", "a001n01").First(&alias).Error)
	var oldCard course.Entity
	check(conn.First(&oldCard, before.CourseId).Error)
	var oldOfferingCount int64
	check(conn.Model(&course.OfferingEntity{}).Where("course_id = ?", oldCard.Id).Count(&oldOfferingCount).Error)
	stats := course.ListCourseStatsByIDs([]uint64{before.CourseId, after.CourseId})
	var statsTasks int64
	check(conn.Model(&taskQueue.Entity{}).Where("type LIKE ?", "course-stats.%").Count(&statsTasks).Error)
	t.Logf("offering id kept=%t; old course=%d new course=%d; alias course=%d; skipped aliases=%d; old card status=%d offerings=%d; old/new review counters=%d/%d; stats rebuild tasks=%d", before.Id == after.Id, before.CourseId, after.CourseId, alias.CourseId, report.AliasesSkipped, oldCard.Status, oldOfferingCount, stats[before.CourseId].ReviewCount, stats[after.CourseId].ReviewCount, statsTasks)
	if before.Id != after.Id || before.CourseId == after.CourseId {
		t.Fatal("setup did not reproduce an offering moving between teacher cards")
	}
	t.Run("class_alias_follows_current_offering", func(t *testing.T) {
		if alias.CourseId != after.CourseId {
			t.Errorf("class alias still points to old course %d; current offering belongs to %d", alias.CourseId, after.CourseId)
		}
	})
	t.Run("search_by_class_code_finds_current_course", func(t *testing.T) {
		rows, _, err := course.ListCourses(course.ListCourseQuery{Keyword: "a001n01", IncludeHidden: true})
		check(err)
		if len(rows) != 1 || rows[0].Id != after.CourseId {
			t.Errorf("class-code search returns old card: %+v", rows)
		}
	})
	t.Run("review_counters_follow_offering_or_rebuild_queued", func(t *testing.T) {
		if statsTasks == 0 && (stats[before.CourseId].ReviewCount != 0 || stats[after.CourseId].ReviewCount != 1) {
			t.Error("review moved with offering but old/new course counters remained unchanged; no rebuild queued")
		}
	})
}

func TestMaterializeFromPkRepairsAlreadyMovedAlias(t *testing.T) {
	migrateMaterializeTables(t)
	seedPkForMaterialize(t)
	conn := db.Connect()
	if _, err := MaterializeFromPk(context.Background(), []uint64{1}); err != nil {
		t.Fatal(err)
	}
	var before course.OfferingEntity
	if err := conn.First(&before).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&pk.TeacherEntity{}).Where("id = ?", 100).Updates(map[string]any{"teacher_code": "T002", "teacher_name": "李四"}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := MaterializeFromPk(context.Background(), []uint64{1}); err != nil {
		t.Fatal(err)
	}
	var moved course.OfferingEntity
	if err := conn.First(&moved).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&course.AliasEntity{}).Where("normalized_value = ?", "a001n01").Update("course_id", before.CourseId).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Unscoped().Where("1=1").Delete(&taskQueue.Entity{}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := MaterializeFromPk(context.Background(), []uint64{1}); err != nil {
		t.Fatal(err)
	}
	var alias course.AliasEntity
	if err := conn.Where("normalized_value = ?", "a001n01").First(&alias).Error; err != nil {
		t.Fatal(err)
	}
	if alias.CourseId != moved.CourseId {
		t.Fatal("repeated materialization left stale class alias")
	}
	var queued int64
	if err := conn.Model(&taskQueue.Entity{}).Where("type LIKE ?", "course-stats.%").Count(&queued).Error; err != nil {
		t.Fatal(err)
	}
	if queued != 1 {
		t.Fatalf("stats repair tasks=%d", queued)
	}
}

func TestMaterializeFromPkKeepsExistingTeacherIdentity(t *testing.T) {
	migrateMaterializeTables(t)
	seedPkForMaterialize(t)
	conn := db.Connect()
	if _, err := MaterializeFromPk(context.Background(), []uint64{1}); err != nil {
		t.Fatal(err)
	}
	var before course.OfferingEntity
	if err := conn.First(&before).Error; err != nil {
		t.Fatal(err)
	}
	// A new teacher sorts ahead, but the existing identity still teaches this class.
	if err := conn.Create(&pk.TeacherEntity{Id: 50, TeachingClassId: 1, TeacherCode: "T000", TeacherName: "李四"}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := MaterializeFromPk(context.Background(), []uint64{1}); err != nil {
		t.Fatal(err)
	}
	var after course.OfferingEntity
	if err := conn.First(&after).Error; err != nil {
		t.Fatal(err)
	}
	if before.CourseId != after.CourseId {
		t.Fatal("adding or reordering teachers changed review identity")
	}
	var links int64
	if err := conn.Model(&course.OfferingInstructorEntity{}).Where("offering_id = ?", after.Id).Count(&links).Error; err != nil {
		t.Fatal(err)
	}
	if links != 2 {
		t.Fatalf("teacher links=%d", links)
	}
}

func TestMaterializeFromPkRejectsIncompleteSnapshots(t *testing.T) {
	for _, tc := range []struct {
		name, status     string
		pages, committed int
		wantError        bool
	}{
		{"running", pk.FetchStatusRunning, 2, 2, true},
		{"partial", pk.FetchStatusFailed, 2, 1, true},
		{"retry complete fetch", pk.FetchStatusFailed, 2, 2, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			migrateMaterializeTables(t)
			seedPkForMaterialize(t)
			conn := db.Connect()
			if err := conn.Create(&pk.FetchLogEntity{CalendarId: 1, Status: tc.status, TotalPages: tc.pages, LastCommittedPage: tc.committed}).Error; err != nil {
				t.Fatal(err)
			}
			_, err := MaterializeFromPk(context.Background(), []uint64{1})
			if (err != nil) != tc.wantError {
				t.Fatalf("error=%v wantError=%t", err, tc.wantError)
			}
			if tc.wantError {
				var count int64
				if err := conn.Model(&course.Entity{}).Count(&count).Error; err != nil {
					t.Fatal(err)
				}
				if count != 0 {
					t.Fatal("rejected snapshot wrote catalog rows")
				}
			}
		})
	}
}

func TestMaterializeFromPkPreservesManualClassAlias(t *testing.T) {
	migrateMaterializeTables(t)
	seedPkForMaterialize(t)
	conn := db.Connect()
	if _, err := MaterializeFromPk(context.Background(), []uint64{1}); err != nil {
		t.Fatal(err)
	}
	var alias course.AliasEntity
	if err := conn.Where("normalized_value = ?", "a001n01").First(&alias).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&alias).Update("source", "manual").Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&pk.TeacherEntity{}).Where("id = ?", 100).Updates(map[string]any{"teacher_code": "T002", "teacher_name": "李四"}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := MaterializeFromPk(context.Background(), []uint64{1}); err != nil {
		t.Fatal(err)
	}
	var after course.AliasEntity
	if err := conn.First(&after, alias.Id).Error; err != nil {
		t.Fatal(err)
	}
	if after.CourseId != alias.CourseId {
		t.Fatal("materializer took ownership of a manual alias")
	}
}

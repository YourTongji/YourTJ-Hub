package topics

import (
	"os"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestReviewCandidateCategoriesSQLite(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	testReviewCandidateCategories(t, conn)
}

func TestReviewCandidateCategoriesPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set")
	}
	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	testReviewCandidateCategories(t, conn)
}

func testReviewCandidateCategories(t *testing.T, conn *gorm.DB) {
	t.Helper()
	raw, err := conn.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = raw.Close() })
	if err := conn.AutoMigrate(&Entity{}, &posts.Entity{}, &postRevisions.Entity{}); err != nil {
		t.Fatal(err)
	}
	tx := conn.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	t.Cleanup(func() { tx.Rollback() })
	ids := []uint64{919501, 919502, 919503, 919504}
	for i, id := range ids {
		revisionID := uint64(0)
		if i > 0 {
			revisionID = id + 200
		}
		topic := Entity{Id: id, FirstPostId: id + 100, Status: 1, ProcessStatus: ProcessStatusPending}
		post := posts.Entity{Id: id + 100, TopicId: id, PostNo: 1, LatestRevisionId: revisionID}
		if err := tx.Create(&topic).Error; err != nil {
			t.Fatal(err)
		}
		if err := tx.Create(&post).Error; err != nil {
			t.Fatal(err)
		}
		if i > 0 {
			categoryID := uint64(2)
			if i == 2 {
				categoryID = 1
			}
			revision := postRevisions.Entity{Id: revisionID, PostId: post.Id, CategoryIds: []uint64{categoryID}, ProcessStatus: posts.ProcessStatusPending}
			if i == 3 {
				revision.CategoryIds = nil
			}
			if err := tx.Create(&revision).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	// Legacy rows remain reviewable; a moved candidate is excluded before pagination.
	query := filterReviewCandidateCategories(tx.Model(&Entity{}).Where("topics.id IN ?", ids), []uint64{1})
	var count int64
	if err := query.Session(&gorm.Session{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("candidate scope total=%d, want 2", count)
	}
	var page []Entity
	if err := query.Order("topics.id ASC").Limit(1).Offset(1).Find(&page).Error; err != nil {
		t.Fatal(err)
	}
	if len(page) != 1 || page[0].Id != ids[2] {
		t.Fatalf("candidate scope page=%+v", page)
	}
}

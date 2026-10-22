package datamigration

import (
	"os"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestBackfillEmptyTopicExcerpts(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	testBackfillEmptyTopicExcerpts(t, conn)
}

func TestBackfillEmptyTopicExcerptsPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set")
	}
	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	testBackfillEmptyTopicExcerpts(t, conn)
}

func testBackfillEmptyTopicExcerpts(t *testing.T, conn *gorm.DB) {
	t.Helper()
	if err := conn.AutoMigrate(&topics.Entity{}, &posts.Entity{}); err != nil {
		t.Fatal(err)
	}
	stamp := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	const base = uint64(13600000)
	cases := []struct {
		content, excerpt, want string
		status, process        int8
		visibility             string
	}{
		{"失眠咯", "", "失眠咯", 1, 0, topics.VisibilityActive},
		{"![pic](/image.png)\n\n绷（）", "", "绷（）", 1, 0, topics.VisibilityActive},
		{"![pic](/image.png)", "", "", 1, 0, topics.VisibilityActive},
		{"new text", "keep existing", "keep existing", 1, 0, topics.VisibilityActive},
		{"draft", "", "", 0, 0, topics.VisibilityActive},
		{"pending", "", "", 1, topics.ProcessStatusPending, topics.VisibilityActive},
		{"hidden", "", "", 1, 0, topics.VisibilityUserDeleted},
	}
	t.Cleanup(func() {
		conn.Unscoped().Where("id > ? AND id <= ?", base, base+507).Delete(&posts.Entity{})
		conn.Unscoped().Where("id > ? AND id <= ?", base, base+507).Delete(&topics.Entity{})
	})
	// More than one batch, with a media-only row that must not stall the cursor.
	for i := 0; i < 507; i++ {
		tc := cases[i%len(cases)]
		id := base + uint64(i) + 1
		topic := topics.Entity{Id: id, FirstPostId: id, Status: tc.status, ProcessStatus: tc.process, VisibilityStatus: tc.visibility, Excerpt: tc.excerpt, UpdatedAt: stamp}
		post := posts.Entity{Id: id, TopicId: id, PostNo: 1, Content: tc.content, ProcessStatus: posts.ProcessStatusNormal, VisibilityStatus: posts.VisibilityActive, LatestRevisionId: 999, PublishedRevisionId: 1}
		if err := conn.Create(&topic).Error; err != nil {
			t.Fatal(err)
		}
		if err := conn.Create(&post).Error; err != nil {
			t.Fatal(err)
		}
	}
	for attempt := 0; attempt < 2; attempt++ {
		if err := BackfillEmptyTopicExcerpts(conn); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 507; i++ {
			var got topics.Entity
			if err := conn.First(&got, base+uint64(i)+1).Error; err != nil {
				t.Fatal(err)
			}
			if got.Excerpt != cases[i%len(cases)].want || !got.UpdatedAt.Equal(stamp) {
				t.Fatalf("attempt %d row %d: excerpt=%q updated=%s", attempt, i, got.Excerpt, got.UpdatedAt)
			}
		}
	}
}

func TestBackfillEmptyTopicExcerptsReturnsScanError(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := BackfillEmptyTopicExcerpts(conn); err == nil {
		t.Fatal("missing schema was silently accepted")
	}
}

package datamigration

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestBackfillLegacyPostsContentTypeArticle(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := conn.AutoMigrate(&posts.Entity{}); err != nil {
		t.Fatalf("migrate posts: %v", err)
	}

	legacyRegularPost := posts.Entity{Id: 1, TopicId: 1, PostNo: 1, ContentType: posts.ContentTypeRegular}
	questionPost := posts.Entity{Id: 2, TopicId: 2, PostNo: 1, ContentType: posts.ContentTypeQuestion}
	thoughtPost := posts.Entity{Id: 3, TopicId: 3, PostNo: 1, ContentType: posts.ContentTypeThought}
	articlePost := posts.Entity{Id: 4, TopicId: 4, PostNo: 1, ContentType: posts.ContentTypeArticle}

	if err := conn.Create(&[]posts.Entity{legacyRegularPost, questionPost, thoughtPost, articlePost}).Error; err != nil {
		t.Fatalf("create posts: %v", err)
	}

	result := BackfillLegacyPostsContentTypeArticleWithDB(conn)
	if result.Failed != 0 || result.Updated != 1 {
		t.Fatalf("unexpected migration result: %#v", result)
	}

	var got []posts.Entity
	if err := conn.Order("id").Find(&got).Error; err != nil {
		t.Fatalf("load posts: %v", err)
	}

	if got[0].ContentType != posts.ContentTypeArticle {
		t.Errorf("expected legacy post 1 to be updated to ContentTypeArticle, got %d", got[0].ContentType)
	}
	if got[1].ContentType != posts.ContentTypeQuestion {
		t.Errorf("expected question post 2 to remain unchanged, got %d", got[1].ContentType)
	}
	if got[2].ContentType != posts.ContentTypeThought {
		t.Errorf("expected thought post 3 to remain unchanged, got %d", got[2].ContentType)
	}
	if got[3].ContentType != posts.ContentTypeArticle {
		t.Errorf("expected article post 4 to remain unchanged, got %d", got[3].ContentType)
	}

	// 幂等性测试：再次执行应 0 更新
	secondResult := BackfillLegacyPostsContentTypeArticleWithDB(conn)
	if secondResult.Updated != 0 || secondResult.Failed != 0 {
		t.Fatalf("expected 0 updates on second run, got %#v", secondResult)
	}
}

package contentdeleteservice

import (
	"fmt"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pointsRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userPoints"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/pointservice"
	"gorm.io/gorm"
)

func assertRestorePoints(t *testing.T, conn *gorm.DB, author uint64, want int64) {
	t.Helper()
	var balance userPoints.Entity
	if err := conn.Where("user_id = ?", author).Take(&balance).Error; err != nil {
		t.Fatal(err)
	}
	var user users.EntityComplete
	if err := conn.Where("id = ?", author).Take(&user).Error; err != nil {
		t.Fatal(err)
	}
	if balance.CurrentPoints != 100+want || user.Prestige != want {
		t.Fatalf("wallet=%d prestige=%d; want wallet=%d prestige=%d", balance.CurrentPoints, user.Prestige, 100+want, want)
	}
}

func TestRestorePreviouslyRewardedReply(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	const topicID uint64 = 950900
	_, author := seedTopicWithOptionalReply(t, conn, topicID, true)
	const replyID = topicID + 200
	seedReplyReward(t, conn, author, replyID)
	assertRestorePoints(t, conn, author, pointservice.PostCreatedReward)
	if _, err := DeletePostByUser(author, replyID); err != nil {
		t.Fatal(err)
	}
	assertRestorePoints(t, conn, author, 0)
	if err := RestoreContent(author, ContentTypePost, replyID); err != nil {
		t.Fatal(err)
	}
	assertPostRemainsActive(t, conn, replyID)
	assertRestorePoints(t, conn, author, pointservice.PostCreatedReward)
	assertReplyRewardUnchanged(t, conn, author, replyID)
	if err := RestoreContent(author, ContentTypePost, replyID); err == nil {
		t.Fatal("already-restored reply was eligible for restoration")
	}
	assertRestorePoints(t, conn, author, pointservice.PostCreatedReward)
	if _, err := DeletePostByUser(author, replyID); err != nil {
		t.Fatal(err)
	}
	assertRestorePoints(t, conn, author, 0)
	installSQLiteTrigger(t, conn, "fail_restore_reply_prestige", fmt.Sprintf(
		"CREATE TRIGGER fail_restore_reply_prestige BEFORE UPDATE OF prestige ON users "+
			"WHEN OLD.id = %d BEGIN SELECT RAISE(ABORT, 'forced prestige failure'); END", author))
	if err := RestoreContent(author, ContentTypePost, replyID); err == nil {
		t.Fatal("restore succeeded despite a failed prestige write")
	}
	assertRestorePoints(t, conn, author, 0)
	var post posts.Entity
	if err := conn.Unscoped().Where("id = ?", replyID).Take(&post).Error; err != nil {
		t.Fatal(err)
	}
	if post.VisibilityStatus != posts.VisibilityUserDeleted || post.RetentionStatus != posts.RetentionRecoverable {
		t.Fatalf("failed restore changed reply state: %s/%s", post.VisibilityStatus, post.RetentionStatus)
	}
	var reversal pointsRecord.Entity
	if err := conn.Where("source_key = ?", fmt.Sprintf("post-deleted:%d", replyID)).Take(&reversal).Error; err != nil {
		t.Fatal(err)
	}
	if reversal.PointsChange != -pointservice.PostCreatedReward {
		t.Fatalf("failed restore changed reversal: %d", reversal.PointsChange)
	}
}

func TestRestoreReplyWithoutPriorRewardDoesNotCreditPoints(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	const topicID uint64 = 951200
	_, author := seedTopicWithOptionalReply(t, conn, topicID, true)
	const replyID = topicID + 200
	if err := conn.Create(&userPoints.Entity{UserId: author, CurrentPoints: 100}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := DeletePostByUser(author, replyID); err != nil {
		t.Fatal(err)
	}
	if err := RestoreContent(author, ContentTypePost, replyID); err != nil {
		t.Fatal(err)
	}
	assertPostRemainsActive(t, conn, replyID)
	assertRestorePoints(t, conn, author, 0)
}

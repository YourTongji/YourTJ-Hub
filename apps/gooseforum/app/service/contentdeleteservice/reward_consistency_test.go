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

func TestDeletePostByUserRollsBackRewardWhenContentDeleteFails(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	const topicID uint64 = 950100
	_, replyAuthorID := seedTopicWithOptionalReply(t, conn, topicID, true)
	const postID = topicID + 200
	seedReplyReward(t, conn, replyAuthorID, postID)

	installSQLiteTrigger(t, conn, "fail_user_post_delete", fmt.Sprintf(
		"CREATE TRIGGER fail_user_post_delete BEFORE UPDATE OF visibility_status ON posts "+
			"WHEN OLD.id = %d AND NEW.visibility_status = '%s' BEGIN "+
			"SELECT RAISE(ABORT, 'forced user content delete failure'); END",
		postID, posts.VisibilityUserDeleted,
	))

	if _, err := DeletePostByUser(replyAuthorID, postID); err == nil {
		t.Fatal("DeletePostByUser succeeded despite forced content update failure")
	}
	assertPostRemainsActive(t, conn, postID)
	assertReplyRewardUnchanged(t, conn, replyAuthorID, postID)
}

func TestDeletePostAsModeratorRollsBackContentWhenRewardReversalFails(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	const topicID uint64 = 950500
	_, replyAuthorID := seedTopicWithOptionalReply(t, conn, topicID, true)
	const postID = topicID + 200
	seedReplyReward(t, conn, replyAuthorID, postID)

	installSQLiteTrigger(t, conn, "fail_moderator_reward_reversal", fmt.Sprintf(
		"CREATE TRIGGER fail_moderator_reward_reversal BEFORE INSERT ON points_record "+
			"WHEN NEW.source_key = 'post-deleted:%d' BEGIN "+
			"SELECT RAISE(ABORT, 'forced reward reversal failure'); END",
		postID,
	))

	if err := DeletePostAsModerator(topicID+99, postID, "forced failure test"); err == nil {
		t.Fatal("DeletePostAsModerator succeeded despite forced reward reversal failure")
	}
	assertPostRemainsActive(t, conn, postID)
	assertReplyRewardUnchanged(t, conn, replyAuthorID, postID)
}

func installSQLiteTrigger(t *testing.T, conn *gorm.DB, name, statement string) {
	t.Helper()
	if err := conn.Exec(statement).Error; err != nil {
		t.Fatalf("install forced-failure trigger: %v", err)
	}
	t.Cleanup(func() {
		if err := conn.Exec("DROP TRIGGER IF EXISTS " + name).Error; err != nil {
			t.Errorf("drop forced-failure trigger %s: %v", name, err)
		}
	})
}

func seedReplyReward(t *testing.T, conn *gorm.DB, userID, postID uint64) {
	t.Helper()
	if err := conn.Model(&users.EntityComplete{}).Where("id = ?", userID).Update("prestige", pointservice.PostCreatedReward).Error; err != nil {
		t.Fatalf("seed user prestige: %v", err)
	}
	if err := conn.Create(&userPoints.Entity{UserId: userID, CurrentPoints: 100 + pointservice.PostCreatedReward}).Error; err != nil {
		t.Fatalf("seed user points: %v", err)
	}
	sourceKey := fmt.Sprintf("post:%d", postID)
	if err := conn.Create(&pointsRecord.Entity{
		UserId:       userID,
		Action:       pointservice.PointsActionPostCreated.Code(),
		PointsChange: pointservice.PostCreatedReward,
		SourceKey:    &sourceKey,
	}).Error; err != nil {
		t.Fatalf("seed post reward: %v", err)
	}
}

func assertPostRemainsActive(t *testing.T, conn *gorm.DB, postID uint64) {
	t.Helper()
	var post posts.Entity
	if err := conn.Unscoped().Where("id = ?", postID).Take(&post).Error; err != nil {
		t.Fatalf("load post after forced failure: %v", err)
	}
	if post.VisibilityStatus != posts.VisibilityActive || post.RetentionStatus != posts.RetentionNormal || post.DeletedAt.Valid {
		t.Fatalf("post state after forced failure = %s/%s deleted=%t, want ACTIVE/NORMAL/not deleted", post.VisibilityStatus, post.RetentionStatus, post.DeletedAt.Valid)
	}
}

func assertReplyRewardUnchanged(t *testing.T, conn *gorm.DB, userID, postID uint64) {
	t.Helper()
	var balance userPoints.Entity
	if err := conn.Where("user_id = ?", userID).Take(&balance).Error; err != nil {
		t.Fatalf("load user points after forced failure: %v", err)
	}
	if balance.CurrentPoints != 100+pointservice.PostCreatedReward {
		t.Errorf("current points after forced failure = %d, want %d", balance.CurrentPoints, 100+pointservice.PostCreatedReward)
	}
	var user users.EntityComplete
	if err := conn.Where("id = ?", userID).Take(&user).Error; err != nil {
		t.Fatalf("load user after forced failure: %v", err)
	}
	if user.Prestige != pointservice.PostCreatedReward {
		t.Errorf("prestige after forced failure = %d, want %d", user.Prestige, pointservice.PostCreatedReward)
	}
	var original pointsRecord.Entity
	if err := conn.Where("source_key = ?", fmt.Sprintf("post:%d", postID)).Take(&original).Error; err != nil {
		t.Fatalf("load original reward after forced failure: %v", err)
	}
	var reversalCount int64
	if err := conn.Model(&pointsRecord.Entity{}).Where("source_key = ?", fmt.Sprintf("post-deleted:%d", postID)).Count(&reversalCount).Error; err != nil {
		t.Fatalf("count reversal after forced failure: %v", err)
	}
	if reversalCount != 0 {
		t.Errorf("reversal records after forced failure = %d, want 0", reversalCount)
	}
}

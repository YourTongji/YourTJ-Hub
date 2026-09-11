package contentdeleteservice

import (
	"errors"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
)

// issue #553 review 修复：重复删除（并发窗口内第二个请求绕过控制器预检、
// 直达行锁分类，或批量/级联路径重入）必须报错退出而非幂等成功，
// 避免重复触发 ContentDeletedEvent/审计/审核日志副作用。

func TestDeletePostByUserTwiceReturnsAlreadyDeleted(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	const topicID = uint64(948300)
	_, replyAuthorID := seedTopicWithOptionalReply(t, conn, topicID, true)
	postID := topicID + 200

	if _, err := DeletePostByUser(replyAuthorID, postID); err != nil {
		t.Fatalf("first delete: %v", err)
	}
	_, err := DeletePostByUser(replyAuthorID, postID)
	var msgErr component.MessageError
	if !errors.As(err, &msgErr) || msgErr.Code != component.MessagePostAlreadyDeleted {
		t.Fatalf("second delete err = %v, want MessagePostAlreadyDeleted", err)
	}
}

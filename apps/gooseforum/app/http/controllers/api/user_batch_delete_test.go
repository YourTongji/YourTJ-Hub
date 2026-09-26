package api

import (
	"errors"
	"fmt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/campus"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/contentDeleteEvent"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationLog"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/optRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pushSubscription"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

func setupBatchDeleteTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(
		&users.EntityComplete{},
		&users.PrivateNoteEntity{},
		&topics.Entity{},
		&posts.Entity{},
		&optRecord.Entity{},
		&moderationLog.Entity{},
		&contentDeleteEvent.Entity{},
		&pushSubscription.Entity{},
		&pk.ScheduleSnapshotEntity{}, &pk.PlanSyncOwner{}, &pk.PlanItem{},
		&campus.Binding{}, &sticker.LibraryOwner{}, &sticker.LibraryEntry{},
	); err != nil {
		t.Fatalf("migrate batch delete tables: %v", err)
	}
	return conn
}

func seedBatchTopics(t *testing.T, conn *gorm.DB, userID uint64, startID uint64, count int) []uint64 {
	t.Helper()
	ids := make([]uint64, 0, count)
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	for i := 0; i < count; i++ {
		topicID := startID + uint64(i)
		postID := startID + uint64(i) + 10_000
		if err := conn.Create(&posts.Entity{Id: postID, TopicId: topicID, PostNo: 1, UserId: userID, Content: "body", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
			t.Fatalf("create first post: %v", err)
		}
		if err := conn.Create(&topics.Entity{
			Id:               topicID,
			Title:            fmt.Sprintf("Batch topic %d", topicID),
			UserId:           userID,
			Status:           1,
			ProcessStatus:    topics.ProcessStatusNormal,
			PostCount:        1,
			PostSeq:          1,
			FirstPostId:      postID,
			VisibilityStatus: topics.VisibilityActive,
			RetentionStatus:  topics.RetentionNormal,
			CreatedAt:        now,
			UpdatedAt:        now,
		}).Error; err != nil {
			t.Fatalf("create topic: %v", err)
		}
		ids = append(ids, topicID)
	}
	t.Cleanup(func() {
		conn.Unscoped().Where("user_id = ?", userID).Delete(&topics.Entity{})
		conn.Unscoped().Where("user_id = ?", userID).Delete(&posts.Entity{})
		conn.Unscoped().Where("actor_id = ?", userID).Delete(&contentDeleteEvent.Entity{})
	})
	return ids
}

// R9：批量删除逐条走删除流程并汇总结果。
func TestBatchDeleteContentDeletesAll(t *testing.T) {
	conn := setupBatchDeleteTestDB(t)
	userID := uint64(9_900_000_001)
	if err := conn.Create(&users.EntityComplete{Id: userID, Username: "batch_owner", IsActivated: 1}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	ids := seedBatchTopics(t, conn, userID, 9_900_000_100, 3)

	res := BatchDeleteContent(component.BetterRequest[BatchDeleteContentReq]{
		UserId: userID,
		Params: BatchDeleteContentReq{ContentType: "topic", ContentIDs: ids},
	})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("BatchDeleteContent failed: %#v", res)
	}
	data, ok := res.Data.Result.(map[string]any)
	if !ok {
		t.Fatalf("result type = %T", res.Data.Result)
	}
	if data["failed"] != 0 || data["succeeded"] != len(ids) {
		t.Fatalf("batch result = %#v", data)
	}
	for _, id := range ids {
		if visible := topics.Get(id); visible.Id != 0 {
			t.Fatalf("topic %d still visible after batch delete", id)
		}
	}
}

// R9：删除频率超限时要求二次确认，force=true 且密码正确后才放行。
func TestBatchDeleteContentRequiresConfirmOverThreshold(t *testing.T) {
	conn := setupBatchDeleteTestDB(t)
	const userPassword = "batch-confirm-password"
	user := users.MakeUser("batch_confirm", userPassword, "batch-confirm@example.com")
	user.Activate()
	if err := conn.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	userID := user.Id
	// 模拟历史已删除 20 条：写入 20 条 content_deleted 事件。
	for i := 0; i < 20; i++ {
		if err := contentDeleteEvent.Record(contentDeleteEvent.Entity{
			EventType:   string(contentDeleteEvent.EventDeleted),
			ContentType: "topic",
			ContentID:   uint64(9_900_001_000 + i),
			ActorID:     userID,
		}); err != nil {
			t.Fatalf("seed delete event: %v", err)
		}
	}
	ids := seedBatchTopics(t, conn, userID, 9_900_000_300, 2)

	// 未 force：应要求二次确认。
	noForce := BatchDeleteContent(component.BetterRequest[BatchDeleteContentReq]{
		UserId: userID,
		Params: BatchDeleteContentReq{ContentType: "topic", ContentIDs: ids},
	})
	if noForce.Data.Code == component.SUCCESS {
		t.Fatalf("expected confirm-required failure, got %#v", noForce)
	}
	if noForce.Data.MessageCode != component.MessageContentBatchConfirmRequired {
		t.Fatalf("messageCode = %s, want %s", noForce.Data.MessageCode, component.MessageContentBatchConfirmRequired)
	}
	for _, id := range ids {
		if visible := topics.Get(id); visible.Id == 0 {
			t.Fatalf("topic %d should not be deleted without force confirm", id)
		}
	}

	// force=true 但密码错误：拒绝，不清空内容（防止被盗会话无脑清空）。
	wrongPassword := BatchDeleteContent(component.BetterRequest[BatchDeleteContentReq]{
		UserId: userID,
		Params: BatchDeleteContentReq{ContentType: "topic", ContentIDs: ids, Force: true, Password: "wrong-password"},
	})
	if wrongPassword.Data.Code == component.SUCCESS {
		t.Fatalf("expected password-rejected failure, got %#v", wrongPassword)
	}
	if wrongPassword.Data.MessageCode != component.MessageAuthInvalidCredentials {
		t.Fatalf("messageCode = %s, want %s", wrongPassword.Data.MessageCode, component.MessageAuthInvalidCredentials)
	}
	for _, id := range ids {
		if visible := topics.Get(id); visible.Id == 0 {
			t.Fatalf("topic %d should not be deleted with wrong password", id)
		}
	}

	// force=true 且密码正确：放行。
	forced := BatchDeleteContent(component.BetterRequest[BatchDeleteContentReq]{
		UserId: userID,
		Params: BatchDeleteContentReq{ContentType: "topic", ContentIDs: ids, Force: true, Password: userPassword},
	})
	if forced.Data.Code != component.SUCCESS {
		t.Fatalf("forced batch delete failed: %#v", forced)
	}
	for _, id := range ids {
		if visible := topics.Get(id); visible.Id != 0 {
			t.Fatalf("topic %d still visible after forced batch delete", id)
		}
	}
}

// R10：注销账号 anonymize 模式保留内容但用户不可见。
func TestAccountCloseAnonymizeKeepsContent(t *testing.T) {
	conn := setupBatchDeleteTestDB(t)
	const userPassword = "close-anonym-password"
	user := users.MakeUser("close_anonym", userPassword, "close-anonym@example.com")
	user.Activate()
	if err := conn.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	userID := user.Id
	if err := conn.Create(&campus.Binding{UserID: userID, IdentityKey: "fictional-school-identity", Revision: "demo", Sealed: "encrypted-fixture"}).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		var count int64
		if err := conn.Model(&campus.Binding{}).Where("user_id = ?", userID).Count(&count).Error; err != nil || count != 0 {
			t.Errorf("campus credential retained after account closure: %v", err)
		}
	})

	ids := seedBatchTopics(t, conn, userID, 9_900_000_500, 2)
	if err := pk.UpsertScheduleSnapshot(&pk.ScheduleSnapshotEntity{UserId: userID, Plans: pk.PlanList{{Id: "p", Name: "private"}}, ActivePlanId: "p"}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := pk.GetScheduleSnapshotByUser(userID); !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Errorf("closed user's cloud snapshot remains: %v", err)
		}
	})

	res := AccountClose(component.BetterRequest[AccountCloseReq]{
		UserId: userID,
		Params: AccountCloseReq{Mode: "anonymize", Password: userPassword},
	})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("AccountClose anonymize failed: %#v", res)
	}

	// 用户应已软删（scoped Get 查不到）。
	if _, err := users.Get(userID); err == nil {
		t.Fatal("closed account should not be retrievable via scoped Get")
	}
	if !users.IsAccountClosed(userID) {
		t.Fatal("expected IsAccountClosed to be true")
	}
	// 内容保留且仍公开（ACTIVE）。
	for _, id := range ids {
		if visible := topics.Get(id); visible.Id == 0 {
			t.Fatalf("topic %d should be kept after anonymize close", id)
		}
	}
}

// R10：注销账号 delete 模式先删除全部内容再注销。
func TestAccountCloseDeleteRemovesContent(t *testing.T) {
	conn := setupBatchDeleteTestDB(t)
	const userPassword = "close-delete-password"
	user := users.MakeUser("close_delete", userPassword, "close-delete@example.com")
	user.Activate()
	if err := conn.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	userID := user.Id
	ids := seedBatchTopics(t, conn, userID, 9_900_000_600, 2)
	if err := pk.UpsertScheduleSnapshot(&pk.ScheduleSnapshotEntity{UserId: userID, Plans: pk.PlanList{{Id: "p", Name: "private"}}, ActivePlanId: "p"}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := pk.GetScheduleSnapshotByUser(userID); !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Errorf("closed user's cloud snapshot remains: %v", err)
		}
	})

	res := AccountClose(component.BetterRequest[AccountCloseReq]{
		UserId: userID,
		Params: AccountCloseReq{Mode: "delete", Password: userPassword},
	})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("AccountClose delete failed: %#v", res)
	}
	for _, id := range ids {
		if visible := topics.Get(id); visible.Id != 0 {
			t.Fatalf("topic %d should be deleted after delete-mode close", id)
		}
	}
	if !users.IsAccountClosed(userID) {
		t.Fatal("expected account to be closed after delete-mode close")
	}
}

// R10：注销账号必须提供正确密码；密码错误时拒绝且不产生任何副作用。
func TestAccountCloseRejectsWrongPassword(t *testing.T) {
	conn := setupBatchDeleteTestDB(t)
	const userPassword = "close-wrong-password"
	user := users.MakeUser("close_wrong", userPassword, "close-wrong@example.com")
	user.Activate()
	if err := conn.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	userID := user.Id
	ids := seedBatchTopics(t, conn, userID, 9_900_000_700, 2)

	res := AccountClose(component.BetterRequest[AccountCloseReq]{
		UserId: userID,
		Params: AccountCloseReq{Mode: "delete", Password: "wrong-password"},
	})
	if res.Data.Code == component.SUCCESS {
		t.Fatalf("expected password-rejected failure, got %#v", res)
	}
	if res.Data.MessageCode != component.MessageAuthInvalidCredentials {
		t.Fatalf("messageCode = %s, want %s", res.Data.MessageCode, component.MessageAuthInvalidCredentials)
	}
	// 账号未注销、内容未被删除。
	if users.IsAccountClosed(userID) {
		t.Fatal("account should not be closed with wrong password")
	}
	for _, id := range ids {
		if visible := topics.Get(id); visible.Id == 0 {
			t.Fatalf("topic %d should not be deleted with wrong password", id)
		}
	}
}

// A failed account close must not irreversibly erase or fence a still-active
// account's private library. The PK prerequisite keeps its existing ordering.
func TestAccountCloseFailureRollsBackStickerLibrary(t *testing.T) {
	for _, stage := range []string{"pk", "account"} {
		t.Run(stage, func(t *testing.T) {
			conn := setupBatchDeleteTestDB(t)
			password := "close-sticker-rollback"
			user := users.MakeUser("close_sticker_"+stage, password, "close-sticker-"+stage+"@example.com")
			user.Activate()
			if err := conn.Create(user).Error; err != nil {
				t.Fatal(err)
			}
			owner := sticker.LibraryOwner{UserID: user.Id, Revision: 7}
			entry := sticker.LibraryEntry{UserID: user.Id, StickerID: 7654321, DisplayName: "private label", SortOrder: 3}
			if err := conn.Create(&owner).Error; err != nil {
				t.Fatal(err)
			}
			if err := conn.Create(&entry).Error; err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				conn.Where("user_id = ?", user.Id).Delete(&sticker.LibraryEntry{})
				conn.Where("user_id = ?", user.Id).Delete(&sticker.LibraryOwner{})
				conn.Unscoped().Delete(user)
			})
			failedTable := users.PrivateNoteEntity{}.TableName()
			if stage == "pk" {
				failedTable = (&pk.ScheduleSnapshotEntity{}).TableName()
			}
			callback := "fail_sticker_close_" + stage
			if err := conn.Callback().Delete().Before("gorm:delete").Register(callback, func(tx *gorm.DB) {
				if tx.Statement.Table == failedTable {
					tx.AddError(errors.New("forced account-close prerequisite failure"))
				}
			}); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = conn.Callback().Delete().Remove(callback) })
			request := component.BetterRequest[AccountCloseReq]{UserId: user.Id, Params: AccountCloseReq{Mode: "anonymize", Password: password}}
			response := AccountClose(request)
			if response.Data.Code == component.SUCCESS {
				t.Fatal("forced close failure was accepted")
			}
			if users.IsAccountClosed(user.Id) {
				t.Fatal("failed account close committed user deletion")
			}
			var retained sticker.LibraryEntry
			if err := conn.First(&retained, "user_id = ? AND sticker_id = ?", user.Id, entry.StickerID).Error; err != nil {
				t.Fatalf("failed close erased membership: %v", err)
			}
			if retained.DisplayName != entry.DisplayName || retained.SortOrder != entry.SortOrder {
				t.Fatal("failed close changed private library metadata")
			}
			if err := conn.First(&owner, "user_id = ?", user.Id).Error; err != nil {
				t.Fatal(err)
			}
			if owner.Closed || owner.Revision != 7 {
				t.Fatalf("failed close committed library fence: %#v", owner)
			}
			if err := conn.Callback().Delete().Remove(callback); err != nil {
				t.Fatal(err)
			}
			response = AccountClose(request)
			if response.Data.Code != component.SUCCESS {
				t.Fatalf("retry failed: %#v", response)
			}
			var count int64
			conn.Model(&sticker.LibraryEntry{}).Where("user_id = ?", user.Id).Count(&count)
			if count != 0 {
				t.Fatal("successful close retained members")
			}
			if err := conn.First(&owner, "user_id = ?", user.Id).Error; err != nil || !owner.Closed {
				t.Fatalf("successful close did not fence old writes: %v", err)
			}
		})
	}
}

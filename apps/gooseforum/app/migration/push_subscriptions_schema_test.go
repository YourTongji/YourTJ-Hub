package migration

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pushSubscription"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestSchemaModelsRegistersPushSubscriptions 验证 SchemaModels 已注册
// pushSubscription 模型（生产迁移入口依赖该注册，注册丢失则表永远不会创建）。
func TestSchemaModelsRegistersPushSubscriptions(t *testing.T) {
	for _, model := range SchemaModels() {
		if _, ok := model.(*pushSubscription.Entity); ok {
			return
		}
	}
	t.Fatal("SchemaModels() does not include *pushSubscription.Entity")
}

// TestPushSubscriptionsSchemaCreatedOnSQLite 验证全新库上 push_subscriptions
// 表可创建，结构完整（endpoint 唯一、凭据列非空）且可正常写入/查询。
func TestPushSubscriptionsSchemaCreatedOnSQLite(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open("file:migration-push-sub-create?mode=memory&cache=shared"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := conn.AutoMigrate(&pushSubscription.Entity{}); err != nil {
		t.Fatalf("AutoMigrate push_subscriptions: %v", err)
	}
	if !conn.Migrator().HasTable("push_subscriptions") {
		t.Fatal("push_subscriptions table missing after AutoMigrate")
	}
	for _, column := range []string{"id", "user_id", "endpoint", "p256dh", "auth", "lang", "created_at", "updated_at", "last_active_at"} {
		if !conn.Migrator().HasColumn(&pushSubscription.Entity{}, column) {
			t.Errorf("push_subscriptions column %q missing after AutoMigrate", column)
		}
	}

	// 写入/查询：一用户多订阅，endpoint 唯一约束生效。
	if err := conn.Create(&pushSubscription.Entity{UserId: 1, Endpoint: "https://push.example/a", P256dh: "k1", Auth: "s1", Lang: "zh"}).Error; err != nil {
		t.Fatalf("insert first subscription: %v", err)
	}
	if err := conn.Create(&pushSubscription.Entity{UserId: 1, Endpoint: "https://push.example/b", P256dh: "k2", Auth: "s2", Lang: "en"}).Error; err != nil {
		t.Fatalf("insert second subscription: %v", err)
	}
	dup := pushSubscription.Entity{UserId: 2, Endpoint: "https://push.example/a", P256dh: "k3", Auth: "s3", Lang: "zh"}
	if err := conn.Create(&dup).Error; err == nil {
		t.Fatal("duplicate endpoint insert succeeded, want unique constraint error")
	}
	var count int64
	if err := conn.Model(&pushSubscription.Entity{}).Where("user_id = ?", 1).Count(&count).Error; err != nil {
		t.Fatalf("count subscriptions: %v", err)
	}
	if count != 2 {
		t.Fatalf("subscription count = %d, want 2", count)
	}
}

// TestPushSubscriptionsSchemaUpgradeFromLegacySubset 模拟存量实例升级：旧库没有
// push_subscriptions 表，AutoMigrate 必须自动补齐新表且不破坏旧表与存量数据。
func TestPushSubscriptionsSchemaUpgradeFromLegacySubset(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open("file:migration-push-sub-upgrade?mode=memory&cache=shared"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// 旧库：只有 users + topics（功能上线前的形态），并插入存量数据
	if err := conn.AutoMigrate(&users.EntityComplete{}, &topics.Entity{}); err != nil {
		t.Fatalf("AutoMigrate legacy subset: %v", err)
	}
	if err := conn.Create(&users.EntityComplete{Username: "legacy-user", Nickname: "legacy"}).Error; err != nil {
		t.Fatalf("insert legacy user: %v", err)
	}
	if conn.Migrator().HasTable("push_subscriptions") {
		t.Fatal("precondition failed: legacy schema should not have push_subscriptions")
	}

	// 部署新二进制：AutoMigrate 补齐 push_subscriptions
	if err := conn.AutoMigrate(&pushSubscription.Entity{}); err != nil {
		t.Fatalf("upgrade AutoMigrate push_subscriptions: %v", err)
	}
	if !conn.Migrator().HasTable("push_subscriptions") {
		t.Fatal("push_subscriptions table missing after upgrade")
	}
	// 旧表数据保留
	var user users.EntityComplete
	if err := conn.First(&user).Error; err != nil || user.Username != "legacy-user" {
		t.Fatalf("legacy user lost after upgrade: %+v err=%v", user, err)
	}
	// 新表可用
	if err := conn.Create(&pushSubscription.Entity{UserId: user.Id, Endpoint: "https://push.example/post-upgrade", P256dh: "k", Auth: "s", Lang: "zh"}).Error; err != nil {
		t.Fatalf("insert subscription after upgrade: %v", err)
	}
}

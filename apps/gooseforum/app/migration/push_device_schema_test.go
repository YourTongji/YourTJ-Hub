package migration

import (
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pushDevice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestSchemaModelsRegistersPushDevice 验证 SchemaModels 已注册 pushDevice
// 模型（生产迁移入口依赖该注册，注册丢失则表永远不会创建）。
func TestSchemaModelsRegistersPushDevice(t *testing.T) {
	for _, model := range SchemaModels() {
		if _, ok := model.(*pushDevice.Entity); ok {
			return
		}
	}
	t.Fatal("SchemaModels() does not include *pushDevice.Entity")
}

// TestPushDeviceSchemaCreatedOnSQLite 验证全新库上 push_device 表可创建，
// 结构完整（token 唯一、user_id/platform 列非空）且可正常写入/查询。
func TestPushDeviceSchemaCreatedOnSQLite(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open("file:migration-push-device-create?mode=memory&cache=shared"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := conn.AutoMigrate(&pushDevice.Entity{}); err != nil {
		t.Fatalf("AutoMigrate push_device: %v", err)
	}
	if !conn.Migrator().HasTable("push_device") {
		t.Fatal("push_device table missing after AutoMigrate")
	}
	for _, column := range []string{"id", "user_id", "platform", "token", "created_at", "last_registered_at", "provider"} {
		if !conn.Migrator().HasColumn(&pushDevice.Entity{}, column) {
			t.Errorf("push_device column %q missing after AutoMigrate", column)
		}
	}

	// 写入/查询：一用户多设备，token 唯一约束生效。
	now := time.Now()
	if err := conn.Create(&pushDevice.Entity{UserId: 1, Platform: "ios", Token: "token-a", LastRegisteredAt: now}).Error; err != nil {
		t.Fatalf("insert first device: %v", err)
	}
	if err := conn.Create(&pushDevice.Entity{UserId: 1, Platform: "android", Token: "token-b", LastRegisteredAt: now}).Error; err != nil {
		t.Fatalf("insert second device: %v", err)
	}
	dup := pushDevice.Entity{UserId: 2, Platform: "ios", Token: "token-a", LastRegisteredAt: now}
	if err := conn.Create(&dup).Error; err == nil {
		t.Fatal("duplicate token insert succeeded, want unique constraint error")
	}
	var count int64
	if err := conn.Model(&pushDevice.Entity{}).Where("user_id = ?", 1).Count(&count).Error; err != nil {
		t.Fatalf("count devices: %v", err)
	}
	if count != 2 {
		t.Fatalf("device count = %d, want 2", count)
	}
}

// TestPushDeviceSchemaUpgradeFromLegacySubset 模拟存量实例升级：旧库没有
// push_device 表，AutoMigrate 必须自动补齐新表且不破坏旧表与存量数据。
func TestPushDeviceSchemaUpgradeFromLegacySubset(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open("file:migration-push-device-upgrade?mode=memory&cache=shared"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// 旧库：只有 users（原生推送上线前的形态），并插入存量数据
	if err := conn.AutoMigrate(&users.EntityComplete{}); err != nil {
		t.Fatalf("AutoMigrate legacy subset: %v", err)
	}
	if err := conn.Create(&users.EntityComplete{Username: "legacy-user", Nickname: "legacy"}).Error; err != nil {
		t.Fatalf("insert legacy user: %v", err)
	}
	if conn.Migrator().HasTable("push_device") {
		t.Fatal("precondition failed: legacy schema should not have push_device")
	}

	// 部署新二进制：AutoMigrate 补齐 push_device
	if err := conn.AutoMigrate(&pushDevice.Entity{}); err != nil {
		t.Fatalf("upgrade AutoMigrate push_device: %v", err)
	}
	if !conn.Migrator().HasTable("push_device") {
		t.Fatal("push_device table missing after upgrade")
	}
	// 旧表数据保留
	var user users.EntityComplete
	if err := conn.First(&user).Error; err != nil || user.Username != "legacy-user" {
		t.Fatalf("legacy user lost after upgrade: %+v err=%v", user, err)
	}
	// 新表可用
	if err := conn.Create(&pushDevice.Entity{UserId: user.Id, Platform: "ios", Token: "post-upgrade", LastRegisteredAt: time.Now()}).Error; err != nil {
		t.Fatalf("insert device after upgrade: %v", err)
	}
}

// Existing APNs/FCM rows must survive adding a non-null provider discriminator.
func TestPushDeviceProviderUpgrade(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open("file:push-provider-upgrade?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Exec(`CREATE TABLE push_device (id integer PRIMARY KEY, user_id integer NOT NULL,
        platform varchar(16) NOT NULL, token varchar(512) NOT NULL UNIQUE,
        created_at datetime, last_registered_at datetime)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Exec(`INSERT INTO push_device (id,user_id,platform,token) VALUES
        (1,1,'ios','old-apns'), (2,2,'android','old-fcm')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.AutoMigrate(&pushDevice.Entity{}); err != nil {
		t.Fatal(err)
	}
	var devices []pushDevice.Entity
	if err := conn.Order("id").Find(&devices).Error; err != nil {
		t.Fatal(err)
	}
	if len(devices) != 2 || devices[0].Provider != "" || devices[0].DeliveryProvider() != "apns" || devices[1].DeliveryProvider() != "fcm" {
		t.Fatal("legacy push ownership or provider routing lost")
	}
}

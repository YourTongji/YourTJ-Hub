package sqlconnect_test

import (
	"os"
	"testing"

	"github.com/leancodebox/GooseForum/app/bundles/connect/sqlconnect"
	"github.com/leancodebox/GooseForum/app/models/chat/imConversations"
	"github.com/leancodebox/GooseForum/app/models/chat/imUserChatConfigs"
	"github.com/leancodebox/GooseForum/app/models/chat/messages"
	"github.com/leancodebox/GooseForum/app/models/forum/badges"
	"github.com/leancodebox/GooseForum/app/models/forum/category"
	"github.com/leancodebox/GooseForum/app/models/forum/dailyStats"
	"github.com/leancodebox/GooseForum/app/models/forum/eventNotification"
	"github.com/leancodebox/GooseForum/app/models/forum/fileUsage"
	"github.com/leancodebox/GooseForum/app/models/forum/migrationMapping"
	"github.com/leancodebox/GooseForum/app/models/forum/moderationLog"
	"github.com/leancodebox/GooseForum/app/models/forum/moderators"
	"github.com/leancodebox/GooseForum/app/models/forum/optRecord"
	"github.com/leancodebox/GooseForum/app/models/forum/pageConfig"
	"github.com/leancodebox/GooseForum/app/models/forum/pointsRecord"
	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/reports"
	"github.com/leancodebox/GooseForum/app/models/forum/role"
	"github.com/leancodebox/GooseForum/app/models/forum/rolePermissionRs"
	"github.com/leancodebox/GooseForum/app/models/forum/taskQueue"
	"github.com/leancodebox/GooseForum/app/models/forum/topicCategoryIndex"
	"github.com/leancodebox/GooseForum/app/models/forum/topicUserAction"
	"github.com/leancodebox/GooseForum/app/models/forum/topicUserStat"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/leancodebox/GooseForum/app/models/forum/userActivities"
	"github.com/leancodebox/GooseForum/app/models/forum/userBadges"
	"github.com/leancodebox/GooseForum/app/models/forum/userFollow"
	"github.com/leancodebox/GooseForum/app/models/forum/userOAuth"
	"github.com/leancodebox/GooseForum/app/models/forum/userPoints"
	"github.com/leancodebox/GooseForum/app/models/forum/userStatistics"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
)

// pgDSN 返回测试用 PostgreSQL DSN；未设置 TEST_PG_DSN 时跳过（CI 默认不跑）。
func pgDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set; skipping PostgreSQL integration test")
	}
	return dsn
}

// TestPostgresConnectAutoMigrate 验证 postgres 连接 + 全部主库模型 AutoMigrate 建表成功。
func TestPostgresConnectAutoMigrate(t *testing.T) {
	dsn := pgDSN(t)

	conn := sqlconnect.GetConnect(sqlconnect.Config{
		Connection:         "postgres",
		DbUrl:              dsn,
		MaxIdleConnections: 1,
		MaxOpenConnections: 2,
		MaxLifeSeconds:     60,
	})
	if conn.Error != nil {
		t.Fatalf("GetConnect(postgres) error: %v", conn.Error)
	}
	if conn.Connect == nil {
		t.Fatal("GetConnect(postgres) returned nil gorm.DB")
	}
	if conn.IsSqlite() {
		t.Fatal("postgres connection reported IsSqlite()=true")
	}
	sqlDB, err := conn.Connect.DB()
	if err != nil {
		t.Fatalf("sqlDB: %v", err)
	}
	defer sqlDB.Close()

	// 与 app/migration migrateSchema 相同的主库实体列表
	entities := []any{
		&badges.Entity{},
		&eventNotification.Entity{},
		&fileUsage.Entity{},
		&moderationLog.Entity{},
		&migrationMapping.Entity{},
		&moderators.Entity{},
		&optRecord.Entity{},
		&pageConfig.Entity{},
		&pointsRecord.Entity{},
		&reports.Entity{},
		&topics.Entity{},
		&posts.Entity{},
		&category.Entity{},
		&topicCategoryIndex.Entity{},
		&topicUserAction.Entity{},
		&topicUserStat.Entity{},
		&role.Entity{},
		&rolePermissionRs.Entity{},
		&taskQueue.Entity{},
		&userFollow.Entity{},
		&userBadges.Entity{},
		&userOAuth.Entity{},
		&userPoints.Entity{},
		&users.EntityComplete{},
		&userStatistics.Entity{},
		&imConversations.Entity{},
		&imUserChatConfigs.Entity{},
		&messages.Entity{},
		&dailyStats.Entity{},
		&userActivities.Entity{},
	}
	if err := conn.Connect.AutoMigrate(entities...); err != nil {
		t.Fatalf("AutoMigrate all models on postgres failed: %v", err)
	}

	// 断言每个实体对应的表存在（gorm 按模型的 TableName() 解析表名）
	for _, entity := range entities {
		if !conn.Connect.Migrator().HasTable(entity) {
			t.Errorf("table for %T missing after AutoMigrate", entity)
		}
	}
}

// TestPostgresTopicCRUD 验证 uint64 主键自增、serializer:json 字段与时间字段在 PG 上读写正常。
func TestPostgresTopicCRUD(t *testing.T) {
	dsn := pgDSN(t)

	conn := sqlconnect.GetConnect(sqlconnect.Config{
		Connection:         "postgres",
		DbUrl:              dsn,
		MaxIdleConnections: 1,
		MaxOpenConnections: 2,
		MaxLifeSeconds:     60,
	})
	sqlDB, err := conn.Connect.DB()
	if err != nil {
		t.Fatalf("sqlDB: %v", err)
	}
	defer sqlDB.Close()

	db := conn.Connect
	if err := db.AutoMigrate(&topics.Entity{}); err != nil {
		t.Fatalf("AutoMigrate topics: %v", err)
	}

	// 插入
	entity := topics.Entity{
		Title:         "PG integration topic",
		CategoryIds:   []uint64{1, 2, 3},
		Status:        1,
		ProcessStatus: 0,
		LikeCount:     10,
	}
	if err := db.Create(&entity).Error; err != nil {
		t.Fatalf("Create topic: %v", err)
	}
	if entity.Id == 0 {
		t.Fatal("topic id not populated after create (uint64 autoincrement broken)")
	}

	// 查询并校验 JSON serializer 字段
	var got topics.Entity
	if err := db.First(&got, entity.Id).Error; err != nil {
		t.Fatalf("First topic: %v", err)
	}
	if got.Title != "PG integration topic" {
		t.Fatalf("title = %q", got.Title)
	}
	if len(got.CategoryIds) != 3 || got.CategoryIds[0] != 1 || got.CategoryIds[2] != 3 {
		t.Fatalf("categoryIds = %v, want [1 2 3]", got.CategoryIds)
	}
	if got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Fatal("timestamps not populated")
	}
	// 默认值列
	if got.Status != 1 {
		t.Fatalf("status = %d", got.Status)
	}

	// 更新
	if err := db.Model(&got).Update("title", "PG integration topic updated").Error; err != nil {
		t.Fatalf("Update topic: %v", err)
	}

	// 软删除（DeletedAt 语义）
	if err := db.Delete(&topics.Entity{}, entity.Id).Error; err != nil {
		t.Fatalf("Delete topic: %v", err)
	}
	var count int64
	if err := db.Model(&topics.Entity{}).Where("id = ?", entity.Id).Count(&count).Error; err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 0 {
		t.Fatalf("soft-deleted topic still visible, count = %d", count)
	}

	// 清理测试数据
	db.Unscoped().Delete(&topics.Entity{}, entity.Id)
}

// TestPostgresDefaultData 验证 versioned migration 依赖的默认数据（category + page_config）可写入。
func TestPostgresDefaultData(t *testing.T) {
	dsn := pgDSN(t)

	conn := sqlconnect.GetConnect(sqlconnect.Config{
		Connection:         "postgres",
		DbUrl:              dsn,
		MaxIdleConnections: 1,
		MaxOpenConnections: 2,
		MaxLifeSeconds:     60,
	})
	sqlDB, err := conn.Connect.DB()
	if err != nil {
		t.Fatalf("sqlDB: %v", err)
	}
	defer sqlDB.Close()

	db := conn.Connect
	if err := db.AutoMigrate(&category.Entity{}, &pageConfig.Entity{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	cat := category.Entity{Name: "PG default", Desc: "integration", Slug: "pg-default", Color: "#000", Icon: ""}
	if err := db.Create(&cat).Error; err != nil {
		t.Fatalf("Create category: %v", err)
	}
	if cat.Id == 0 {
		t.Fatal("category id not populated")
	}

	cfg := pageConfig.Entity{PageType: pageConfig.Version, Config: `{"v":12}`}
	if err := db.Create(&cfg).Error; err != nil {
		t.Fatalf("Create page_config: %v", err)
	}

	db.Exec("DELETE FROM categories WHERE id = ?", cat.Id)
	db.Exec("DELETE FROM page_config WHERE page_type = ?", pageConfig.Version)
}

package migration

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openFileUsageIndexTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	conn, err := gorm.Open(
		sqlite.Open("file:fileusage-index-"+name+"?mode=memory&cache=shared"),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)},
	)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return conn
}

// seedFileUsagePlannerData 写入代表性数据并执行 ANALYZE。无统计信息时 SQLite
// 的静态成本模型在两个单列索引之间近似打平，规划器选择与数据无关（空表上
// 可能选中 idx_file_usage_status——status='ACTIVE' 几乎匹配全表，恰恰是最差
// 选择）。ANALYZE 写入真实选择率后，规划器稳定选择高选择率的 file_name 索引；
// 这也是生产 PostgreSQL（auto-analyze）下的真实行为：status=ACTIVE 匹配近乎
// 全表，file_name= 只匹配个位数行。
func seedFileUsagePlannerData(t *testing.T, conn *gorm.DB) {
	t.Helper()
	rows := make([]fileUsage.Entity, 0, 200)
	for filename := range 40 {
		for variant := range 5 {
			status := fileUsage.UsageStatusActive
			usageType := fileUsage.UsageInlineImage
			switch variant {
			case 3:
				status = fileUsage.UsageStatusRecovering
			case 4:
				status = fileUsage.UsageStatusPurged
				usageType = fileUsage.UsageUploadOwner
			}
			rows = append(rows, fileUsage.Entity{
				FileName:   fmt.Sprintf("2026/09/01/planner-%02d.png", filename),
				TargetType: fileUsage.TargetTopic,
				TargetId:   uint64(filename*10 + variant),
				UsageType:  usageType,
				UserId:     1,
				Status:     status,
			})
		}
	}
	if err := conn.CreateInBatches(rows, 100).Error; err != nil {
		t.Fatalf("seed planner rows: %v", err)
	}
	if err := conn.Exec("ANALYZE").Error; err != nil {
		t.Fatalf("analyze: %v", err)
	}
}

// TestFileNameLookupsUseIndex 钉住公开图片读取路径的索引契约（图片速率优化
// P0）：/file/img/* 每次请求都按 file_name 做引用检查（HasAnyReferences /
// HasActiveReferences 的 COUNT 查询）。唯一索引 idx_file_usage_target_file 的
// 前缀是 (target_type, target_id, usage_type)，file_name 排在第四位、前缀不
// 匹配，该查询只能全表扫描并随附件量线性劣化；模型必须声明独立的
// idx_file_usage_file_name 让引用检查走索引查找。
func TestFileNameLookupsUseIndex(t *testing.T) {
	conn := openFileUsageIndexTestDB(t, t.Name())
	if err := conn.AutoMigrate(&fileUsage.Entity{}); err != nil {
		t.Fatalf("AutoMigrate file_usages: %v", err)
	}
	seedFileUsagePlannerData(t, conn)

	cases := []struct {
		name string
		sql  string
		args []any
	}{
		{
			name: "HasAnyReferences",
			sql:  "SELECT count(*) FROM file_usages WHERE file_name = ? AND usage_type <> ?",
			args: []any{"2026/09/01/planner-07.png", fileUsage.UsageUploadOwner},
		},
		{
			name: "HasActiveReferences",
			sql:  "SELECT count(*) FROM file_usages WHERE file_name = ? AND status = ? AND usage_type <> ?",
			args: []any{"2026/09/01/planner-07.png", fileUsage.UsageStatusActive, fileUsage.UsageUploadOwner},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var rows []struct {
				ID      int
				Parent  int
				Notused int
				Detail  string
			}
			if err := conn.Raw("EXPLAIN QUERY PLAN "+tc.sql, tc.args...).Scan(&rows).Error; err != nil {
				t.Fatalf("explain query plan: %v", err)
			}
			joined := make([]string, 0, len(rows))
			used := false
			for _, row := range rows {
				joined = append(joined, row.Detail)
				if strings.Contains(row.Detail, "USING INDEX idx_file_usage_file_name") {
					used = true
				}
			}
			if !used {
				t.Fatalf("query plan does not use idx_file_usage_file_name: %s", strings.Join(joined, " | "))
			}
		})
	}
}

// legacyFileUsageEntity 旧版形态（与存量库一致）：只有 (target_type, target_id,
// usage_type, file_name) 复合唯一索引，没有独立 file_name 索引。
type legacyFileUsageEntity struct {
	Id         uint64 `gorm:"primaryKey;column:id;autoIncrement;not null;"`
	FileName   string `gorm:"column:file_name;type:varchar(512);not null;default:'';uniqueIndex:idx_file_usage_target_file,priority:4;"`
	TargetType string `gorm:"column:target_type;type:varchar(32);not null;default:'';uniqueIndex:idx_file_usage_target_file,priority:1;"`
	TargetId   uint64 `gorm:"column:target_id;not null;default:0;uniqueIndex:idx_file_usage_target_file,priority:2;"`
	UsageType  string `gorm:"column:usage_type;type:varchar(32);not null;default:'';uniqueIndex:idx_file_usage_target_file,priority:3;"`
	UserId     uint64 `gorm:"column:user_id;not null;default:0;"`
	Status     string `gorm:"column:status;type:varchar(32);not null;default:'ACTIVE';index:idx_file_usage_status,priority:1;"`
	ExpiresAt  *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (legacyFileUsageEntity) TableName() string { return "file_usages" }

// exerciseFileNameIndexUpgrade 在给定连接上执行完整升级路径并断言结果：
// 旧 schema（无 file_name 索引）→ AutoMigrate 新模型补齐索引 → 幂等重跑。
func exerciseFileNameIndexUpgrade(t *testing.T, conn *gorm.DB) {
	t.Helper()
	if err := conn.Migrator().DropTable(&fileUsage.Entity{}); err != nil {
		t.Fatalf("reset file_usages: %v", err)
	}
	if err := conn.AutoMigrate(&legacyFileUsageEntity{}); err != nil {
		t.Fatalf("migrate legacy schema: %v", err)
	}
	if conn.Migrator().HasIndex(&fileUsage.Entity{}, "idx_file_usage_file_name") {
		t.Fatal("precondition failed: legacy schema should not have idx_file_usage_file_name")
	}
	seed := &fileUsage.Entity{FileName: "2026/09/01/legacy.png", TargetType: fileUsage.TargetTopic, TargetId: 1, UsageType: fileUsage.UsageInlineImage, UserId: 1}
	if err := conn.Create(seed).Error; err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	// 部署新二进制：AutoMigrate 补齐索引
	if err := conn.AutoMigrate(&fileUsage.Entity{}); err != nil {
		t.Fatalf("upgrade AutoMigrate file_usages: %v", err)
	}
	if !conn.Migrator().HasIndex(&fileUsage.Entity{}, "idx_file_usage_file_name") {
		t.Fatal("idx_file_usage_file_name missing after upgrade AutoMigrate")
	}
	var count int64
	if err := conn.Model(&fileUsage.Entity{}).Where("file_name = ?", seed.FileName).Count(&count).Error; err != nil {
		t.Fatalf("count legacy rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("legacy row count after upgrade = %d, want 1", count)
	}
	// 幂等：每次启动都会 AutoMigrate，重复执行不得报错。
	if err := conn.AutoMigrate(&fileUsage.Entity{}); err != nil {
		t.Fatalf("idempotent AutoMigrate file_usages: %v", err)
	}
}

// TestFileNameIndexAddedOnLegacySchema 模拟存量库升级（SQLite）：旧 schema 上
// AutoMigrate 新模型必须自动补齐 idx_file_usage_file_name，且存量数据不受影响。
func TestFileNameIndexAddedOnLegacySchema(t *testing.T) {
	conn := openFileUsageIndexTestDB(t, t.Name()+"-upgrade")
	exerciseFileNameIndexUpgrade(t, conn)
}

// TestSchemaFileNameIndexAddedOnLegacySchemaOnPostgreSQL 同上，PostgreSQL 版
// （生产默认库方言，模型/迁移变更必须过 PG 门禁）。TestSchema 前缀是刻意的：
// CI 的 ci-backend-pg job 以 -run 'TestSchema' 圈定迁移包 PG 用例，同名模式的
// 新用例自动纳入（workflow 注释即约定），PG 门禁之外的命名会在 CI 中静默跳过。
// 依赖 YOURTJ_TEST_PG_URL，未设置时跳过。
func TestSchemaFileNameIndexAddedOnLegacySchemaOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set; skipping PostgreSQL file_usage index test")
	}
	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	exerciseFileNameIndexUpgrade(t, conn)
}

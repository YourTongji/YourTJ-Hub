package migration

import (
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/db4fileconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/setting"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/imConversations"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/imUserChatConfigs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/messages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/badges"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/contentDeleteEvent"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/dailyStats"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/migrationMapping"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationLog"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderators"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/networkAccessLog"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/oidcAccessTokens"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/oidcAuthRequests"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/optRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pointsRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/reports"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/role"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicCategoryIndex"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserStat"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userActivities"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userBadges"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userFollow"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userOAuth"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userPoints"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userSessions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userStatistics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userTotp"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userTotpChallenges"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userTotpRecoveryCodes"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

func M() {
	if !setting.UseMigration() {
		return
	}
	migrateSchema()
	runVersionedDataMigrations()
}

func migrateSchema() {
	var err error

	db := dbconnect.Connect()
	if err = validateUniqueUsernames(db); err != nil {
		slog.Error("dbconnect migration preflight failed", "err", err)
		os.Exit(1)
	}
	if err = upgradeCourseReviewLegacySchema(db); err != nil {
		slog.Error("dbconnect course_review legacy upgrade failed", "err", err)
		os.Exit(1)
	}
	if err = db.AutoMigrate(SchemaModels()...); err != nil {
		// 迁移失败必须立即退出（非零码），否则服务会带着残缺 schema 继续启动，
		// 登录/注册等依赖新表的接口在运行期才会报错，故障被发现时已影响线上。
		// 进程管理器会因此把实例标记为启动失败，触发回滚或告警。
		slog.Error("dbconnect migration err", "err", err)
		os.Exit(1)
	}
	slog.Info("dbconnect migration end")

	db4file := db4fileconnect.Connect()
	if err = db4file.AutoMigrate(
		&filedata.Entity{},
	); err != nil {
		slog.Error("db4fileconnect migration err", "err", err)
		os.Exit(1)
	}
	slog.Info("db4fileconnect migration end")
}

type duplicateUsername struct {
	Username string
	Count    int64
}

// validateUniqueUsernames fails before AutoMigrate attempts to add the global
// username unique index. Identity rows are never rewritten automatically; an
// operator must resolve blank or duplicate legacy usernames deliberately.
func validateUniqueUsernames(db *gorm.DB) error {
	if !db.Migrator().HasTable("users") {
		return nil
	}

	var blankCount int64
	if err := db.Table("users").Where("username = ?", "").Count(&blankCount).Error; err != nil {
		return fmt.Errorf("inspect blank usernames: %w", err)
	}
	var duplicates []duplicateUsername
	if err := db.Table("users").
		Select("username, COUNT(*) AS count").
		Where("username <> ?", "").
		Group("username").
		Having("COUNT(*) > 1").
		Limit(10).
		Scan(&duplicates).Error; err != nil {
		return fmt.Errorf("inspect duplicate usernames: %w", err)
	}
	if blankCount == 0 && len(duplicates) == 0 {
		return nil
	}
	return fmt.Errorf(
		"cannot create uniq_users_username: found %d blank username row(s) and duplicate usernames %v; assign every user a non-empty globally unique username before restarting",
		blankCount,
		duplicates,
	)
}

// SchemaModels 返回主库（db.default）的全部迁移模型。
// 迁移与 PostgreSQL 兼容性测试共用同一份清单，避免两处维护漂移。
func SchemaModels() []any {
	return []any{
		&badges.Entity{},
		&course.Entity{},
		&course.AliasEntity{},
		&course.TermEntity{},
		&course.OfferingEntity{},
		&course.InstructorEntity{},
		&course.OfferingInstructorEntity{},
		&course.ImportRunEntity{},
		&course.SourceRefEntity{},
		&course.ReviewEntity{},
		&course.HelpfulEntity{},
		&course.CourseStatsEntity{},
		&course.OfferingStatsEntity{},
		&eventNotification.Entity{},
		&fileUsage.Entity{},
		&moderationLog.Entity{},
		&migrationMapping.Entity{},
		&moderators.Entity{},
		&optRecord.Entity{},
		&pageConfig.Entity{},
		&pointsRecord.Entity{},
		&reports.Entity{},
		&agents.Entity{},
		&topics.Entity{},
		&posts.Entity{},
		&category.Entity{},
		&topicCategoryIndex.Entity{},
		&topicUserAction.Entity{},
		&postUserAction.Entity{},
		&topicUserStat.Entity{},
		&contentDeleteEvent.Entity{},
		&role.Entity{},
		&networkAccessLog.Entity{},
		&rolePermissionRs.Entity{},
		&taskQueue.Entity{},
		&userFollow.Entity{},
		&userBadges.Entity{},
		&oidcAuthRequests.Entity{},
		&oidcAccessTokens.Entity{},
		&userOAuth.Entity{},
		&userPoints.Entity{},
		&userSessions.Entity{},
		&userTotp.Entity{},
		&userTotpRecoveryCodes.Entity{},
		&userTotpChallenges.Entity{},
		&users.EntityComplete{},
		&userStatistics.Entity{},
		&imConversations.Entity{},
		&imUserChatConfigs.Entity{},
		&messages.Entity{},
		&dailyStats.Entity{},
		&userActivities.Entity{},
	}
}

// upgradeCourseReviewLegacySchema 把存量 course_review 表升级到 B3 清理所需
// 的新形态（issue #175，security 复审 F1）：
//   - author_user_id NOT NULL → nullable（清理置 NULL 释放唯一索引占位）
//   - deleted_at 从 gorm.DeletedAt 语义列 → 普通时间列（隔离窗口锚点）
//
// PostgreSQL：AutoMigrate 对 DROP NOT NULL 走 ALTER COLUMN，不重建表、
// 不丢数据，无需干预（由 PG 迁移测试覆盖）。
// SQLite：GORM 的整表重建（temp 表 + INSERT SELECT）只复制"列变化检测"
// 涉及的列——author_user_id NOT NULL→nullable 触发重建时**不复制该列**，
// 全部落 DEFAULT 0，与存量真实作者值冲突（唯一索引 2067），AutoMigrate 直接
// 失败。因此 SQLite 下先手工全列重建（临时表 + 全列复制 + 改名 + 重建索引），
// 保留 author_user_id 数据，再让 AutoMigrate 走新形态。
// 仅检测旧形态（author_user_id 为 NOT NULL）时执行；新库/已升级库跳过。
// 方言判断用 db.Dialector.Name()（测试连接与全局连接解耦）。
func upgradeCourseReviewLegacySchema(db *gorm.DB) error {
	if db.Dialector.Name() != "sqlite" {
		return nil // PG 由 AutoMigrate ALTER COLUMN 处理
	}
	if !db.Migrator().HasTable(courseReviewTableName) {
		return nil // 全新库，AutoMigrate 直接建新形态
	}
	// 检查 author_user_id 是否仍为 NOT NULL（旧形态）：
	// 读 sqlite_master 的表定义 SQL 做匹配——PRAGMA table_info 的 notnull
	// 列在 glebarez/sqlite 下无法可靠映射到 bool 字段，弃用。
	var ddl struct {
		SQL string
	}
	if err := db.Raw("SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?", courseReviewTableName).Scan(&ddl).Error; err != nil {
		return fmt.Errorf("inspect course_review DDL: %w", err)
	}
	lower := strings.ToLower(ddl.SQL)
	if !strings.Contains(lower, "author_user_id") {
		return nil // 非预期表，交给 AutoMigrate
	}
	// 旧形态判定：author_user_id 列定义段含 NOT NULL（新模型为 nullable）。
	re := regexp.MustCompile(`author_user_id[^,]*`)
	colDef := re.FindString(lower)
	if !strings.Contains(colDef, "not null") {
		return nil // 已是新形态（nullable），交给 AutoMigrate
	}

	// 旧形态：手工全列重建（保留 author_user_id 数据与唯一索引）
	// 1) 临时表（新形态：author_user_id 可空、deleted_at 普通列）
	if err := db.Exec(`CREATE TABLE course_review__upgrade (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		offering_id INTEGER NOT NULL DEFAULT 0,
		author_user_id INTEGER,
		rating INTEGER,
		content TEXT NOT NULL DEFAULT '',
		is_anonymous INTEGER NOT NULL DEFAULT 0,
		status INTEGER NOT NULL DEFAULT 0,
		legacy_helpful_count INTEGER NOT NULL DEFAULT 0,
		source TEXT NOT NULL DEFAULT '',
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	)`).Error; err != nil {
		return fmt.Errorf("create course_review upgrade table: %w", err)
	}
	// 2) 全列复制（含 author_user_id——gorm 重建会漏掉的列）
	if err := db.Exec(`INSERT INTO course_review__upgrade
		(id, offering_id, author_user_id, rating, content, is_anonymous, status,
		 legacy_helpful_count, source, created_at, updated_at, deleted_at)
		SELECT id, offering_id, author_user_id, rating, content, is_anonymous, status,
		       legacy_helpful_count, source, created_at, updated_at, deleted_at
		FROM course_review`).Error; err != nil {
		return fmt.Errorf("copy course_review data: %w", err)
	}
	// 3) 原子替换 + 重建索引
	if err := db.Exec(`DROP TABLE course_review`).Error; err != nil {
		return fmt.Errorf("drop legacy course_review: %w", err)
	}
	if err := db.Exec(`ALTER TABLE course_review__upgrade RENAME TO course_review`).Error; err != nil {
		return fmt.Errorf("rename course_review: %w", err)
	}
	if err := db.Exec(`CREATE UNIQUE INDEX uniq_course_review_offering_author ON course_review (offering_id, author_user_id)`).Error; err != nil {
		return fmt.Errorf("recreate course_review unique index: %w", err)
	}
	for _, idx := range []string{
		"CREATE INDEX idx_course_review_offering ON course_review (offering_id)",
		"CREATE INDEX idx_course_review_author ON course_review (author_user_id)",
		"CREATE INDEX idx_course_review_status ON course_review (status)",
	} {
		if err := db.Exec(idx).Error; err != nil {
			return fmt.Errorf("recreate course_review index: %w", err)
		}
	}
	return nil
}

const courseReviewTableName = "course_review"

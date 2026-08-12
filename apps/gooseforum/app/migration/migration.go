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
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
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
		&pk.CalendarEntity{},
		&pk.LanguageEntity{},
		&pk.CourseNatureEntity{},
		&pk.CourseNatureByCalendarEntity{},
		&pk.AssessmentEntity{},
		&pk.CampusEntity{},
		&pk.FacultyEntity{},
		&pk.MajorEntity{},
		&pk.MajorCourseEntity{},
		&pk.CourseDetailEntity{},
		&pk.TeacherEntity{},
		&pk.TeacherTimeslotEntity{},
		&pk.FetchLogEntity{},
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
		// 原表缺失：先检测半迁移状态（oierxjn 复审 P1）——旧的非事务重建
		// 若在 DROP course_review 成功、RENAME 之前中断，course_review 不
		// 存在但 course_review__upgrade 仍含已复制的历史课评。此时必须
		// rename 恢复临时表（含数据），而不是当全新库建空表（否则历史数据
		// 滞留 upgrade 表、业务不可见）。
		if db.Migrator().HasTable("course_review__upgrade") {
			slog.Warn("migration: course_review missing but course_review__upgrade present — restoring half-migrated data")
			err := db.Transaction(func(tx *gorm.DB) error {
				if err := tx.Exec(`ALTER TABLE course_review__upgrade RENAME TO course_review`).Error; err != nil {
					return fmt.Errorf("restore course_review from half-migrated upgrade table: %w", err)
				}
				return nil
			})
			if err != nil {
				return err
			}
			// 恢复后继续走下方旧形态检测（upgrade 表为 gorm 模型驱动的新
			// 形态——author_user_id nullable，检测会跳过重建，直接返回）。
			// 若恢复的是更早的非事务版本遗留（author_user_id NOT NULL），
			// 则进入正常重建流程。
		} else {
			return nil // 全新库，AutoMigrate 直接建新形态
		}
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

	// 旧形态：手工全列重建（保留 author_user_id 数据与唯一索引）。
	// 整个序列（删旧索引 → CREATE temp → INSERT SELECT → DROP → RENAME）
	// 包进 db.Transaction：SQLite 事务性 DDL 下，任意点崩溃（如 DROP 后、
	// RENAME 前进程退出）全量回滚，旧表完整、重启重试（security F1 与
	// spec F1 同根因——此前 autocommit 的崩溃窗口会把存量数据滞留在孤儿
	// 临时表，preflight 走"全新库"分支静默丢数据，或临时表残留导致重启
	// 卡死）。
	//
	// 孤儿临时表兜底（spec F1）：事务回滚覆盖普通错误与可恢复崩溃；进程
	// SIGKILL 等无法回滚的极端场景下，SQLite 崩溃恢复语义（journal/WAL）
	// 通常也会撤销未提交 DDL，但为绝对自愈，此处主动清理历史残留的
	// course_review__upgrade（若有）——CREATE 无 IF NOT EXISTS，残留会令
	// 重启永久卡死（already exists → os.Exit(1)）。清理先于事务，带 Warn
	// 日志便于诊断。
	//
	// 多实例并发（security 复审 F2）：两实例同时 preflight 时后到者 CREATE
	// temp 报 already exists → 返回错误 → 迁移失败退出 → 进程管理器重启 →
	// 先到者已完成则跳过。数据安全（fail-safe）但有一次启动失败噪音；
	// SQLite 无 advisory lock，事务包裹 + 孤儿清理兜底后错误重试成本低，
	// 容忍现状。
	if db.Migrator().HasTable("course_review__upgrade") {
		slog.Warn("migration: dropping orphan course_review__upgrade from a previous interrupted upgrade")
		if err := db.Exec(`DROP TABLE course_review__upgrade`).Error; err != nil {
			return fmt.Errorf("drop orphan course_review__upgrade: %w", err)
		}
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		// 1) 先删旧表索引（SQLite 索引名 schema 级唯一；旧表的
		// idx_course_review_* / uniq_course_review_offering_author 与
		// CreateTable 将建的索引同名冲突）。事务回滚时 DROP INDEX 一并
		// 撤销，旧表索引完整。
		for _, idx := range []string{
			"uniq_course_review_offering_author",
			"idx_course_review_offering",
			"idx_course_review_author",
			"idx_course_review_status",
		} {
			if err := tx.Migrator().DropIndex(&course.ReviewEntity{}, idx); err != nil {
				return fmt.Errorf("drop legacy course_review index %s: %w", idx, err)
			}
		}
		// 2) 临时表（新形态：author_user_id 可空、deleted_at 普通列）。
		// 用 gorm Migrator.CreateTable 模型驱动生成——与 ReviewEntity 精确
		// 一致。手写 DDL 与 gorm 期望的细微差异会触发 AutoMigrate 的渐进
		// 整表重建（每轮只复制部分列，多轮累积丢列，实测 6 轮后丢数据）；
		// 模型驱动后 AutoMigrate 无差异可检，不再重建。⚠️ 未来加列自动跟随
		// （security 复审 F3 消除硬编码漂移）。
		if err := tx.Table("course_review__upgrade").Migrator().CreateTable(&course.ReviewEntity{}); err != nil {
			return fmt.Errorf("create course_review upgrade table: %w", err)
		}
		// 3) 全列复制（含 author_user_id——gorm 自动重建会漏掉的列）
		if err := tx.Exec(`INSERT INTO course_review__upgrade
			(id, offering_id, author_user_id, rating, content, is_anonymous, status,
			 legacy_helpful_count, source, created_at, updated_at, deleted_at)
			SELECT id, offering_id, author_user_id, rating, content, is_anonymous, status,
			       legacy_helpful_count, source, created_at, updated_at, deleted_at
			FROM course_review`).Error; err != nil {
			return fmt.Errorf("copy course_review data: %w", err)
		}
		// 4) 原子替换（temp 已含唯一索引与普通索引，RENAME 后随表保留）
		if err := tx.Exec(`DROP TABLE course_review`).Error; err != nil {
			return fmt.Errorf("drop legacy course_review: %w", err)
		}
		if err := tx.Exec(`ALTER TABLE course_review__upgrade RENAME TO course_review`).Error; err != nil {
			return fmt.Errorf("rename course_review: %w", err)
		}
		return nil
	})
	return err
}

const courseReviewTableName = "course_review"

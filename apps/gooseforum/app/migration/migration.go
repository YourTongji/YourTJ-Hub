package migration

import (
	_ "embed"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
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
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
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
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaceEditors"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaces"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPageRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiSyncRuns"
	"gorm.io/gorm"
)

// M runs the schema migration and versioned data migration, returning an
// error on any failure. It does not decide process policy: callers choose
// whether a returned error is fatal (serve via the startup gate, the standalone
// migrate command) or tolerated (other CLI commands). Sentinels let callers
// distinguish "deferred, retry later" (ErrRetryLater) and
// "another instance holds the migration lock" (ErrLockUnavailable) from a hard
// failure.
func M() error {
	if !setting.UseMigration() {
		return nil
	}
	if err := migrateSchema(); err != nil {
		return err
	}
	return runVersionedDataMigrations()
}

func migrateSchema() error {
	var err error

	db := dbconnect.Connect()
	if err = validateUniqueUserEmails(db); err != nil {
		return fmt.Errorf("dbconnect migration email uniqueness preflight failed: %w", err)
	}
	if err = validateUniqueUsernames(db); err != nil {
		return fmt.Errorf("dbconnect migration preflight failed: %w", err)
	}
	if err = upgradeCourseReviewLegacySchema(db); err != nil {
		return fmt.Errorf("dbconnect course_review legacy upgrade failed: %w", err)
	}
	if err = dedupeWikiRevisionNumbers(db); err != nil {
		return fmt.Errorf("dbconnect wiki revision dedupe failed: %w", err)
	}
	if err = upgradeImportRunCompositeIndex(db); err != nil {
		return fmt.Errorf("dbconnect course_import_run index upgrade failed: %w", err)
	}
	if err = upgradeCourseTeacherIdentity(db); err != nil {
		return fmt.Errorf("dbconnect course teacher identity upgrade failed: %w", err)
	}
	if err = upgradeCourseAggregation(db); err != nil {
		return fmt.Errorf("dbconnect course aggregation upgrade failed: %w", err)
	}
	if err = upgradeCourseAiSummaryStatusIndex(db); err != nil {
		return fmt.Errorf("dbconnect course_ai_summary status index upgrade failed: %w", err)
	}
	if err = db.AutoMigrate(SchemaModels()...); err != nil {
		// 迁移失败必须上层按非零码退出，否则服务会带着残缺 schema 继续启动，
		// 登录/注册等依赖新表的接口在运行期才会报错，故障被发现时已影响线上。
		// 进程管理器会因此把实例标记为启动失败，触发回滚或告警。
		return fmt.Errorf("dbconnect migration err: %w", err)
	}
	slog.Info("dbconnect migration end")

	db4file := db4fileconnect.Connect()
	if err = db4file.AutoMigrate(
		&filedata.Entity{},
	); err != nil {
		return fmt.Errorf("db4fileconnect migration err: %w", err)
	}
	slog.Info("db4fileconnect migration end")
	return nil
}

type duplicateUsername struct {
	Username string
	Count    int64
}

type duplicateUserEmail struct {
	Email string
	Count int64
}

// validateUniqueUserEmails fails before AutoMigrate attempts to add the
// partial unique index for non-empty user emails. Empty emails are valid for
// bot/OAuth accounts and are intentionally excluded from the index. Identity
// rows are never rewritten automatically: legacy duplicate non-empty emails
// require an operator to resolve the conflicting accounts before startup.
func validateUniqueUserEmails(db *gorm.DB) error {
	if !db.Migrator().HasTable("users") {
		return nil
	}

	var duplicates []duplicateUserEmail
	if err := db.Table("users").
		Select("email, COUNT(*) AS count").
		Where("email IS NOT NULL AND email <> ?", "").
		Group("email").
		Having("COUNT(*) > 1").
		Order("email ASC").
		Limit(10).
		Scan(&duplicates).Error; err != nil {
		return fmt.Errorf("inspect duplicate user emails: %w", err)
	}
	if len(duplicates) == 0 {
		return nil
	}
	return fmt.Errorf(
		"cannot create uniq_users_email_nonempty: found duplicate non-empty user emails %v; resolve each conflicting account before restarting",
		duplicates,
	)
}

// validateUniqueUsernames fails before AutoMigrate attempts to add the global
// username unique index. Identity rows are never rewritten automatically; an
// operator must resolve blank or duplicate legacy usernames deliberately.
// upgradeImportRunCompositeIndex 升级 course_import_run 的幂等唯一索引（issue #183）：
// 旧版本为 manifest_hash 单列唯一（uniq_course_import_run_manifest），新模型声明
// (kind, manifest_hash) 复合唯一、索引名不变。两个必须显式处理的坑：
//  1. GORM AutoMigrate 按索引名判重，同名旧索引不会被重建——不删旧索引，存量库上
//     同一 manifest 包的 catalog/reviews 两次导入（相同 manifest_hash）会撞旧唯一约束。
//  2. 依赖 AutoMigrate 补 kind 列会触发 SQLite 整表重建，实测存量行数据丢失；
//     必须显式 ALTER TABLE ADD COLUMN（带默认值，保留存量数据）。
func upgradeImportRunCompositeIndex(db *gorm.DB) error {
	if !db.Migrator().HasTable("course_import_run") {
		return nil // 全新库：AutoMigrate 直接建全表 + 复合索引。
	}
	if !db.Migrator().HasColumn(&course.ImportRunEntity{}, "kind") {
		if err := db.Exec("ALTER TABLE course_import_run ADD COLUMN kind VARCHAR(32) NOT NULL DEFAULT 'catalog'").Error; err != nil {
			return fmt.Errorf("add course_import_run.kind column: %w", err)
		}
		slog.Info("dbconnect course_import_run.kind column added (default 'catalog')")
	}
	if db.Migrator().HasIndex(&course.ImportRunEntity{}, "uniq_course_import_run_manifest") {
		if err := db.Migrator().DropIndex(&course.ImportRunEntity{}, "uniq_course_import_run_manifest"); err != nil {
			return fmt.Errorf("drop legacy course_import_run unique index: %w", err)
		}
		slog.Info("dbconnect course_import_run legacy unique index dropped, will be recreated as (kind, manifest_hash)")
	}
	return nil
}

// upgradeCourseTeacherIdentity 把 course 身份从「一门课一个 primary_code」
// 升级为「(primary_code, teacher_id) 复合身份」（issue #326）：
//  1. teacher_id 列：存量库显式 ALTER TABLE ADD COLUMN（NOT NULL DEFAULT 0，
//     0 = 无教师哨兵值），不依赖 AutoMigrate——SQLite 依赖 AutoMigrate 补列会
//     整表重建丢数据。
//  2. 唯一索引 uniq_course_primary_code（单列）→ uniq_course_code_teacher
//     (primary_code, teacher_id) 复合：GORM AutoMigrate 按索引名判重，
//     同名旧索引不会被重建，必须先显式 DropIndex 旧索引。
func upgradeCourseTeacherIdentity(db *gorm.DB) error {
	if !db.Migrator().HasTable(courseTableName) {
		return nil // 全新库：AutoMigrate 直接建全表 + 复合索引。
	}
	if !db.Migrator().HasColumn(&course.Entity{}, "teacher_id") {
		// 带默认值 0（无教师）的 NOT NULL 列：存量行回填 0，符合新模型。
		if err := db.Exec("ALTER TABLE course ADD COLUMN teacher_id BIGINT NOT NULL DEFAULT 0").Error; err != nil {
			return fmt.Errorf("add course.teacher_id column: %w", err)
		}
		slog.Info("dbconnect course.teacher_id column added (default 0)")
		// 回填身份教师：存量库从 offering 教师关联取每门课的首位教师
		// （id 最小可见 offering 的 id 最小 instructor），与物化/导入的
		// "教学班首位教师"语义一致；无 offering 或教师关联的课程保持 0。
		// 旧模型 primary_code 单列唯一 → 每 code 只有一行，回填不会造成
		// (code, teacher_id) 复合唯一冲突。
		if err := backfillCourseTeacherIdentity(db); err != nil {
			return fmt.Errorf("backfill course.teacher_id: %w", err)
		}
	}
	if db.Migrator().HasIndex(&course.Entity{}, "uniq_course_primary_code") {
		if err := db.Migrator().DropIndex(&course.Entity{}, "uniq_course_primary_code"); err != nil {
			return fmt.Errorf("drop legacy course primary_code unique index: %w", err)
		}
		slog.Info("dbconnect course legacy unique index dropped, will be recreated as (primary_code, teacher_id)")
	}
	return nil
}

// upgradeCourseAiSummaryStatusIndex 删除 course_ai_summary.status 列的冗余索引
// （#342/#343 review）：status 低基数（2 值）且查询全部按 course_id 主键访问。
// 只移除模型上的 index tag 不够——GORM AutoMigrate 按索引名判重，不会删除模型
// 上已不再声明的索引；存量库（#342 合入时建出了 idx_course_ai_summary_status）
// 必须显式 DropIndex，全新库无此索引直接跳过。
func upgradeCourseAiSummaryStatusIndex(db *gorm.DB) error {
	if !db.Migrator().HasTable("course_ai_summary") {
		return nil // 全新库：AutoMigrate 按新模型建表（无 status 索引）。
	}
	if db.Migrator().HasIndex(&course.CourseAiSummaryEntity{}, "idx_course_ai_summary_status") {
		if err := db.Migrator().DropIndex(&course.CourseAiSummaryEntity{}, "idx_course_ai_summary_status"); err != nil {
			return fmt.Errorf("drop legacy course_ai_summary status index: %w", err)
		}
		slog.Info("dbconnect course_ai_summary legacy status index dropped")
	}
	return nil
}

// upgradeCourseAggregation 为课评聚合新增列（Blueprint 课评聚合 Phase 1）：
//   - course.review_scope / course.team_key：三档课评范围（teacher/team/course）与团队键；
//   - course_offering.teaching_class_id：教学班持久关联（offering 权威写入源 = PK 物化链）；
//   - course_instructor.teacher_code：教师身份主锚（替代仅有 (name, department) 自然键）。
//
// 存量库显式 ALTER TABLE ADD COLUMN（带默认值），不依赖 AutoMigrate——SQLite 依赖
// AutoMigrate 补列会整表重建丢数据。
// 回填（一次性，双方言）：
//   - offering.teaching_class_id ← course_source_ref.external_id 首段（"{class_id}-{course_ext}"，
//     EntityTypeOffering；other-* 虚拟班非数字首段保持 0）；
//   - instructor.teacher_code ← course_source_ref.external_id（EntityTypeInstructor；
//     syn-{id} 合成工号原样保留，作为稳定身份键）。
func upgradeCourseAggregation(db *gorm.DB) error {
	if !db.Migrator().HasTable(courseTableName) {
		return nil // 全新库：AutoMigrate 直接建全表 + 索引。
	}
	if !db.Migrator().HasColumn(&course.Entity{}, "review_scope") {
		if err := db.Exec("ALTER TABLE course ADD COLUMN review_scope VARCHAR(16) NOT NULL DEFAULT 'teacher'").Error; err != nil {
			return fmt.Errorf("add course.review_scope column: %w", err)
		}
		slog.Info("dbconnect course.review_scope column added (default 'teacher')")
	}
	if !db.Migrator().HasColumn(&course.Entity{}, "team_key") {
		if err := db.Exec("ALTER TABLE course ADD COLUMN team_key VARCHAR(64) NOT NULL DEFAULT ''").Error; err != nil {
			return fmt.Errorf("add course.team_key column: %w", err)
		}
		slog.Info("dbconnect course.team_key column added (default '')")
	}
	if db.Migrator().HasTable(&course.OfferingEntity{}) {
		if !db.Migrator().HasColumn(&course.OfferingEntity{}, "teaching_class_id") {
			if err := db.Exec("ALTER TABLE course_offering ADD COLUMN teaching_class_id BIGINT NOT NULL DEFAULT 0").Error; err != nil {
				return fmt.Errorf("add course_offering.teaching_class_id column: %w", err)
			}
			slog.Info("dbconnect course_offering.teaching_class_id column added (default 0)")
		}
		if err := backfillOfferingTeachingClass(db); err != nil {
			return fmt.Errorf("backfill course_offering.teaching_class_id: %w", err)
		}
	}
	if db.Migrator().HasTable(&course.InstructorEntity{}) {
		if !db.Migrator().HasColumn(&course.InstructorEntity{}, "teacher_code") {
			if err := db.Exec("ALTER TABLE course_instructor ADD COLUMN teacher_code VARCHAR(64) NOT NULL DEFAULT ''").Error; err != nil {
				return fmt.Errorf("add course_instructor.teacher_code column: %w", err)
			}
			slog.Info("dbconnect course_instructor.teacher_code column added (default '')")
		}
		if err := backfillInstructorTeacherCode(db); err != nil {
			return fmt.Errorf("backfill course_instructor.teacher_code: %w", err)
		}
	}
	return nil
}

// backfillOfferingTeachingClass 从 course_source_ref 回填 offering.teaching_class_id：
// external_id 格式 "{class_id}-{course_ext}"（converter 生成），首段即一系统教学班 id。
// other-* 虚拟班（非数字首段）与无 source_ref 的 offering 保持 0。
func backfillOfferingTeachingClass(db *gorm.DB) error {
	if !db.Migrator().HasTable("course_source_ref") {
		return nil
	}
	type refRow struct {
		ExternalId string
		LocalId    uint64
	}
	var rows []refRow
	backfilled := 0
	// 分批处理（review nit：整表全量读入内存后逐行 UPDATE → FindInBatches 分批，
	// 课程域量级上涨时不占峰值内存）。
	if err := db.Table("course_source_ref").
		Select("external_id, local_id").
		Where("entity_type = ?", course.EntityTypeOffering).
		Where("external_id <> ''").
		Order("id ASC").
		FindInBatches(&rows, 500, func(tx *gorm.DB, batch int) error {
			for _, r := range rows {
				classID, ok := parseTeachingClassID(r.ExternalId)
				if !ok {
					continue
				}
				res := tx.Table("course_offering").
					Where("id = ? AND teaching_class_id = 0", r.LocalId).
					Update("teaching_class_id", classID)
				if res.Error != nil {
					return res.Error
				}
				if res.RowsAffected > 0 {
					backfilled++
				}
			}
			return nil
		}).Error; err != nil {
		return err
	}
	if backfilled > 0 {
		slog.Info("dbconnect course_offering.teaching_class_id backfilled from source_ref", "offerings", backfilled)
	}
	return nil
}

// parseTeachingClassID 从 offering external_id（"{class_id}-{course_ext}"）解析教学班 id；
// 非数字首段（如 other-* 虚拟班）返回 false。
func parseTeachingClassID(externalID string) (uint64, bool) {
	head, _, _ := strings.Cut(externalID, "-")
	id, err := strconv.ParseUint(strings.TrimSpace(head), 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return id, true
}

// backfillInstructorTeacherCode 从 course_source_ref 回填 course_instructor.teacher_code：
// instructor 的 external id 即 teacherCode（或合成 "syn-{teacher_id}"，原样保留作稳定身份键）。
func backfillInstructorTeacherCode(db *gorm.DB) error {
	if !db.Migrator().HasTable("course_source_ref") {
		return nil
	}
	type refRow struct {
		ExternalId string
		LocalId    uint64
	}
	var rows []refRow
	backfilled := 0
	if err := db.Table("course_source_ref").
		Select("external_id, local_id").
		Where("entity_type = ?", course.EntityTypeInstructor).
		Where("external_id <> ''").
		Order("id ASC").
		FindInBatches(&rows, 500, func(tx *gorm.DB, batch int) error {
			for _, r := range rows {
				code := strings.TrimSpace(r.ExternalId)
				if code == "" {
					continue
				}
				res := tx.Table("course_instructor").
					Where("id = ? AND teacher_code = ''", r.LocalId).
					Update("teacher_code", code)
				if res.Error != nil {
					return res.Error
				}
				if res.RowsAffected > 0 {
					backfilled++
				}
			}
			return nil
		}).Error; err != nil {
		return err
	}
	if backfilled > 0 {
		slog.Info("dbconnect course_instructor.teacher_code backfilled from source_ref", "instructors", backfilled)
	}
	return nil
}

func backfillCourseTeacherIdentity(db *gorm.DB) error {
	if !db.Migrator().HasTable("course_offering") || !db.Migrator().HasTable("course_offering_instructor") {
		return nil // 全新库/无课程域：AutoMigrate 随后建表，无需回填。
	}
	type offeringTeacher struct {
		CourseId     uint64
		OfferingId   uint64
		InstructorId uint64
	}
	var rows []offeringTeacher
	if err := db.Table("course_offering o").
		Select("o.course_id, o.id AS offering_id, oi.instructor_id").
		Joins("JOIN course_offering_instructor oi ON oi.offering_id = o.id").
		Where("o.deleted_at IS NULL").
		Order("o.course_id ASC, o.id ASC, oi.instructor_id ASC").
		Scan(&rows).Error; err != nil {
		return err
	}
	seen := make(map[uint64]struct{}, len(rows))
	for _, r := range rows {
		if _, ok := seen[r.CourseId]; ok {
			continue
		}
		seen[r.CourseId] = struct{}{}
		if err := db.Table(courseTableName).
			Where("id = ? AND teacher_id = 0", r.CourseId).
			Update("teacher_id", r.InstructorId).Error; err != nil {
			return err
		}
	}
	if len(seen) > 0 {
		slog.Info("dbconnect course.teacher_id backfilled from offering instructors", "courses", len(seen))
	}
	return nil
}

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
		&course.DislikeEntity{},
		&course.CourseStatsEntity{},
		&course.OfferingStatsEntity{},
		&course.CourseAiSummaryEntity{},
		&course.CourseUserActionEntity{},
		&course.RelationEntity{},
		// PK 排课数据域（Issue #187 / #186）：13 表。
		&pk.CalendarEntity{},
		&pk.CampusEntity{},
		&pk.FacultyEntity{},
		&pk.LanguageEntity{},
		&pk.AssessmentEntity{},
		&pk.CourseNatureEntity{},
		&pk.CourseNatureByCalendarEntity{},
		&pk.MajorEntity{},
		&pk.MajorCourseEntity{},
		&pk.CourseDetailEntity{},
		&pk.TeacherEntity{},
		&pk.TeacherTimeslotEntity{},
		&pk.FetchLogEntity{},
		&pk.SettingEntity{},
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
		&postRevisions.Entity{},
		&category.Entity{},
		&topicCategoryIndex.Entity{},
		&topicUserAction.Entity{},
		&postUserAction.Entity{},
		&wikiNamespaces.Entity{},
		&wikiNamespaceEditors.Entity{},
		&wikiPages.Entity{},
		&wikiPageRevisions.Entity{},
		&wikiSyncRuns.Entity{},
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

const courseTableName = "course"

// dedupeWikiRevisionNumbers 在 AutoMigrate 创建 uniq_wiki_rev_page_no 唯一索引前，
// 清理存量 wiki_page_revisions 中 (page_id, revision_no) 重复的行（旧版并发写入
// 或回滚残留可能产生同号修订；唯一索引不区分 deleted_at，软删行同样占用索引槽，
// 因此对全部行含软删去重）。每组保留 id 最大（最新写入）的一行，其余物理删除。
func dedupeWikiRevisionNumbers(db *gorm.DB) error {
	if !db.Migrator().HasTable("wiki_page_revisions") {
		return nil
	}
	var dups []struct {
		PageId     uint64
		RevisionNo int
		Cnt        int64
	}
	if err := db.Table("wiki_page_revisions").Unscoped().
		Select("page_id, revision_no, COUNT(*) AS cnt").
		Group("page_id, revision_no").
		Having("COUNT(*) > 1").
		Scan(&dups).Error; err != nil {
		return fmt.Errorf("inspect duplicate wiki revisions: %w", err)
	}
	if len(dups) == 0 {
		return nil
	}
	slog.Warn("migration: found duplicate wiki revision numbers", "groups", len(dups))
	for _, d := range dups {
		var keepID uint64
		if err := db.Table("wiki_page_revisions").Unscoped().
			Where("page_id = ?", d.PageId).
			Where("revision_no = ?", d.RevisionNo).
			Order("id DESC").
			Limit(1).
			Pluck("id", &keepID).Error; err != nil {
			return fmt.Errorf("find keep wiki revision: %w", err)
		}
		// Unscoped + Delete = 物理删除：软删行仍占用唯一索引，必须真正删除。
		if err := db.Table("wiki_page_revisions").Unscoped().
			Where("page_id = ?", d.PageId).
			Where("revision_no = ?", d.RevisionNo).
			Where("id <> ?", keepID).
			Delete(&wikiPageRevisions.Entity{}).Error; err != nil {
			return fmt.Errorf("dedupe wiki revisions (page=%d rev=%d): %w", d.PageId, d.RevisionNo, err)
		}
	}
	slog.Info("migration: wiki revision dedupe done", "groups", len(dups))
	return nil
}

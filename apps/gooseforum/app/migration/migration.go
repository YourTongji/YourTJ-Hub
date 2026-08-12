package migration

import (
	_ "embed"
	"fmt"
	"log/slog"
	"os"

	"github.com/leancodebox/GooseForum/app/bundles/connect/db4fileconnect"
	"github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/bundles/setting"
	"github.com/leancodebox/GooseForum/app/models/chat/imConversations"
	"github.com/leancodebox/GooseForum/app/models/chat/imUserChatConfigs"
	"github.com/leancodebox/GooseForum/app/models/chat/messages"
	"github.com/leancodebox/GooseForum/app/models/filemodel/filedata"
	"github.com/leancodebox/GooseForum/app/models/forum/agents"
	"github.com/leancodebox/GooseForum/app/models/forum/badges"

	"github.com/leancodebox/GooseForum/app/models/forum/category"
	"github.com/leancodebox/GooseForum/app/models/forum/contentDeleteEvent"
	"github.com/leancodebox/GooseForum/app/models/forum/course"
	"github.com/leancodebox/GooseForum/app/models/forum/dailyStats"
	"github.com/leancodebox/GooseForum/app/models/forum/eventNotification"
	"github.com/leancodebox/GooseForum/app/models/forum/fileUsage"
	"github.com/leancodebox/GooseForum/app/models/forum/migrationMapping"
	"github.com/leancodebox/GooseForum/app/models/forum/moderationLog"
	"github.com/leancodebox/GooseForum/app/models/forum/moderators"
	"github.com/leancodebox/GooseForum/app/models/forum/networkAccessLog"
	"github.com/leancodebox/GooseForum/app/models/forum/oidcAccessTokens"
	"github.com/leancodebox/GooseForum/app/models/forum/oidcAuthRequests"
	"github.com/leancodebox/GooseForum/app/models/forum/optRecord"
	"github.com/leancodebox/GooseForum/app/models/forum/pageConfig"
	"github.com/leancodebox/GooseForum/app/models/forum/pointsRecord"
	"github.com/leancodebox/GooseForum/app/models/forum/postUserAction"
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
	"github.com/leancodebox/GooseForum/app/models/forum/userSessions"
	"github.com/leancodebox/GooseForum/app/models/forum/userStatistics"
	"github.com/leancodebox/GooseForum/app/models/forum/userTotp"
	"github.com/leancodebox/GooseForum/app/models/forum/userTotpChallenges"
	"github.com/leancodebox/GooseForum/app/models/forum/userTotpRecoveryCodes"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
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

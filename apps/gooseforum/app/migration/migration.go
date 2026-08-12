package migration

import (
	_ "embed"
	"fmt"
	"log/slog"
	"os"

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

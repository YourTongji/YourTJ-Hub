package migration

import (
	_ "embed"
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
	"github.com/leancodebox/GooseForum/app/models/forum/dailyStats"
	"github.com/leancodebox/GooseForum/app/models/forum/eventNotification"
	"github.com/leancodebox/GooseForum/app/models/forum/fileUsage"
	"github.com/leancodebox/GooseForum/app/models/forum/migrationMapping"
	"github.com/leancodebox/GooseForum/app/models/forum/moderationLog"
	"github.com/leancodebox/GooseForum/app/models/forum/moderators"
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

// SchemaModels 返回主库（db.default）的全部迁移模型。
// 迁移与 PostgreSQL 兼容性测试共用同一份清单，避免两处维护漂移。
func SchemaModels() []any {
	return []any{
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
		&agents.Entity{},
		&topics.Entity{},
		&posts.Entity{},
		&category.Entity{},
		&topicCategoryIndex.Entity{},
		&topicUserAction.Entity{},
		&postUserAction.Entity{},
		&topicUserStat.Entity{},
		&role.Entity{},
		&rolePermissionRs.Entity{},
		&taskQueue.Entity{},
		&userFollow.Entity{},
		&userBadges.Entity{},
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

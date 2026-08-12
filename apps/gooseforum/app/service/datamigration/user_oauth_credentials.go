package datamigration

import (
	"fmt"

	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"gorm.io/gorm"
)

// UserOAuthCredentialsResult 汇报 user_o_auth 表明文凭据列的清理结果。
type UserOAuthCredentialsResult struct {
	Dropped    []string
	Failed     int
	LastFailed string
}

// DropUserOAuthTokenColumns 删除 user_o_auth 表中无业务消费的明文 OAuth
// 凭据列（access_token/refresh_token/token_expiry/scopes/raw_user_data），
// 对应 Issue #131：第三方凭据不再落库。
//
// 生产主库为 SQLite；逐列使用 HasColumn 守卫后执行单列 DROP COLUMN，
// 既满足 SQLite 不支持单语句多列 DROP，也天然幂等（全新库上为 no-op）。
// 任一列 DROP 失败即返回并保留失败信息，由上层决定不推进迁移版本，
// 服务可正常启动并在下次重启时重试。
func DropUserOAuthTokenColumns() UserOAuthCredentialsResult {
	return DropUserOAuthTokenColumnsWithDB(db.Connect())
}

func DropUserOAuthTokenColumnsWithDB(conn *gorm.DB) UserOAuthCredentialsResult {
	result := UserOAuthCredentialsResult{}
	if !conn.Migrator().HasTable("user_o_auth") {
		return result
	}

	for _, column := range []string{"access_token", "refresh_token", "token_expiry", "scopes", "raw_user_data"} {
		if !conn.Migrator().HasColumn("user_o_auth", column) {
			continue
		}
		if err := conn.Exec("ALTER TABLE user_o_auth DROP COLUMN " + column).Error; err != nil {
			failUserOAuthCredentialsMigration(&result, "drop_"+column, err)
			return result
		}
		result.Dropped = append(result.Dropped, column)
	}
	return result
}

func failUserOAuthCredentialsMigration(result *UserOAuthCredentialsResult, step string, err error) {
	result.Failed++
	result.LastFailed = fmt.Sprintf("%s: %s", step, err.Error())
}

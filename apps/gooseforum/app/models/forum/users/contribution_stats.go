package users

import (
	"context"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"gorm.io/gorm"
)

// ActorAuthorIDsQuery 是身份所有者提供给「内容贡献统计」的标量子查询：
// 返回指定主体类型且未注销（未软删）的账号 id。
//
// 冻结账号仍会返回：冻结只阻止继续写入，不改变其历史公开内容的统计归属
// （贡献统计的口径只排除匿名行与已注销账号，见 research/issue-1145-design.md R4/R6）。
func ActorAuthorIDsQuery(ctx context.Context, actorType int8) *gorm.DB {
	return ActorAuthorIDsQueryTx(db.ConnectContext(ctx), actorType)
}

// ActorAuthorIDsQueryTx 是 ActorAuthorIDsQuery 的显式连接变体，
// 供需要复用同一连接/方言的聚合查询（含 PostgreSQL 方言测试）使用。
func ActorAuthorIDsQueryTx(conn *gorm.DB, actorType int8) *gorm.DB {
	return conn.
		Table(tableName).
		Select(pid).
		Where(fieldDeletedAt+" IS NULL").
		Where("actor_type = ?", actorType)
}

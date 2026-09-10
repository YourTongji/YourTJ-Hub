package dbconnect

import (
	"context"
	"sync"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/closer"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/sqlconnect"

	"gorm.io/gorm"
)

//func init() {
//	bootstrap.AddDInit(connectDB)
//}

var (
	once = new(sync.Once)
)

var dbConnect sqlconnect.Connect

func Connect() *gorm.DB {
	once.Do(func() {
		dbConnect = sqlconnect.ConnectByPrefix("db.default")
		// 注册到全局关闭管理器
		closer.RegisterPriorityContext(closer.PriorityDatabase, func(context.Context) error {
			dbConnect.Close()
			return nil
		})
	})
	return dbConnect.Connect
}

// ConnectContext returns the default database handle bound to ctx. Queries
// started from request or worker code should use this entry point so driver
// cancellation reaches the database instead of continuing in the background.
func ConnectContext(ctx context.Context) *gorm.DB {
	if ctx == nil {
		ctx = context.Background()
	}
	return Connect().WithContext(ctx)
}

func IsSqlite() bool {
	Connect()
	return dbConnect.IsSqlite()
}

// Close 关闭数据库连接
func Close() {
	dbConnect.Close()
}

func BackupSQLiteHandle() {
	dbConnect.BackupSQLiteHandle()
}

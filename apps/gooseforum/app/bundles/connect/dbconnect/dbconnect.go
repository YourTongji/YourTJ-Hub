package dbconnect

import (
	"fmt"
	"sync"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/closer"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/sqlconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"

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
		if preferences.IsTestMode() {
			dbConnect = sqlconnect.GetConnect(sqlconnect.TestConfig())
		} else {
			dbConfig := preferences.GetExclusivePreferences("db.default")
			dbConnect = sqlconnect.GetConnectByPreferences(dbConfig)
		}
		if dbConnect.Error != nil {
			// 连接失败时立即失败，避免 nil *gorm.DB 在后续 AutoMigrate 上解引用崩溃
			panic(fmt.Sprintf("dbconnect: %v", dbConnect.Error))
		}
		// 注册到全局关闭管理器
		closer.RegisterPriority(closer.PriorityDatabase, func() error {
			dbConnect.Close()
			return nil
		})
	})
	return dbConnect.Connect
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

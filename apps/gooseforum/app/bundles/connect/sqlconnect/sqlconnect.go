package sqlconnect

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/logging"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/setting"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var debug = setting.IsDebug()

type Config struct {
	Connection         string
	DbUrl              string
	DbPath             string
	MaxIdleConnections int
	MaxOpenConnections int
	MaxLifeSeconds     int
}

type Connect struct {
	Config  Config
	Connect *gorm.DB
	Error   error
	Init    bool
}

func TestConfig() Config {
	return Config{
		Connection:         "sqlite",
		DbPath:             ":memory:",
		MaxIdleConnections: 1,
		MaxOpenConnections: 1,
		MaxLifeSeconds:     60,
	}
}

// ConnectByPrefix 按配置前缀建立数据库连接（测试模式走内存 sqlite），
// 连接失败时立即 panic，避免 nil *gorm.DB 在后续 AutoMigrate 上解引用崩溃。
func ConnectByPrefix(prefix string) Connect {
	if preferences.IsTestMode() {
		return GetConnect(TestConfig())
	}
	dbConfig := preferences.GetExclusivePreferences(prefix)
	dbConnect := GetConnectByPreferences(dbConfig)
	if dbConnect.Error != nil {
		panic(fmt.Sprintf("dbconnect(%s): %v", prefix, dbConnect.Error))
	}
	return dbConnect
}

func (itself *Connect) IsSqlite() bool {
	return itself.Config.Connection == "sqlite"
}

func GetConnectByPreferences(preferences preferences.ExclusivePreferences) Connect {
	c := Config{
		Connection:         preferences.Get(`connection`, `sqlite`),
		DbUrl:              preferences.Get(`url`),
		DbPath:             preferences.Get(`path`, `:memory:`),
		MaxIdleConnections: preferences.GetInt(`maxIdleConnections`, 2),
		MaxOpenConnections: preferences.GetInt(`maxOpenConnections`, 2),
		MaxLifeSeconds:     preferences.GetInt(`maxLifeSeconds`, 60),
	}
	return GetConnect(c)
}

// GetConnect 初始化模型
func GetConnect(config Config) Connect {
	var dbIns *gorm.DB
	var err error
	switch config.Connection {
	case "sqlite":
		slog.Info("use sqlite")
		dbIns, err = connectSqlLiteDB(config.DbPath)
	case "postgres":
		slog.Info("use postgres")
		dbIns, err = connectPostgresDB(config.DbUrl)
	default:
		// 未知连接类型显式报错，避免配置拼错悄悄回退到 sqlite
		err = fmt.Errorf("unsupported db connection type %q (supported: sqlite, postgres)", config.Connection)
		slog.Error(err.Error())
		return Connect{Config: config, Connect: nil, Error: err}
	}

	if err != nil {
		slog.Error(err.Error())
		return Connect{Config: config, Connect: dbIns, Error: err}
	}

	if debug {
		slog.Info("开启debug")
		dbIns = dbIns.Debug()
	}

	// 获取底层的 sqlDB
	sqlDB, err := dbIns.DB()
	if err != nil {
		slog.Error(err.Error())
		return Connect{Config: config, Connect: dbIns, Error: err}
	}
	// 设置最大连接数
	sqlDB.SetMaxOpenConns(config.MaxOpenConnections)
	// 设置最大空闲连接数
	sqlDB.SetMaxIdleConns(config.MaxIdleConnections)
	// 设置每个链接的过期时间
	sqlDB.SetConnMaxLifetime(time.Duration(config.MaxLifeSeconds) * time.Second)
	return Connect{Config: config, Connect: dbIns, Error: err}
}

func connectPostgresDB(dbUrl string) (*gorm.DB, error) {
	// 初始化 PostgreSQL 连接信息
	// DSN 推荐 key=value 格式:
	// host=localhost user=yourtj password=yourtj dbname=yourtj port=5432 sslmode=disable
	gormConfig := postgres.New(postgres.Config{
		DSN: dbUrl,
	})

	db, err := gorm.Open(gormConfig, &gorm.Config{
		Logger:         logging.NewGormLoggerWithDefault(),
		TranslateError: true,
	})
	return db, err
}

func connectSqlLiteDB(dbPath string) (*gorm.DB, error) {
	if dbPath == "" {
		dbPath = ":memory:"
	} else if dbPath == ":memory:" {
		// ":memory:"
	} else if err := createFileIfNotExists(dbPath); err != nil {
		return nil, err
	}

	// 构建 SQLite DSN，启用 WAL 模式和其他优化配置
	dsn := buildSQLiteDSN(dbPath, map[string]string{
		"journal_mode":       "WAL",
		"cache_size":         "-20000",
		"synchronous":        "NORMAL",
		"journal_size_limit": "1048576",
		"wal_autocheckpoint": "1000",
		"page_size":          "8192",
		"busy_timeout":       "5000",
	})

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:         logging.NewGormLoggerWithDefault(),
		TranslateError: true,
	})

	return db, err
}

func createFileIfNotExists(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(filePath), 0750); err != nil {
			return err
		}

		if err := os.WriteFile(filePath, []byte(""), 0600); err != nil {
			return err
		}
	}
	return nil
}

func buildSQLiteDSN(filepath string, config map[string]string) string {
	// 如果 filepath 已经包含参数，则拆分路径和参数
	basePath := filepath
	existingParams := make(map[string]string)
	if before, after, ok := strings.Cut(filepath, "?"); ok {
		basePath = before
		paramStr := after

		// 解析已有的参数
		for param := range strings.SplitSeq(paramStr, "&") {
			if after, ok := strings.CutPrefix(param, "_pragma="); ok {
				// 提取 _pragma 的值
				pragmaValue := after
				// 拆分 key 和 value
				if pidx := strings.Index(pragmaValue, "("); pidx != -1 {
					key := pragmaValue[:pidx]
					value := pragmaValue[pidx+1 : len(pragmaValue)-1] // 去掉括号
					existingParams[key] = value
				}
			}
		}
	}

	// 构建新的参数
	var params []string
	for key, value := range config {
		// 如果参数已经存在，跳过
		if _, exists := existingParams[key]; exists {
			continue
		}
		// 添加新的参数
		params = append(params, fmt.Sprintf("_pragma=%s(%s)", key, value))
	}

	// 如果有已有的参数，添加到 params 中
	if len(existingParams) > 0 {
		for key, value := range existingParams {
			params = append(params, fmt.Sprintf("_pragma=%s(%s)", key, value))
		}
	}

	// 拼接路径和参数
	if len(params) > 0 {
		return basePath + "?" + strings.Join(params, "&")
	}
	return basePath
}

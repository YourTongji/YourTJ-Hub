package sqlconnect

import (
	"fmt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
	"github.com/glebarez/sqlite"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/fileopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"gorm.io/gorm"
)

func getBackUpDir() string {
	return preferences.Get("db.backupDir")
}

func (itself *Connect) GenerateBackupPath(backupDir string) string {
	if !itself.IsSqlite() || itself.Connect == nil {
		return ""
	}
	sourcePath := itself.Config.DbPath
	// 处理内存数据库的特殊标识
	if sourcePath == ":memory:" {
		return filepath.Join(backupDir, fmt.Sprintf("memory_%s.db", time.Now().Format("20060102_150405")))
	}

	// 提取源文件名（不含扩展名）
	baseName := filepath.Base(sourcePath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)

	// 生成带时间戳的备份文件名
	timestamp := time.Now().Format("20060102_150405")
	return filepath.Join(backupDir, fmt.Sprintf("%s_%s.db", timestamp, nameWithoutExt))
}

func (itself *Connect) BackupSQLiteHandle() {
	if !itself.IsSqlite() || itself.Connect == nil {
		return
	}
	if !preferences.GetBool("db.backupSqlite", false) {
		return
	}
	backupDir := getBackUpDir()
	backupPath := itself.GenerateBackupPath(backupDir)
	slog.Info("backupDir DirExistOrCreate", "err", fileopt.DirExistOrCreate(backupDir))
	err := backupSQLite(itself.Connect, backupPath)
	slog.Info("backupSQLite", "err", err)
	keep := max(preferences.GetInt("db.keep", 7), 1)
	cleanOldBackups(itself.Config.DbPath, keep)
}

// Scrub only a private backup copy, then publish it atomically. VACUUM after
// deletion removes raw records from free pages as well as logical tables.
func backupSQLite(source *gorm.DB, backupPath string) error {
	dir, err := os.MkdirTemp(filepath.Dir(backupPath), ".feed-backup-")
	if err != nil {
		return err
	}
	defer func() {
		if cleanupErr := os.RemoveAll(dir); cleanupErr != nil {
			slog.Warn("remove temporary feed backup directory", "error", cleanupErr)
		}
	}()
	temporary := filepath.Join(dir, "copy.db")
	if err = source.Exec("VACUUM main INTO ?", temporary).Error; err != nil {
		return err
	}
	copyDB, err := gorm.Open(sqlite.Open(temporary), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := copyDB.DB()
	if err != nil {
		return err
	}
	defer func() { _ = sqlDB.Close() }()
	err = copyDB.Transaction(func(tx *gorm.DB) error {
		for _, table := range append(append([]string{}, feed.RawTables...), "feed_state", "topic_rank_schedule") {
			if tx.Migrator().HasTable(table) {
				if e := tx.Exec("DELETE FROM " + table).Error; e != nil {
					return e
				}
			}
		}
		if tx.Migrator().HasColumn("topics", "rank_ready") {
			return tx.Table("topics").Where("1 = 1").UpdateColumn("rank_ready", false).Error
		}
		return nil
	})
	if err != nil {
		return err
	}
	if err = copyDB.Exec("VACUUM").Error; err != nil {
		return err
	}
	if err = sqlDB.Close(); err != nil {
		return err
	}
	if err = os.Chmod(temporary, 0600); err != nil {
		return err
	}
	return os.Rename(temporary, backupPath)
}

func cleanOldBackups(sourcePath string, keep int) {
	// 获取同源的所有备份文件
	// 提取源文件名（不含扩展名）
	baseName := filepath.Base(sourcePath)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)
	searchFileName := filepath.Join(getBackUpDir(), fmt.Sprintf("*%s*.db", nameWithoutExt))
	files, err := filepath.Glob(searchFileName)
	if err != nil {
		slog.Error("cleanOldBackups err", "err", err)
		return
	}

	// 按修改时间排序（旧文件在前）
	slices.SortFunc(files, func(a, b string) int {
		infoA, _ := os.Stat(a)
		infoB, _ := os.Stat(b)
		if infoA.ModTime().Before(infoB.ModTime()) {
			return -1
		}
		if infoA.ModTime().After(infoB.ModTime()) {
			return 1
		}
		return 0
	})

	// 删除超量旧备份
	if len(files) > keep {
		for _, f := range files[:len(files)-keep] {
			if err = os.Remove(f); err != nil {
				slog.Error("cleanOldBackups 删除失败", "file", f, "err", err)
			}
		}
	}
}

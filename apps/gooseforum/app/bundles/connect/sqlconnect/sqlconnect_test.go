package sqlconnect

import (
	"testing"
	"testing/synctest"
	"time"

	"gorm.io/gorm/logger"
)

func TestInMemoryTestDatabaseSurvivesLongRunningSuite(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		conn := GetConnect(TestConfig())
		if conn.Error != nil {
			t.Fatal(conn.Error)
		}
		// SQL errors must not start a process-wide log worker inside this bubble.
		conn.Connect.Logger = conn.Connect.Logger.LogMode(logger.Silent)
		pool, err := conn.Connect.DB()
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := pool.Close(); err != nil {
				t.Error(err)
			}
		}()
		if err := conn.Connect.Exec("CREATE TABLE lifetime_fixture (value INTEGER)").Error; err != nil {
			t.Fatal(err)
		}
		if err := conn.Connect.Exec("INSERT INTO lifetime_fixture VALUES (7)").Error; err != nil {
			t.Fatal(err)
		}
		// Virtual time reproduces a suite lasting over one minute without a
		// minute-long test delay. Closing the only connection destroys :memory:.
		time.Sleep(61 * time.Second)
		var value int
		if err := conn.Connect.Raw("SELECT value FROM lifetime_fixture").Scan(&value).Error; err != nil {
			t.Fatalf("long-running suite lost its in-memory database: %v", err)
		}
		if value != 7 {
			t.Fatalf("fixture data = %d, want 7", value)
		}
	})
}

func TestTestConfigUsesSingleConnectionInMemorySQLite(t *testing.T) {
	cfg := TestConfig()

	if cfg.Connection != "sqlite" {
		t.Fatalf("Connection = %q, want sqlite", cfg.Connection)
	}
	if cfg.DbPath != ":memory:" {
		t.Fatalf("DbPath = %q, want :memory:", cfg.DbPath)
	}
	if cfg.MaxOpenConnections != 1 {
		t.Fatalf("MaxOpenConnections = %d, want 1", cfg.MaxOpenConnections)
	}
	if cfg.MaxIdleConnections != 1 {
		t.Fatalf("MaxIdleConnections = %d, want 1", cfg.MaxIdleConnections)
	}
}

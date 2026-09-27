package notificationservice

import (
	"context"
	"fmt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"os"
	"testing"
	"time"
)

func TestNotificationsWaitForConcurrentBlockOnPostgreSQL(t *testing.T) {
	for _, batch := range []bool{false, true} {
		t.Run(fmt.Sprint(batch), func(t *testing.T) {
			open := notificationPostgreSQL(t)
			conn := open("notifications-monitor")
			writerName := fmt.Sprintf("notification-writer-%d", time.Now().UnixNano())
			writer := open(writerName)
			if err := conn.AutoMigrate(&users.EntityComplete{}, &users.BlockEntity{}, &eventNotification.Entity{}); err != nil {
				t.Fatal(err)
			}
			if err := conn.Create(&[]users.EntityComplete{{Id: 1, Username: "actor"}, {Id: 2, Username: "blocked"}, {Id: 3, Username: "allowed"}}).Error; err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			block := conn.WithContext(ctx).Begin()
			defer func() { _ = block.Rollback().Error }()
			if err := users.LockInteractionUsers(block, 1, 2); err != nil {
				t.Fatal(err)
			}
			if err := block.Create(&users.BlockEntity{OwnerID: 2, TargetUserID: 1}).Error; err != nil {
				t.Fatal(err)
			}
			notifications := []*eventNotification.Entity{{UserId: 2, EventType: eventNotification.EventTypeMention, Payload: eventNotification.NotificationPayload{ActorId: 1}}}
			if batch {
				notifications = append(notifications, &eventNotification.Entity{UserId: 3, EventType: eventNotification.EventTypeMention, Payload: eventNotification.NotificationPayload{ActorId: 1}})
			}
			type outcome struct {
				rows []*eventNotification.Entity
				err  error
			}
			done := make(chan outcome, 1)
			go func() {
				rows, err := persistInteractions(writer.WithContext(ctx), notifications)
				done <- outcome{rows, err}
			}()
			for {
				var waiting int64
				if err := conn.Raw("SELECT count(*) FROM pg_stat_activity WHERE application_name = ? AND wait_event_type = 'Lock'", writerName).Scan(&waiting).Error; err != nil {
					t.Fatal(err)
				}
				if waiting > 0 {
					break
				}
				select {
				case result := <-done:
					t.Fatalf("notification did not wait for block commit: %d rows, %v", len(result.rows), result.err)
				case <-ctx.Done():
					t.Fatal("notification never reached participant lock")
				case <-time.After(10 * time.Millisecond):
				}
			}
			if err := block.Commit().Error; err != nil {
				t.Fatal(err)
			}
			select {
			case result := <-done:
				if result.err != nil {
					t.Fatal(result.err)
				}
				want := 0
				if batch {
					want = 1
				}
				if len(result.rows) != want {
					t.Fatalf("committed rows=%d want=%d", len(result.rows), want)
				}
			case <-ctx.Done():
				t.Fatal("notification did not finish after block commit")
			}
			var count int64
			if err := conn.Model(&eventNotification.Entity{}).Where("user_id = ?", 2).Count(&count).Error; err != nil || count != 0 {
				t.Fatalf("blocked notification persisted: %d, %v", count, err)
			}
		})
	}
}

func notificationPostgreSQL(t *testing.T) func(string) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set")
	}
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	admin := stdlib.OpenDB(*config)
	schema := fmt.Sprintf("notification_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = admin.Exec("DROP SCHEMA " + schema + " CASCADE"); _ = admin.Close() })
	return func(name string) *gorm.DB {
		cfg := config.Copy()
		cfg.RuntimeParams["search_path"] = schema
		cfg.RuntimeParams["application_name"] = name
		cfg.RuntimeParams["statement_timeout"] = "10000"
		pool := stdlib.OpenDB(*cfg)
		pool.SetMaxOpenConns(4)
		conn, err := gorm.Open(postgres.New(postgres.Config{Conn: pool}), &gorm.Config{TranslateError: true})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = pool.Close() })
		return conn
	}
}

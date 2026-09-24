package chatservice

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/imConversations"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/imUserChatConfigs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/messages"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Each test owns a schema and distinct pools so blocked transactions cannot
// accidentally share the only connection or affect other PostgreSQL fixtures.
func chatPostgreSQL(t *testing.T) func(string) *gorm.DB {
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
	schema := fmt.Sprintf("chat_test_%d", time.Now().UnixNano())
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

func TestChatReadStatesConsistentOnPostgreSQL(t *testing.T) {
	for _, pauseTable := range []string{"im_user_chat_configs", "messages"} {
		t.Run("after_"+pauseTable, func(t *testing.T) {
			open := chatPostgreSQL(t)
			reader, monitor := open("chat-read-state"), open("chat-read-monitor")
			writerName := fmt.Sprintf("chat-write-%d", time.Now().UnixNano())
			writer := open(writerName)
			if err := reader.AutoMigrate(&imConversations.Entity{}, &imUserChatConfigs.Entity{}, &messages.Entity{}); err != nil {
				t.Fatal(err)
			}
			if err := reader.Create(&imConversations.Entity{Id: 1, Type: 1, LastMsgTime: time.Now()}).Error; err != nil {
				t.Fatal(err)
			}
			if err := reader.Create(&imUserChatConfigs.Entity{UserId: 2, PeerId: 1, ConvId: 1, UnreadCount: 1}).Error; err != nil {
				t.Fatal(err)
			}
			if err := reader.Create(&messages.Entity{Id: 1, ConvId: 1, SenderId: 1, Content: "incoming", MsgType: 1}).Error; err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			paused, resume := make(chan struct{}), make(chan struct{})
			var once sync.Once
			const callback = "test:pause_read_state_snapshot"
			if err := reader.Callback().Query().After("gorm:query").Register(callback, func(tx *gorm.DB) {
				if tx.Statement.Table == pauseTable {
					once.Do(func() {
						close(paused)
						select {
						case <-resume:
						case <-ctx.Done():
						}
					})
				}
			}); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = reader.Callback().Query().Remove(callback) })
			type readOutcome struct {
				result *MessageReadStatesResult
				err    error
			}
			readDone := make(chan readOutcome, 1)
			go func() {
				result, err := getMessageReadStates(reader.WithContext(ctx), 2, 1, []uint64{1})
				readDone <- readOutcome{result, err}
			}()
			select {
			case <-paused:
			case <-ctx.Done():
				t.Fatal("read did not reach the coordinated query")
			}
			writeDone := make(chan error, 1)
			go func() { _, err := markVisibleRead(writer.WithContext(ctx), 2, 1, []uint64{1}); writeDone <- err }()
			// Observe either a committed writer (the old READ COMMITTED race) or its
			// actual PostgreSQL lock wait. No scheduling sleep decides the assertion.
			writerFinished := false
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()
		awaitWriter:
			for {
				select {
				case err := <-writeDone:
					if err != nil {
						t.Fatal(err)
					}
					writerFinished = true
					break awaitWriter
				case <-ctx.Done():
					t.Fatal("writer neither committed nor waited for the conversation lock")
				case <-ticker.C:
					var blocked bool
					if err := monitor.Raw("SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE application_name = ? AND wait_event_type = 'Lock')", writerName).Scan(&blocked).Error; err != nil {
						t.Fatal(err)
					}
					if blocked {
						break awaitWriter
					}
				}
			}
			close(resume)
			outcome := <-readDone
			if !writerFinished {
				if err := <-writeDone; err != nil {
					t.Fatal(err)
				}
			}
			if outcome.err != nil {
				t.Fatal(outcome.err)
			}
			if len(outcome.result.Items) != 1 || outcome.result.UnreadCount != uint(1-outcome.result.Items[0].IsRead) {
				t.Fatalf("read flags and unread counter must describe one snapshot: %+v", outcome.result)
			}
			current, err := getMessageReadStates(reader, 2, 1, []uint64{1})
			if err != nil || current.Items[0].IsRead != 1 || current.UnreadCount != 0 {
				t.Fatalf("next read must see the completed write: %+v %v", current, err)
			}
		})
	}
}

func TestChatCounterInterleavingOnPostgreSQL(t *testing.T) {
	open := chatPostgreSQL(t)
	conn := open("chat-counter-interleaving")
	if err := conn.AutoMigrate(&imConversations.Entity{}, &imUserChatConfigs.Entity{}, &messages.Entity{}); err != nil {
		t.Fatal(err)
	}
	conv := imConversations.Entity{Type: 1, LastMsgTime: time.Now()}
	if err := conn.Create(&conv).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&[]imUserChatConfigs.Entity{
		{UserId: 1, PeerId: 2, ConvId: conv.Id},
		{UserId: 2, PeerId: 1, ConvId: conv.Id, UnreadCount: 1},
	}).Error; err != nil {
		t.Fatal(err)
	}
	initial := messages.Entity{ConvId: conv.Id, SenderId: 1, Content: "first", MsgType: 1}
	if err := conn.Create(&initial).Error; err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	const sends = 12
	start, done := make(chan struct{}), make(chan error, 2*sends)
	for range sends {
		go func() { <-start; _, err := sendMessage(conn.WithContext(ctx), 1, 2, "new", 1); done <- err }()
		go func() {
			<-start
			_, err := markVisibleRead(conn.WithContext(ctx), 2, conv.Id, []uint64{initial.Id})
			done <- err
		}()
	}
	close(start)
	for range 2 * sends {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	var config imUserChatConfigs.Entity
	if err := conn.Where("conv_id = ? AND user_id = ?", conv.Id, 2).First(&config).Error; err != nil {
		t.Fatal(err)
	}
	var incoming int64
	if err := conn.Model(&messages.Entity{}).Where("conv_id = ? AND sender_id = 1 AND is_read = 0", conv.Id).Count(&incoming).Error; err != nil {
		t.Fatal(err)
	}
	if config.UnreadCount != sends || incoming != sends {
		t.Fatalf("counter=%d, unread rows=%d, want %d after one effective acknowledgement and concurrent sends", config.UnreadCount, incoming, sends)
	}
}

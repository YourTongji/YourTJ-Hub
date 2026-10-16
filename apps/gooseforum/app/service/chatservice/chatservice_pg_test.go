package chatservice

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/imConversations"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/imUserChatConfigs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/messages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/pushpolicy"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestChatLongMessageOnPostgreSQL(t *testing.T) {
	open := chatPostgreSQL(t)
	conn := open("chat-long-message")
	if err := conn.AutoMigrate(&users.EntityComplete{}, &users.BlockEntity{}, &imConversations.Entity{}, &imUserChatConfigs.Entity{}, &messages.Entity{}); err != nil {
		t.Fatal(err)
	}
	for _, content := range []string{
		strings.Repeat("a", 255),
		strings.Repeat("a", 256),
		strings.Repeat("中", 255),
		strings.Repeat("🙂", 256),
		strings.Repeat("[:sticker:Atari哭哭:]\n", 18),
		strings.Repeat("长消息", 2000),
	} {
		convID, err := sendMessage(conn, 1, 2, content, 1)
		if err != nil {
			t.Fatalf("send %d characters: %v", utf8.RuneCountInString(content), err)
		}
		var message messages.Entity
		if err := conn.Where("conv_id = ?", convID).Order("id DESC").First(&message).Error; err != nil {
			t.Fatal(err)
		}
		if message.Content != content {
			t.Fatal("full message body was truncated")
		}
		var conversation imConversations.Entity
		if err := conn.First(&conversation, convID).Error; err != nil {
			t.Fatal(err)
		}
		if !utf8.ValidString(conversation.LastMsgContent) || utf8.RuneCountInString(conversation.LastMsgContent) > 255 {
			t.Fatal("preview exceeds storage limit or splits a Unicode character")
		}
		if utf8.RuneCountInString(content) <= 255 && conversation.LastMsgContent != content {
			t.Fatal("short preview changed")
		}
		if strings.HasPrefix(content, "[:sticker:") && strings.Count(conversation.LastMsgContent, "[:sticker:") != strings.Count(conversation.LastMsgContent, ":]") {
			t.Fatal("preview split a sticker token")
		}
	}
}

func TestChatRetryKeysUpgradeAndConcurrencyOnPostgreSQL(t *testing.T) {
	open := chatPostgreSQL(t)
	conn := open("chat-retry-key-upgrade")
	if err := conn.AutoMigrate(&users.EntityComplete{}, &users.BlockEntity{}, &imConversations.Entity{}, &imUserChatConfigs.Entity{}, &messages.Entity{}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&[]users.EntityComplete{{Id: 1, Username: "sender"}, {Id: 2, Username: "peer"}}).Error; err != nil {
		t.Fatal(err)
	}
	var convID uint64
	for range 2 {
		var err error
		convID, err = sendMessage(conn, 1, 2, "legacy message", 1)
		if err != nil {
			t.Fatal(err)
		}
	}
	// Reproduce the deployed schema: existing messages have no retry-key column.
	if err := conn.Migrator().DropColumn(&messages.Entity{}, "client_message_id"); err != nil {
		t.Fatal(err)
	}
	if err := conn.AutoMigrate(&messages.Entity{}); err != nil {
		t.Fatal(err)
	}
	var legacy int64
	if err := conn.Model(&messages.Entity{}).Where("client_message_id IS NULL").Count(&legacy).Error; err != nil || legacy != 2 {
		t.Fatalf("legacy rows must retain distinct NULL keys: %d, %v", legacy, err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	const attempts = 12
	start, done := make(chan struct{}), make(chan error, attempts)
	for range attempts {
		go func() {
			<-start
			id, err := sendMessage(conn.WithContext(ctx), 1, 2, "retried", 1, "one-intent")
			if err == nil && id != convID {
				err = fmt.Errorf("retry returned conversation %d, want %d", id, convID)
			}
			done <- err
		}()
	}
	close(start)
	for range attempts {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	var count int64
	if err := conn.Model(&messages.Entity{}).Count(&count).Error; err != nil || count != 3 {
		t.Fatalf("concurrent retries must add one message: count=%d, err=%v", count, err)
	}
	var peer imUserChatConfigs.Entity
	if err := conn.Where("conv_id = ? AND user_id = 2", convID).First(&peer).Error; err != nil {
		t.Fatal(err)
	}
	if peer.UnreadCount != 3 {
		t.Fatalf("retries inflated unread count to %d", peer.UnreadCount)
	}
}

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
			if err := reader.AutoMigrate(&users.EntityComplete{}, &users.BlockEntity{}, &imConversations.Entity{}, &imUserChatConfigs.Entity{}, &messages.Entity{}); err != nil {
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
	if err := conn.AutoMigrate(&users.EntityComplete{}, &users.BlockEntity{}, &imConversations.Entity{}, &imUserChatConfigs.Entity{}, &messages.Entity{}); err != nil {
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

// Fixed base64url keys of the exact byte lengths webpush's Enabled() requires
// (87-char / 65-byte P-256 public key, 43-char / 32-byte private key). They are
// not a valid cryptographic pair; the enqueue path only checks length/encoding,
// so the web push channel is exercised without any provider traffic.
const (
	chatTestVapidPublicKey  = "CxIZICcuNTxDSlFYX2ZtdHuCiZCXnqWss7rByM_W3eTr8vkABw4VHCMqMTg_Rk1UW2JpcHd-hYyTmqGor7a9xMs"
	chatTestVapidPrivateKey = "HSQrMjlAR05VXGNqcXh_ho2Um6KpsLe-xczT2uHo7_Y"
)

func TestChatPushOutboxFailureIsolationOnPostgreSQL(t *testing.T) {
	conn := chatPostgreSQL(t)("chat-push-failure-isolation")
	if err := conn.AutoMigrate(&users.EntityComplete{}, &users.BlockEntity{}, &imConversations.Entity{}, &imUserChatConfigs.Entity{}, &messages.Entity{}, &taskQueue.Entity{}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&[]users.EntityComplete{{Id: 1, Username: "sender"}, {Id: 2, Username: "peer"}}).Error; err != nil {
		t.Fatal(err)
	}
	for _, kv := range []struct{ key, val string }{
		{"push.jpush.app_key", "test-only"},
		{"push.jpush.master_secret", "test-only"},
		{"webpush.vapid_public_key", chatTestVapidPublicKey},
		{"webpush.vapid_private_key", chatTestVapidPrivateKey},
	} {
		old := preferences.GetRaw(kv.key)
		preferences.Set(kv.key, kv.val)
		t.Cleanup(func() { preferences.Set(kv.key, old) })
	}
	if err := conn.Exec("ALTER TABLE task_queue ADD CONSTRAINT test_reject_native_push CHECK (type NOT LIKE 'nativepush.%')").Error; err != nil {
		t.Fatal(err)
	}
	failedInsert := false
	const callback = "test:observe_failed_native_push"
	if err := conn.Callback().Create().After("gorm:create").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "task_queue" && tx.Error != nil {
			failedInsert = true
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Callback().Create().Remove(callback) })
	convID, err := sendMessageWithEffects(conn, 1, 2, "message survives native push failure", 1, false, "failed-native-outbox")
	if err != nil {
		t.Fatalf("native push failure aborted message transaction: %v", err)
	}
	if !failedInsert {
		t.Fatal("native push SQL insert failure was not exercised")
	}
	var message messages.Entity
	if err := conn.Where("sender_id = 1 AND client_message_id = ?", "failed-native-outbox").First(&message).Error; err != nil {
		t.Fatal(err)
	}
	var peer imUserChatConfigs.Entity
	if err := conn.Where("conv_id = ? AND user_id = 2", convID).First(&peer).Error; err != nil || peer.UnreadCount != 1 {
		t.Fatalf("membership/unread did not commit despite native push failure: %+v %v", peer, err)
	}
	var webCount, nativeCount int64
	if err := conn.Model(&taskQueue.Entity{}).Where("type LIKE 'webpush.%'").Count(&webCount).Error; err != nil || webCount != 1 {
		t.Fatalf("native push failure rolled back web task too: web=%d err=%v", webCount, err)
	}
	if err := conn.Model(&taskQueue.Entity{}).Where("type LIKE 'nativepush.%'").Count(&nativeCount).Error; err != nil || nativeCount != 0 {
		t.Fatalf("failed native outbox left %d native tasks: %v", nativeCount, err)
	}
	if err := conn.Exec("ALTER TABLE task_queue DROP CONSTRAINT test_reject_native_push").Error; err != nil {
		t.Fatal(err)
	}
	for range 2 {
		id, err := sendMessageWithEffects(conn, 1, 2, "successful outbox", 1, false, "successful-outbox")
		if err != nil || id != convID {
			t.Fatalf("success/replay: conversation=%d err=%v", id, err)
		}
	}
	if err := conn.Model(&taskQueue.Entity{}).Where("type LIKE 'webpush.%'").Count(&webCount).Error; err != nil || webCount != 2 {
		t.Fatalf("success/replay produced %d web tasks, want 2: %v", webCount, err)
	}
	if err := conn.Model(&taskQueue.Entity{}).Where("type LIKE 'nativepush.%'").Count(&nativeCount).Error; err != nil || nativeCount != 1 {
		t.Fatalf("success/replay produced %d native tasks, want 1: %v", nativeCount, err)
	}
	var task taskQueue.Entity
	if err := conn.Where("type LIKE 'nativepush.%'").First(&task).Error; err != nil {
		t.Fatal(err)
	}
	var payload pushpolicy.Task
	if err := json.Unmarshal([]byte(task.TaskJson), &payload); err != nil {
		t.Fatal(err)
	}
	message = messages.Entity{}
	if err := conn.Where("sender_id = 1 AND client_message_id = ?", "successful-outbox").First(&message).Error; err != nil {
		t.Fatal(err)
	}
	if payload.SourceKind != pushpolicy.ChatMessage || payload.SourceId != message.Id || payload.UserId != 2 || task.DedupKey == nil {
		t.Fatalf("wrong committed source identity: %+v", task)
	}
	if err := conn.Where("conv_id = ? AND user_id = 2", convID).First(&peer).Error; err != nil || peer.UnreadCount != 2 {
		t.Fatalf("replay changed unread count: %+v %v", peer, err)
	}
}

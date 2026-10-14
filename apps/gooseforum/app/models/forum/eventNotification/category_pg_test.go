package eventNotification

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestNotificationCategoriesOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set")
	}
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	admin := stdlib.OpenDB(*config)
	schema := fmt.Sprintf("notification_category_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = admin.Exec("DROP SCHEMA " + schema + " CASCADE"); _ = admin.Close() })
	config.RuntimeParams["search_path"] = schema
	pool := stdlib.OpenDB(*config)
	t.Cleanup(func() { _ = pool.Close() })
	conn, err := gorm.Open(postgres.New(postgres.Config{Conn: pool}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Exec("CREATE TABLE event_notification (id BIGSERIAL PRIMARY KEY, user_id BIGINT, event_type VARCHAR(16), payload JSON, is_read BOOLEAN DEFAULT FALSE)").Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Exec("INSERT INTO event_notification (user_id,event_type,payload) VALUES (1,'post_reply','{}')").Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.AutoMigrate(&Entity{}); err != nil {
		t.Fatal(err)
	}
	if !conn.Migrator().HasIndex(&Entity{}, "idx_user_mention_id") {
		t.Fatal("missing user/mention/id index")
	}
	var old Entity
	if err := conn.First(&old, 1).Error; err != nil || old.HasDirectMention || old.Payload.HasDirectMention {
		t.Fatalf("historical reason was invented: %+v %v", old, err)
	}
	rows := []Entity{
		{Id: 2, UserId: 1, EventType: EventTypePostReply, HasDirectMention: true, Payload: NotificationPayload{HasDirectMention: true}},
		{Id: 3, UserId: 1, EventType: EventTypeComment},
		{Id: 4, UserId: 1, EventType: EventTypeLike},
		{Id: 5, UserId: 1, EventType: EventTypeMention, IsRead: true},
		{Id: 6, UserId: 1, EventType: "legacy"},
		{Id: 7, UserId: 2, EventType: EventTypePostReply, HasDirectMention: true},
		{Id: 8, UserId: 1, EventType: EventTypeWikiUpdated},
	}
	if err := conn.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		category string
		unread   bool
		cursor   uint64
		want     uint64
	}{{"mentions", true, 0, 2}, {"mentions", false, 0, 5}, {"mentions", false, 5, 2}, {"replies", true, 0, 3}, {"replies", true, 3, 1}, {"likes", true, 0, 4}, {"following", true, 0, 8}, {"system", true, 0, 6}}
	for _, tt := range tests {
		got, err := queryByUserId(conn, 1, 1, tt.cursor, tt.unread, tt.category)
		if err != nil || len(got) != 1 || got[0].Id != tt.want {
			t.Fatalf("%s unread=%v cursor=%d got=%+v err=%v", tt.category, tt.unread, tt.cursor, got, err)
		}
	}
}

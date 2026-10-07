package posts

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type seenSQLRecorder struct {
	logger.Interface
	sql string
}

func (r *seenSQLRecorder) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	sql, _ := fc()
	if strings.Contains(sql, "marks") {
		r.sql = sql
	}
	r.Interface.Trace(ctx, begin, fc, err)
}
func TestNewPublicReplyCutoffsOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set")
	}
	log := &seenSQLRecorder{Interface: logger.Default.LogMode(logger.Silent)}
	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: log})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := conn.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	schema := fmt.Sprintf("feed_seen_%d", time.Now().UnixNano())
	if err := conn.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Exec("SET search_path TO " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Exec("SET jit = off").Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		conn.Exec("SET search_path TO public")
		conn.Exec("DROP SCHEMA " + schema + " CASCADE")
		if err := sqlDB.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := conn.AutoMigrate(&Entity{}, &users.EntityComplete{}, &users.BlockEntity{}, &feed.SeenState{}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []uint64{1, 2} {
		if err := conn.Create(&users.EntityComplete{Id: id, Username: fmt.Sprintf("u%d", id), Email: fmt.Sprintf("u%d@invalid.test", id)}).Error; err != nil {
			t.Fatal(err)
		}
	}
	at := time.Now().UTC().Add(-time.Minute)
	cutoffs := map[uint64]time.Time{}
	rows := []Entity{}
	for id := uint64(1); id <= 300; id++ {
		cutoffs[id] = at
		for no := uint64(2); no <= 21; no++ {
			user := uint64(1)
			if no == 21 {
				user = 2
			}
			public := at.Add(time.Second)
			rows = append(rows, Entity{TopicId: id, UserId: user, PostNo: no, FirstPublicAt: &public})
		}
	}
	if err := conn.CreateInBatches(&rows, 100).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Exec("ANALYZE posts").Error; err != nil {
		t.Fatal(err)
	}
	durations := []time.Duration{}
	for i := 0; i < 20; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		start := time.Now()
		reply, err := newPublicRepliesAmong(conn.WithContext(ctx), 1, cutoffs)
		durations = append(durations, time.Since(start))
		cancel()
		if err != nil {
			t.Fatalf("%v: %s", err, log.sql[max(0, len(log.sql)-1600):])
		}
		if len(reply) != 300 {
			t.Fatalf("eligible replies=%d", len(reply))
		}
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	t.Logf("synthetic 300 cutoffs/6000 replies, one SELECT per call, p95=%s", durations[18])
	var plan []struct {
		Text string `gorm:"column:QUERY PLAN"`
	}
	if err := conn.Raw("EXPLAIN (ANALYZE, BUFFERS) " + log.sql).Scan(&plan).Error; err != nil {
		t.Fatal(err)
	}
	usedIndex := false
	for _, line := range plan {
		if strings.Contains(line.Text, "posts_topic") {
			usedIndex = true
		}
		if strings.Contains(line.Text, "Index") || strings.Contains(line.Text, "Execution Time") {
			t.Log(line.Text)
		}
	}
	if !usedIndex {
		t.Fatal("bounded reply query did not use a topic index")
	}
	// Composite upsert SQL and zero-increment replay are exercised on PG too.
	state := feed.SeenState{UserID: 1, TopicID: 1, LastSeenAt: at, SeenContentAt: at, LastSeenProofID: "one", ExpiresAt: at.Add(30 * 24 * time.Hour)}
	if err := feed.UpsertSeenTx(conn, []feed.SeenState{state}); err != nil {
		t.Fatal(err)
	}
	state.LastSeenAt = at.Add(time.Second)
	state.ExpiresAt = state.ExpiresAt.Add(time.Second)
	if err := feed.UpsertSeenTx(conn, []feed.SeenState{state}); err != nil {
		t.Fatal(err)
	}
	var saved feed.SeenState
	if err := conn.First(&saved, "user_id = ? AND topic_id = ?", 1, 1).Error; err != nil {
		t.Fatal(err)
	}
	if !saved.LastSeenAt.Equal(at.Truncate(time.Microsecond)) && saved.LastSeenAt.Sub(at).Abs() > time.Microsecond {
		t.Fatal("PG replay extended last seen")
	}
}

func TestNewPublicReplyCutoffsAndQueryPlanOnSQLite(t *testing.T) {
	log := &seenSQLRecorder{Interface: logger.Default.LogMode(logger.Silent)}
	conn, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: log})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := conn.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := conn.AutoMigrate(&Entity{}, &users.EntityComplete{}, &users.BlockEntity{}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&users.EntityComplete{Id: 2, Username: "peer", Email: "peer@invalid.test"}).Error; err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 7, 8, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	old, fresh, immediate := at.Add(-time.Second), at.Add(time.Second), at.Add(100*time.Microsecond)
	if err := conn.Create(&[]Entity{{TopicId: 31, UserId: 2, PostNo: 2, FirstPublicAt: &old}, {TopicId: 32, UserId: 2, PostNo: 2, FirstPublicAt: &fresh}, {TopicId: 33, UserId: 2, PostNo: 2, FirstPublicAt: &immediate}}).Error; err != nil {
		t.Fatal(err)
	}
	result, err := newPublicRepliesAmong(conn, 1, map[uint64]time.Time{31: at.UTC(), 32: at.UTC(), 33: at.UTC()})
	if err != nil {
		t.Fatal(err)
	}
	if result[31] || !result[32] || !result[33] {
		t.Fatalf("timezone-normalized cutoffs: %v", result)
	}
	var plan []struct{ Detail string }
	if err := conn.Raw("EXPLAIN QUERY PLAN " + log.sql).Scan(&plan).Error; err != nil {
		t.Fatal(err)
	}
	usedIndex := false
	for _, row := range plan {
		t.Log(row.Detail)
		if strings.Contains(row.Detail, "SEARCH posts USING INDEX idx_posts_topic_") {
			usedIndex = true
		}
	}
	if !usedIndex {
		t.Fatal("reply query must start from a per-topic index probe")
	}
}

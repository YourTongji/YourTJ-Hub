package campusservice

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/campus"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pointsRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userPoints"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userStatistics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestSchoolLoginConcurrentRegistrationOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set")
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	admin := stdlib.OpenDB(*cfg)
	defer func() { _ = admin.Close() }()
	schema := fmt.Sprintf("school_login_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec("DROP SCHEMA " + schema + " CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	cfg.RuntimeParams["search_path"] = schema
	sql := stdlib.OpenDB(*cfg)
	defer func() { _ = sql.Close() }()
	conn, err := gorm.Open(postgres.New(postgres.Config{Conn: sql}), &gorm.Config{TranslateError: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.AutoMigrate(&users.EntityComplete{}, &campus.Binding{}, &userPoints.Entity{}, &pointsRecord.Entity{}, &userStatistics.Entity{}); err != nil {
		t.Fatal(err)
	}
	s := New(Config{EncryptionKey: strings.Repeat("e", 32), IdentityKey: strings.Repeat("i", 32)}, campus.Store{DB: conn}, &fakeProvider{id: "2359999"})
	policy := pageConfig.SecurityAndRegistration{EnableSignup: true, MaxDailySignups: -1}
	var wg sync.WaitGroup
	for range 8 {
		state, browser, err := s.StartLogin("/", "en")
		if err != nil {
			t.Fatal(err)
		}
		wg.Go(func() { _, _ = s.Login(context.Background(), browser, state, "code", policy) })
	}
	wg.Wait()
	for _, model := range []any{&users.EntityComplete{}, &campus.Binding{}, &userPoints.Entity{}, &pointsRecord.Entity{}, &userStatistics.Entity{}} {
		var count int64
		if err := conn.Model(model).Count(&count).Error; err != nil || count != 1 {
			t.Fatalf("%T count=%d error=%v", model, count, err)
		}
	}
	result, err := signIn(t, s, policy)
	if err != nil || result.Created || result.User.IsActivated != users.ActivationSuccess {
		t.Fatalf("existing login failed: %v", err)
	}
}

package oauthservice

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userOAuth"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestUnbindPreservesLastLoginMethodOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set")
	}
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	admin := stdlib.OpenDB(*config)
	defer func() { _ = admin.Close() }()
	schema := fmt.Sprintf("oauth_unbind_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = admin.Exec("DROP SCHEMA " + schema + " CASCADE") }()
	config.RuntimeParams["search_path"] = schema
	pool := stdlib.OpenDB(*config)
	defer func() { _ = pool.Close() }()
	conn, err := gorm.Open(postgres.New(postgres.Config{Conn: pool}), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.AutoMigrate(&users.EntityComplete{}, &userOAuth.Entity{}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&users.EntityComplete{Id: 1, Username: "oauth-only"}).Error; err != nil {
		t.Fatal(err)
	}
	for _, provider := range []string{ProviderGitHub, ProviderGoogle} {
		if err := conn.Create(&userOAuth.Entity{UserId: 1, Provider: provider, ProviderUid: provider + "-identity"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, provider := range []string{ProviderGitHub, ProviderGoogle} {
		wg.Go(func() { <-start; results <- unbindOAuth(ctx, conn, 1, provider) })
	}
	close(start)
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	var count int64
	if err := conn.Model(&userOAuth.Entity{}).Where("user_id = ?", 1).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if successes != 1 || count != 1 {
		t.Fatalf("successful unlinks=%d remaining=%d", successes, count)
	}
}

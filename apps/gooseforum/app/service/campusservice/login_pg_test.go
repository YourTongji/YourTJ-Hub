package campusservice

import (
	"context"
	"errors"
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

func loginPostgres(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set")
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	admin := stdlib.OpenDB(*cfg)
	t.Cleanup(func() { _ = admin.Close() })
	schema := fmt.Sprintf("school_login_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec("DROP SCHEMA " + schema + " CASCADE"); err != nil {
			t.Error(err)
		}
	})
	cfg.RuntimeParams["search_path"] = schema
	sql := stdlib.OpenDB(*cfg)
	t.Cleanup(func() { _ = sql.Close() })
	conn, err := gorm.Open(postgres.New(postgres.Config{Conn: sql}), &gorm.Config{TranslateError: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.AutoMigrate(&users.EntityComplete{}, &campus.Binding{}, &campus.IdentityReservation{}, &userPoints.Entity{}, &pointsRecord.Entity{}, &userStatistics.Entity{}); err != nil {
		t.Fatal(err)
	}
	return conn
}

func TestSchoolLoginConcurrentRegistrationOnPostgreSQL(t *testing.T) {
	conn := loginPostgres(t)
	s := New(Config{EncryptionKey: strings.Repeat("e", 32), IdentityKey: strings.Repeat("i", 32)}, campus.Store{DB: conn}, &fakeProvider{id: "2359999"})
	policy := pageConfig.SecurityAndRegistration{EnableSignup: true, MaxDailySignups: -1}
	var wg sync.WaitGroup
	for range 8 {
		state, browser, err := s.StartLogin("/", "en")
		if err != nil {
			t.Fatal(err)
		}
		wg.Go(func() {
			result, err := s.Login(context.Background(), browser, state, "code", policy)
			if err == nil && result.Registration != "" {
				_, _ = completeTestRegistration(s, result.Registration, policy)
			}
		})
	}
	wg.Wait()
	for _, model := range []any{&users.EntityComplete{}, &campus.Binding{}, &campus.IdentityReservation{}, &userPoints.Entity{}, &pointsRecord.Entity{}, &userStatistics.Entity{}} {
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

func TestSchoolLoginDailyQuotaOnPostgreSQL(t *testing.T) {
	for _, rollback := range []bool{false, true} {
		t.Run(fmt.Sprintf("rollback=%t", rollback), func(t *testing.T) {
			conn := loginPostgres(t)
			first := conn.Begin()
			if first.Error != nil {
				t.Fatal(first.Error)
			}
			t.Cleanup(func() { first.Rollback() })
			user := &users.EntityComplete{Username: "first", Email: "2351111@tongji.edu.cn"}
			if err := users.CreateVerifiedAccountTx(first, user, 1); err != nil {
				t.Fatal(err)
			}
			s := New(Config{EncryptionKey: strings.Repeat("e", 32), IdentityKey: strings.Repeat("i", 32)}, campus.Store{DB: conn}, &fakeProvider{})
			result := make(chan error, 1)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			go func() {
				_, _, err := s.loginAccount(ctx, Credentials{Subject: "second", StudentID: "2352222"}, "en", pageConfig.SecurityAndRegistration{EnableSignup: true, MaxDailySignups: 1}, &Registration{Username: "quota_user", PasswordHash: "test-hash"})
				result <- err
			}()
			var err error
			finished := false
			select {
			case err = <-result:
				finished = true
			case <-time.After(200 * time.Millisecond):
			}
			if rollback {
				if e := first.Rollback().Error; e != nil {
					t.Fatal(e)
				}
			} else if e := first.Commit().Error; e != nil {
				t.Fatal(e)
			}
			if !finished {
				err = <-result
			}
			if rollback && err != nil {
				t.Fatalf("rollback did not release quota: %v", err)
			}
			if !rollback && !errors.Is(err, users.ErrSignupQuota) {
				t.Fatalf("competing signup error=%v, want quota rejection", err)
			}
			var count int64
			if err := conn.Unscoped().Model(&users.EntityComplete{}).Count(&count).Error; err != nil || count != 1 {
				t.Fatalf("accounts=%d error=%v", count, err)
			}
		})
	}
}

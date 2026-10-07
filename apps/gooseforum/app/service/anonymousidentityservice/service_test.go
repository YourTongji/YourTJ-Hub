package anonymousidentityservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/anonymousnames"
	identity "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/anonymousIdentity"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setup(t *testing.T, pg bool) Service {
	t.Helper()
	var conn *gorm.DB
	var err error
	if pg {
		dsn := os.Getenv("YOURTJ_TEST_PG_URL")
		if dsn == "" {
			t.Skip("YOURTJ_TEST_PG_URL required")
		}
		admin, e := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		if e != nil {
			t.Fatal(e)
		}
		schema := fmt.Sprintf("anonymous_test_%d", time.Now().UnixNano())
		if err := admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := admin.Exec("DROP SCHEMA " + schema + " CASCADE").Error; err != nil {
				t.Error(err)
			}
			sql, err := admin.DB()
			if err != nil {
				t.Error(err)
				return
			}
			if err := sql.Close(); err != nil {
				t.Error(err)
			}
		})
		conn, err = gorm.Open(postgres.Open(dsn+" search_path="+schema), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	} else {
		conn, err = gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&_busy_timeout=5000"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	}
	if err != nil {
		t.Fatal(err)
	}
	sql, err := conn.DB()
	if err != nil {
		t.Fatal(err)
	}
	if !pg {
		sql.SetMaxOpenConns(1)
	}
	t.Cleanup(func() {
		if err := sql.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := conn.AutoMigrate(&users.EntityComplete{}, &identity.Persona{}, &identity.Binding{}, &identity.Quota{}, &identity.Batch{}, &identity.RevealAudit{}, &rolePermissionRs.Entity{}); err != nil {
		t.Fatal(err)
	}
	for i := uint64(1); i <= 3; i++ {
		if err := conn.Create(&users.EntityComplete{Id: i, Username: fmt.Sprintf("private-owner-%d", i), RoleId: i}).Error; err != nil {
			t.Fatal(err)
		}
	}
	now := time.Date(2024, 2, 29, 15, 0, 0, 0, time.UTC)
	return Service{conn, func() time.Time { return now }, func() ([]string, error) {
		return []string{"C++", "中华人民共和国道路交通安全法实施条例", "人", "数学", "星辰", "春天", "通济", "大学", "上海", "同学"}, nil
	}}
}

func exerciseCleanup(t *testing.T, s Service) {
	t.Helper()
	day, _ := anonymousnames.Day(s.Now())
	cutoff, _ := anonymousnames.Day(s.Now().Add(-7 * 24 * time.Hour))
	old, _ := anonymousnames.Day(s.Now().Add(-8 * 24 * time.Hour))
	uid := strings.Repeat("a", 32)
	rows := []any{
		&identity.Quota{OwnerID: 1, Day: old, Used: 10},
		&identity.Quota{OwnerID: 1, Day: cutoff, Used: 10},
		&identity.Quota{OwnerID: 1, Day: day, Used: 10},
		&identity.Persona{UID: uid, Name: "同学", AvatarSeed: uid},
		&identity.Binding{OwnerID: 1, PersonaUID: uid},
		&identity.RevealAudit{ActorID: 2, OwnerID: 1, PersonaUID: uid, Reason: "retained audit"},
	}
	for _, row := range rows {
		if err := s.DB.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	for range 2 {
		if err := s.Cleanup(); err != nil {
			t.Fatal(err)
		}
	}
	var quotas []identity.Quota
	if err := s.DB.Order("day").Find(&quotas).Error; err != nil || len(quotas) != 2 || quotas[0].Day != cutoff || quotas[1].Day != day || quotas[1].Used != 10 {
		t.Fatalf("quota cleanup changed active/boundary counters or retained expired rows: %+v, %v", quotas, err)
	}
	for _, model := range []any{&identity.Persona{}, &identity.Binding{}, &identity.RevealAudit{}} {
		var count int64
		if err := s.DB.Model(model).Count(&count).Error; err != nil || count != 1 {
			t.Fatalf("cleanup removed retained %T: %d, %v", model, count, err)
		}
	}
}

func TestCleanupRetainsCurrentQuotaAndPrivateEvidence(t *testing.T) {
	exerciseCleanup(t, setup(t, false))
}

func TestPostgreSQLCleanupRetainsCurrentQuotaAndPrivateEvidence(t *testing.T) {
	exerciseCleanup(t, setup(t, true))
}

func exerciseQuota(t *testing.T, s Service) {
	t.Helper()
	day, _ := anonymousnames.Day(s.Now())
	var successes atomic.Int64
	var unexpected atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := s.Generate(1, day, fmt.Sprintf("request-%03d", i))
			if err == nil {
				successes.Add(1)
			} else if !errors.Is(err, ErrQuota) {
				unexpected.Add(1)
			}
		}(i)
	}
	wg.Wait()
	if successes.Load() != 10 || unexpected.Load() != 0 {
		t.Fatalf("success=%d unexpected=%d", successes.Load(), unexpected.Load())
	}
	state, err := s.State(1)
	if err != nil {
		t.Fatal(err)
	}
	if state.Remaining != 0 || len(state.Batches) != 10 {
		t.Fatal(state)
	}
	batch := state.Batches[0]
	replay, err := s.Generate(1, day, batch.RequestKey)
	if err != nil || replay.ID != batch.ID {
		t.Fatalf("retry: %v %v", replay, err)
	}
	var ids sync.Map
	unexpected.Store(0)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, err := s.Confirm(1, batch.ID, 0)
			if err != nil {
				unexpected.Add(1)
			}
			ids.Store(p.PublicUID, true)
		}()
	}
	wg.Wait()
	count := 0
	ids.Range(func(key, value any) bool { count++; return true })
	if count != 1 || unexpected.Load() != 0 {
		t.Fatalf("personas=%d errors=%d", count, unexpected.Load())
	}
	var bindings int64
	s.DB.Model(&identity.Binding{}).Count(&bindings)
	if bindings != 1 {
		t.Fatal(bindings)
	}
	if _, err := s.Generate(1, day, "locked-request"); !errors.Is(err, ErrLocked) {
		t.Fatal(err)
	}
	if _, err := s.Confirm(2, batch.ID, 0); !errors.Is(err, ErrCandidate) {
		t.Fatal(err)
	}
	if _, err := s.Confirm(1, batch.ID, 11); !errors.Is(err, ErrCandidate) {
		t.Fatal(err)
	}
	state, err = s.State(1)
	if err != nil {
		t.Fatal(err)
	}
	if state.NameChangeAvailableAt.Day() != 28 || state.NameChangeAvailableAt.Month() != time.February {
		t.Fatal(state)
	}
	originalUID := state.Persona.PublicUID
	available := *state.NameChangeAvailableAt
	if err := s.SetDisabled(1, true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Resolve(1, "persona"); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
	if err := s.SetDisabled(1, false); err != nil {
		t.Fatal(err)
	}
	uid, err := s.Resolve(1, "persona")
	if err != nil || uid != originalUID {
		t.Fatal(uid, err)
	}
	s.Now = func() time.Time { return available }
	day, _ = anonymousnames.Day(s.Now())
	renew, err := s.Generate(1, day, "rename-request")
	if err != nil {
		t.Fatal(err)
	}
	same, err := s.Confirm(1, renew.ID, 0)
	if err != nil || same.PublicUID != originalUID {
		t.Fatal(same, err)
	}
	state, _ = s.State(1)
	if !state.NameChangeAvailableAt.Equal(available) {
		t.Fatal("no-op reset lock")
	}
	changed, err := s.Confirm(1, renew.ID, 1)
	if err != nil || changed.PublicUID != originalUID || changed.AvatarURL != same.AvatarURL {
		t.Fatal(changed, err)
	}
	state, _ = s.State(1)
	if !state.NameChangeAvailableAt.After(available) {
		t.Fatal("rename did not lock")
	}
	raw, err := json.Marshal(state.Persona)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "owner") || strings.Contains(string(raw), "seed") {
		t.Fatal(string(raw))
	}
	s.Now = func() time.Time { return available.Add(25 * time.Hour) }
	if _, err := s.Confirm(1, renew.ID, 1); !errors.Is(err, ErrCandidate) {
		t.Fatal(err)
	}
}
func TestQuotaCreationAndRename(t *testing.T)           { exerciseQuota(t, setup(t, false)) }
func TestPostgreSQLQuotaCreationAndRename(t *testing.T) { exerciseQuota(t, setup(t, true)) }
func TestGenerationFailureDoesNotCharge(t *testing.T) {
	s := setup(t, false)
	day, _ := anonymousnames.Day(s.Now())
	s.Draw = func() ([]string, error) { return nil, errors.New("entropy failed") }
	if _, err := s.Generate(1, day, "failed-request"); err == nil {
		t.Fatal("success")
	}
	state, err := s.State(1)
	if err != nil || state.Remaining != 10 || len(state.Batches) != 0 {
		t.Fatal(state, err)
	}
}

func TestInvalidGenerationRequestKeyDoesNotCharge(t *testing.T) {
	s := setup(t, false)
	day, _ := anonymousnames.Day(s.Now())
	for _, key := range []string{"", "short", strings.Repeat("x", 129)} {
		_, err := s.Generate(1, day, key)
		if err == nil || ErrorCode(err) != "common.request.invalidParams" {
			t.Fatalf("invalid request key returned %v, want invalid parameters", err)
		}
	}
	state, err := s.State(1)
	if err != nil || state.Remaining != 10 || len(state.Batches) != 0 {
		t.Fatalf("invalid request consumed quota: %+v, %v", state, err)
	}
}

func exerciseGovernanceAndClosure(t *testing.T, s Service) {
	t.Helper()
	day, _ := anonymousnames.Day(s.Now())
	b, err := s.Generate(1, day, "govern-request")
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.Confirm(1, b.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Govern(2, p.PublicUID, "abuse investigation", true); err != nil {
		t.Fatal(err)
	}
	var owner users.EntityComplete
	s.DB.First(&owner, 1)
	if !owner.AnonymousGovernanceBlocked {
		t.Fatal("main identity not restricted")
	}
	if _, err := s.Resolve(1, "persona"); err == nil {
		t.Fatal("governance bypass")
	}
	if err := s.SetDisabled(1, false); err == nil {
		t.Fatal("self reactivation bypass")
	}
	state, err := s.State(1)
	if err != nil || !state.GovernanceDisabled || state.Persona.PublicUID != p.PublicUID {
		t.Fatal(state, err)
	}
	s.DB.Model(&owner).Update("is_frozen", users.StatusFrozen)
	if err := s.Govern(2, p.PublicUID, "appeal accepted", false); err != nil {
		t.Fatal(err)
	}
	s.DB.First(&owner, 1)
	if owner.AnonymousGovernanceBlocked || owner.IsFrozen != users.StatusFrozen {
		t.Fatal("restore cleared unrelated freeze")
	}
	if _, err := s.Resolve(1, "persona"); err == nil {
		t.Fatal("frozen account wrote")
	}
	s.DB.Model(&owner).Update("is_frozen", users.StatusNormal)
	if uid, err := s.Resolve(1, "persona"); err != nil || uid != p.PublicUID {
		t.Fatal(uid, err)
	}
	if err := s.DB.Delete(&owner).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := s.Resolve(1, "persona"); err == nil {
		t.Fatal("closed account wrote")
	}
	s.DB.Create(&rolePermissionRs.Entity{RoleId: 2, PermissionId: permission.RevealAnonymousIdentity.Id(), Effective: 1})
	result, err := s.Reveal(2, p.PublicUID, "closed account audit", "")
	if err != nil || result.UserID != 1 {
		t.Fatal(result, err)
	}
	var audits int64
	s.DB.Model(&identity.RevealAudit{}).Where("persona_uid = ?", p.PublicUID).Count(&audits)
	if audits != 3 {
		t.Fatal("retained audits", audits)
	}
}
func TestGovernanceAndClosure(t *testing.T) { exerciseGovernanceAndClosure(t, setup(t, false)) }
func TestPostgreSQLGovernanceAndClosure(t *testing.T) {
	exerciseGovernanceAndClosure(t, setup(t, true))
}

func TestPersistFailureRollsBackQuota(t *testing.T) {
	s := setup(t, false)
	if err := s.DB.Callback().Create().Before("gorm:create").Register("fail-anonymous-batch", func(tx *gorm.DB) {
		if tx.Statement.Table == (identity.Batch{}).TableName() {
			_ = tx.AddError(errors.New("disk failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	day, _ := anonymousnames.Day(s.Now())
	if _, err := s.Generate(1, day, "persist-failure"); err == nil {
		t.Fatal("success")
	}
	state, err := s.State(1)
	if err != nil || state.Remaining != 10 || len(state.Batches) != 0 {
		t.Fatal(state, err)
	}
}
func TestRevealIsExplicitAuditedAndFailClosed(t *testing.T) {
	s := setup(t, false)
	day, _ := anonymousnames.Day(s.Now())
	b, err := s.Generate(1, day, "first-request")
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.Confirm(1, b.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	s.DB.Create(&rolePermissionRs.Entity{RoleId: 2, PermissionId: permission.Admin.Id(), Effective: 1})
	if _, err := s.Reveal(2, p.PublicUID, "investigation", "trace"); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
	s.DB.Create(&rolePermissionRs.Entity{RoleId: 2, PermissionId: permission.RevealAnonymousIdentity.Id(), Effective: 1})
	if _, err := s.Reveal(2, p.PublicUID, " ", "trace"); err == nil {
		t.Fatal("blank reason")
	}
	owner, err := s.Reveal(2, p.PublicUID, "investigation", "trace")
	if err != nil || owner.UserID != 1 {
		t.Fatal(owner, err)
	}
	var n int64
	s.DB.Model(&identity.RevealAudit{}).Count(&n)
	if n != 1 {
		t.Fatal(n)
	}
	if err := s.DB.Migrator().DropTable(&identity.RevealAudit{}); err != nil {
		t.Fatal(err)
	}
	owner, err = s.Reveal(2, p.PublicUID, "investigation", "trace")
	if err == nil || owner.UserID != 0 {
		t.Fatal("disclosed despite audit failure", owner, err)
	}
}

func TestGovernanceAuditFailureRollsBackBothIdentityRestrictions(t *testing.T) {
	s := setup(t, false)
	day, _ := anonymousnames.Day(s.Now())
	b, err := s.Generate(1, day, "govern-audit-failure")
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.Confirm(1, b.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.DB.Callback().Create().Before("gorm:create").Register("fail-governance-audit", func(tx *gorm.DB) {
		if tx.Statement.Table == (identity.RevealAudit{}).TableName() {
			_ = tx.AddError(errors.New("audit storage unavailable"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.Govern(2, p.PublicUID, "required audit", true); err == nil {
		t.Fatal("governance succeeded without audit")
	}
	state, err := s.State(1)
	if err != nil || state.GovernanceDisabled {
		t.Fatal("partial persona restriction", state, err)
	}
	if uid, err := s.Resolve(1, "persona"); err != nil || uid != p.PublicUID {
		t.Fatal("partial owner restriction", uid, err)
	}
}

func TestPostgreSQLPublicPersonaQueries(t *testing.T) {
	s := setup(t, true)
	if err := s.DB.AutoMigrate(&posts.Entity{}, &topics.Entity{}); err != nil {
		t.Fatal(err)
	}
	if err := s.DB.Create(&topics.Entity{Id: 1, FirstPostId: 1, Status: 1}).Error; err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 20; i++ {
		if err := s.DB.Create(&posts.Entity{Id: uint64(i), TopicId: 1, PostNo: uint64(i), IsAnonymous: true, PersonaUID: fmt.Sprintf("%032d", i)}).Error; err != nil {
			t.Fatal(err)
		}
	}
	participants, err := posts.PersonaParticipants(s.DB, []uint64{1})
	if err != nil || len(participants) != 12 {
		t.Fatal("bounded PG window query", participants, err)
	}
	_, count, _, err := posts.PublicPersonaPosts(s.DB, fmt.Sprintf("%032d", 1), 1)
	if err != nil || count != 1 {
		t.Fatal("PG profile visibility", count, err)
	}
	if err := s.DB.Model(&posts.Entity{}).Where("id = 1").Update("retention_status", posts.RetentionPurged).Error; err != nil {
		t.Fatal(err)
	}
	_, count, replies, err := posts.PublicPersonaPosts(s.DB, fmt.Sprintf("%032d", 2), 1)
	if err != nil || count != 0 || replies != 0 {
		t.Fatal("purged parent exposed replies", count, replies, err)
	}
}

// A request queued on another persona write must use the day when its lock is acquired.
func TestMidnightAfterOwnerLock(t *testing.T) {
	for _, operation := range []string{"generate", "confirm", "state"} {
		t.Run(operation, func(t *testing.T) {
			s := setup(t, false)
			now := time.Date(2026, 10, 6, 15, 59, 59, 0, time.UTC)
			s.Now = func() time.Time { return now }
			day, _ := anonymousnames.Day(now)
			batch, err := s.Generate(1, day, "before-midnight")
			if err != nil {
				t.Fatal(err)
			}
			err = s.DB.Callback().Update().Before("gorm:update").Register("test:cross_midnight", func(tx *gorm.DB) {
				if tx.Statement.Table == "users" {
					now = time.Date(2026, 10, 6, 16, 0, 0, 0, time.UTC)
				}
			})
			if err != nil {
				t.Fatal(err)
			}
			switch operation {
			case "generate":
				_, err = s.Generate(1, day, "queued-midnight")
				if !errors.Is(err, ErrCandidate) {
					t.Fatalf("queued old-day generate: %v", err)
				}
			case "confirm":
				_, err = s.Confirm(1, batch.ID, 0)
				if !errors.Is(err, ErrCandidate) {
					t.Fatalf("queued expired confirmation: %v", err)
				}
			case "state":
				state, err := s.State(1)
				if err != nil {
					t.Fatal(err)
				}
				if state.Day != "2026-10-07" || state.Remaining != 10 || len(state.Batches) != 0 {
					t.Fatalf("stale locked snapshot: %+v", state)
				}
			}
		})
	}
}

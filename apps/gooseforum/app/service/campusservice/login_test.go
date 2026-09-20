package campusservice

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pointsRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userPoints"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userStatistics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

func loginSetup(t *testing.T) (*Service, *fakeProvider, pageConfig.SecurityAndRegistration) {
	t.Helper()
	s, p := setup(t)
	p.id = "2351234"
	if err := s.store.DB.AutoMigrate(&users.EntityComplete{}, &userPoints.Entity{}, &pointsRecord.Entity{}, &userStatistics.Entity{}); err != nil {
		t.Fatal(err)
	}
	return s, p, pageConfig.SecurityAndRegistration{EnableSignup: true, MaxDailySignups: -1, EnableEmailVerification: true, AllowedDomains: []string{"tongji.edu.cn"}}
}

func signIn(t *testing.T, s *Service, policy pageConfig.SecurityAndRegistration) (LoginResult, error) {
	t.Helper()
	state, browser, err := s.StartLogin("/campus", "de-DE")
	if err != nil {
		t.Fatal(err)
	}
	return s.Login(context.Background(), browser, state, "code", policy)
}

func TestSchoolLoginProvisionsActivatedPrivateAccountAndReusesBinding(t *testing.T) {
	s, p, policy := loginSetup(t)
	result, err := signIn(t, s, policy)
	if err != nil {
		t.Fatal(err)
	}
	user := result.User
	if !result.Created || user.Id == 0 || user.Email != p.id+"@tongji.edu.cn" || user.IsActivated != users.ActivationSuccess || user.ActivatedAt == nil || user.Password != "" || user.RoleId != 0 || user.Locale != "de" {
		t.Fatal("incomplete or privileged signup")
	}
	if strings.Contains(user.Username, p.id) || strings.Contains(user.Nickname, p.id) {
		t.Fatal("public student ID")
	}
	b, err := s.store.Get(user.Id)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.Sealed, p.id) || strings.Contains(b.Sealed, "private-access") {
		t.Fatal("plaintext credentials")
	}
	var credential Credentials
	if err := s.config.open(user.Id, b.Sealed, &credential); err != nil || credential.StudentID != p.id {
		t.Fatal("credentials not bound to new user")
	}
	// Login keeps existing account fields and does not depend on registration being open.
	if err := s.store.DB.Model(user).Updates(map[string]any{"email": "changed@example.test", "is_activated": users.ActivationPending}).Error; err != nil {
		t.Fatal(err)
	}
	policy.EnableSignup = false
	again, err := signIn(t, s, policy)
	if err != nil || again.Created || again.User.Id != user.Id || again.User.Email != "changed@example.test" || again.User.IsActivated != users.ActivationPending {
		t.Fatalf("existing account login: %v", err)
	}
	current, _ := s.store.Get(user.Id)
	if current.Revision == b.Revision || !current.CreatedAt.Equal(b.CreatedAt) {
		t.Fatal("login did not renew credentials while preserving binding date")
	}
}

func TestSchoolLoginCookieStateReplayExpiryAndProviderFailure(t *testing.T) {
	s, p, policy := loginSetup(t)
	state, browser, _ := s.StartLogin("/api/oauth/authorize/callback?id=original", "en")
	for _, pair := range [][2]string{{"", state}, {"foreign", state}, {browser, "login.wrong"}} {
		if _, err := s.Login(context.Background(), pair[0], pair[1], "code", policy); !errors.Is(err, ErrFlow) {
			t.Fatal("foreign callback accepted")
		}
	}
	result, err := s.Login(context.Background(), browser, state, "code", policy)
	if err != nil || result.Redirect != "/api/oauth/authorize/callback?id=original" {
		t.Fatalf("continuation lost: %v", err)
	}
	if _, err = s.Login(context.Background(), browser, state, "code", policy); !errors.Is(err, ErrFlow) {
		t.Fatal("replay accepted")
	}
	state, browser, _ = s.StartLogin("/", "")
	s.logins[state].Expires = time.Now().Add(-time.Second)
	if _, err = s.Login(context.Background(), browser, state, "code", policy); !errors.Is(err, ErrFlow) {
		t.Fatal("expired callback accepted")
	}
	state, browser, _ = s.StartLogin("/", "")
	p.err = ErrUpstream
	if _, err = s.Login(context.Background(), browser, state, "code", policy); !errors.Is(err, ErrUpstream) {
		t.Fatal("provider failure accepted")
	}
	p.err = nil
	if _, err = s.Login(context.Background(), browser, state, "code", policy); !errors.Is(err, ErrFlow) {
		t.Fatal("failed callback replay accepted")
	}
}

func TestSchoolLoginRejectsEmailClaimsAndRollsBackAccount(t *testing.T) {
	for _, scenario := range []string{"email", "pending", "closed", "pointsFailure", "bindingFailure"} {
		t.Run(scenario, func(t *testing.T) {
			s, p, policy := loginSetup(t)
			before := int64(0)
			switch scenario {
			case "email", "pending", "closed":
				now := time.Now()
				old := &users.EntityComplete{Username: "existing", Email: p.id + "@tongji.edu.cn", Password: "unchanged"}
				if scenario == "pending" {
					old.PendingEmail = old.Email
					old.PendingEmailAt = &now
					old.Email = "old@example.test"
				}
				if err := s.store.DB.Create(old).Error; err != nil {
					t.Fatal(err)
				}
				if scenario == "closed" {
					if err := s.store.DB.Delete(old).Error; err != nil {
						t.Fatal(err)
					}
				}
				before = 1
			case "pointsFailure":
				if err := s.store.DB.Migrator().DropTable(&userPoints.Entity{}); err != nil {
					t.Fatal(err)
				}
			default:
				if err := s.store.DB.Callback().Create().Before("gorm:create").Register("reject_binding", func(tx *gorm.DB) {
					if tx.Statement.Table == "campus_identity_bindings" {
						_ = tx.AddError(errors.New("binding failed"))
					}
				}); err != nil {
					t.Fatal(err)
				}
			}
			result, err := signIn(t, s, policy)
			if err == nil || result.User != nil {
				t.Fatal("conflicting or partial account accepted")
			}
			if scenario == "email" || scenario == "pending" || scenario == "closed" {
				if !errors.Is(err, users.ErrEmailOccupied) {
					t.Fatalf("wrong conflict: %v", err)
				}
			}
			var count int64
			s.store.DB.Unscoped().Model(&users.EntityComplete{}).Count(&count)
			if count != before {
				t.Fatal("orphan account created")
			}
		})
	}
}

func TestSchoolLoginEnforcesPolicyAndAccountState(t *testing.T) {
	for _, scenario := range []string{"disabled", "domain", "quota", "invalidID", "frozen", "bot", "deleted"} {
		t.Run(scenario, func(t *testing.T) {
			s, p, policy := loginSetup(t)
			switch scenario {
			case "disabled":
				policy.EnableSignup = false
			case "domain":
				policy.AllowedDomains = []string{"elsewhere.test"}
			case "quota":
				policy.MaxDailySignups = 0
			case "invalidID":
				p.id = "other@evil.test"
			default:
				original, err := signIn(t, s, policy)
				if err != nil {
					t.Fatal(err)
				}
				switch scenario {
				case "frozen":
					s.store.DB.Model(original.User).Update("is_frozen", users.StatusFrozen)
				case "bot":
					s.store.DB.Model(original.User).Update("actor_type", users.ActorTypeBot)
				case "deleted":
					s.store.DB.Delete(original.User)
				}
			}
			if _, err := signIn(t, s, policy); err == nil {
				t.Fatal("rejected account accepted")
			}
		})
	}
}

func TestSchoolLoginConcurrentCallbacksNeverDuplicateAccount(t *testing.T) {
	s, _, policy := loginSetup(t)
	var wg sync.WaitGroup
	for range 8 {
		state, browser, err := s.StartLogin("/", "zh")
		if err != nil {
			t.Fatal(err)
		}
		wg.Go(func() { _, _ = s.Login(context.Background(), browser, state, "code", policy) })
	}
	wg.Wait()
	var count int64
	if err := s.store.DB.Model(&users.EntityComplete{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("accounts=%d err=%v", count, err)
	}
}

func TestSchoolLoginPurposesAndCancelledAuthorization(t *testing.T) {
	s, _, policy := loginSetup(t)
	state, browser, _ := s.StartLogin("/campus", "zh")
	bindingState, err := s.Start(8, "forum-session", "bind")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Login(context.Background(), browser, bindingState, "code", policy); !errors.Is(err, ErrFlow) {
		t.Fatal("binding state used to log in")
	}
	if err := s.Callback(context.Background(), 8, "forum-session", state, "code"); !errors.Is(err, ErrFlow) {
		t.Fatal("login state used to bind")
	}
	result, err := s.Login(context.Background(), browser, state, "", policy)
	if !errors.Is(err, ErrFlow) || result.Redirect != "/campus" {
		t.Fatal("cancelled flow lost continuation")
	}
	if _, err := s.Login(context.Background(), browser, state, "code", policy); !errors.Is(err, ErrFlow) {
		t.Fatal("cancelled authorization replayed")
	}
}

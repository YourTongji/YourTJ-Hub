package campusservice

import (
	"context"
	"crypto/subtle"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/campus"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
	"gorm.io/gorm"
)

var errRegistrationRequired = errors.New("campus_registration_required")
var ErrSignupDisabled = errors.New("campus_signup_disabled")
var ErrAccountUnavailable = errors.New("campus_account_unavailable")

// Login state is a different purpose from a signed-in binding transaction. The
// cookie proof is never carried in the authorization URL or accepted as state.
type loginAttempt struct {
	Browser, Nonce, Verifier, Redirect, Locale string
	Expires                                    time.Time
}

type LoginResult struct {
	User         *users.EntityComplete
	Redirect     string
	Registration string
	Created      bool
}

func IsLoginState(state string) bool { return strings.HasPrefix(state, "login.") }

func (s *Service) StartLogin(redirect, locale string) (address, browser string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for state, attempt := range s.logins {
		if !attempt.Expires.After(now) {
			delete(s.logins, state)
		}
	}
	if len(s.logins) >= 1000 {
		return "", "", ErrUpstream
	}
	state := "login." + random()
	attempt := &loginAttempt{Browser: random(), Nonce: random(), Verifier: random(), Redirect: redirect, Locale: locale, Expires: now.Add(10 * time.Minute)}
	s.logins[state] = attempt
	time.AfterFunc(time.Until(attempt.Expires), func() { s.mu.Lock(); delete(s.logins, state); s.mu.Unlock() })
	return s.provider.Authorize(state, attempt.Nonce, attempt.Verifier, "login"), attempt.Browser, nil
}

func (s *Service) Login(ctx context.Context, browser, state, code string, policy pageConfig.SecurityAndRegistration) (LoginResult, error) {
	s.mu.Lock()
	attempt := s.logins[state]
	if attempt == nil || browser == "" || subtle.ConstantTimeCompare([]byte(attempt.Browser), []byte(browser)) != 1 || !attempt.Expires.After(time.Now()) {
		s.mu.Unlock()
		return LoginResult{}, ErrFlow
	}
	delete(s.logins, state) // consume before any network call, including provider failures
	s.mu.Unlock()
	result := LoginResult{Redirect: attempt.Redirect}
	if code == "" {
		return result, ErrFlow
	}
	credentials, err := s.provider.Exchange(ctx, code, attempt.Verifier)
	if err != nil {
		return result, err
	}
	credentials, err = s.provider.Identity(ctx, credentials, attempt.Nonce)
	if err != nil {
		return result, err
	}
	if !attempt.Expires.After(time.Now()) {
		return result, ErrFlow
	}
	result.User, result.Created, err = s.loginAccount(ctx, credentials, attempt.Locale, policy, nil)
	if errors.Is(err, errRegistrationRequired) {
		result.Registration, err = s.prepareRegistration(credentials, attempt)
	}
	return result, err
}

// student_info is the identity authority. Only an unambiguous numeric student ID
// may become an email address; arbitrary upstream strings never become addresses.
var studentNumber = regexp.MustCompile(`^[0-9]{5,20}$`)

func (s *Service) loginAccount(ctx context.Context, credentials Credentials, locale string, policy pageConfig.SecurityAndRegistration, registration *Registration) (*users.EntityComplete, bool, error) {
	if credentials.Subject == "" || !studentNumber.MatchString(credentials.StudentID) {
		return nil, false, ErrAuthorization
	}
	key := fingerprint(s.config.IdentityKey, credentials.StudentID)
	var user *users.EntityComplete
	created := false
	err := s.store.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		store := campus.Store{DB: tx}
		binding, err := store.GetByIdentity(key)
		if err == nil {
			if registration != nil {
				return campus.ErrIdentityUsed
			}
			current, err := users.GetForAuthenticationTx(tx, binding.UserID)
			if err != nil || current.IsBot() || current.IsFrozen == users.StatusFrozen {
				return ErrAccountUnavailable
			}
			user = &current
			sealed, err := s.config.seal(current.Id, credentials)
			if err != nil {
				return err
			}
			return store.Replace(campus.Binding{UserID: current.Id, IdentityKey: key, Revision: random(), Sealed: sealed}, binding.Revision)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := store.ReserveSignup(key); err != nil {
			return err
		}
		if !policy.EnableSignup {
			return ErrSignupDisabled
		}
		domainAllowed := len(policy.AllowedDomains) == 0
		for _, domain := range policy.AllowedDomains {
			if strings.EqualFold(strings.TrimSpace(domain), "tongji.edu.cn") {
				domainAllowed = true
			}
		}
		if !domainAllowed {
			return ErrSignupDisabled
		}
		if err := users.CheckEmailClaimTx(tx, credentials.StudentID+"@tongji.edu.cn", 0); err != nil {
			return err
		}
		if registration == nil {
			return errRegistrationRequired
		}
		username := registration.Username
		if _, err := moderationservice.CheckUsernameAllowedWithConfig(username, policy); err != nil {
			return ErrSignupDisabled
		}
		user, err = userservice.CreateVerifiedAccountTx(tx, username, credentials.StudentID+"@tongji.edu.cn", locale, policy.MaxDailySignups)
		if err != nil {
			return err
		}
		// Account, password, identity binding and initial rewards commit together.
		if err := users.UpdatePasswordHashTx(tx, user, registration.PasswordHash); err != nil {
			return err
		}
		sealed, err := s.config.seal(user.Id, credentials)
		if err != nil {
			return err
		}
		if err := store.Replace(campus.Binding{UserID: user.Id, IdentityKey: key, Revision: random(), Sealed: sealed}, ""); err != nil {
			return err
		}
		created = true
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return user, created, nil
}

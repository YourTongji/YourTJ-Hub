package campusservice

import (
	"context"
	"crypto/subtle"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
)

// Registration contains validated public fields. The HTTP boundary enforces the
// same username/password policy as password registration before hashing.
type Registration struct{ Username, PasswordHash string }

type pendingRegistration struct {
	sealed, redirect, locale, csrf string
	expires                        time.Time
	busy                           bool
}

type RegistrationStatus struct {
	CSRFToken string    `json:"csrfToken"`
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func (s *Service) prepareRegistration(credentials Credentials, attempt *loginAttempt) (string, error) {
	sealed, err := s.config.seal(0, credentials)
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for key, pending := range s.registrations {
		if !pending.expires.After(now) {
			delete(s.registrations, key)
		}
	}
	if len(s.registrations) >= 1000 || !attempt.Expires.After(now) {
		return "", ErrFlow
	}
	ticket := random()
	s.registrations[ticket] = &pendingRegistration{sealed: sealed, redirect: attempt.Redirect, locale: attempt.Locale, csrf: random(), expires: attempt.Expires}
	time.AfterFunc(time.Until(attempt.Expires), func() { s.mu.Lock(); delete(s.registrations, ticket); s.mu.Unlock() })
	return ticket, nil
}

// The cookie is HttpOnly. The independent CSRF value is readable only through a
// same-origin JSON request; a bearer token never substitutes for either proof.
func (s *Service) RegistrationStatus(ticket string) (RegistrationStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pending := s.registrations[ticket]
	if pending == nil || !pending.expires.After(time.Now()) {
		return RegistrationStatus{}, ErrFlow
	}
	var credentials Credentials
	if err := s.config.open(0, pending.sealed, &credentials); err != nil {
		return RegistrationStatus{}, err
	}
	return RegistrationStatus{CSRFToken: pending.csrf, Email: credentials.StudentID + "@tongji.edu.cn", ExpiresAt: pending.expires}, nil
}

func (s *Service) CompleteRegistration(ctx context.Context, ticket, csrf string, fields Registration, policy pageConfig.SecurityAndRegistration) (LoginResult, error) {
	s.mu.Lock()
	pending := s.registrations[ticket]
	if pending == nil || pending.busy || !pending.expires.After(time.Now()) || csrf == "" || subtle.ConstantTimeCompare([]byte(csrf), []byte(pending.csrf)) != 1 || fields.Username == "" || fields.PasswordHash == "" {
		s.mu.Unlock()
		return LoginResult{}, ErrFlow
	}
	pending.busy = true
	s.mu.Unlock()
	success := false
	defer func() {
		s.mu.Lock()
		if success {
			delete(s.registrations, ticket)
		} else {
			pending.busy = false
		}
		s.mu.Unlock()
	}()
	result := LoginResult{Redirect: pending.redirect}
	var credentials Credentials
	if err := s.config.open(0, pending.sealed, &credentials); err != nil {
		return result, err
	}
	var err error
	result.User, result.Created, err = s.loginAccount(ctx, credentials, pending.locale, policy, &fields)
	success = err == nil
	return result, err
}

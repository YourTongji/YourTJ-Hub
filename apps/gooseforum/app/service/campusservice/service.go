package campusservice

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/campus"
	"gorm.io/gorm"
)

type BindingView struct {
	MaskedID           string `json:"maskedId"`
	BoundAt            string `json:"boundAt"`
	Revision           string `json:"revision"`
	NeedsAuthorization bool   `json:"needsAuthorization"`
}
type CandidateView struct {
	MaskedID  string `json:"maskedId"`
	Mode      string `json:"mode"`
	ExpiresAt string `json:"expiresAt"`
}
type Status struct {
	Enabled   bool           `json:"enabled"`
	Binding   *BindingView   `json:"binding"`
	Candidate *CandidateView `json:"candidate"`
}
type pending struct {
	UserID                                                    uint64
	Session, State, Nonce, Verifier, Mode, Previous, Identity string
	Expires                                                   time.Time
	Consumed                                                  bool
	Credentials                                               *Credentials
}

var lifecycleLocks [64]sync.Mutex

type Service struct {
	config   Config
	store    campus.Store
	provider Provider
	mu       sync.Mutex
	pending  map[uint64]*pending
	logins   map[string]*loginAttempt
	allowed  func(uint64) bool
}

func New(c Config, s campus.Store, p Provider) *Service {
	return &Service{config: c, store: s, provider: p, pending: make(map[uint64]*pending), logins: make(map[string]*loginAttempt)}
}
func (s *Service) dropPending(id uint64) { s.mu.Lock(); delete(s.pending, id); s.mu.Unlock() }
func (s *Service) Status(id uint64, session string) (Status, error) {
	result := Status{Enabled: true}
	b, e := s.store.Get(id)
	if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
		return result, e
	}
	if e == nil {
		var c Credentials
		err := s.config.open(id, b.Sealed, &c)
		masked := "••••"
		if err == nil {
			masked = Mask(c.StudentID)
		}
		result.Binding = &BindingView{masked, b.CreatedAt.UTC().Format(time.RFC3339), b.Revision, b.NeedsAuthorization || err != nil}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.pending[id]
	if p != nil && p.Session == session && p.Expires.After(time.Now()) && p.Credentials != nil {
		result.Candidate = &CandidateView{Mask(p.Credentials.StudentID), p.Mode, p.Expires.UTC().Format(time.RFC3339)}
	}
	return result, nil
}
func (s *Service) Start(id uint64, session, mode string) (string, error) {
	lock := &lifecycleLocks[id%64]
	lock.Lock()
	defer lock.Unlock()
	if s.allowed != nil && !s.allowed(id) {
		return "", ErrAuthorization
	}
	if session == "" || (mode != "bind" && mode != "replace" && mode != "reauthorize") {
		return "", ErrFlow
	}
	b, e := s.store.Get(id)
	if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
		return "", e
	}
	if (mode == "bind" && e == nil) || (mode != "bind" && e != nil) {
		return "", ErrFlow
	}
	p := &pending{UserID: id, Session: session, State: random(), Nonce: random(), Verifier: random(), Mode: mode, Previous: b.Revision, Identity: b.IdentityKey, Expires: time.Now().Add(10 * time.Minute)}
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range s.pending {
		if v.Expires.Before(time.Now()) {
			delete(s.pending, k)
		}
	}
	if len(s.pending) >= 1000 {
		return "", ErrUpstream
	}
	s.pending[id] = p
	time.AfterFunc(time.Until(p.Expires), func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.pending[id] == p {
			delete(s.pending, id)
		}
	})
	return s.provider.Authorize(p.State, p.Nonce, p.Verifier, mode), nil
}
func (s *Service) Callback(ctx context.Context, id uint64, session, state, code string) error {
	s.mu.Lock()
	p := s.pending[id]
	if p == nil || p.Session != session || p.State != state || p.Consumed || p.Expires.Before(time.Now()) || code == "" {
		s.mu.Unlock()
		return ErrFlow
	}
	p.Consumed = true
	copy := *p
	s.mu.Unlock()
	c, e := s.provider.Exchange(ctx, code, copy.Verifier)
	if e == nil {
		c, e = s.provider.Identity(ctx, c, copy.Nonce)
	}
	if e == nil && copy.Mode == "reauthorize" && fingerprint(s.config.IdentityKey, c.StudentID) != copy.Identity {
		e = ErrConflict
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending[id] != p {
		return ErrFlow
	}
	if e != nil {
		delete(s.pending, id)
		return e
	}
	p.Credentials = &c
	p.Verifier = ""
	p.Nonce = ""
	return nil
}
func (s *Service) Confirm(id uint64, session string) error {
	lock := &lifecycleLocks[id%64]
	lock.Lock()
	defer lock.Unlock()
	if s.allowed != nil && !s.allowed(id) {
		s.dropPending(id)
		return ErrAuthorization
	}
	s.mu.Lock()
	p := s.pending[id]
	if p == nil || p.Session != session || p.Expires.Before(time.Now()) || p.Credentials == nil {
		s.mu.Unlock()
		return ErrFlow
	}
	delete(s.pending, id)
	copy := *p
	s.mu.Unlock()
	sealed, e := s.config.seal(id, copy.Credentials)
	if e != nil {
		return e
	}
	b := campus.Binding{UserID: id, IdentityKey: fingerprint(s.config.IdentityKey, copy.Credentials.StudentID), Revision: random(), Sealed: sealed}
	if e = s.store.Replace(b, copy.Previous); e != nil {
		return ErrConflict
	}
	return nil
}
func (s *Service) Unbind(ctx context.Context, id uint64, revision string) error {
	lock := &lifecycleLocks[id%64]
	lock.Lock()
	defer lock.Unlock()
	b, e := s.store.Get(id)
	if e != nil {
		return e
	}
	if b.Revision != revision {
		return campus.ErrChanged
	}
	if e = s.store.Delete(id, revision); e != nil {
		return e
	}
	s.dropPending(id)
	var c Credentials
	if s.config.open(id, b.Sealed, &c) == nil {
		s.provider.Revoke(ctx, c)
	}
	return nil
}
func (s *Service) credential(ctx context.Context, id uint64, force bool, rejected ...string) (campus.Binding, Credentials, error) {
	lock := &lifecycleLocks[id%64]
	lock.Lock()
	defer lock.Unlock()
	b, e := s.store.Get(id)
	var c Credentials
	if e != nil {
		return b, c, ErrAuthorization
	}
	if b.NeedsAuthorization {
		return b, c, ErrAuthorization
	}
	if e = s.config.open(id, b.Sealed, &c); e != nil {
		return b, c, ErrAuthorization
	}
	if ((!force) || (len(rejected) > 0 && c.Access != rejected[0])) && c.ExpiresAt > time.Now().Add(45*time.Second).Unix() {
		return b, c, nil
	}
	if e = s.store.Acquire(b); e != nil {
		return b, c, e
	}
	renewed, e := s.provider.Refresh(ctx, c)
	if e != nil {
		finishErr := s.store.Finish(b, b.Sealed, errors.Is(e, ErrAuthorization))
		if finishErr != nil {
			return b, c, finishErr
		}
		return b, c, e
	}
	sealed, e := s.config.seal(id, renewed)
	if e != nil {
		_ = s.store.Finish(b, b.Sealed, false)
		return b, c, e
	}
	if e = s.store.Finish(b, sealed, false); e != nil {
		return b, c, e
	}
	return b, renewed, nil
}
func (s *Service) Dataset(ctx context.Context, id uint64, key string) (Dataset, error) {
	if _, ok := datasets[key]; !ok || key == "identity" {
		return Dataset{}, ErrFlow
	}
	data, e := s.readPrivate(ctx, id, func(c Credentials) (any, error) {
		data, err := s.provider.Data(ctx, key, c.Access)
		if err == nil && key == "profile" {
			rows := objects(data)
			if len(rows) != 1 || str(rows[0]["userId"]) != c.StudentID {
				return nil, ErrAuthorization
			}
		}
		return data, err
	})
	if e != nil {
		if errors.Is(e, ErrAuthorization) || errors.Is(e, campus.ErrChanged) {
			return Dataset{}, e
		}
		return emptyDataset(key, "unavailable"), nil
	}
	return normalize(key, data), nil
}

// Every private projection uses the same refresh and binding-revision fence.
func (s *Service) readPrivate(ctx context.Context, id uint64, read func(Credentials) (any, error)) (any, error) {
	b, c, e := s.credential(ctx, id, false)
	if e != nil {
		return nil, e
	}
	data, e := read(c)
	if errors.Is(e, ErrAuthorization) {
		b, c, e = s.credential(ctx, id, true, c.Access)
		if e == nil {
			data, e = read(c)
		}
	}
	current, check := s.store.Get(id)
	if check != nil || current.Revision != b.Revision {
		return nil, campus.ErrChanged
	}
	return data, e
}

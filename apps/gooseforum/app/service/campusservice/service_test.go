package campusservice

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/campus"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type fakeProvider struct {
	id               string
	err              error
	refreshes        atomic.Int32
	revocations      atomic.Int32
	entered, release chan struct{}
}

func (p *fakeProvider) Authorize(a, b, c, d string) string { return a }
func (p *fakeProvider) Exchange(context.Context, string, string) (Credentials, error) {
	return Credentials{Access: "private-access", Refresh: "private-refresh", ExpiresAt: time.Now().Add(time.Hour).Unix()}, p.err
}
func (p *fakeProvider) Identity(_ context.Context, c Credentials, _ string) (Credentials, error) {
	c.StudentID = p.id
	c.Subject = "subject-" + p.id
	return c, nil
}
func (p *fakeProvider) Refresh(_ context.Context, c Credentials) (Credentials, error) {
	p.refreshes.Add(1)
	c.ExpiresAt = time.Now().Add(time.Hour).Unix()
	return c, p.err
}
func (p *fakeProvider) Data(context.Context, string, string) (any, error) {
	if p.entered != nil {
		close(p.entered)
		<-p.release
	}
	return map[string]any{"name": "test term", "week": float64(1)}, nil
}
func (p *fakeProvider) Revoke(context.Context, Credentials)                  { p.revocations.Add(1) }
func (p *fakeProvider) Message(context.Context, string, string) (any, error) { return nil, ErrUpstream }
func setup(t *testing.T) (*Service, *fakeProvider) {
	t.Helper()
	db, e := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		t.Fatal(e)
	}
	sql, e := db.DB()
	if e != nil {
		t.Fatal(e)
	}
	sql.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sql.Close() })
	if e = db.AutoMigrate(&campus.Binding{}); e != nil {
		t.Fatal(e)
	}
	p := &fakeProvider{id: "student-123456"}
	return New(Config{EncryptionKey: strings.Repeat("e", 32), IdentityKey: strings.Repeat("i", 32)}, campus.Store{DB: db}, p), p
}
func prepare(t *testing.T, s *Service, id uint64, mode string) {
	t.Helper()
	state, e := s.Start(id, "session", mode)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Callback(context.Background(), id, "session", state, "code"); e != nil {
		t.Fatal(e)
	}
}
func bind(t *testing.T, s *Service, id uint64) {
	t.Helper()
	prepare(t, s, id, "bind")
	if e := s.Confirm(id, "session"); e != nil {
		t.Fatal(e)
	}
}
func TestBindingUniquenessAndAtomicReplacement(t *testing.T) {
	s, p := setup(t)
	bind(t, s, 1)
	prepare(t, s, 2, "bind")
	if !errors.Is(s.Confirm(2, "session"), ErrConflict) {
		t.Fatal("same official identity bound twice")
	}
	p.id = "other-654321"
	bind(t, s, 2)
	old, _ := s.store.Get(2)
	p.id = "student-123456"
	prepare(t, s, 2, "replace")
	if s.Confirm(2, "session") == nil {
		t.Fatal("occupied replacement succeeded")
	}
	after, _ := s.store.Get(2)
	if after.Revision != old.Revision || after.Sealed != old.Sealed {
		t.Fatal("failed replacement destroyed old connection")
	}
	if e := s.Unbind(context.Background(), 1, "stale"); e == nil {
		t.Fatal("stale unbind accepted")
	}
	first, _ := s.store.Get(1)
	if e := s.Unbind(context.Background(), 1, first.Revision); e != nil {
		t.Fatal(e)
	}
	prepare(t, s, 2, "replace")
	if e := s.Confirm(2, "session"); e != nil {
		t.Fatal(e)
	}
}
func TestAuthorizationSessionStateReplayAndExpiry(t *testing.T) {
	s, _ := setup(t)
	state, e := s.Start(1, "session", "bind")
	if e != nil {
		t.Fatal(e)
	}
	for _, args := range [][2]string{{"other", state}, {"session", "wrong"}} {
		if s.Callback(context.Background(), 1, args[0], args[1], "code") == nil {
			t.Fatal("foreign callback accepted")
		}
	}
	if e = s.Callback(context.Background(), 1, "session", state, "code"); e != nil {
		t.Fatal(e)
	}
	if s.Callback(context.Background(), 1, "session", state, "code") == nil {
		t.Fatal("replay accepted")
	}
	if s.Confirm(1, "foreign") == nil {
		t.Fatal("foreign confirmation accepted")
	}
	s.pending[1].Expires = time.Now().Add(-time.Second)
	if s.Confirm(1, "session") == nil {
		t.Fatal("expired confirmation accepted")
	}
}
func TestRefreshSerializedFailureReservesIdentity(t *testing.T) {
	s, p := setup(t)
	bind(t, s, 1)
	b, _ := s.store.Get(1)
	var c Credentials
	_ = s.config.open(1, b.Sealed, &c)
	c.ExpiresAt = 0
	sealed, _ := s.config.seal(1, c)
	_ = s.store.Finish(b, sealed, false)
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, e := s.credential(context.Background(), 1, false)
			if e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	if p.refreshes.Load() != 1 {
		t.Fatal("parallel refresh rotated token more than once")
	}
	p.err = ErrAuthorization
	_, _, e := s.credential(context.Background(), 1, true)
	if !errors.Is(e, ErrAuthorization) {
		t.Fatal(e)
	}
	b, e = s.store.Get(1)
	if e != nil || !b.NeedsAuthorization {
		t.Fatal("expired refresh removed binding")
	}
}
func TestLateDataCannotSurviveUnbind(t *testing.T) {
	s, p := setup(t)
	bind(t, s, 1)
	p.entered = make(chan struct{})
	p.release = make(chan struct{})
	result := make(chan error)
	go func() { _, e := s.Dataset(context.Background(), 1, "calendar"); result <- e }()
	<-p.entered
	b, _ := s.store.Get(1)
	if e := s.Unbind(context.Background(), 1, b.Revision); e != nil {
		t.Fatal(e)
	}
	close(p.release)
	if !errors.Is(<-result, campus.ErrChanged) {
		t.Fatal("old identity data returned after unlink")
	}
	if s.Confirm(1, "session") == nil {
		t.Fatal("deleted pending identity restored")
	}
}
func TestEncryptionBoundToUserAndEnvironment(t *testing.T) {
	s, _ := setup(t)
	sealed, e := s.config.seal(1, Credentials{Access: "secret-token", StudentID: "secret-student"})
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(sealed, "secret") {
		t.Fatal("plaintext stored")
	}
	var c Credentials
	if s.config.open(2, sealed, &c) == nil {
		t.Fatal("row swapping accepted")
	}
	other := s.config
	other.EncryptionKey = strings.Repeat("x", 32)
	if other.open(1, sealed, &c) == nil {
		t.Fatal("foreign environment decrypted credentials")
	}
}
func TestReauthorizeCannotChangeIdentity(t *testing.T) {
	s, p := setup(t)
	bind(t, s, 1)
	p.id = "other-student"
	state, _ := s.Start(1, "session", "reauthorize")
	if !errors.Is(s.Callback(context.Background(), 1, "session", state, "code"), ErrConflict) {
		t.Fatal("reauthorize replaced identity")
	}
}
func TestProjectionExcludesPrivateFields(t *testing.T) {
	v := map[string]any{"list": []any{map[string]any{"studentId": "private-id", "cardNo": "private-ticket", "studentName": "private-name", "score": "425", "writtenSubjectName": "CET4", "calendarYearTermCn": "term"}}}
	d := normalize("cet", v)
	if len(d.Rows) != 1 || d.Rows[0][2] != "425" {
		t.Fatal("missing score")
	}
	for _, row := range d.Rows {
		for _, c := range row {
			if strings.Contains(c, "private") {
				t.Fatal("private metadata leaked")
			}
		}
	}
}

func TestGradeProjectionDoesNotInventZeroForUnpublishedGPA(t *testing.T) {
	v := map[string]any{"term": []any{
		map[string]any{"termName": "pending", "averagePoint": "未公布"},
		map[string]any{"termName": "missing"},
		map[string]any{"termName": "nonfinite", "averagePoint": "NaN"},
		map[string]any{"termName": "zero", "averagePoint": "0"},
	}}
	d := normalize("grades", v)
	if len(d.Series) != 1 || d.Series[0].Label != "zero" || d.Series[0].Value != 0 {
		t.Fatalf("only published numeric values should be charted: %#v", d.Series)
	}
}

func TestClosedAccountCannotConfirmOrStart(t *testing.T) {
	s, _ := setup(t)
	prepare(t, s, 1, "bind")
	s.allowed = func(uint64) bool { return false }
	if s.Confirm(1, "session") == nil {
		t.Fatal("closed account bound identity")
	}
	if _, e := s.Start(1, "session", "bind"); e == nil {
		t.Fatal("closed account started authorization")
	}
}
func TestStaleCredentialCannotAcquireRefreshLease(t *testing.T) {
	s, _ := setup(t)
	bind(t, s, 1)
	old, _ := s.store.Get(1)
	if e := s.store.Finish(old, "new-ciphertext", false); e != nil {
		t.Fatal(e)
	}
	if e := s.store.Acquire(old); e == nil {
		t.Fatal("old refresh credential acquired lease")
	}
}

func TestBoundAtTracksIdentityReplacementOnly(t *testing.T) {
	s, p := setup(t)
	bind(t, s, 1)
	original := time.Now().Add(-24 * time.Hour).UTC().Truncate(time.Second)
	if err := s.store.DB.Model(&campus.Binding{}).Where("user_id = ?", 1).Update("created_at", original).Error; err != nil {
		t.Fatal(err)
	}
	prepare(t, s, 1, "reauthorize")
	if err := s.Confirm(1, "session"); err != nil {
		t.Fatal(err)
	}
	state, _ := s.Status(1, "session")
	if state.Binding.BoundAt != original.Format(time.RFC3339) {
		t.Fatal("reauthorization changed original binding time")
	}
	p.id = "replacement-student"
	prepare(t, s, 1, "replace")
	if err := s.Confirm(1, "session"); err != nil {
		t.Fatal(err)
	}
	state, _ = s.Status(1, "session")
	at, err := time.Parse(time.RFC3339, state.Binding.BoundAt)
	if err != nil || !at.After(original) {
		t.Fatal("replacement retained previous identity binding time")
	}
}

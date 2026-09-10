package storageservice

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type fakeObject struct {
	data        []byte
	contentType string
}

// fakeDirectUploadProvider is an in-memory Provider + DirectUploadProvider.
type fakeDirectUploadProvider struct {
	objects    map[string]fakeObject
	now        time.Time
	presignErr error
}

func newFakeDirectUploadProvider() *fakeDirectUploadProvider {
	return &fakeDirectUploadProvider{objects: map[string]fakeObject{}, now: time.Now()}
}

func (p *fakeDirectUploadProvider) Save(_ context.Context, name string, data []byte, contentType string) error {
	p.objects[name] = fakeObject{data: data, contentType: contentType}
	return nil
}

func (p *fakeDirectUploadProvider) Get(_ context.Context, name string) ([]byte, string, error) {
	obj, ok := p.objects[name]
	if !ok {
		return nil, "", ErrNotFound
	}
	return obj.data, obj.contentType, nil
}

func (p *fakeDirectUploadProvider) Delete(_ context.Context, name string) error {
	delete(p.objects, name)
	return nil
}

func (p *fakeDirectUploadProvider) Exists(_ context.Context, name string) (bool, error) {
	_, ok := p.objects[name]
	return ok, nil
}

func (p *fakeDirectUploadProvider) PresignUpload(_ context.Context, request DirectUploadRequest) (*DirectUpload, error) {
	if p.presignErr != nil {
		return nil, p.presignErr
	}
	if err := validateDirectUploadRequest(request.Name, request.ContentType, request.Size); err != nil {
		return nil, err
	}
	return &DirectUpload{
		URL:       "https://fake.example.com/" + request.Name,
		Method:    "POST",
		Fields:    map[string]string{"key": request.Name},
		ExpiresAt: p.now.Add(time.Hour),
	}, nil
}

func (p *fakeDirectUploadProvider) VerifyUpload(_ context.Context, request DirectUploadRequest) error {
	obj, ok := p.objects[request.Name]
	if !ok {
		return ErrNotFound
	}
	if int64(len(obj.data)) != request.Size {
		return fmt.Errorf("s3 object size mismatch: got %d, want %d", len(obj.data), request.Size)
	}
	if !strings.EqualFold(obj.contentType, request.ContentType) {
		return fmt.Errorf("s3 object content type mismatch: got %q, want %q", obj.contentType, request.ContentType)
	}
	return nil
}

// fakeMetadataStore is an in-memory DirectUploadMetadataStore.
type fakeMetadataStore struct {
	rows   map[string]FileMetadata
	status map[string]string
	nextID uint64
}

func newFakeMetadataStore() *fakeMetadataStore {
	return &fakeMetadataStore{rows: map[string]FileMetadata{}, status: map[string]string{}}
}

func (s *fakeMetadataStore) CreateFileMetadata(_ context.Context, userId uint64, name, fileType string, size int64) (*FileMetadata, error) {
	if _, ok := s.rows[name]; ok {
		return nil, fmt.Errorf("file already exists: %s", name)
	}
	s.nextID++
	row := FileMetadata{Id: s.nextID, Name: name, Type: fileType, Size: size, UserId: userId, CreatedAt: time.Now()}
	s.rows[name] = row
	s.status[name] = "pending"
	return &row, nil
}

func (s *fakeMetadataStore) GetPendingFileMetadataByName(_ context.Context, name string) (*FileMetadata, error) {
	row, ok := s.rows[name]
	if !ok || s.status[name] != "pending" {
		return nil, errors.New("file not found")
	}
	return &row, nil
}

func (s *fakeMetadataStore) GetFileMetadataByName(_ context.Context, name string) (*FileMetadata, error) {
	row, ok := s.rows[name]
	if !ok || s.status[name] != "ready" {
		return nil, errors.New("file not found")
	}
	return &row, nil
}

func (s *fakeMetadataStore) MarkFileReady(_ context.Context, name string) (*FileMetadata, error) {
	row, ok := s.rows[name]
	if !ok {
		return nil, errors.New("file metadata not found")
	}
	s.status[name] = "ready"
	return &row, nil
}

func (s *fakeMetadataStore) ListPendingFilesBefore(_ context.Context, before time.Time, limit int) ([]FileMetadata, error) {
	var items []FileMetadata
	for name, row := range s.rows {
		if s.status[name] == "pending" && row.CreatedAt.Before(before) {
			items = append(items, row)
		}
	}
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (s *fakeMetadataStore) DeleteByName(_ context.Context, name string) error {
	delete(s.rows, name)
	delete(s.status, name)
	return nil
}

func setupDirectUploadTest(t *testing.T) (*fakeDirectUploadProvider, *fakeMetadataStore) {
	t.Helper()
	provider := newFakeDirectUploadProvider()
	store := newFakeMetadataStore()
	SetCurrentForTest(provider)
	RegisterDirectUploadMetadataStore(store)
	t.Cleanup(func() {
		SetCurrentForTest(nil)
		RegisterDirectUploadMetadataStore(nil)
	})
	return provider, store
}

func TestBeginDirectUploadLifecycle(t *testing.T) {
	provider, store := setupDirectUploadTest(t)
	ctx := context.Background()

	session, err := BeginDirectUpload(ctx, DirectUploadRequest{
		Name: "2026/09/01/a.png", ContentType: "image/png", Size: 4, UserId: 1,
	})
	if err != nil {
		t.Fatalf("BeginDirectUpload() error = %v", err)
	}
	if session.Metadata.Name != "2026/09/01/a.png" || session.Metadata.UserId != 1 {
		t.Fatalf("session metadata = %+v, want name/userId set", session.Metadata)
	}
	if session.Upload.Method != http.MethodPost || session.Upload.URL == "" {
		t.Fatalf("session upload = %+v, want POST url", session.Upload)
	}
	if store.status[session.Metadata.Name] != "pending" {
		t.Fatalf("row status = %q, want pending", store.status[session.Metadata.Name])
	}

	// Simulate the browser upload, then complete with a content validator.
	provider.objects[session.Metadata.Name] = fakeObject{data: []byte("png!"), contentType: "image/png"}
	metadata, err := CompleteDirectUpload(ctx, CompleteDirectUploadRequest{
		Name:      session.Metadata.Name,
		UserId:    1,
		Validator: func(_ io.Reader, _ string) error { return nil },
	})
	if err != nil {
		t.Fatalf("CompleteDirectUpload() error = %v", err)
	}
	if metadata.Name != session.Metadata.Name {
		t.Fatalf("completed metadata name = %q, want %q", metadata.Name, session.Metadata.Name)
	}
	if store.status[metadata.Name] != "ready" {
		t.Fatalf("row status = %q, want ready", store.status[metadata.Name])
	}
}

func TestCompleteDirectUploadOwnerMismatch(t *testing.T) {
	_, store := setupDirectUploadTest(t)
	ctx := context.Background()

	session, err := BeginDirectUpload(ctx, DirectUploadRequest{
		Name: "2026/09/01/b.png", ContentType: "image/png", Size: 4, UserId: 1,
	})
	if err != nil {
		t.Fatalf("BeginDirectUpload() error = %v", err)
	}
	_, err = CompleteDirectUpload(ctx, CompleteDirectUploadRequest{Name: session.Metadata.Name, UserId: 2})
	if !errors.Is(err, ErrDirectUploadOwnerMismatch) {
		t.Fatalf("CompleteDirectUpload() error = %v, want ErrDirectUploadOwnerMismatch", err)
	}
	if store.status[session.Metadata.Name] != "pending" {
		t.Fatalf("row status = %q, want still pending after owner mismatch", store.status[session.Metadata.Name])
	}
}

func TestCompleteDirectUploadDoubleCompleteIdempotent(t *testing.T) {
	provider, _ := setupDirectUploadTest(t)
	ctx := context.Background()

	session, err := BeginDirectUpload(ctx, DirectUploadRequest{
		Name: "2026/09/01/c.png", ContentType: "image/png", Size: 4, UserId: 1,
	})
	if err != nil {
		t.Fatalf("BeginDirectUpload() error = %v", err)
	}
	provider.objects[session.Metadata.Name] = fakeObject{data: []byte("png!"), contentType: "image/png"}
	if _, err := CompleteDirectUpload(ctx, CompleteDirectUploadRequest{Name: session.Metadata.Name, UserId: 1}); err != nil {
		t.Fatalf("first CompleteDirectUpload() error = %v", err)
	}
	// Second complete for the same user is idempotent.
	metadata, err := CompleteDirectUpload(ctx, CompleteDirectUploadRequest{Name: session.Metadata.Name, UserId: 1})
	if err != nil {
		t.Fatalf("duplicate CompleteDirectUpload() error = %v, want idempotent success", err)
	}
	if metadata.Name != session.Metadata.Name {
		t.Fatalf("duplicate complete metadata name = %q, want %q", metadata.Name, session.Metadata.Name)
	}
}

func TestCompleteDirectUploadInvalidObjectRollsBack(t *testing.T) {
	provider, store := setupDirectUploadTest(t)
	ctx := context.Background()

	session, err := BeginDirectUpload(ctx, DirectUploadRequest{
		Name: "2026/09/01/d.png", ContentType: "image/png", Size: 4, UserId: 1,
	})
	if err != nil {
		t.Fatalf("BeginDirectUpload() error = %v", err)
	}
	provider.objects[session.Metadata.Name] = fakeObject{data: []byte("png!"), contentType: "image/png"}
	_, err = CompleteDirectUpload(ctx, CompleteDirectUploadRequest{
		Name:      session.Metadata.Name,
		UserId:    1,
		Validator: func(_ io.Reader, _ string) error { return ErrDirectUploadInvalidObject },
	})
	if !errors.Is(err, ErrDirectUploadInvalidObject) {
		t.Fatalf("CompleteDirectUpload() error = %v, want ErrDirectUploadInvalidObject", err)
	}
	if _, ok := provider.objects[session.Metadata.Name]; ok {
		t.Fatal("object still present after invalid-object rollback")
	}
	if _, ok := store.rows[session.Metadata.Name]; ok {
		t.Fatal("pending row still present after invalid-object rollback")
	}
}

func TestAbortDirectUpload(t *testing.T) {
	provider, store := setupDirectUploadTest(t)
	ctx := context.Background()

	session, err := BeginDirectUpload(ctx, DirectUploadRequest{
		Name: "2026/09/01/e.png", ContentType: "image/png", Size: 4, UserId: 1,
	})
	if err != nil {
		t.Fatalf("BeginDirectUpload() error = %v", err)
	}
	provider.objects[session.Metadata.Name] = fakeObject{data: []byte("png!"), contentType: "image/png"}
	if err := AbortDirectUpload(ctx, CompleteDirectUploadRequest{Name: session.Metadata.Name, UserId: 1}); err != nil {
		t.Fatalf("AbortDirectUpload() error = %v", err)
	}
	if _, ok := provider.objects[session.Metadata.Name]; ok {
		t.Fatal("object still present after abort")
	}
	if _, ok := store.rows[session.Metadata.Name]; ok {
		t.Fatal("pending row still present after abort")
	}
}

func TestAbortDirectUploadOwnerMismatch(t *testing.T) {
	provider, _ := setupDirectUploadTest(t)
	ctx := context.Background()

	session, err := BeginDirectUpload(ctx, DirectUploadRequest{
		Name: "2026/09/01/f.png", ContentType: "image/png", Size: 4, UserId: 1,
	})
	if err != nil {
		t.Fatalf("BeginDirectUpload() error = %v", err)
	}
	provider.objects[session.Metadata.Name] = fakeObject{data: []byte("png!"), contentType: "image/png"}
	err = AbortDirectUpload(ctx, CompleteDirectUploadRequest{Name: session.Metadata.Name, UserId: 2})
	if !errors.Is(err, ErrDirectUploadOwnerMismatch) {
		t.Fatalf("AbortDirectUpload() error = %v, want ErrDirectUploadOwnerMismatch", err)
	}
	if _, ok := provider.objects[session.Metadata.Name]; !ok {
		t.Fatal("object removed despite owner mismatch")
	}
}

func TestCleanupPending(t *testing.T) {
	provider, store := setupDirectUploadTest(t)
	ctx := context.Background()

	oldSession, err := BeginDirectUpload(ctx, DirectUploadRequest{
		Name: "2026/08/01/old.png", ContentType: "image/png", Size: 4, UserId: 1,
	})
	if err != nil {
		t.Fatalf("BeginDirectUpload(old) error = %v", err)
	}
	// Backdate the old row so it falls before the cleanup cutoff.
	oldRow := store.rows[oldSession.Metadata.Name]
	oldRow.CreatedAt = time.Now().Add(-3 * time.Hour)
	store.rows[oldSession.Metadata.Name] = oldRow
	provider.objects[oldSession.Metadata.Name] = fakeObject{data: []byte("png!"), contentType: "image/png"}

	newSession, err := BeginDirectUpload(ctx, DirectUploadRequest{
		Name: "2026/09/01/new.png", ContentType: "image/png", Size: 4, UserId: 1,
	})
	if err != nil {
		t.Fatalf("BeginDirectUpload(new) error = %v", err)
	}
	provider.objects[newSession.Metadata.Name] = fakeObject{data: []byte("png!"), contentType: "image/png"}

	removed, err := CleanupPending(ctx, time.Now().Add(-2*time.Hour), 500)
	if err != nil {
		t.Fatalf("CleanupPending() error = %v", err)
	}
	if removed != 1 {
		t.Fatalf("CleanupPending() removed = %d, want 1", removed)
	}
	if _, ok := provider.objects[oldSession.Metadata.Name]; ok {
		t.Fatal("old object still present after cleanup")
	}
	if _, ok := provider.objects[newSession.Metadata.Name]; !ok {
		t.Fatal("fresh object removed by cleanup")
	}
}

func TestBeginDirectUploadRollsBackOnPresignFailure(t *testing.T) {
	provider, store := setupDirectUploadTest(t)
	ctx := context.Background()
	provider.presignErr = errors.New("presign failed")

	_, err := BeginDirectUpload(ctx, DirectUploadRequest{
		Name: "2026/09/01/g.png", ContentType: "image/png", Size: 4, UserId: 1,
	})
	if err == nil {
		t.Fatal("BeginDirectUpload() error = nil, want presign failure")
	}
	if _, ok := store.rows["2026/09/01/g.png"]; ok {
		t.Fatal("pending row not rolled back after presign failure")
	}
}

func TestBeginDirectUploadUnsupported(t *testing.T) {
	SetCurrentForTest(localOnlyProvider{})
	RegisterDirectUploadMetadataStore(newFakeMetadataStore())
	t.Cleanup(func() {
		SetCurrentForTest(nil)
		RegisterDirectUploadMetadataStore(nil)
	})
	_, err := BeginDirectUpload(context.Background(), DirectUploadRequest{
		Name: "2026/09/01/h.png", ContentType: "image/png", Size: 4, UserId: 1,
	})
	if !errors.Is(err, ErrDirectUploadUnsupported) {
		t.Fatalf("BeginDirectUpload() error = %v, want ErrDirectUploadUnsupported", err)
	}
	if SupportsDirectUpload() {
		t.Fatal("SupportsDirectUpload() = true for local-only provider")
	}
}

func TestBeginDirectUploadValidation(t *testing.T) {
	setupDirectUploadTest(t)
	ctx := context.Background()
	tests := []struct {
		name        string
		contentType string
		size        int64
	}{
		{name: "", contentType: "image/png", size: 4},
		{name: "a.png", contentType: "", size: 4},
		{name: "a.png", contentType: "image/png", size: 0},
	}
	for _, tt := range tests {
		if _, err := BeginDirectUpload(ctx, DirectUploadRequest{Name: tt.name, ContentType: tt.contentType, Size: tt.size, UserId: 1}); err == nil {
			t.Fatalf("BeginDirectUpload(%q, %q, %d) error = nil, want validation error", tt.name, tt.contentType, tt.size)
		}
	}
}

func TestNewUploadName(t *testing.T) {
	name := NewUploadName("Photo.PNG", "2026/09/01")
	if !strings.HasPrefix(name, "2026/09/01/") || !strings.HasSuffix(name, ".png") {
		t.Fatalf("NewUploadName() = %q, want 2026/09/01/<uuid>.png", name)
	}
}

// localOnlyProvider implements Provider but not DirectUploadProvider.
type localOnlyProvider struct{}

func (localOnlyProvider) Save(_ context.Context, name string, data []byte, contentType string) error {
	return nil
}
func (localOnlyProvider) Get(_ context.Context, name string) ([]byte, string, error) {
	return nil, "", ErrNotFound
}
func (localOnlyProvider) Delete(_ context.Context, name string) error         { return nil }
func (localOnlyProvider) Exists(_ context.Context, name string) (bool, error) { return false, nil }

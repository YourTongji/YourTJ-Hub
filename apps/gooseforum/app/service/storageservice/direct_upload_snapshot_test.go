package storageservice

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

// VerifyUpload sees the original object; the subsequent read can observe a
// replacement or fail independently, as it can with a replayed upload policy.
type snapshotReadProvider struct {
	*fakeDirectUploadProvider
	snapshot []byte
	readErr  error
}

func (p *snapshotReadProvider) Get(_ context.Context, name string) ([]byte, string, error) {
	if p.readErr != nil {
		return nil, "", p.readErr
	}
	p.objects[name] = fakeObject{data: p.snapshot, contentType: "image/png"}
	return p.snapshot, "image/png", nil
}

type snapshotRangeProvider struct {
	*snapshotReadProvider
	t *testing.T
}

func (p *snapshotRangeProvider) GetRange(ctx context.Context, name string, offset, length int64) ([]byte, string, error) {
	if offset != 0 || length != 5 {
		p.t.Fatalf("snapshot range = %d, %d; want 0, declared size + 1", offset, length)
	}
	return p.Get(ctx, name)
}

func TestCompleteDirectUploadSnapshotFailureCleanup(t *testing.T) {
	readErr := errors.New("temporary read failure")
	deleteErr := errors.New("temporary delete failure")
	for _, ranged := range []bool{false, true} {
		for _, tc := range []struct {
			name      string
			snapshot  string
			readErr   error
			deleteErr error
		}{
			{name: "short", snapshot: "bad"},
			{name: "long", snapshot: "extra"},
			{name: "read error", readErr: readErr},
			{name: "delete error", snapshot: "bad", deleteErr: deleteErr},
		} {
			mode := "full/"
			if ranged {
				mode = "range/"
			}
			t.Run(mode+tc.name, func(t *testing.T) {
				base, metadata := setupDirectUploadTest(t)
				ctx := context.Background()
				source, err := metadata.CreateFileMetadata(ctx, 1, "snapshot/source.png", "image/png", 4)
				if err != nil {
					t.Fatal(err)
				}
				base.objects[source.Name] = fakeObject{data: []byte("good"), contentType: "image/png"}
				base.deleteErr = tc.deleteErr
				provider := &snapshotReadProvider{fakeDirectUploadProvider: base, snapshot: []byte(tc.snapshot), readErr: tc.readErr}
				SetCurrentForTest(provider)
				if ranged {
					SetCurrentForTest(&snapshotRangeProvider{snapshotReadProvider: provider, t: t})
				}
				_, err = CompleteDirectUpload(ctx, CompleteDirectUploadRequest{
					Name: source.Name, UserId: 1,
					Validator: func(io.Reader) (string, error) {
						t.Fatal("invalid or unreadable snapshot reached validation")
						return "", nil
					},
				})
				wantErr := ErrDirectUploadInvalidObject
				if tc.readErr != nil {
					wantErr = tc.readErr
				}
				if !errors.Is(err, wantErr) || (tc.deleteErr != nil && !errors.Is(err, tc.deleteErr)) {
					t.Fatalf("complete error = %v; want %v and delete error %v", err, wantErr, tc.deleteErr)
				}
				keep := tc.readErr != nil || tc.deleteErr != nil
				_, objectExists := base.objects[source.Name]
				_, rowExists := metadata.rows[source.Name]
				if objectExists != keep || rowExists != keep {
					t.Fatalf("after failure: object=%t, metadata=%t; want retained=%t", objectExists, rowExists, keep)
				}
				if len(metadata.completed) != 0 || len(metadata.rows) > 1 {
					t.Fatal("failed snapshot created a published or reserved copy")
				}
				if tc.readErr != nil {
					provider.readErr = nil
					provider.snapshot = []byte("good")
					if _, err := CompleteDirectUpload(ctx, CompleteDirectUploadRequest{Name: source.Name, UserId: 1}); err != nil {
						t.Fatalf("retry after transient read error: %v", err)
					}
				}
				if tc.deleteErr != nil {
					base.deleteErr = nil
					removed, err := CleanupPending(ctx, time.Now().Add(time.Hour), 10)
					if err != nil || removed != 1 || len(base.objects) != 0 || len(metadata.rows) != 0 {
						t.Fatalf("cleanup retry: removed=%d, error=%v", removed, err)
					}
				}
			})
		}
	}
}

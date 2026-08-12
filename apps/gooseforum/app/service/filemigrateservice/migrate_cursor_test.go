package filemigrateservice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/storageservice"
)

// fakeProvider simulates object storage with optional per-object failure counts.
// failures[name] is the number of Save attempts that fail before succeeding.
// saveCounts tracks every Save invocation per object so tests can assert no
// object is re-uploaded during cursor-stall re-scans.
type fakeProvider struct {
	objects    map[string][]byte
	failures   map[string]int
	failAll    bool
	saveCounts map[string]int
}

func newFakeProvider() *fakeProvider {
	return &fakeProvider{
		objects:    map[string][]byte{},
		failures:   map[string]int{},
		saveCounts: map[string]int{},
	}
}

func (p *fakeProvider) Save(ctx context.Context, name string, data []byte, _ string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.saveCounts[name]++
	if p.failAll {
		return errors.New("simulated provider failure")
	}
	if n := p.failures[name]; n > 0 {
		p.failures[name] = n - 1
		return errors.New("simulated transient failure")
	}
	p.objects[name] = append([]byte(nil), data...)
	return nil
}

func (p *fakeProvider) Get(_ context.Context, name string) ([]byte, string, error) {
	data, ok := p.objects[name]
	if !ok {
		return nil, "", storageservice.ErrNotFound
	}
	return data, "application/octet-stream", nil
}

func (p *fakeProvider) Delete(_ context.Context, name string) error {
	delete(p.objects, name)
	return nil
}

func (p *fakeProvider) Exists(_ context.Context, name string) (bool, error) {
	_, ok := p.objects[name]
	return ok, nil
}

// fakeTable mirrors the file_data table in memory so the injected query/clear
// functions behave like the real ones (including clearing Data after a
// successful migration, which makes re-fetched rows show as already migrated).
type fakeTable struct {
	rows []*filedata.Entity
}

func (t *fakeTable) queryByID(startID uint64, limit int) []*filedata.Entity {
	var out []*filedata.Entity
	for _, r := range t.rows {
		if r.Id > startID {
			out = append(out, r)
			if len(out) >= limit {
				break
			}
		}
	}
	return out
}

func (t *fakeTable) clearContent(name string) error {
	for _, r := range t.rows {
		if r.Name == name {
			r.Data = nil
			return nil
		}
	}
	return errors.New("file not found")
}

func (t *fakeTable) byName(name string) *filedata.Entity {
	for _, r := range t.rows {
		if r.Name == name {
			return r
		}
	}
	return nil
}

func entity(id uint64, name string, data []byte) *filedata.Entity {
	return &filedata.Entity{Id: id, Name: name, Type: "image/png", Data: data}
}

// TestMigrateFilesRetriesFailedObject reproduces the issue: a middle object
// fails while later objects succeed. The cursor must stop before the failed
// object (not skip it), and the object must be retried until it succeeds.
func TestMigrateFilesRetriesFailedObject(t *testing.T) {
	table := &fakeTable{rows: []*filedata.Entity{
		entity(1, "a.png", []byte("aaa")),
		entity(2, "b.png", []byte("bbb")),
		entity(3, "c.png", []byte("ccc")),
	}}
	provider := newFakeProvider()
	provider.failures["b.png"] = 1 // row 2 fails exactly once

	var cursors []uint64
	onProgress := func(id uint64, _, _ int64) { cursors = append(cursors, id) }

	gotP, gotF, err := migrateFiles(context.Background(), provider, table.queryByID, table.clearContent, 0, true, nil, onProgress)
	if err != nil {
		t.Fatalf("migrateFiles() error = %v, want nil", err)
	}
	if gotP != 3 || gotF != 1 {
		t.Fatalf("migrateFiles() = (%d, %d), want (3, 1)", gotP, gotF)
	}
	// The cursor must freeze at id 1 while row 2 is failing, then advance past
	// it once row 2 is retried. A cursor jump straight to 3 would be the bug.
	if len(cursors) != 2 || cursors[0] != 1 || cursors[1] != 3 {
		t.Fatalf("cursor sequence = %v, want [1 3]", cursors)
	}
	// Every object must be uploaded and every BLOB cleared in the end.
	for _, name := range []string{"a.png", "b.png", "c.png"} {
		if _, _, err := provider.Get(context.Background(), name); err != nil {
			t.Fatalf("object %s missing after migration: %v", name, err)
		}
		if table.byName(name).Data != nil {
			t.Fatalf("blob %s not cleared after successful migration", name)
		}
	}
}

// TestMigrateFilesAbortsOnPersistentFailure verifies that a permanently failing
// object aborts the run with an error (so the task is not marked success), keeps
// its BLOB so it stays retryable, and does not block objects after it.
func TestMigrateFilesAbortsOnPersistentFailure(t *testing.T) {
	table := &fakeTable{rows: []*filedata.Entity{
		entity(1, "a.png", []byte("aaa")),
		entity(2, "b.png", []byte("bbb")),
		entity(3, "c.png", []byte("ccc")),
	}}
	provider := newFakeProvider()
	provider.failures["b.png"] = 1 << 30 // permanently failing

	_, gotF, err := migrateFiles(context.Background(), provider, table.queryByID, table.clearContent, 0, true, nil, nil)
	if err == nil {
		t.Fatal("migrateFiles() error = nil with persistent failure, want abort error")
	}
	if !strings.Contains(err.Error(), "abort") {
		t.Fatalf("migrateFiles() error = %q, want an abort message", err)
	}
	if gotF != 1 {
		t.Fatalf("migrateFiles() failed = %d, want 1 distinct failed object", gotF)
	}
	// The failed row must keep its BLOB so a later run can retry it.
	if table.byName("b.png").Data == nil {
		t.Fatal("failed row b.png BLOB was cleared; it must remain retryable")
	}
	// Objects after the failed one are still migrated (progress), not skipped.
	if _, _, err := provider.Get(context.Background(), "c.png"); err != nil {
		t.Fatalf("object c.png not migrated despite failure on b.png: %v", err)
	}
	if table.byName("c.png").Data != nil {
		t.Fatal("blob c.png not cleared after successful migration")
	}
}

// TestMigrateFilesResumeFromPersistedCursor simulates a failed task being
// retried: run 1 aborts leaving the cursor frozen before the failed object, and
// a later run resumed from that cursor migrates the remaining object.
func TestMigrateFilesResumeFromPersistedCursor(t *testing.T) {
	table := &fakeTable{rows: []*filedata.Entity{
		entity(1, "a.png", []byte("aaa")),
		entity(2, "b.png", []byte("bbb")),
	}}
	provider := newFakeProvider()
	provider.failures["b.png"] = 1 << 30 // storage broken during run 1

	var cursor uint64
	_, _, err := migrateFiles(context.Background(), provider, table.queryByID, table.clearContent, 0, true, nil, func(id uint64, _, _ int64) { cursor = id })
	if err == nil {
		t.Fatal("run 1 error = nil, want abort")
	}
	if cursor != 1 {
		t.Fatalf("run 1 cursor = %d, want 1 (frozen before failed row b.png)", cursor)
	}

	// The storage issue is fixed; a new task resumes from the persisted cursor.
	provider.failures["b.png"] = 0
	gotP, gotF, err := migrateFiles(context.Background(), provider, table.queryByID, table.clearContent, cursor, true, nil, nil)
	if err != nil {
		t.Fatalf("run 2 error = %v, want nil", err)
	}
	if gotP != 1 || gotF != 0 {
		t.Fatalf("run 2 = (%d, %d), want (1, 0)", gotP, gotF)
	}
	if table.byName("b.png").Data != nil {
		t.Fatal("run 2 should clear b.png BLOB after successful migration")
	}
}

// TestMigrateFilesSkipsAlreadyMigratedRows verifies that empty rows (already
// migrated/cleared) advance the cursor without re-uploading, while still being
// counted as handled so a re-run reports the full total.
func TestMigrateFilesSkipsAlreadyMigratedRows(t *testing.T) {
	table := &fakeTable{rows: []*filedata.Entity{
		entity(1, "a.png", nil),
		entity(2, "b.png", []byte("bbb")),
		entity(3, "c.png", nil),
	}}
	provider := newFakeProvider()

	gotP, gotF, err := migrateFiles(context.Background(), provider, table.queryByID, table.clearContent, 0, true, nil, nil)
	if err != nil {
		t.Fatalf("migrateFiles() error = %v, want nil", err)
	}
	if gotP != 3 || gotF != 0 {
		t.Fatalf("migrateFiles() = (%d, %d), want (3, 0)", gotP, gotF)
	}
	if _, _, err := provider.Get(context.Background(), "b.png"); err != nil {
		t.Fatalf("object b.png not migrated: %v", err)
	}
	if len(provider.objects) != 1 {
		t.Fatalf("provider has %d objects, want 1 (empty rows must not be re-uploaded)", len(provider.objects))
	}
	if provider.saveCounts["b.png"] != 1 {
		t.Fatalf("b.png uploaded %d times, want 1", provider.saveCounts["b.png"])
	}
}

// TestMigrateFilesPropagatesContextCancellation verifies a cancelled context
// stops the run instead of looping forever.
func TestMigrateFilesPropagatesContextCancellation(t *testing.T) {
	table := &fakeTable{rows: []*filedata.Entity{entity(1, "a.png", []byte("aaa"))}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := migrateFiles(ctx, newFakeProvider(), table.queryByID, table.clearContent, 0, true, nil, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("migrateFiles() error = %v, want context.Canceled", err)
	}
}

// TestMigrateFilesDoesNotReuploadTrailingRowsOnStall is the clearAfterMigrate
// = false regression: a permanently-failing object must not cause trailing rows
// to be re-uploaded or re-counted on every stall re-scan round.
func TestMigrateFilesDoesNotReuploadTrailingRowsOnStall(t *testing.T) {
	table := &fakeTable{rows: []*filedata.Entity{
		entity(1, "a.png", []byte("aaa")),
		entity(2, "b.png", []byte("bbb")),
		entity(3, "c.png", []byte("ccc")),
	}}
	provider := newFakeProvider()
	provider.failures["b.png"] = 1 << 30 // permanently failing, clearAfterMigrate=false

	gotP, gotF, err := migrateFiles(context.Background(), provider, table.queryByID, table.clearContent, 0, false, nil, nil)
	if err == nil {
		t.Fatal("migrateFiles() error = nil with persistent failure, want abort error")
	}
	if !strings.Contains(err.Error(), "b.png") {
		t.Fatalf("migrateFiles() error = %q, want it to name the failing object", err)
	}
	if gotP != 2 || gotF != 1 {
		t.Fatalf("migrateFiles() = (%d, %d), want (2, 1)", gotP, gotF)
	}
	// Trailing rows are uploaded exactly once, not once per stall round.
	if provider.saveCounts["c.png"] != 1 {
		t.Fatalf("c.png uploaded %d times, want exactly 1", provider.saveCounts["c.png"])
	}
	if provider.saveCounts["a.png"] != 1 {
		t.Fatalf("a.png uploaded %d times, want exactly 1", provider.saveCounts["a.png"])
	}
	// The failed object keeps its BLOB (still retryable) and was never stored.
	if _, _, err := provider.Get(context.Background(), "b.png"); !errors.Is(err, storageservice.ErrNotFound) {
		t.Fatalf("b.png unexpectedly present in provider: %v", err)
	}
	if table.byName("b.png").Data == nil {
		t.Fatal("failed row b.png BLOB was cleared; it must remain retryable")
	}
	// With clearAfterMigrate=false blobs of successfully migrated rows stay.
	if table.byName("c.png").Data == nil {
		t.Fatal("c.png BLOB was cleared despite clearAfterMigrate=false")
	}
}

// TestMigrateFilesFailsFastOnProviderOutage verifies a bucket-wide outage
// aborts after ~maxConsecutiveFailures failed uploads instead of re-scanning
// the whole window for ~50 rounds (regression guard for fail-fast behavior).
func TestMigrateFilesFailsFastOnProviderOutage(t *testing.T) {
	const rowCount = 60
	rows := make([]*filedata.Entity, 0, rowCount)
	for i := 1; i <= rowCount; i++ {
		rows = append(rows, entity(uint64(i), fmt.Sprintf("f%02d.png", i), []byte("data")))
	}
	provider := newFakeProvider()
	provider.failAll = true

	_, _, err := migrateFiles(context.Background(), provider, (&fakeTable{rows: rows}).queryByID, (&fakeTable{rows: rows}).clearContent, 0, true, nil, nil)
	if err == nil {
		t.Fatal("migrateFiles() error = nil under full outage, want abort error")
	}
	if !strings.Contains(err.Error(), "abort") {
		t.Fatalf("migrateFiles() error = %q, want an abort message", err)
	}
	// Fail-fast: abort after ~maxConsecutiveFailures failed PUTs in the first
	// batch, not one attempt per row per stall round (which would be ~3000).
	total := 0
	for _, c := range provider.saveCounts {
		total += c
	}
	if total >= maxConsecutiveFailures+rowCount {
		t.Fatalf("provider outage caused %d Save attempts, want fail-fast near %d", total, maxConsecutiveFailures)
	}
}

// TestMigrateFilesAbortsReportsProgress verifies the abort path reports the
// frozen cursor and the real distinct failed count through onProgress before
// returning, so RunMigrateTask persists a truthful taskJson (a bucket-wide
// outage must not show failed=0) and a retry resumes from the frozen cursor.
func TestMigrateFilesAbortsReportsProgress(t *testing.T) {
	const rowCount = 60
	rows := make([]*filedata.Entity, 0, rowCount)
	for i := 1; i <= rowCount; i++ {
		rows = append(rows, entity(uint64(i), fmt.Sprintf("f%02d.png", i), []byte("data")))
	}
	provider := newFakeProvider()
	provider.failAll = true

	var calls int
	var lastCursor uint64
	var lastProcessed, lastFailed int64
	onProgress := func(id uint64, processed, failed int64) {
		calls++
		lastCursor, lastProcessed, lastFailed = id, processed, failed
	}

	gotP, gotF, err := migrateFiles(context.Background(), provider, (&fakeTable{rows: rows}).queryByID, (&fakeTable{rows: rows}).clearContent, 0, true, nil, onProgress)
	if err == nil {
		t.Fatal("migrateFiles() error = nil under full outage, want abort error")
	}
	// The abort path must have reported the frozen cursor and the real failure
	// count exactly once before returning.
	if calls != 1 {
		t.Fatalf("onProgress called %d times, want 1 (the abort report)", calls)
	}
	if lastFailed != gotF || lastFailed == 0 {
		t.Fatalf("abort reported failed = %d, want returned %d (real distinct count)", lastFailed, gotF)
	}
	if lastCursor != 0 || lastProcessed != 0 {
		t.Fatalf("abort reported cursor=%d processed=%d, want (0, 0) with no success", lastCursor, lastProcessed)
	}
	if gotP != 0 {
		t.Fatalf("migrateFiles() processed = %d, want 0 under full outage", gotP)
	}
}

// TestMigrateFilesCumulativeProgressAcrossRetries is the regression for review
// #152 P2: a worker retry must not overwrite the task-level cumulative progress
// with a single run's local counters. Run 1 migrates a, freezes the cursor on b
// (still migrating c after it); run 2 resumes from the frozen cursor once b
// recovers. Because processed is derived from the persisted cursor via
// countUpTo, the final report is the cumulative 3/3 — not run 2's local count
// of 2, which would make a fully-migrated task display "2/3".
func TestMigrateFilesCumulativeProgressAcrossRetries(t *testing.T) {
	table := &fakeTable{rows: []*filedata.Entity{
		entity(1, "a.png", []byte("aaa")),
		entity(2, "b.png", []byte("bbb")),
		entity(3, "c.png", []byte("ccc")),
	}}
	provider := newFakeProvider()
	provider.failures["b.png"] = 1 << 30 // b permanently fails during run 1

	// Mirrors filedata.CountFilesUpTo: rows with id <= cursor are migrated.
	countUpTo := func(cursor uint64) int64 {
		var n int64
		for _, r := range table.rows {
			if r.Id <= cursor {
				n++
			}
		}
		return n
	}

	var run1Cursor uint64
	var run1Processed int64
	_, _, err := migrateFiles(context.Background(), provider, table.queryByID, table.clearContent, 0, true, countUpTo, func(id uint64, processed, _ int64) {
		run1Cursor, run1Processed = id, processed
	})
	if err == nil {
		t.Fatal("run 1 error = nil, want abort (b.png keeps failing)")
	}
	if run1Cursor != 1 || run1Processed != 1 {
		t.Fatalf("run 1 reported cursor=%d processed=%d, want (1, 1) frozen before b.png", run1Cursor, run1Processed)
	}

	// b recovers; run 2 resumes from the frozen cursor.
	provider.failures["b.png"] = 0
	var run2Cursor uint64
	var run2Processed, run2Failed int64
	_, _, err = migrateFiles(context.Background(), provider, table.queryByID, table.clearContent, run1Cursor, true, countUpTo, func(id uint64, processed, failed int64) {
		run2Cursor, run2Processed, run2Failed = id, processed, failed
	})
	if err != nil {
		t.Fatalf("run 2 error = %v, want nil after b recovers", err)
	}
	// The cumulative progress must reach 3/3 — not run 2's local count of 2 —
	// so a fully migrated task never displays "2/3".
	if run2Cursor != 3 || run2Processed != 3 || run2Failed != 0 {
		t.Fatalf("run 2 reported cursor=%d processed=%d failed=%d, want (3, 3, 0) cumulative", run2Cursor, run2Processed, run2Failed)
	}
}

// cancelOnSaveN wraps fakeProvider and cancels the context right before the
// Nth Save, simulating a shutdown mid-batch.
type cancelOnSaveN struct {
	*fakeProvider
	cancel context.CancelFunc
	n      int
	count  int
}

func (p *cancelOnSaveN) Save(ctx context.Context, name string, data []byte, typ string) error {
	p.count++
	if p.count == p.n {
		p.cancel()
	}
	return p.fakeProvider.Save(ctx, name, data, typ)
}

// TestMigrateFilesMidBatchCancelDoesNotCountAsFailure verifies that a context
// cancellation in the middle of a batch is propagated as the cancel error and
// is not counted as a storage failure (which would inflate failed and could
// trigger a misleading fail-fast abort).
func TestMigrateFilesMidBatchCancelDoesNotCountAsFailure(t *testing.T) {
	table := &fakeTable{rows: []*filedata.Entity{
		entity(1, "a.png", []byte("aaa")),
		entity(2, "b.png", []byte("bbb")),
		entity(3, "c.png", []byte("ccc")),
	}}
	// a.png succeeds, then the context is cancelled before b.png is saved.
	ctx, cancel := context.WithCancel(context.Background())
	provider := &cancelOnSaveN{fakeProvider: newFakeProvider(), cancel: cancel, n: 2}

	var lastCursor uint64
	var lastProcessed, lastFailed int64
	onProgress := func(id uint64, processed, failed int64) {
		lastCursor, lastProcessed, lastFailed = id, processed, failed
	}

	gotP, gotF, err := migrateFiles(ctx, provider, table.queryByID, table.clearContent, 0, true, nil, onProgress)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("migrateFiles() error = %v, want context.Canceled", err)
	}
	// The cancelled Save must not be counted as a failed object.
	if gotF != 0 {
		t.Fatalf("migrateFiles() failed = %d, want 0 (cancel is not a storage failure)", gotF)
	}
	if gotP != 1 {
		t.Fatalf("migrateFiles() processed = %d, want 1 (only a.png succeeded before cancel)", gotP)
	}
	// Progress is reported up to the cancel point so the persisted cursor is
	// not stale and a retry resumes from where the run actually stopped.
	if lastCursor != 1 || lastProcessed != 1 || lastFailed != 0 {
		t.Fatalf("onProgress reported cursor=%d processed=%d failed=%d, want (1, 1, 0)", lastCursor, lastProcessed, lastFailed)
	}
}

package migration

import "errors"

// ErrLockUnavailable reports that the versioned data migration lock could not
// be acquired because another instance is (or recently was) migrating. It is
// non-fatal: schema has already been applied and data migrations will retry on
// a later start, so serve proceeds and the standalone migrate command reports
// "skipped" and exits successfully.
var ErrLockUnavailable = errors.New("migration: versioned migration lock unavailable")

// ErrRetryLater reports that a versioned data migration was deferred (e.g.
// Meilisearch unavailable) and the migration version was intentionally not
// advanced. It is non-fatal: serve proceeds with degraded functionality and
// the standalone migrate command reports "deferred" and exits successfully.
var ErrRetryLater = errors.New("migration: deferred, will retry on next run")

// Deferred reports whether err is a non-fatal migration outcome that must not
// block startup or fail a release step: either the versioned migration lock is
// held by another instance, or a data migration was deferred to a later run.
func Deferred(err error) bool {
	return errors.Is(err, ErrRetryLater) || errors.Is(err, ErrLockUnavailable)
}

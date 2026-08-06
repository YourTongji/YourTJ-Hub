// Package storageservice provides a pluggable file storage abstraction.
//
// The active provider is selected by the storage settings page config
// (local SQLite BLOB by default, S3-compatible object storage when configured).
// The local provider implementation lives in the file model package and is
// registered through RegisterLocalFactory to avoid an import cycle.
package storageservice

import (
	"context"
	"errors"
	"reflect"
	"sync"

	"github.com/leancodebox/GooseForum/app/models/forum/pageConfig"
	"github.com/leancodebox/GooseForum/app/models/hotdataserve"
)

// Provider abstracts object storage backends (SQLite BLOB, S3-compatible).
type Provider interface {
	// Save stores data under name with the given content type.
	Save(ctx context.Context, name string, data []byte, contentType string) error
	// Get returns the stored bytes and content type for name.
	Get(ctx context.Context, name string) ([]byte, string, error)
	// Delete removes the object identified by name.
	Delete(ctx context.Context, name string) error
	// Exists reports whether an object exists for name.
	Exists(ctx context.Context, name string) (bool, error)
}

// Provider names used by the storage settings config.
const (
	ProviderLocal = "local"
	ProviderS3    = "s3"
)

// ErrNotFound is returned by Get/Exists when the object does not exist.
var ErrNotFound = errors.New("storage object not found")

var (
	mu            sync.RWMutex
	current       Provider
	currentFinger any
	localFactory  func() Provider
)

// RegisterLocalFactory installs the factory that builds the local provider.
// The local provider lives in the file model package to avoid an import cycle.
func RegisterLocalFactory(f func() Provider) {
	mu.Lock()
	defer mu.Unlock()
	localFactory = f
}

// IsLocalProvider reports whether the configured provider is the local one.
func IsLocalProvider() bool {
	cfg := hotdataserve.GetStorageSettingsConfigCache()
	return cfg.Provider != ProviderS3
}

// Current returns the active provider, rebuilt whenever storage settings change.
func Current() Provider {
	cfg := hotdataserve.GetStorageSettingsConfigCache()
	mu.Lock()
	defer mu.Unlock()
	if current != nil && reflect.DeepEqual(currentFinger, cfg) {
		return current
	}
	p := buildFromConfig(cfg)
	if p == nil && localFactory != nil {
		p = localFactory()
	}
	if p == nil {
		panic("storageservice: no storage provider available")
	}
	current = p
	currentFinger = cfg
	return current
}

func buildFromConfig(cfg pageConfig.StorageSettings) Provider {
	switch cfg.Provider {
	case ProviderS3:
		p, err := NewS3Provider(cfg)
		if err != nil {
			return nil
		}
		return p
	default:
		if localFactory == nil {
			return nil
		}
		return localFactory()
	}
}

// PublicAccessPath returns the public URL path for a stored file name.
// When a public URL prefix is configured, it is used instead of the
// local /file/img proxy route so clients can load from CDN/object storage
// directly. Existing markdown content keeps working through the proxy route.
func PublicAccessPath(name string) string {
	cfg := hotdataserve.GetStorageSettingsConfigCache()
	if cfg.PublicUrlPrefix != "" {
		return trimRightSlash(cfg.PublicUrlPrefix) + "/" + trimLeftSlash(name)
	}
	return "/file/img/" + name
}

func trimRightSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

func trimLeftSlash(s string) string {
	for len(s) > 0 && s[0] == '/' {
		s = s[1:]
	}
	return s
}

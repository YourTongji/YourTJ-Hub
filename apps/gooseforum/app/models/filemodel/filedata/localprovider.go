package filedata

import (
	"context"

	"github.com/leancodebox/GooseForum/app/service/storageservice"
)

// localProvider stores file bytes in the SQLite BLOB column, which is the
// default behavior of the forum. It is registered as the local storage
// provider implementation so storageservice never needs to import this package.
type localProvider struct{}

func (localProvider) Save(_ context.Context, name string, data []byte, contentType string) error {
	entity := GetByName(name)
	if entity.Id == 0 {
		// Row missing: create one (used by migration/import paths that write
		// through the provider abstraction while the active provider is local).
		entity = Entity{
			Name: name,
			Type: contentType,
			Data: data,
		}
		return builder().Create(&entity).Error
	}
	entity.Data = data
	entity.Type = contentType
	return builder().Where("id = ?", entity.Id).Update("content", data).Error
}

func (localProvider) Get(_ context.Context, name string) ([]byte, string, error) {
	entity := GetByName(name)
	if entity.Id == 0 {
		return nil, "", storageservice.ErrNotFound
	}
	return entity.Data, entity.Type, nil
}

func (localProvider) Delete(_ context.Context, name string) error {
	return DeleteByName(name)
}

func (localProvider) Exists(_ context.Context, name string) (bool, error) {
	entity := GetByName(name)
	return entity.Id != 0, nil
}

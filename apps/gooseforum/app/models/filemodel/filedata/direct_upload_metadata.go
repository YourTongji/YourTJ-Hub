package filedata

import (
	"context"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/storageservice"
)

// directUploadMetadataStore adapts the filedata repository to the
// storageservice.DirectUploadMetadataStore interface. It is registered in
// init() so the direct upload lifecycle can create/read/update pending rows
// without storageservice importing this package (which would be a cycle).
type directUploadMetadataStore struct{}

func init() {
	storageservice.RegisterDirectUploadMetadataStore(directUploadMetadataStore{})
}

func (directUploadMetadataStore) CreateFileMetadata(ctx context.Context, userId uint64, name, fileType string, size int64) (*storageservice.FileMetadata, error) {
	entity, err := CreateFileMetadata(ctx, userId, name, fileType, size)
	if err != nil {
		return nil, err
	}
	return toFileMetadata(entity), nil
}

func (directUploadMetadataStore) GetPendingFileMetadataByName(ctx context.Context, name string) (*storageservice.FileMetadata, error) {
	entity, err := GetPendingFileMetadataByNameContext(ctx, name)
	if err != nil {
		return nil, err
	}
	return toFileMetadata(entity), nil
}

func (directUploadMetadataStore) GetFileMetadataByName(ctx context.Context, name string) (*storageservice.FileMetadata, error) {
	entity, err := GetFileMetadataByNameContext(ctx, name)
	if err != nil {
		return nil, err
	}
	return toFileMetadata(entity), nil
}

func (directUploadMetadataStore) MarkFileReady(ctx context.Context, name string) (*storageservice.FileMetadata, error) {
	entity, err := MarkFileReady(ctx, name)
	if err != nil {
		return nil, err
	}
	return toFileMetadata(entity), nil
}

func (directUploadMetadataStore) ListPendingFilesBefore(ctx context.Context, before time.Time, limit int) ([]storageservice.FileMetadata, error) {
	entities, err := ListPendingFilesBefore(ctx, before, limit)
	if err != nil {
		return nil, err
	}
	items := make([]storageservice.FileMetadata, 0, len(entities))
	for index := range entities {
		items = append(items, *toFileMetadata(&entities[index]))
	}
	return items, nil
}

func (directUploadMetadataStore) DeleteByName(ctx context.Context, name string) error {
	return DeleteByNameContext(ctx, name)
}

func toFileMetadata(entity *Entity) *storageservice.FileMetadata {
	return &storageservice.FileMetadata{
		Id:        entity.Id,
		Name:      entity.Name,
		Type:      entity.Type,
		Size:      entity.Size,
		UserId:    entity.UserId,
		CreatedAt: entity.CreatedAt,
		UpdatedAt: entity.UpdatedAt,
	}
}

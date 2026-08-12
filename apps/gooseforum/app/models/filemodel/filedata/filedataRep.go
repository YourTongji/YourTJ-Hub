package filedata

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/storageservice"

	"github.com/google/uuid"
)

type FileResource struct {
	Id        uint64    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Size      int64     `json:"size"`
	UserId    uint64    `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
	URL       string    `json:"url"`
	Data      []byte    `json:"-"`
}

type FileResourcePageResult struct {
	List     []FileResource
	Page     int
	PageSize int
	MaxId    int64
}

var supportedImageTypes = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".bmp":  "image/bmp",
}

// CheckImageType returns the image MIME type for supported extensions.
func CheckImageType(filename string) (string, error) {
	ext := strings.ToLower(path.Ext(filename))
	if contentType, ok := supportedImageTypes[ext]; ok {
		return contentType, nil
	}
	return "", fmt.Errorf("unsupported image type: %s", ext)
}

func create(entity *Entity) int64 {
	result := builder().Create(entity)
	return result.RowsAffected
}

func GetByName(name string) (entity Entity) {
	builder().Where(queryopt.Eq(fieldName, name)).First(&entity)
	return
}

func SaveFile(userId uint64, name string, fileType string, data []byte) (*Entity, error) {
	if GetByName(name).Id != 0 {
		return nil, fmt.Errorf("file already exists: %s", name)
	}
	entity := &Entity{
		Name:   name,
		Type:   fileType,
		Data:   data,
		UserId: userId,
	}
	affected := create(entity)
	if affected == 0 {
		return nil, errors.New("failed to save file, possibly duplicate name")
	}
	// In object storage mode the bytes are mirrored to the provider. The BLOB
	// column is kept as fallback so reads stay correct during migration.
	if !storageservice.IsLocalProvider() {
		if err := storageservice.Current().Save(context.Background(), name, data, fileType); err != nil {
			_ = DeleteByName(name)
			return nil, fmt.Errorf("save to storage provider: %w", err)
		}
	}
	return entity, nil
}

func GetFileByName(name string) (*Entity, error) {
	entity := GetByName(name)
	if entity.Id == 0 {
		return nil, errors.New("file not found")
	}
	if storageservice.IsLocalProvider() {
		return &entity, nil
	}
	data, contentType, err := storageservice.Current().Get(context.Background(), name)
	if err == nil {
		entity.Data = data
		entity.Type = contentType
		return &entity, nil
	}
	if !errors.Is(err, storageservice.ErrNotFound) {
		return nil, err
	}
	// Fall back to the BLOB column for legacy rows not yet migrated.
	if entity.Data != nil {
		return &entity, nil
	}
	return nil, errors.New("file not found")
}

// DeleteByName removes the file row and, in object storage mode, the object.
func DeleteByName(name string) error {
	if !storageservice.IsLocalProvider() {
		if err := storageservice.Current().Delete(context.Background(), name); err != nil && !errors.Is(err, storageservice.ErrNotFound) {
			return err
		}
	}
	return builder().Where(queryopt.Eq(fieldName, name)).Delete(&Entity{}).Error
}

// QueryById returns rows with id greater than startId, ascending (migration cursor).
func QueryById(startId uint64, limit int) (entities []*Entity) {
	builder().Where(queryopt.Gt("id", startId)).Limit(limit).Order(queryopt.Asc("id")).Find(&entities)
	return
}

// CountFiles returns the total number of file rows.
func CountFiles() int64 {
	var count int64
	builder().Count(&count)
	return count
}

// CountFilesUpTo returns the number of file rows with id <= cursor. The
// migration cursor only advances past successfully migrated (or already empty)
// rows, so this is the task-level cumulative count of migrated objects. It is
// derived from the persisted cursor instead of per-run local counters, so a
// worker retry never overwrites the accumulated progress with a partial count.
func CountFilesUpTo(cursor uint64) int64 {
	var count int64
	builder().Where(queryopt.Le("id", cursor)).Count(&count)
	return count
}

// ClearContentByName clears the BLOB column after a successful migration.
func ClearContentByName(name string) error {
	return builder().Where(queryopt.Eq(fieldName, name)).Update("content", nil).Error
}

func FileResourcePage(page, pageSize int) FileResourcePageResult {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 50 {
		pageSize = 50
	}

	var maxId int64
	builder().Select("id").Order("id DESC").Limit(1).Scan(&maxId)
	upperId := maxId - int64((page-1)*pageSize)

	var list []FileResource
	builder().
		Where("id <= ?", upperId).
		Select("id, name, assert_type AS type, LENGTH(content) AS size, user_id, created_at").
		Order("id DESC").
		Limit(pageSize).
		Scan(&list)
	for index := range list {
		list[index].URL = list[index].GetAccessPath()
	}
	return FileResourcePageResult{List: list, Page: page, PageSize: pageSize, MaxId: maxId}
}

func (itself FileResource) GetAccessPath() string {
	return accessPath(itself.Name)
}

// CountDailyUploads returns the number of files uploaded by a user today.
func CountDailyUploads(userId uint64) int64 {
	return CountUserUploadsToday(userId)
}

func SaveFileFromUpload(userId uint64, fileData []byte, filename string, customPath string) (*Entity, error) {
	if len(fileData) > MaxFileSize {
		return nil, fmt.Errorf("file size exceeds maximum limit of %dMB", MaxFileSize/(1024*1024))
	}

	contentType, err := CheckImageType(filename)
	if err != nil {
		return nil, err
	}

	fileExt := path.Ext(filename)
	newFilename := fmt.Sprintf("%s/%s%s",
		customPath,
		uuid.New().String(),
		fileExt)

	return SaveFile(userId, newFilename, contentType, fileData)
}

const (
	MaxFileSize = 4 * 1024 * 1024 // 4MB
	AvatarPath  = "avatars"
)

type AvatarUpload struct {
	Filename string
	Data     []byte
}

// SaveAvatar stores an uploaded avatar file.
func SaveAvatar(userId uint64, fileData []byte, filename string) (*Entity, error) {
	avatarPath := fmt.Sprintf("%s/avatar_%d_%d",
		AvatarPath,
		userId,
		time.Now().Unix())

	return SaveFileFromUpload(userId, fileData, filename, avatarPath)
}

func SaveAvatarSet(userId uint64, uploads []AvatarUpload) ([]*Entity, error) {
	if len(uploads) == 0 {
		return nil, errors.New("avatar files are required")
	}
	if len(uploads) > 2 {
		return nil, errors.New("avatar files exceed maximum limit of 2")
	}

	avatarPath := fmt.Sprintf("%s/%d/%d", AvatarPath, userId, time.Now().UnixNano())
	avatarNames := []string{"avatar", "avatar_medium"}
	entities := make([]*Entity, 0, len(uploads))

	for index, upload := range uploads {
		if len(upload.Data) > MaxFileSize {
			return nil, fmt.Errorf("file size exceeds maximum limit of %dMB", MaxFileSize/(1024*1024))
		}

		contentType, err := CheckImageType(upload.Filename)
		if err != nil {
			return nil, err
		}

		fileExt := strings.ToLower(path.Ext(upload.Filename))
		entity, err := SaveFile(userId, fmt.Sprintf("%s/%s%s", avatarPath, avatarNames[index], fileExt), contentType, upload.Data)
		if err != nil {
			return nil, err
		}
		entities = append(entities, entity)
	}

	return entities, nil
}

// CountUserUploadsInTimeRange counts uploads for a user within a time range.
func CountUserUploadsInTimeRange(userId uint64, startTime, endTime time.Time) int64 {
	var count int64
	builder().Where("user_id = ? AND created_at >= ? AND created_at <= ?", userId, startTime, endTime).Count(&count)
	return count
}

// CountUserUploadsToday counts uploads for a user today.
func CountUserUploadsToday(userId uint64) int64 {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour).Add(-time.Nanosecond)
	return CountUserUploadsInTimeRange(userId, startOfDay, endOfDay)
}

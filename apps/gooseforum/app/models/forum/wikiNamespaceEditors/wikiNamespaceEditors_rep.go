package wikiNamespaceEditors

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"gorm.io/gorm"
)

// ListByNamespace 返回某 namespace 的贡献者列表。
func ListByNamespace(namespace string) []*Entity {
	var entities []*Entity
	builder().
		Where(queryopt.Eq("namespace", namespace)).
		Order(queryopt.Asc("id")).
		Find(&entities)
	return entities
}

// UserIDsByNamespace 返回某 namespace 的贡献者 user_id 列表（去重）。
func UserIDsByNamespace(namespace string) []uint64 {
	entities := ListByNamespace(namespace)
	ids := make([]uint64, 0, len(entities))
	seen := map[uint64]struct{}{}
	for _, e := range entities {
		if _, ok := seen[e.UserId]; ok {
			continue
		}
		seen[e.UserId] = struct{}{}
		ids = append(ids, e.UserId)
	}
	return ids
}

func IsEditor(namespace string, userId uint64) bool {
	var count int64
	builder().
		Where(queryopt.Eq("namespace", namespace)).
		Where(queryopt.Eq("user_id", userId)).
		Count(&count)
	return count > 0
}

// SetEditorsTx 在给定事务内整表替换某 namespace 的贡献者列表。
func SetEditorsTx(tx *gorm.DB, namespace string, userIds []uint64, addedBy uint64) error {
	if err := tx.Table(tableName).
		Where(queryopt.Eq("namespace", namespace)).
		Delete(&Entity{}).Error; err != nil {
		return err
	}
	for _, userId := range userIds {
		if userId == 0 {
			continue
		}
		entity := &Entity{
			Namespace: namespace,
			UserId:    userId,
			AddedBy:   addedBy,
		}
		if err := tx.Table(tableName).Create(entity).Error; err != nil {
			return err
		}
	}
	return nil
}

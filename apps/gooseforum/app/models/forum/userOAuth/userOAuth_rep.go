package userOAuth

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Create 创建OAuth记录
func Create(entity *Entity) error {
	return builder().Create(entity).Error
}

// Delete 删除OAuth记录
func Delete(id uint64) error {
	return builder().Delete(&Entity{}, id).Error
}

// GetByProviderAndUID 根据提供商和UID获取OAuth记录
func GetByProviderAndUID(provider, providerUID string) *Entity {
	var entity Entity
	err := builder().Where("provider = ? AND provider_uid = ?", provider, providerUID).First(&entity).Error
	if err != nil {
		return nil
	}
	return &entity
}

// GetByUserIDAndProvider 根据用户ID和提供商获取OAuth记录
func GetByUserIDAndProvider(userID uint64, provider string) *Entity {
	var entity Entity
	err := builder().Where("user_id = ? AND provider = ?", userID, provider).First(&entity).Error
	if err != nil {
		return nil
	}
	return &entity
}

//func saveAll(entities []*Entity) int64 {
//	result := builder().Save(entities)
//	return result.RowsAffected
//}

//func deleteEntity(entity *Entity) int64 {
//	result := builder().Delete(entity)
//	return result.RowsAffected
//}

//func all() (entities []*Entity) {
//	builder().Find(&entities)
//	return
//}

// GetByIdentityTx can lock the binding after the caller locks its owning user.
func GetByIdentityTx(tx *gorm.DB, provider, subject string, lock bool) (Entity, error) {
	if lock {
		tx = tx.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var binding Entity
	err := tx.Where("provider = ? AND provider_uid = ?", provider, subject).First(&binding).Error
	return binding, err
}

func GetByOwnerTx(tx *gorm.DB, userID uint64, provider string) (Entity, error) {
	var binding Entity
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("provider = ? AND user_id = ?", provider, userID).First(&binding).Error
	return binding, err
}

// SaveAppleBindingTx stores only the encrypted revocation credential. Provider
// access tokens and raw identity tokens are never retained.
func SaveAppleBindingTx(tx *gorm.DB, binding *Entity, encrypted string) error {
	if binding.Id != 0 {
		return tx.Model(binding).Update("apple_refresh_token", encrypted).Error
	}
	binding.AppleRefreshToken = encrypted
	return tx.Create(binding).Error
}

func DeleteTx(tx *gorm.DB, binding *Entity) error { return tx.Delete(binding).Error }

func ListByOwnerTx(tx *gorm.DB, userID uint64, providers []string) ([]Entity, error) {
	var bindings []Entity
	err := tx.Where("user_id = ? AND provider IN ?", userID, providers).Find(&bindings).Error
	return bindings, err
}

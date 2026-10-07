package rolePermissionRs

import "gorm.io/gorm"

func HasExplicitTx(tx *gorm.DB, roleID, permissionID uint64) (bool, error) {
	var count int64
	err := tx.Model(&Entity{}).Where("role_id = ? AND permission_id = ? AND effective = 1", roleID, permissionID).Count(&count).Error
	return count > 0, err
}

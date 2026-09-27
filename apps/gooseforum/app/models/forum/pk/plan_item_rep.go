package pk

import (
	"errors"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrPlanOwnerClosed = errors.New("schedule owner closed")

var ErrPlanConflict = errors.New("plan revision changed")
var ErrPlanDeleted = errors.New("plan was deleted")
var ErrPlanQuota = errors.New("plan quota exceeded")
var ErrSnapshotRetired = errors.New("snapshot writes retired for this account")

const MaxPlanItems = 10

func lockPlanOwner(tx *gorm.DB, userID uint64) (PlanSyncOwner, error) {
	owner := PlanSyncOwner{UserID: userID}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&owner).Error; err != nil {
		return owner, err
	}
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&owner, "user_id = ?", userID).Error
	if err == nil && owner.Closed {
		return owner, ErrPlanOwnerClosed
	}
	return owner, err
}

// Migration and first read/write share the owner's lock with legacy writes.
func migratePlanItems(tx *gorm.DB, userID uint64) error {
	owner, err := lockPlanOwner(tx, userID)
	if err != nil {
		return err
	}
	if owner.Migrated {
		return nil
	}
	var legacy ScheduleSnapshotEntity
	err = tx.Where("user_id = ?", userID).First(&legacy).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err == nil {
		for _, plan := range legacy.Plans {
			item := PlanItem{UserID: userID, PlanID: plan.Id, Revision: 1, Payload: plan, UpdatedAt: legacy.UpdatedAt}
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
		}
	}
	return tx.Model(&PlanSyncOwner{}).Where("user_id = ?", userID).Update("migrated", true).Error
}

func ListPlanItems(userID uint64) ([]PlanItem, error) { return listPlanItems(db.Connect(), userID) }
func listPlanItems(conn *gorm.DB, userID uint64) ([]PlanItem, error) {
	items := []PlanItem{}
	err := conn.Transaction(func(tx *gorm.DB) error {
		if err := migratePlanItems(tx, userID); err != nil {
			return err
		}
		return tx.Where("user_id = ?", userID).Order("plan_id").Find(&items).Error
	})
	return items, err
}

// SavePlanItem never treats a missing positive revision as creation. On conflict,
// current contains the observed remote item and is returned without another read.
func SavePlanItem(userID uint64, plan PlanPayload, base int64) (*PlanItem, error) {
	return savePlanItem(db.Connect(), userID, plan, base)
}
func savePlanItem(conn *gorm.DB, userID uint64, plan PlanPayload, base int64) (*PlanItem, error) {
	var current *PlanItem
	var conflict error
	err := conn.Transaction(func(tx *gorm.DB) error {
		if err := migratePlanItems(tx, userID); err != nil {
			return err
		}
		var existing PlanItem
		err := tx.Where("user_id = ? AND plan_id = ?", userID, plan.Id).First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if base != 0 {
				conflict = ErrPlanDeleted
				return nil
			}
			var count int64
			if err := tx.Model(&PlanItem{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
				return err
			}
			if count >= MaxPlanItems {
				conflict = ErrPlanQuota
				return nil
			}
			current = &PlanItem{UserID: userID, PlanID: plan.Id, Revision: 1, Payload: plan, UpdatedAt: time.Now().UTC().Truncate(time.Microsecond)}
			return tx.Create(current).Error
		}
		current = &existing
		if base != existing.Revision || base >= 9007199254740991 {
			conflict = ErrPlanConflict
			return nil
		}
		next := PlanItem{Payload: plan, Revision: base + 1, UpdatedAt: time.Now().UTC().Truncate(time.Microsecond)}
		result := tx.Model(&PlanItem{}).Where("user_id = ? AND plan_id = ? AND revision = ?", userID, plan.Id, base).Select("payload", "revision", "updated_at").Updates(&next)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrPlanConflict
		}
		next.UserID, next.PlanID = userID, plan.Id
		current = &next
		return nil
	})
	if err != nil {
		return nil, err
	}
	return current, conflict
}

func DeletePlanItem(userID uint64, planID string, base int64) (*PlanItem, error) {
	return deletePlanItem(db.Connect(), userID, planID, base)
}
func deletePlanItem(conn *gorm.DB, userID uint64, planID string, base int64) (*PlanItem, error) {
	var current *PlanItem
	var conflict error
	err := conn.Transaction(func(tx *gorm.DB) error {
		if err := migratePlanItems(tx, userID); err != nil {
			return err
		}
		var existing PlanItem
		err := tx.Where("user_id = ? AND plan_id = ?", userID, planID).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if existing.Revision != base {
			current = &existing
			conflict = ErrPlanConflict
			return nil
		}
		return tx.Where("user_id = ? AND plan_id = ? AND revision = ?", userID, planID, base).Delete(&PlanItem{}).Error
	})
	if err != nil {
		return nil, err
	}
	return current, conflict
}

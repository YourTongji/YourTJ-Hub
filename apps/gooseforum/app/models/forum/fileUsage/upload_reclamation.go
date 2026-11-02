package fileUsage

import (
	"errors"
	"slices"

	"gorm.io/gorm"
)

// Ownership rows are also the durable arbitration point between a publisher
// and orphan cleanup. Keep PURGED tombstones: dropping the row would let a
// publisher that already resolved the file register a dangling reference.
const UsageStatusReclaiming = "RECLAIMING"

var ErrUploadReclaimed = errors.New("upload expired; upload the attachment again")

func lockUploadOwnerTx(tx *gorm.DB, name string) error {
	// A no-op UPDATE takes a row lock on PostgreSQL and a write lock on SQLite.
	// Unlike SELECT FOR UPDATE, this also serializes SQLite read/write upgrades.
	return tx.Session(&gorm.Session{NewDB: true}).Model(&Entity{}).
		Where("file_name = ? AND usage_type = ?", name, UsageUploadOwner).
		UpdateColumn("status", gorm.Expr("status")).Error
}

// GuardUploadReferencesTx holds the ownership locks until the content/reference
// transaction commits. Multi-file writers acquire them in filename order.
func GuardUploadReferencesTx(tx *gorm.DB, names []string) error {
	names = slices.Clone(names)
	slices.Sort(names)
	for _, name := range slices.Compact(names) {
		if err := lockUploadOwnerTx(tx, name); err != nil {
			return err
		}
		var blocked int64
		if err := tx.Session(&gorm.Session{NewDB: true}).Model(&Entity{}).
			Where("file_name = ? AND usage_type = ? AND status IN ?", name, UsageUploadOwner,
				[]string{UsageStatusReclaiming, UsageStatusPurged}).Count(&blocked).Error; err != nil {
			return err
		}
		if blocked != 0 {
			return ErrUploadReclaimed
		}
	}
	return nil
}

// BeforeCreate covers all reference writers, including direct transaction
// inserts for moderation revisions and managed assets.
func (entity *Entity) BeforeCreate(tx *gorm.DB) error {
	if entity.UsageType == UsageUploadOwner {
		return nil
	}
	return GuardUploadReferencesTx(tx, []string{entity.FileName})
}

// CheckOwnedUploadAvailable is an early error for expired attachments even
// after the file row has gone. The transactional guard remains authoritative.
func CheckOwnedUploadAvailable(userID uint64, name string) error {
	var blocked int64
	if err := builder().Where("file_name = ? AND user_id = ? AND usage_type = ? AND status IN ?",
		name, userID, UsageUploadOwner, []string{UsageStatusReclaiming, UsageStatusPurged}).Count(&blocked).Error; err != nil {
		return err
	}
	if blocked != 0 {
		return ErrUploadReclaimed
	}
	return nil
}

// ClaimOrphanUpload closes reference admission before touching the separate
// file database or object store. A failed delete is retried from RECLAIMING.
func ClaimOrphanUpload(name string) (claimed bool, err error) {
	return claimOrphanUpload(builder(), name)
}

func claimOrphanUpload(conn *gorm.DB, name string) (claimed bool, err error) {
	err = conn.Transaction(func(tx *gorm.DB) error {
		if err := lockUploadOwnerTx(tx, name); err != nil {
			return err
		}
		var owners []Entity
		if err := tx.Where("file_name = ? AND usage_type = ?", name, UsageUploadOwner).Find(&owners).Error; err != nil {
			return err
		}
		if len(owners) == 0 {
			return nil
		}
		for _, owner := range owners {
			if owner.Status == UsageStatusPurged {
				return nil
			}
		}
		var references int64
		if err := tx.Model(&Entity{}).Where("file_name = ? AND usage_type <> ?", name, UsageUploadOwner).Count(&references).Error; err != nil {
			return err
		}
		if references != 0 {
			return nil
		}
		if err := tx.Model(&Entity{}).Where("file_name = ? AND usage_type = ?", name, UsageUploadOwner).
			UpdateColumn("status", UsageStatusReclaiming).Error; err != nil {
			return err
		}
		claimed = true
		return nil
	})
	return claimed && err == nil, err
}

func FinishOrphanUpload(name string) error {
	return builder().Where("file_name = ? AND usage_type = ? AND status = ?", name, UsageUploadOwner, UsageStatusReclaiming).
		UpdateColumn("status", UsageStatusPurged).Error
}

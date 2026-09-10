package pk

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// ValidateMaterializeSnapshotTx rejects partial imports. With repeatable-read
// isolation, a sync starting later cannot mix old classes with new teachers.
// The active sync may read its own completed fetch under its fenced lease.
func ValidateMaterializeSnapshotTx(tx *gorm.DB, audience Audience, calendars []uint64, claim *FetchLogEntity) error {
	if claim != nil {
		if err := RenewFetchLogLeaseTx(tx, claim); err != nil {
			return err
		}
	}
	for _, id := range calendars {
		if _, err := GetCalendarByAudienceIDTx(tx, audience, id); err != nil {
			return fmt.Errorf("materialize: lookup calendar %d（尚未同步到本地）: %w", id, err)
		}
		var log FetchLogEntity
		err := tx.Model(&FetchLogEntity{}).Where("audience = ? AND calendar_id = ?", audience, ScopeID(audience, id)).Order("id DESC").First(&log).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		} // Imported historical snapshots have no fetch log.
		if err != nil {
			return err
		}
		ownsLease := claim != nil && claim.Id == log.Id && claim.LeaseVersion == log.LeaseVersion
		if log.Status == FetchStatusRunning && !ownsLease {
			return fmt.Errorf("学期 %d 正在同步，请完成后再物化", id)
		}
		if log.Status != FetchStatusCompleted && (log.TotalPages == 0 || log.LastCommittedPage < log.TotalPages) {
			return fmt.Errorf("学期 %d 的排课数据尚未完整抓取，请先完成同步", id)
		}
	}
	return nil
}

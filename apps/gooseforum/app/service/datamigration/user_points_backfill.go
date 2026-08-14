package datamigration

import (
	"fmt"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pointsRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userPoints"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

const initialUserPoints int64 = 100

type UserPointsBackfillResult struct {
	Backfilled int
	Failed     int
	LastFailed string
}

func BackfillMissingUserPoints() UserPointsBackfillResult {
	return BackfillMissingUserPointsWithDB(db.Connect())
}

func BackfillMissingUserPointsWithDB(conn *gorm.DB) UserPointsBackfillResult {
	result := UserPointsBackfillResult{}
	if !conn.Migrator().HasTable("users") || !conn.Migrator().HasTable("user_points") || !conn.Migrator().HasTable("points_record") {
		return result
	}

	err := conn.Transaction(func(tx *gorm.DB) error {
		var missingUserIDs []uint64
		if err := tx.Table("users AS e").
			Select("e.id").
			Joins("LEFT JOIN user_points AS up ON up.user_id = e.id").
			Where("up.user_id IS NULL AND e.deleted_at IS NULL AND e.actor_type = ?", users.ActorTypeHuman).
			Order("e.id").
			Scan(&missingUserIDs).Error; err != nil {
			return err
		}
		if len(missingUserIDs) == 0 {
			return nil
		}

		type ledgerSummary struct {
			UserID    uint64 `gorm:"column:user_id"`
			Total     int64  `gorm:"column:total"`
			InitCount int64  `gorm:"column:init_count"`
		}
		var summaries []ledgerSummary
		if err := tx.Table("points_record").
			Select("user_id, COALESCE(SUM(points_change), 0) AS total, SUM(CASE WHEN action = 'init' THEN 1 ELSE 0 END) AS init_count").
			Where("user_id IN ?", missingUserIDs).
			Group("user_id").
			Scan(&summaries).Error; err != nil {
			return err
		}
		ledgerByUser := make(map[uint64]ledgerSummary, len(summaries))
		for _, summary := range summaries {
			ledgerByUser[summary.UserID] = summary
		}

		now := time.Now()
		balances := make([]userPoints.Entity, 0, len(missingUserIDs))
		records := make([]pointsRecord.Entity, 0, len(missingUserIDs))
		for _, userID := range missingUserIDs {
			summary := ledgerByUser[userID]
			currentPoints := summary.Total
			if summary.InitCount == 0 {
				currentPoints += initialUserPoints
				records = append(records, pointsRecord.Entity{
					UserId:       userID,
					Action:       "init",
					PointsChange: initialUserPoints,
					CreatedAt:    now,
				})
			}
			balances = append(balances, userPoints.Entity{
				UserId:        userID,
				CurrentPoints: currentPoints,
				CreatedAt:     now,
				UpdatedAt:     now,
			})
		}
		if err := tx.CreateInBatches(&balances, 500).Error; err != nil {
			return err
		}
		if len(records) > 0 {
			if err := tx.CreateInBatches(&records, 500).Error; err != nil {
				return err
			}
		}
		result.Backfilled = len(missingUserIDs)
		return nil
	})
	if err != nil {
		result.Failed = 1
		result.LastFailed = fmt.Sprintf("backfill_missing_user_points: %s", err.Error())
	}
	return result
}

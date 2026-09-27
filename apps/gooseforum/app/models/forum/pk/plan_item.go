package pk

import "time"

// PlanSyncOwner serializes migration and quota checks within the scheduler domain.
// The marker survives an empty collection so old clients cannot revive its snapshot.
type PlanSyncOwner struct {
	UserID   uint64 `gorm:"primaryKey;autoIncrement:false"`
	Closed   bool   `gorm:"not null;default:false"`
	Migrated bool   `gorm:"not null;default:false"`
}

func (*PlanSyncOwner) TableName() string { return "pk_plan_sync_owner" }

// PlanItem has an independent revision; device UI preferences never enter Payload.
type PlanItem struct {
	UserID    uint64      `gorm:"primaryKey;autoIncrement:false" json:"-"`
	PlanID    string      `gorm:"primaryKey;type:varchar(64)" json:"-"`
	Revision  int64       `gorm:"not null" json:"revision"`
	Payload   PlanPayload `gorm:"type:json;serializer:json;not null" json:"plan"`
	UpdatedAt time.Time   `json:"updatedAt"`
}

func (*PlanItem) TableName() string { return "pk_plan_item" }

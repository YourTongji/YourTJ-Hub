package course

const offeringInstructorTableName = "course_offering_instructor"

type OfferingInstructorEntity struct {
	OfferingId   uint64 `gorm:"primaryKey;column:offering_id;not null;index:idx_offering_instructor_instructor;" json:"offeringId"`
	InstructorId uint64 `gorm:"primaryKey;column:instructor_id;not null;" json:"instructorId"`
	Role         string `gorm:"column:role;type:varchar(32);not null;default:'';" json:"role"`
}

func (itself *OfferingInstructorEntity) TableName() string {
	return offeringInstructorTableName
}

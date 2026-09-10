package pk

import "time"

const scheduleSnapshotTableName = "pk_schedule_snapshot"

// PlanList 排课方案列表（pk_schedule_snapshot.plans JSON 列的存储形态）。
type PlanList []PlanPayload

// PlanPayload 镜像前端持久化方案（resource/src/site/types/pk.ts 的 PkPlan，
// mobile core 同名类型）。字段与 JSON 键逐字段对齐 web localStorage 的
// pk.plans 序列化形态，云端快照直接复用该 schema；前端未提交的未知字段
// 不保留（两端客户端均以契约为准做 sanitize）。
type PlanPayload struct {
	Id              string                `json:"id"`
	Name            string                `json:"name"`
	CreatedAt       int64                 `json:"createdAt"`
	StagedCourses   []StagedCoursePayload `json:"stagedCourses"`
	SelectedCourses []string              `json:"selectedCourses"`
	CustomEvents    []CustomEventPayload  `json:"customEvents"`
}

// StagedCoursePayload 镜像 PkStagedCourse。status（0 未选 / 1 备选 / 2 已选）
// 是前端持久化字段，随快照原样保存。
type StagedCoursePayload struct {
	CourseCode         string                `json:"courseCode"`
	CourseName         string                `json:"courseName"`
	CourseNameReserved string                `json:"courseNameReserved"`
	Credit             float64               `json:"credit"`
	CourseType         string                `json:"courseType"`
	CourseNature       []string              `json:"courseNature"`
	Teacher            []TeacherPayload      `json:"teacher"`
	Status             int                   `json:"status"`
	CourseDetail       []CourseDetailPayload `json:"courseDetail"`
}

// CourseDetailPayload 镜像 PkCourseDetail（一个教学班）。
type CourseDetailPayload struct {
	ArrangementInfo  []ArrangementPayload `json:"arrangementInfo"`
	Campus           string               `json:"campus"`
	Code             string               `json:"code"`
	TeachingClassId  *uint64              `json:"teachingClassId"`
	IsExclusive      *bool                `json:"isExclusive"`
	Status           *int                 `json:"status"`
	Teachers         []TeacherPayload     `json:"teachers"`
	TeachingLanguage string               `json:"teachingLanguage"`
}

// ArrangementPayload 镜像 PkArrangement（一次上课安排）。
type ArrangementPayload struct {
	ArrangementText string `json:"arrangementText"`
	OccupyDay       int    `json:"occupyDay"`
	OccupyTime      []int  `json:"occupyTime"`
	OccupyWeek      []int  `json:"occupyWeek"`
	OccupyRoom      string `json:"occupyRoom"`
	TeacherAndCode  string `json:"teacherAndCode"`
}

// TeacherPayload 镜像 PkTeacher。
type TeacherPayload struct {
	TeacherName string `json:"teacherName"`
	TeacherCode string `json:"teacherCode"`
}

// CustomEventPayload 镜像 PkCustomEvent（自定义「有事」占位）。
type CustomEventPayload struct {
	Id       string `json:"id"`
	Label    string `json:"label"`
	Day      int    `json:"day"`
	Sections []int  `json:"sections"`
	Weeks    []int  `json:"weeks"`
}

// MajorSelectionPayload 镜像 PkMajorSelection（学期/年级/专业三元组）。
type MajorSelectionPayload struct {
	CalendarId *int    `json:"calendarId"`
	Grade      *int    `json:"grade"`
	Major      *string `json:"major"`
	MajorName  *string `json:"majorName"`
}

// WeekViewPayload 镜像 PkWeekView；Week 为 nil 表示「全部周次」堆叠视图。
type WeekViewPayload struct {
	Week       *int `json:"week"`
	UseCurrent bool `json:"useCurrent"`
}

// ScheduleSnapshotEntity 排课方案云端快照（issue #537）：每用户一行，
// plans/activePlanId/majorSelected/weekView 四字段整体替换写入（服务端
// updated_at 是唯一权威同步时钟，客户端不写时钟）。方案含用户自选课程与
// 自定义占位等个人数据，仅本人可读写，账号注销即删（与 pushDevice 同语义）。
type ScheduleSnapshotEntity struct {
	Id            uint64                `gorm:"primaryKey;column:id;autoIncrement;not null;"`
	UserId        uint64                `gorm:"column:user_id;not null;default:0;uniqueIndex"`
	Plans         PlanList              `gorm:"column:plans;type:json;serializer:json;not null;"`
	ActivePlanId  string                `gorm:"column:active_plan_id;type:varchar(64);not null;default:'';"`
	MajorSelected MajorSelectionPayload `gorm:"column:major_selected;type:json;serializer:json;not null;"`
	WeekView      WeekViewPayload       `gorm:"column:week_view;type:json;serializer:json;not null;"`
	CreatedAt     time.Time             `gorm:"column:created_at;autoCreateTime;<-:create;"`
	// UpdatedAt 服务端权威同步时钟：每次 upsert 刷新，GET 下发给客户端存为
	// pk.syncedAt 用于下次进页的新旧判定；客户端永不回传该值。
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime;"`
}

func (itself *ScheduleSnapshotEntity) TableName() string {
	return scheduleSnapshotTableName
}

package course

import (
	"time"
)

const reviewTableName = "course_review"

// Entity 课程评价：挂在 offering 上；原生评价 author_user_id 必填，
// 导入的历史匿名评价以 0 占位；rating 为 1..5，历史 0 转 NULL 不计平均。
// 复合唯一索引 (offering_id, author_user_id) 在数据库层保证"同一用户对同一
// offering 至多一条"（author_user_id=0 是值而非 NULL，同样约束 legacy 行
// 每 offering 至多一条）；应用层查重负责语义错误映射，索引兜底并发写。
// 隔离窗口清理（courseservice.CleanupExpiredReviewsBatch）把 author_user_id
// 置 NULL 并清空 content：NULL 在唯一索引中彼此不冲突（SQL 标准，
// SQLite/PostgreSQL/MySQL 一致），因此同 offering 的多条已清理行可共存，
// 且同用户可重新写评新建行。
// DeletedAt 是隔离窗口锚点（普通 *time.Time 列，非 GORM 软删语义）：作者
// 删除时写入（MarkReviewDeletedFromTx），清理任务按 deleted_at 超过窗口
// 扫描；全库统一用 status 表达逻辑删除，避免 GORM 软删过滤隐式隐藏 deleted
// 行（Table 查询同样生效）。
type ReviewEntity struct {
	Id                 uint64     `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	OfferingId         uint64     `gorm:"column:offering_id;not null;default:0;index:idx_course_review_offering;uniqueIndex:uniq_course_review_offering_author,priority:1;" json:"offeringId"`
	AuthorUserId       *uint64    `gorm:"column:author_user_id;index:idx_course_review_author;uniqueIndex:uniq_course_review_offering_author,priority:2;" json:"authorUserId"`
	Rating             *int       `gorm:"column:rating;type:int;" json:"rating"`
	Content            string     `gorm:"column:content;type:text;not null;default:'';" json:"content"`
	IsAnonymous        bool       `gorm:"column:is_anonymous;not null;default:false;" json:"isAnonymous"`
	Status             int8       `gorm:"column:status;not null;default:0;index:idx_course_review_status;" json:"status"`
	LegacyHelpfulCount int        `gorm:"column:legacy_helpful_count;not null;default:0;" json:"legacyHelpfulCount"`
	Source             string     `gorm:"column:source;type:varchar(64);not null;default:'';" json:"source"`
	CreatedAt          time.Time  `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt          time.Time  `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt          *time.Time `gorm:"column:deleted_at;" json:"-"`
}

// 评价状态
const (
	ReviewStatusVisible int8 = 0 // 可见
	ReviewStatusHidden  int8 = 1 // 隐藏（对普通用户 404）
	ReviewStatusDeleted int8 = 2 // 删除（隔离窗口后清理正文与作者关联）
)

// AuthorID 返回作者用户 ID；隔离窗口清理把 author_user_id 置 NULL 后统一按 0
// 处理（与历史 legacy 行 author_user_id=0 的口径一致）。
func (itself ReviewEntity) AuthorID() uint64 {
	if itself.AuthorUserId == nil {
		return 0
	}
	return *itself.AuthorUserId
}

// 评价来源（source 列）：原生评价留空，历史导入固定为 legacy-import。
const ReviewSourceLegacyImport string = "legacy-import"

func (itself *ReviewEntity) TableName() string {
	return reviewTableName
}

package courseservice

import (
	"errors"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"gorm.io/gorm"
)

// AdminReviewItem 管理端评价列表项：不泄露匿名作者身份，仅 kind/label。
type AdminReviewItem struct {
	Id         uint64             `json:"id"`
	OfferingId uint64             `json:"offeringId"`
	CourseId   uint64             `json:"courseId"`
	CourseCode string             `json:"courseCode"`
	CourseName string             `json:"courseName"`
	Rating     *int               `json:"rating"`
	Content    string             `json:"content"`
	Status     int8               `json:"status"` // 0 可见 / 1 隐藏 / 2 删除
	Author     ReviewAuthorPayload `json:"author"`
	CreatedAt  string             `json:"createdAt"`
	UpdatedAt  string             `json:"updatedAt"`
}

// AdminReviewQuery 管理端评价检索条件。
type AdminReviewQuery struct {
	Keyword  string // 课程名/主课号/评价正文
	Status   int8   // -1 全部；0/1/2 按状态过滤
	Cursor   uint64
	PageSize int
}

// AdminReviewPage 管理端评价分页结果。
type AdminReviewPage struct {
	Items      []AdminReviewItem `json:"items"`
	NextCursor uint64            `json:"nextCursor"`
	HasNext    bool              `json:"hasNext"`
}

// AdminReviewUpdateInput 管理端编辑评价请求：指针区分“缺省”与“显式置空”。
type AdminReviewUpdateInput struct {
	Rating  *int    `json:"rating"`
	Content *string `json:"content"`
}

// AdminReviewList 返回管理端评价分页（含隐藏/删除状态，跨课程名/课号/正文搜索）。
func AdminReviewList(q AdminReviewQuery) (AdminReviewPage, error) {
	pageSize := q.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 50 {
		pageSize = 50
	}
	entities, err := course.ListReviewsForAdmin(course.AdminReviewQuery{
		Keyword:  q.Keyword,
		Status:   q.Status,
		Cursor:   q.Cursor,
		PageSize: pageSize + 1,
	})
	if err != nil {
		return AdminReviewPage{}, err
	}
	hasNext := len(entities) > pageSize
	page := entities
	if hasNext {
		page = entities[:pageSize]
	}
	items, err := buildAdminReviewItems(page)
	if err != nil {
		return AdminReviewPage{}, err
	}
	nextCursor := uint64(0)
	if hasNext && len(page) > 0 {
		nextCursor = page[len(page)-1].Id
	}
	return AdminReviewPage{Items: items, NextCursor: nextCursor, HasNext: hasNext}, nil
}

// buildAdminReviewItems 批量回填课程信息（offering → course）并构造管理端 DTO。
func buildAdminReviewItems(entities []course.ReviewEntity) ([]AdminReviewItem, error) {
	if len(entities) == 0 {
		return []AdminReviewItem{}, nil
	}
	offeringIds := make([]uint64, 0, len(entities))
	for _, e := range entities {
		offeringIds = append(offeringIds, e.OfferingId)
	}
	offeringMap := course.GetOfferingMapByIds(offeringIds)
	courseIds := make([]uint64, 0, len(offeringMap))
	for _, o := range offeringMap {
		courseIds = append(courseIds, o.CourseId)
	}
	courseMap := course.GetMapByIds(courseIds)
	items := make([]AdminReviewItem, 0, len(entities))
	for _, e := range entities {
		item := AdminReviewItem{
			Id:         e.Id,
			OfferingId: e.OfferingId,
			Rating:     e.Rating,
			Content:    e.Content,
			Status:     e.Status,
			Author:     adminReviewAuthor(e),
			CreatedAt:  e.CreatedAt.Format(time.RFC3339),
			UpdatedAt:  e.UpdatedAt.Format(time.RFC3339),
		}
		if offering, ok := offeringMap[e.OfferingId]; ok {
			item.CourseId = offering.CourseId
			if c := courseMap[offering.CourseId]; c != nil {
				item.CourseCode = c.PrimaryCode
				item.CourseName = c.Name
			}
		}
		items = append(items, item)
	}
	return items, nil
}

// adminReviewAuthor 构造不泄露匿名作者身份的 kind/label（与公开 DTO 口径一致）。
func adminReviewAuthor(entity course.ReviewEntity) ReviewAuthorPayload {
	if entity.IsAnonymous || entity.AuthorUserId == 0 {
		return ReviewAuthorPayload{Kind: "anonymous", Label: "匿名同学"}
	}
	if entity.Source == "legacy-import" || (entity.Source != "" && entity.Source != "native") {
		return ReviewAuthorPayload{Kind: "legacy", Label: "历史匿名评价"}
	}
	return ReviewAuthorPayload{Kind: "member", Label: "同学"}
}

// AdminUpdateReview 管理端编辑评价（rating/content）；评分变化仅对可见评价同步 stats delta，
// 隐藏评价已在 hide 时扣减、不重复计入。使用 CAS 评分更新避免并发丢更新。
func AdminUpdateReview(reviewId uint64, input AdminReviewUpdateInput) (ReviewPayload, error) {
	if input.Rating != nil && (*input.Rating < 1 || *input.Rating > 5) {
		return ReviewPayload{}, ErrRatingOutOfRange
	}
	if input.Content != nil && len([]rune(*input.Content)) > ReviewContentMaxLength {
		return ReviewPayload{}, ErrReviewContentTooLong
	}
	for attempt := 0; attempt < 3; attempt++ {
		var payload ReviewPayload
		err := dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
			entity, err := course.GetReviewTx(tx, reviewId)
			if err != nil {
				return ErrReviewNotFound
			}
			if entity.Status == course.ReviewStatusDeleted {
				return ErrReviewNotFound
			}
			oldRating := 0
			if entity.Rating != nil {
				oldRating = *entity.Rating
			}
			newRating := oldRating
			if input.Rating != nil {
				newRating = *input.Rating
			}
			if input.Content != nil {
				if err := tx.Table((&course.ReviewEntity{}).TableName()).
					Where("id = ?", reviewId).
					Update("content", *input.Content).Error; err != nil {
					return err
				}
			}
			if newRating != oldRating {
				ok, err := course.UpdateReviewRatingFromTx(tx, reviewId, oldRating, newRating)
				if err != nil {
					return err
				}
				if !ok {
					return errReviewRatingConflict
				}
				if entity.Status == course.ReviewStatusVisible {
					offering, err := course.GetOfferingTx(tx, entity.OfferingId)
					if err != nil {
						return err
					}
					// rating NULL → n：对未评分评价补评分时 rating_count 需 +1，
					// 否则平均分按偏小的计数计算导致 ratingAvg 虚高。
					deltaCount := 0
					if oldRating == 0 {
						deltaCount = 1
					}
					if err := course.UpsertCourseStatsTx(tx, offering.CourseId, deltaCount, newRating-oldRating, 0); err != nil {
						return err
					}
					if err := course.UpsertOfferingStatsTx(tx, offering.Id, deltaCount, newRating-oldRating, 0); err != nil {
						return err
					}
				}
			}
			refreshed, err := course.GetReviewTx(tx, reviewId)
			if err != nil {
				return err
			}
			payload = buildReviewPayload(refreshed, 0, 0)
			return nil
		})
		if err == nil {
			return payload, nil
		}
		if !errors.Is(err, errReviewRatingConflict) {
			return ReviewPayload{}, err
		}
	}
	return ReviewPayload{}, errReviewRatingConflict
}

// AdminDeleteReview 管理端硬删除评价（物理删除 + helpful 清理），同步 stats delta；幂等。
// 隔离窗口内（status=deleted）的评价仅物理清理；可见评价扣减 stats，隐藏评价不重复扣减。
func AdminDeleteReview(reviewId uint64) error {
	return dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		entity, err := course.GetReviewTx(tx, reviewId)
		if err != nil {
			return ErrReviewNotFound
		}
		if entity.Status == course.ReviewStatusVisible {
			offering, err := course.GetOfferingTx(tx, entity.OfferingId)
			if err != nil {
				return err
			}
			rating := 0
			if entity.Rating != nil {
				rating = *entity.Rating
			}
			if rating > 0 {
				if err := course.UpsertCourseStatsTx(tx, offering.CourseId, -1, -rating, -1); err != nil {
					return err
				}
				if err := course.UpsertOfferingStatsTx(tx, offering.Id, -1, -rating, -1); err != nil {
					return err
				}
			} else {
				if err := course.UpsertCourseStatsTx(tx, offering.CourseId, 0, 0, -1); err != nil {
					return err
				}
				if err := course.UpsertOfferingStatsTx(tx, offering.Id, 0, 0, -1); err != nil {
					return err
				}
			}
		}
		return hardDeleteReviewTx(tx, reviewId)
	})
}

// hardDeleteReviewTx 事务内物理删除评价及其 helpful 标记。
func hardDeleteReviewTx(tx *gorm.DB, reviewId uint64) error {
	if err := tx.Unscoped().Table((&course.HelpfulEntity{}).TableName()).
		Where("review_id = ?", reviewId).
		Delete(&course.HelpfulEntity{}).Error; err != nil {
		return err
	}
	return tx.Unscoped().Table((&course.ReviewEntity{}).TableName()).
		Where("id = ?", reviewId).
		Delete(&course.ReviewEntity{}).Error
}

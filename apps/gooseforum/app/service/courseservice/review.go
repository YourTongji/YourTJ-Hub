package courseservice

import (
	"errors"
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/http/controllers/markdown2html"
	"github.com/leancodebox/GooseForum/app/models/forum/course"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/service/searchservice"
	"gorm.io/gorm"
)

// 稳定错误 sentinel：控制器据此映射语义 HTTP 状态（400/403/404/409/500）。
var (
	// ErrReviewNotFound 评价不存在或不可见。
	ErrReviewNotFound = errors.New("review not found")
	// ErrReviewNotOwned 不能修改/删除他人评价。
	ErrReviewNotOwned = errors.New("review not owned by user")
	// ErrReviewDuplicate 同一用户对同一 offering 已评价。
	ErrReviewDuplicate = errors.New("review already exists for this offering")
	// ErrOfferingNotFound 开课实例不存在或不可见。
	ErrOfferingNotFound = errors.New("offering not found")
	// ErrRatingOutOfRange rating 越界（非 1..5）。
	ErrRatingOutOfRange = errors.New("rating must be 1..5")
	// ErrReviewContentEmpty 正文为空。
	ErrReviewContentEmpty = errors.New("content is required")
)

// ReviewAuthorPayload 评价作者（公开展示用，匿名时不泄漏身份）。
type ReviewAuthorPayload struct {
	Kind  string `json:"kind"`  // anonymous / member / legacy
	Label string `json:"label"` // 展示名
}

// ReviewViewerPayload 当前用户的个性化状态。
type ReviewViewerPayload struct {
	CanEdit   bool `json:"canEdit"`
	CanDelete bool `json:"canDelete"`
	IsHelpful bool `json:"isHelpful"`
}

// ReviewPayload 公开评价 DTO：匿名内容不得包含用户 ID/用户名/头像。
type ReviewPayload struct {
	Id           uint64              `json:"id"`
	OfferingId   uint64              `json:"offeringId"`
	Rating       *int                `json:"rating"`
	ContentHtml  string              `json:"contentHtml"`
	Author       ReviewAuthorPayload `json:"author"`
	Viewer       ReviewViewerPayload `json:"viewer"`
	HelpfulCount int64               `json:"helpfulCount"`
	CreatedAt    string              `json:"createdAt"`
	UpdatedAt    string              `json:"updatedAt"`
}

// CreateReviewInput 写评请求。
type CreateReviewInput struct {
	OfferingId  uint64 `json:"offeringId"`
	Rating      int    `json:"rating"`
	Content     string `json:"content"`
	IsAnonymous bool   `json:"isAnonymous"`
}

// UpdateReviewInput 编辑评价请求。
type UpdateReviewInput struct {
	Rating      *int   `json:"rating"`
	Content     string `json:"content"`
	IsAnonymous *bool  `json:"isAnonymous"`
}

// CreateReview 登录用户为 offering 写评价；与 stats delta 和搜索任务同事务提交。
func CreateReview(userId uint64, input CreateReviewInput) (ReviewPayload, error) {
	if userId == 0 {
		return ReviewPayload{}, ErrReviewNotOwned
	}
	if input.Rating < 1 || input.Rating > 5 {
		return ReviewPayload{}, ErrRatingOutOfRange
	}
	if input.Content == "" {
		return ReviewPayload{}, ErrReviewContentEmpty
	}
	var payload ReviewPayload
	err := dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		offering, err := course.GetOfferingTx(tx, input.OfferingId)
		if err != nil || offering.Status != course.OfferingStatusVisible {
			return ErrOfferingNotFound
		}
		// 唯一约束：同一用户对同一 offering 最多一条
		existing, err := course.FindReviewByOfferingAndUserTx(tx, input.OfferingId, userId)
		if err == nil && existing.Id > 0 {
			return ErrReviewDuplicate
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		rating := input.Rating
		entity := course.ReviewEntity{
			OfferingId:   input.OfferingId,
			AuthorUserId: userId,
			Rating:       &rating,
			Content:      input.Content,
			IsAnonymous:  input.IsAnonymous,
			Status:       course.ReviewStatusVisible,
		}
		if err := course.CreateReviewTx(tx, &entity); err != nil {
			return err
		}
		// stats delta（同事务）
		if err := course.UpsertCourseStatsTx(tx, offering.CourseId, 1, rating, 1); err != nil {
			return err
		}
		if err := course.UpsertOfferingStatsTx(tx, offering.Id, 1, rating, 1); err != nil {
			return err
		}
		payload = buildReviewPayload(entity, userId, int64(0))
		return nil
	})
	if err != nil {
		return ReviewPayload{}, err
	}
	// 事务已提交后再回填 member 作者名：users.Get 需要独立连接，
	// 在事务内调用会在单连接 SQLite 测试环境下死锁。
	fillReviewAuthorLabel(&payload, userId)
	return payload, nil
}

// UpdateReview 作者更新自己的评价（rating/content/anonymous）。
func UpdateReview(userId, reviewId uint64, input UpdateReviewInput) (ReviewPayload, error) {
	var payload ReviewPayload
	err := dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		entity, err := course.GetReviewTx(tx, reviewId)
		if err != nil {
			return ErrReviewNotFound
		}
		if entity.Status != course.ReviewStatusVisible {
			return ErrReviewNotFound
		}
		if entity.AuthorUserId != userId {
			return ErrReviewNotOwned
		}
		// 旧 rating 与新 rating 的 stats delta
		oldRating := 0
		if entity.Rating != nil {
			oldRating = *entity.Rating
		}
		newRating := oldRating
		if input.Rating != nil {
			if *input.Rating < 1 || *input.Rating > 5 {
				return ErrRatingOutOfRange
			}
			newRating = *input.Rating
			entity.Rating = input.Rating
		}
		entity.Content = input.Content
		if input.IsAnonymous != nil {
			entity.IsAnonymous = *input.IsAnonymous
		}
		if err := course.SaveReviewTx(tx, &entity); err != nil {
			return err
		}
		offering, err := course.GetOfferingTx(tx, entity.OfferingId)
		if err != nil {
			return err
		}
		// 更新 stats（替换旧 rating 的贡献）
		if newRating != oldRating {
			if err := course.UpsertCourseStatsTx(tx, offering.CourseId, 0, newRating-oldRating, 0); err != nil {
				return err
			}
			if err := course.UpsertOfferingStatsTx(tx, offering.Id, 0, newRating-oldRating, 0); err != nil {
				return err
			}
		}
		if err := searchservice.EnqueueCourseSearchTask(tx, offering.CourseId); err != nil {
			return err
		}
		payload = buildReviewPayload(entity, userId, int64(0))
		return nil
	})
	if err != nil {
		return ReviewPayload{}, err
	}
	fillReviewAuthorLabel(&payload, userId)
	return payload, nil
}

// DeleteReview 作者删除评价（隔离窗口语义由 status=deleted 表达，正文保留待清理）。
func DeleteReview(userId, reviewId uint64) error {
	return dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		entity, err := course.GetReviewTx(tx, reviewId)
		if err != nil {
			return ErrReviewNotFound
		}
		if entity.AuthorUserId != userId {
			return ErrReviewNotOwned
		}
		rating := 0
		if entity.Rating != nil {
			rating = *entity.Rating
		}
		if err := course.UpdateReviewStatusTx(tx, reviewId, course.ReviewStatusDeleted); err != nil {
			return err
		}
		offering, err := course.GetOfferingTx(tx, entity.OfferingId)
		if err != nil {
			return err
		}
		if rating > 0 {
			if err := course.UpsertCourseStatsTx(tx, offering.CourseId, -1, -rating, -1); err != nil {
				return err
			}
			if err := course.UpsertOfferingStatsTx(tx, offering.Id, -1, -rating, -1); err != nil {
				return err
			}
		}
		return searchservice.EnqueueCourseSearchTask(tx, offering.CourseId)
	})
}

// SetReviewHelpful 幂等设置/取消 helpful。
func SetReviewHelpful(userId, reviewId uint64, helpful bool) error {
	if userId == 0 {
		return ErrReviewNotOwned
	}
	return dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		entity, err := course.GetReviewTx(tx, reviewId)
		if err != nil || entity.Status != course.ReviewStatusVisible {
			return ErrReviewNotFound
		}
		if helpful {
			if err := course.CreateHelpfulTx(tx, &course.HelpfulEntity{ReviewId: reviewId, UserId: userId}); err != nil {
				// 唯一约束冲突视为已标记（幂等）
				return nil
			}
			return nil
		}
		return course.DeleteHelpfulTx(tx, reviewId, userId)
	})
}

// ListReviewsByOffering 返回 offering 的可见评价列表（匿名 DTO）。
func ListReviewsByOffering(offeringId, viewerId uint64) ([]ReviewPayload, error) {
	entities, err := course.ListReviewsByOffering(offeringId)
	if err != nil {
		return nil, err
	}
	return listReviewPayloads(entities, viewerId)
}

// ListReviewsByCourse 返回课程下所有可见 offering 的评价（时间倒序，匿名 DTO）。
func ListReviewsByCourse(courseId, viewerId uint64) ([]ReviewPayload, error) {
	offerings, err := course.ListOfferingsByCourse(courseId)
	if err != nil {
		return nil, err
	}
	offeringIds := make([]uint64, 0, len(offerings))
	for _, o := range offerings {
		offeringIds = append(offeringIds, o.Id)
	}
	if len(offeringIds) == 0 {
		return []ReviewPayload{}, nil
	}
	entities, err := course.ListReviewsByOfferings(offeringIds)
	if err != nil {
		return nil, err
	}
	return listReviewPayloads(entities, viewerId)
}

// SetReviewVisibility 审核隐藏/恢复评价；与 stats delta 和搜索任务同事务提交。
// 幂等：状态已为目标值时直接成功。已删除（隔离窗口）的评价不可恢复。
func SetReviewVisibility(reviewId uint64, hidden bool) error {
	return dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		entity, err := course.GetReviewTx(tx, reviewId)
		if err != nil {
			return ErrReviewNotFound
		}
		if entity.Status == course.ReviewStatusDeleted {
			return ErrReviewNotFound
		}
		target := course.ReviewStatusVisible
		if hidden {
			target = course.ReviewStatusHidden
		}
		if entity.Status == target {
			return nil
		}
		if err := course.UpdateReviewStatusTx(tx, reviewId, target); err != nil {
			return err
		}
		offering, err := course.GetOfferingTx(tx, entity.OfferingId)
		if err != nil {
			return err
		}
		delta := 1
		if hidden {
			delta = -1
		}
		rating := 0
		if entity.Rating != nil {
			rating = *entity.Rating
		}
		// 隐藏/恢复时同步调整 stats 投影；无 rating 的 legacy 评价只影响 review_count。
		if rating > 0 {
			if err := course.UpsertCourseStatsTx(tx, offering.CourseId, delta, delta*rating, delta); err != nil {
				return err
			}
			if err := course.UpsertOfferingStatsTx(tx, offering.Id, delta, delta*rating, delta); err != nil {
				return err
			}
		} else {
			if err := course.UpsertCourseStatsTx(tx, offering.CourseId, 0, 0, delta); err != nil {
				return err
			}
			if err := course.UpsertOfferingStatsTx(tx, offering.Id, 0, 0, delta); err != nil {
				return err
			}
		}
		return searchservice.EnqueueCourseSearchTask(tx, offering.CourseId)
	})
}

// listReviewPayloads 批量构造公开 DTO：匿名/legacy 评价不泄漏身份；
// member 评价的作者名批量回填（避免 N+1 查询）。
func listReviewPayloads(entities []course.ReviewEntity, viewerId uint64) ([]ReviewPayload, error) {
	reviewIds := make([]uint64, 0, len(entities))
	authorIds := make([]uint64, 0, len(entities))
	for _, e := range entities {
		reviewIds = append(reviewIds, e.Id)
		if !e.IsAnonymous && e.AuthorUserId > 0 {
			authorIds = append(authorIds, e.AuthorUserId)
		}
	}
	helpfulCounts := course.CountHelpfulByReviewIds(reviewIds)
	var userMap map[uint64]*users.EntityComplete
	if len(authorIds) > 0 {
		userMap = users.GetMapByIds(authorIds)
	}
	myHelpful := make(map[uint64]bool)
	if viewerId > 0 {
		for _, id := range reviewIds {
			if _, err := course.GetHelpful(id, viewerId); err == nil {
				myHelpful[id] = true
			}
		}
	}
	payloads := make([]ReviewPayload, 0, len(entities))
	for _, e := range entities {
		p := buildReviewPayload(e, viewerId, helpfulCounts[e.Id])
		if myHelpful[e.Id] {
			p.Viewer.IsHelpful = true
		}
		if p.Author.Kind == "member" {
			if user, ok := userMap[e.AuthorUserId]; ok && user != nil {
				p.Author.Label = user.Username
				if user.Nickname != "" {
					p.Author.Label = user.Nickname
				}
			}
		}
		payloads = append(payloads, p)
	}
	return payloads, nil
}

// fillReviewAuthorLabel 为写路径单条 member 评价回填展示用户名；匿名/legacy 不查询、不泄漏。
func fillReviewAuthorLabel(p *ReviewPayload, authorUserId uint64) {
	if p == nil || p.Author.Kind != "member" || authorUserId == 0 {
		return
	}
	user, err := users.Get(authorUserId)
	if err != nil || user.Id == 0 {
		return
	}
	p.Author.Label = user.Username
	if user.Nickname != "" {
		p.Author.Label = user.Nickname
	}
}

// buildReviewPayload 构造公开 DTO：匿名评价不泄漏任何身份信息。
func buildReviewPayload(entity course.ReviewEntity, viewerId uint64, helpfulCount int64) ReviewPayload {
	author := ReviewAuthorPayload{Kind: "member", Label: "同学"}
	if entity.IsAnonymous || entity.AuthorUserId == 0 {
		author = ReviewAuthorPayload{Kind: "anonymous", Label: "匿名同学"}
	} else if entity.Source != "" && entity.Source != "native" {
		author = ReviewAuthorPayload{Kind: "legacy", Label: "历史匿名评价"}
	}
	if entity.Source == "legacy-import" {
		author = ReviewAuthorPayload{Kind: "legacy", Label: "历史匿名评价"}
	}
	viewer := ReviewViewerPayload{
		CanEdit:   entity.AuthorUserId > 0 && entity.AuthorUserId == viewerId,
		CanDelete: entity.AuthorUserId > 0 && entity.AuthorUserId == viewerId,
	}
	return ReviewPayload{
		Id:           entity.Id,
		OfferingId:   entity.OfferingId,
		Rating:       entity.Rating,
		ContentHtml:  markdown2html.PostMarkdownToHTML(entity.Content),
		Author:       author,
		Viewer:       viewer,
		HelpfulCount: helpfulCount,
		CreatedAt:    entity.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    entity.UpdatedAt.Format(time.RFC3339),
	}
}

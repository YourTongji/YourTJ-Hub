package courseservice

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

// ReviewContentMaxLength 评价正文最大长度（rune 计数，与帖子正文上限一致）。
const ReviewContentMaxLength = 50000

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
	// ErrReviewContentTooLong 正文超过上限。
	ErrReviewContentTooLong = errors.New("content too long")
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
	Content      string              `json:"content"`
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
// Content 为指针以区分"缺省（nil，保留原正文）"与"显式空串（清空正文）"，
// 与 OpenAPI 契约的 PATCH 部分更新语义一致。
type UpdateReviewInput struct {
	Rating      *int    `json:"rating"`
	Content     *string `json:"content"`
	IsAnonymous *bool   `json:"isAnonymous"`
}

// CreateReview 登录用户为 offering 写评价；与 stats delta 同事务提交。
// 课程搜索文档当前不携带课评字段，评价写路径一律不入队搜索任务
// （课程数据变化才由目录导入/审核入口入队）。
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
	if len([]rune(input.Content)) > ReviewContentMaxLength {
		return ReviewPayload{}, ErrReviewContentTooLong
	}
	var payload ReviewPayload
	err := dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		offering, err := course.GetOfferingTx(tx, input.OfferingId)
		if err != nil || offering.Status != course.OfferingStatusVisible {
			return ErrOfferingNotFound
		}
		rating := input.Rating
		// 唯一约束：同一用户对同一 offering 最多一条（唯一索引不含 status，
		// 软删的 deleted 行仍占用唯一键，查重必须覆盖全部状态）。
		existing, err := course.FindReviewByOfferingAndUserTx(tx, input.OfferingId, userId)
		if err == nil && existing.Id > 0 {
			if existing.Status != course.ReviewStatusDeleted {
				return ErrReviewDuplicate
			}
			// deleted 行：恢复重写（复用唯一键），并重新累加 stats
			// （删除时已扣减，恢复时按新 rating/content 重新计入）。
			ok, err := course.ReactivateReviewTx(tx, existing.Id, &rating, input.Content, input.IsAnonymous)
			if err != nil {
				return err
			}
			if !ok {
				// 并发恢复已发生：另一事务已把该行恢复为 visible，视为重复。
				return ErrReviewDuplicate
			}
			if err := course.UpsertCourseStatsTx(tx, offering.CourseId, 1, rating, 1); err != nil {
				return err
			}
			if err := course.UpsertOfferingStatsTx(tx, offering.Id, 1, rating, 1); err != nil {
				return err
			}
			refreshed, err := course.GetReviewTx(tx, existing.Id)
			if err != nil {
				return err
			}
			payload = buildReviewPayload(refreshed, userId, int64(0))
			return nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		entity := course.ReviewEntity{
			OfferingId:   input.OfferingId,
			AuthorUserId: userId,
			Rating:       &rating,
			Content:      input.Content,
			IsAnonymous:  input.IsAnonymous,
			Status:       course.ReviewStatusVisible,
		}
		if err := course.CreateReviewTx(tx, &entity); err != nil {
			// 并发兜底：数据库唯一索引 (offering_id, author_user_id) 冲突
			// 映射为同一语义错误，与事务内查重一致。
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return ErrReviewDuplicate
			}
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

// errReviewRatingConflict 内部 sentinel：并发 PATCH 评分时 CAS 未命中（另一事务已改评分），
// 调用方重试整个事务（事务已回滚，无副作用）。
var errReviewRatingConflict = errors.New("review rating changed concurrently")

// UpdateReview 作者更新自己的评价（rating/content/anonymous）。
// 评分更新使用带旧评分条件的 CAS：仅当 rating 仍是读取时的旧值才更新并累加 stats delta；
// 并发 PATCH 时 CAS 未命中的事务回滚重试，避免两笔事务基于同一旧值重复累加 delta。
func UpdateReview(userId, reviewId uint64, input UpdateReviewInput) (ReviewPayload, error) {
	for attempt := 0; attempt < 3; attempt++ {
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
			}
			// PATCH 部分更新：content 缺省（nil）时保留原正文；显式空串才清空。
			if input.Content != nil {
				if len([]rune(*input.Content)) > ReviewContentMaxLength {
					return ErrReviewContentTooLong
				}
				if err := tx.Table((&course.ReviewEntity{}).TableName()).
					Where("id = ?", reviewId).
					Update("content", *input.Content).Error; err != nil {
					return err
				}
			}
			if input.IsAnonymous != nil {
				if err := tx.Table((&course.ReviewEntity{}).TableName()).
					Where("id = ?", reviewId).
					Update("is_anonymous", *input.IsAnonymous).Error; err != nil {
					return err
				}
			}
			// 评分变化：CAS 更新（WHERE rating = 旧值），成功才调整 stats delta。
			if newRating != oldRating {
				ok, err := course.UpdateReviewRatingFromTx(tx, reviewId, oldRating, newRating)
				if err != nil {
					return err
				}
				if !ok {
					return errReviewRatingConflict // 事务回滚后重试
				}
				offering, err := course.GetOfferingTx(tx, entity.OfferingId)
				if err != nil {
					return err
				}
				if err := course.UpsertCourseStatsTx(tx, offering.CourseId, 0, newRating-oldRating, 0); err != nil {
					return err
				}
				if err := course.UpsertOfferingStatsTx(tx, offering.Id, 0, newRating-oldRating, 0); err != nil {
					return err
				}
			}
			refreshed, err := course.GetReviewTx(tx, reviewId)
			if err != nil {
				return err
			}
			payload = buildReviewPayload(refreshed, userId, int64(0))
			return nil
		})
		if err == nil {
			fillReviewAuthorLabel(&payload, userId)
			return payload, nil
		}
		if !errors.Is(err, errReviewRatingConflict) {
			return ReviewPayload{}, err
		}
	}
	return ReviewPayload{}, errReviewRatingConflict
}

// DeleteReview 作者删除评价（隔离窗口语义由 status=deleted 表达，正文保留待清理）。
// 幂等：已删除（含隔离窗口后的清理）直接成功；仅当评价仍可见时扣减 stats，
// 避免隐藏（SetReviewVisibility 已扣）后再删导致双重扣减。
// 状态转换使用 CAS（WHERE status = 旧值）：并发 hide 与 delete 同时到达时，
// 只有拿到转换权的事务会调整 stats，另一个 RowsAffected=0 直接幂等成功。
func DeleteReview(userId, reviewId uint64) error {
	return dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		entity, err := course.GetReviewTx(tx, reviewId)
		if err != nil {
			return ErrReviewNotFound
		}
		if entity.AuthorUserId != userId {
			return ErrReviewNotOwned
		}
		if entity.Status == course.ReviewStatusDeleted {
			return nil
		}
		converted, err := course.UpdateReviewStatusFromTx(tx, reviewId, entity.Status, course.ReviewStatusDeleted)
		if err != nil {
			return err
		}
		if !converted {
			// 另一事务已并发改变状态（隐藏/删除）；删除语义已由该事务完成，
			// 当前事务不重复扣减 stats。
			return nil
		}
		// 仅当评价原为可见时才扣减 stats（隐藏时 SetReviewVisibility 已扣）。
		// 与 SetReviewVisibility 口径一致：无 rating 的评价也扣 review_count。
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
		return nil
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
				// 仅唯一约束冲突视为已标记（幂等）；其余错误如实上报，避免吞掉 DB 故障。
				if errors.Is(err, gorm.ErrDuplicatedKey) {
					return nil
				}
				return err
			}
			return nil
		}
		return course.DeleteHelpfulTx(tx, reviewId, userId)
	})
}

// ReviewListMaxItems 评价列表单次返回上限（防止热门课程响应无界；分页由后续 slice 增强）。
// 注意：B2 分页（issue #174）上线后 HTTP 列表走 ListReviewsPage，旧路径
// ListReviewsByOffering/ListReviewsByCourse 仅供内部调用（如无 HTTP 调用方时
// 待后续清理，PR #201 security F3）。
const ReviewListMaxItems = 200

// ---- B2: cursor 分页（issue #174） ----

// DefaultReviewPageSize 评价列表默认页大小；MaxReviewPageSize 上限。
const (
	DefaultReviewPageSize = 20
	MaxReviewPageSize     = 50
)

// ErrReviewInvalidCursor 非法 cursor（格式/取值错误，控制器映射 400）。
var ErrReviewInvalidCursor = errors.New("invalid review cursor")

// ReviewPageResult cursor 分页结果。
type ReviewPageResult struct {
	List       []ReviewPayload `json:"list"`
	NextCursor string          `json:"nextCursor,omitempty"`
	Total      int64           `json:"total"`
}

// ReviewCursor 复合游标（offering_id, review_id）。
// Course 级列表按 (offering_id DESC, id DESC) 排序，cursor 是上一页
// 最后一条的 (offeringId, id)；offering 级列表只用 reviewId。
type ReviewCursor struct {
	OfferingId uint64
	ReviewId   uint64
}

// EncodeCursor 编码 cursor 为明文 "offeringId:reviewId"。
func EncodeCursor(c ReviewCursor) string {
	return fmt.Sprintf("%d:%d", c.OfferingId, c.ReviewId)
}

// DecodeCursor 解析 cursor；非法格式返回 ErrReviewInvalidCursor。
func DecodeCursor(raw string) (ReviewCursor, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ReviewCursor{}, nil
	}
	parts := strings.Split(raw, ":")
	if len(parts) != 2 {
		return ReviewCursor{}, ErrReviewInvalidCursor
	}
	oid, err1 := strconv.ParseUint(parts[0], 10, 64)
	rid, err2 := strconv.ParseUint(parts[1], 10, 64)
	if err1 != nil || err2 != nil {
		return ReviewCursor{}, ErrReviewInvalidCursor
	}
	return ReviewCursor{OfferingId: oid, ReviewId: rid}, nil
}

// ListReviewsPage 按 cursor 分页返回课程（或指定 offering）的可见评价。
// pageSize 默认 20、上限 50；结果多取一条判断 hasNext（无重复无遗漏）。
// total 为当前筛选下的可见评价总数（offering 过滤时同口径）。
func ListReviewsPage(courseId, offeringId, viewerId uint64, cursor ReviewCursor, pageSize int) (ReviewPageResult, error) {
	if pageSize <= 0 {
		pageSize = DefaultReviewPageSize
	}
	if pageSize > MaxReviewPageSize {
		pageSize = MaxReviewPageSize
	}
	query := course.ReviewPageQuery{
		CourseId:         courseId,
		OfferingId:       offeringId,
		CursorOfferingId: cursor.OfferingId,
		CursorReviewId:   cursor.ReviewId,
		Limit:            pageSize + 1,
	}
	entities, err := course.ListReviewsPage(query)
	if err != nil {
		return ReviewPageResult{}, err
	}
	hasNext := len(entities) > pageSize
	if hasNext {
		entities = entities[:pageSize]
	}
	payloads, err := listReviewPayloads(entities, viewerId)
	if err != nil {
		return ReviewPageResult{}, err
	}
	result := ReviewPageResult{
		List:  payloads,
		Total: 0,
	}
	if offeringId > 0 {
		result.Total, err = course.CountVisibleReviewsByOffering(offeringId)
	} else {
		result.Total, err = course.CountVisibleReviewsByCourse(courseId)
	}
	if err != nil {
		return ReviewPageResult{}, err
	}
	if hasNext {
		last := entities[len(entities)-1]
		result.NextCursor = EncodeCursor(ReviewCursor{OfferingId: last.OfferingId, ReviewId: last.Id})
	}
	return result, nil
}

// ListReviewsByOffering 返回 offering 的可见评价列表（匿名 DTO，最多 ReviewListMaxItems 条）。
func ListReviewsByOffering(offeringId, viewerId uint64) ([]ReviewPayload, error) {
	entities, err := course.ListReviewsByOffering(offeringId)
	if err != nil {
		return nil, err
	}
	if len(entities) > ReviewListMaxItems {
		entities = entities[:ReviewListMaxItems]
	}
	return listReviewPayloads(entities, viewerId)
}

// ListReviewsByCourse 返回课程下所有可见 offering 的评价（时间倒序，匿名 DTO，
// 最多 ReviewListMaxItems 条）。
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
	if len(entities) > ReviewListMaxItems {
		entities = entities[:ReviewListMaxItems]
	}
	return listReviewPayloads(entities, viewerId)
}

// SetReviewVisibility 审核隐藏/恢复评价；与 stats delta 同事务提交。
// 幂等：状态已为目标值时直接成功。已删除（隔离窗口）的评价不可恢复。
// 状态转换使用 CAS（WHERE status = 旧值）：并发 hide/delete 双写时只有
// 拿到转换权的事务调整 stats；CAS 未命中则重新读取，按最新状态决定
// 幂等成功（已被并发转为目标态）或 404（已被删除）。
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
		converted, err := course.UpdateReviewStatusFromTx(tx, reviewId, entity.Status, target)
		if err != nil {
			return err
		}
		if !converted {
			// 并发状态转换已发生：重新读取，按最新状态判定幂等成功或 404。
			latest, err := course.GetReviewTx(tx, reviewId)
			if err != nil {
				return ErrReviewNotFound
			}
			if latest.Status == target {
				return nil
			}
			return ErrReviewNotFound
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
		return nil
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
		// 批量查询当前用户的 helpful 标记，避免逐条 GetHelpful 的 N+1；
		// 查询错误如实向上返回（原先被静默吞掉会误报 isHelpful=false）。
		ids, err := course.ListHelpfulReviewIDsByUser(viewerId, reviewIds)
		if err != nil {
			return nil, err
		}
		myHelpful = ids
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
// helpfulCount 为原生 helpful 计数；历史导入的 legacy_helpful_count 一并计入总数，
// 保证 legacy 评价展示的 helpful 数与源数据一致。
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
		Content:      entity.Content,
		ContentHtml:  markdown2html.PostMarkdownToHTML(entity.Content),
		Author:       author,
		Viewer:       viewer,
		HelpfulCount: helpfulCount + int64(entity.LegacyHelpfulCount),
		CreatedAt:    entity.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    entity.UpdatedAt.Format(time.RFC3339),
	}
}

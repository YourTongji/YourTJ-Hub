package dataservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicCategoryIndex"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserStat"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"

	"gorm.io/gorm"
)

// ImportError 单行导入错误。
type ImportError struct {
	Line   int    `json:"line"`
	Table  string `json:"table"`
	Reason string `json:"reason"`
}

// ImportReport 导入结果报告。
type ImportReport struct {
	Total    int           `json:"total"`
	Success  int           `json:"success"`
	Skipped  int           `json:"skipped"`
	Failed   int           `json:"failed"`
	Errors   []ImportError `json:"errors"`
	Imported []string      `json:"importedTables"`
	TaskID   uint64        `json:"taskId,omitempty"`
	Status   string        `json:"status,omitempty"`
}

// ImportData 导入 JSON 数据（仅支持 JSON 格式）。
// 支持两种结构：数组 `[{...}]` 或对象 `{"users":[...],"topics":[...],"posts":[...]}`。
// 按 users → topics → posts → postRevisions → 派生表顺序导入；已存在记录跳过（幂等）。
// 导入完成后重建话题 invariants（首末帖指针、计数、post_seq、参与者统计、
// 分类索引），保证 round-trip 后结构与源库一致且可继续回复（issue #135）。
// postRevisions 依赖 posts 存在（外键校验），在 posts 之后导入。
var errImportRowsFailed = errors.New("导入存在失败行，事务已回滚")

func ImportData(ctx context.Context, data []byte, format string) (*ImportReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if format != "json" {
		return nil, fmt.Errorf("%w: 导入仅支持 JSON 格式", ErrImportInvalidFormat)
	}
	parsed, err := parseImportJSON(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrImportInvalidFormat, err)
	}
	report := &ImportReport{
		Errors:   []ImportError{},
		Imported: []string{},
	}
	txErr := dbconnect.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		importUsers(tx, parsed["users"], report)
		importTopics(tx, parsed["topics"], report)
		importPosts(tx, parsed["posts"], report)
		importPostRevisions(tx, parsed["postRevisions"], report)
		importTopicCategoryIndexes(tx, parsed["topicCategoryIndex"], report)
		importTopicUserStats(tx, parsed["topicUserStat"], report)
		if report.Failed > 0 {
			return errImportRowsFailed
		}
		if err := resetPostgresSequences(tx); err != nil {
			return err
		}
		if len(parsed["topics"]) > 0 {
			return rebuildTopicInvariants(tx)
		}
		return nil
	})
	if txErr != nil && !errors.Is(txErr, errImportRowsFailed) {
		return nil, txErr
	}
	for _, t := range []string{"users", "topics", "posts", "postRevisions", "topicCategoryIndex", "topicUserStat"} {
		if _, ok := parsed[t]; ok {
			report.Imported = append(report.Imported, t)
		}
	}
	return report, nil
}

// resetPostgresSequences 导入显式主键后推进各表的 sequence。
// 覆盖 users/topics/posts/post_revisions 及两张派生表 topic_category_index/
// topic_user_stat（均 autoIncrement 主键，PR #160 review, warning 1）：PG 上
// 显式主键写入不推进序列，若漏推，下一次 INSERT 可能复用已导入 ID 触发
// 主键冲突。SQLite 无 sequence 概念，无需处理。
func resetPostgresSequences(db *gorm.DB) error {
	if dbconnect.IsSqlite() {
		return nil
	}
	for _, table := range []string{"users", "topics", "posts", "post_revisions", "topic_category_index", "topic_user_stat"} {
		if err := db.Exec(fmt.Sprintf(
			`SELECT setval(pg_get_serial_sequence('%s', 'id'), GREATEST((SELECT COALESCE(MAX(id), 1) FROM %s), 1), true)`,
			table, table,
		)).Error; err != nil {
			return fmt.Errorf("resetPostgresSequences: 推进 %s 序列失败: %w", table, err)
		}
	}
	return nil
}

// parseImportJSON 解析导入 JSON 为按表分组的行。
func parseImportJSON(data []byte) (map[string][]map[string]any, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, fmt.Errorf("导入文件为空")
	}
	// 先尝试对象形式
	var obj map[string][]map[string]any
	if err := json.Unmarshal(data, &obj); err == nil {
		result := map[string][]map[string]any{}
		for _, t := range []string{"users", "topics", "posts", "postRevisions", "topicCategoryIndex", "topicUserStat"} {
			if rows, ok := obj[t]; ok {
				result[t] = rows
			}
		}
		if len(result) == 0 {
			return nil, fmt.Errorf("导入文件不含 users/topics/posts 数据")
		}
		return result, nil
	}
	// 数组形式（每个元素需带 table 字段或按顺序推断为 users）
	var arr []map[string]any
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %v", err)
	}
	result := map[string][]map[string]any{}
	for _, row := range arr {
		table, _ := row["table"].(string)
		if table == "" {
			table = "users" // 数组形式默认按 users 处理
		}
		if !AllowedExportTables[table] {
			return nil, fmt.Errorf("未知表: %s", table)
		}
		delete(row, "table")
		result[table] = append(result[table], row)
	}
	return result, nil
}

func importUsers(db *gorm.DB, rows []map[string]any, report *ImportReport) {
	for i, row := range rows {
		line := i + 1
		report.Total++
		id := rowUint64(row, "id")
		if id > 0 {
			var existingByID users.EntityComplete
			if err := db.First(&existingByID, id).Error; err == nil && existingByID.Id > 0 {
				report.Skipped++
				continue
			}
		}
		username, _ := row["username"].(string)
		username = strings.TrimSpace(username)
		if username == "" {
			report.Failed++
			report.Errors = append(report.Errors, ImportError{Line: line, Table: "users", Reason: "username 必填"})
			continue
		}
		var existing users.EntityComplete
		// 幂等：id 已在上方检查；此处按 username/email 兜底
		if id > 0 {
			if err := db.First(&existing, id).Error; err == nil && existing.Id > 0 {
				report.Skipped++
				continue
			}
		} else if err := db.Where("username = ?", username).First(&existing).Error; err == nil && existing.Id > 0 {
			report.Skipped++
			continue
		}
		if email, _ := row["email"].(string); email != "" {
			if err := db.Where("email = ?", email).First(&existing).Error; err == nil && existing.Id > 0 {
				report.Skipped++
				continue
			}
		}
		user := users.EntityComplete{
			Id:          id,
			Username:    username,
			Email:       rowString(row, "email"),
			Nickname:    rowString(row, "nickname"),
			Bio:         rowString(row, "bio"),
			Signature:   rowString(row, "signature"),
			Website:     rowString(row, "website"),
			AvatarUrl:   rowString(row, "avatarUrl"),
			Prestige:    rowInt64(row, "prestige"),
			IsFrozen:    int8(rowInt64(row, "isFrozen")),
			IsActivated: int8(rowInt64(row, "isActivated")),
			RoleId:      rowUint64(row, "roleId"),
		}
		if err := db.Create(&user).Error; err != nil {
			report.Failed++
			report.Errors = append(report.Errors, ImportError{Line: line, Table: "users", Reason: err.Error()})
			continue
		}
		report.Success++
	}
}

func importTopics(db *gorm.DB, rows []map[string]any, report *ImportReport) {
	for i, row := range rows {
		line := i + 1
		report.Total++
		id := rowUint64(row, "id")
		if id == 0 {
			report.Failed++
			report.Errors = append(report.Errors, ImportError{Line: line, Table: "topics", Reason: "id 必填"})
			continue
		}
		title := strings.TrimSpace(rowString(row, "title"))
		if title == "" {
			report.Failed++
			report.Errors = append(report.Errors, ImportError{Line: line, Table: "topics", Reason: "title 必填"})
			continue
		}
		userID := rowUint64(row, "userId")
		if userID == 0 {
			report.Failed++
			report.Errors = append(report.Errors, ImportError{Line: line, Table: "topics", Reason: "userId 必填"})
			continue
		}
		var user users.EntityComplete
		if err := db.First(&user, userID).Error; err != nil {
			report.Failed++
			report.Errors = append(report.Errors, ImportError{Line: line, Table: "topics", Reason: "userId 不存在"})
			continue
		}
		// 校验分类存在
		var categoryIDs []uint64
		if cats, ok := row["categoryIds"].(string); ok && cats != "" {
			_ = json.Unmarshal([]byte(cats), &categoryIDs)
		}
		for _, cid := range categoryIDs {
			var cat category.Entity
			if err := db.First(&cat, cid).Error; err != nil {
				report.Failed++
				reason := fmt.Sprintf("分类 %d 不存在", cid)
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					reason = "分类校验失败"
				}
				report.Errors = append(report.Errors, ImportError{Line: line, Table: "topics", Reason: reason})
				categoryIDs = nil
				break
			}
		}
		if categoryIDs == nil && len(rowString(row, "categoryIds")) > 0 {
			// 分类校验失败：仅记一次失败，跳过该行，避免 Failed/Success 双计数，
			// 也避免写入无分类主题导致重试按 ID 跳过脏数据。
			continue
		}
		var existing topics.Entity
		if err := db.First(&existing, id).Error; err == nil && existing.Id > 0 {
			report.Skipped++
			continue
		}
		topic := topics.Entity{
			Id:            id,
			Title:         title,
			CategoryIds:   categoryIDs,
			UserId:        userID,
			Status:        int8(rowInt64(row, "status")),
			ProcessStatus: int8(rowInt64(row, "processStatus")),
			Excerpt:       rowString(row, "excerpt"),
			FirstImageURL: rowString(row, "firstImageUrl"),
		}
		// 回填话题 invariants（issue #135）。导出文件带完整字段时原样恢复；
		// 旧格式（无这些字段）时先落 0，由 rebuildTopicInvariants 从 posts 推导补齐。
		topic.PostCount = rowUint64(row, "postCount")
		topic.ReplyCount = rowUint64(row, "replyCount")
		topic.PostSeq = rowUint64(row, "postSeq")
		topic.FirstPostId = rowUint64(row, "firstPostId")
		topic.LastPostId = rowUint64(row, "lastPostId")
		topic.LikeCount = rowUint64(row, "likeCount")
		topic.ViewCount = rowUint64(row, "viewCount")
		topic.PinWeight = int(rowInt64(row, "pinWeight"))
		if lp := rowString(row, "lastPostedAt"); lp != "" {
			if t, err := time.Parse(time.RFC3339Nano, lp); err == nil {
				topic.LastPostedAt = &t
			}
		}
		var posters []topics.Poster
		if ps := rowString(row, "posters"); ps != "" {
			_ = json.Unmarshal([]byte(ps), &posters)
		}
		topic.Posters = posters
		var imageURLs []string
		if imgs := rowString(row, "imageUrls"); imgs != "" {
			_ = json.Unmarshal([]byte(imgs), &imageURLs)
		}
		topic.ImageUrls = imageURLs
		if err := db.Create(&topic).Error; err != nil {
			report.Failed++
			report.Errors = append(report.Errors, ImportError{Line: line, Table: "topics", Reason: err.Error()})
			continue
		}
		report.Success++
	}
}

func importTopicCategoryIndexes(db *gorm.DB, rows []map[string]any, report *ImportReport) {
	for i, row := range rows {
		line := i + 1
		report.Total++
		id := rowUint64(row, "id")
		topicID := rowUint64(row, "topicId")
		categoryID := rowUint64(row, "categoryId")
		if id == 0 || topicID == 0 || categoryID == 0 {
			report.Failed++
			report.Errors = append(report.Errors, ImportError{Line: line, Table: "topicCategoryIndex", Reason: "id/topicId/categoryId 必填"})
			continue
		}
		var topic topics.Entity
		if err := db.First(&topic, topicID).Error; err != nil {
			report.Failed++
			report.Errors = append(report.Errors, ImportError{Line: line, Table: "topicCategoryIndex", Reason: "topicId 不存在"})
			continue
		}
		// 外键一致性：与 importTopics 校验分类存在保持一致（PR #160 review, suggestion 6）
		var cat category.Entity
		if err := db.First(&cat, categoryID).Error; err != nil {
			report.Failed++
			reason := "categoryId 不存在"
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				reason = "categoryId 校验失败"
			}
			report.Errors = append(report.Errors, ImportError{Line: line, Table: "topicCategoryIndex", Reason: reason})
			continue
		}
		var existing topicCategoryIndex.Entity
		if err := db.First(&existing, id).Error; err == nil && existing.Id > 0 {
			report.Skipped++
			continue
		}
		entity := topicCategoryIndex.Entity{
			Id:         id,
			TopicId:    topicID,
			CategoryId: categoryID,
			Effective:  int(rowInt64(row, "effective")),
		}
		if err := db.Create(&entity).Error; err != nil {
			report.Failed++
			report.Errors = append(report.Errors, ImportError{Line: line, Table: "topicCategoryIndex", Reason: err.Error()})
			continue
		}
		report.Success++
	}
}

func importTopicUserStats(db *gorm.DB, rows []map[string]any, report *ImportReport) {
	for i, row := range rows {
		line := i + 1
		report.Total++
		id := rowUint64(row, "id")
		topicID := rowUint64(row, "topicId")
		userID := rowUint64(row, "userId")
		if id == 0 || topicID == 0 || userID == 0 {
			report.Failed++
			report.Errors = append(report.Errors, ImportError{Line: line, Table: "topicUserStat", Reason: "id/topicId/userId 必填"})
			continue
		}
		var topic topics.Entity
		if err := db.First(&topic, topicID).Error; err != nil {
			report.Failed++
			report.Errors = append(report.Errors, ImportError{Line: line, Table: "topicUserStat", Reason: "topicId 不存在"})
			continue
		}
		// 外键一致性：与 importPosts 校验用户存在保持一致（PR #160 review, suggestion 6）
		var user users.EntityComplete
		if err := db.First(&user, userID).Error; err != nil {
			report.Failed++
			report.Errors = append(report.Errors, ImportError{Line: line, Table: "topicUserStat", Reason: "userId 不存在"})
			continue
		}
		var existing topicUserStat.Entity
		if err := db.First(&existing, id).Error; err == nil && existing.Id > 0 {
			report.Skipped++
			continue
		}
		// lastReplyAt 缺省用零值而非 time.Now()：旧格式文件缺该字段时
		// 不臆造"最后回复时间"（该话题后续会被 rebuildTopicUserStats 或
		// 正常发帖流程修正；PR #160 复审 🟡 可选）。
		var lastReplyAt time.Time
		if lr := rowString(row, "lastReplyAt"); lr != "" {
			if t, err := time.Parse(time.RFC3339Nano, lr); err == nil {
				lastReplyAt = t
			}
		}
		entity := topicUserStat.Entity{
			Id:          id,
			TopicId:     topicID,
			UserId:      userID,
			ReplyCount:  uint32(rowInt64(row, "replyCount")),
			LastReplyAt: lastReplyAt,
		}
		if err := db.Create(&entity).Error; err != nil {
			report.Failed++
			report.Errors = append(report.Errors, ImportError{Line: line, Table: "topicUserStat", Reason: err.Error()})
			continue
		}
		report.Success++
	}
}

// rebuildTopicInvariants 在导入全部落库后重建话题 invariants：
//   - post_seq / first_post_id / last_post_id / post_count / reply_count /
//     posters / last_posted_at 按实际 posts 推导，保证与源库一致（issue #135）；
//   - topic_category_index 缺失时按 topics.categoryIds 补齐，避免分类/搜索关联丢失；
//   - topic_user_stat 缺失（count==0）时从 posts 重建参与者统计——
//     覆盖默认 UI 路径（仅导出 users/topics/posts，新格式 topics 无需回填
//     指针但仍需重建统计，PR #160 review, warning 2）。
//
// 仅当话题缺失关键字段时补算指针，避免覆盖新导出格式中导出的精确值。
func rebuildTopicInvariants(db *gorm.DB) error {
	var list []topics.Entity
	if err := db.Find(&list).Error; err != nil {
		return fmt.Errorf("rebuildTopicInvariants: 加载话题列表失败: %w", err)
	}
	for _, topic := range list {
		// 判断该话题是否需要按 posts 回填指针：invariants 缺失（旧格式导出，
		// postSeq/firstPostId/lastPostId/lastPostedAt 全 0 或空）即回填。
		needsBackfill := topic.PostSeq == 0 || topic.FirstPostId == 0 || topic.LastPostId == 0 || topic.LastPostedAt == nil
		if !needsBackfill {
			// 新格式导出：指针/计数已精确恢复，仅补齐缺失的派生表。
			// 分类索引缺失时按 topics.categoryIds 补齐；
			// 参与者统计缺失（默认 UI 路径不导出该表）时从 posts 重建。
			if err := ensureTopicCategoryIndexes(db, topic.Id); err != nil {
				return err
			}
			if err := ensureTopicUserStats(db, topic.Id); err != nil {
				return err
			}
			continue
		}
		var postList []*posts.Entity
		// 仅统计 visibility_status=ACTIVE 的帖子（与 rebuildTopicUserStats /
		// postservice.RebuildTopicPostStats 严格对齐，PR #160 复审 🟡）。
		if err := db.Where("topic_id = ? AND visibility_status = ?", topic.Id, posts.VisibilityActive).
			Order("post_no asc").Order("id asc").
			Find(&postList).Error; err != nil {
			return fmt.Errorf("rebuildTopicInvariants: 加载话题 %d posts 失败: %w", topic.Id, err)
		}
		// 无 posts 的话题（空话题在源库不存在）跳过，避免全 0 误判后白扫
		if len(postList) == 0 {
			continue
		}
		var (
			firstPostID uint64
			lastPostID  uint64
			maxPostNo   uint64
			lastTime    *time.Time
		)
		for _, p := range postList {
			if firstPostID == 0 {
				firstPostID = p.Id
			}
			lastPostID = p.Id
			if p.PostNo > maxPostNo {
				maxPostNo = p.PostNo
			}
			if lastTime == nil || p.CreatedAt.After(*lastTime) {
				t := p.CreatedAt
				lastTime = &t
			}
		}
		if firstPostID == 0 {
			continue
		}
		updates := map[string]any{
			"post_seq":      maxPostNo,
			"first_post_id": firstPostID,
			"last_post_id":  lastPostID,
			"post_count":    len(postList),
		}
		// reply_count 与 post_count 一致重算（首帖不计回复）：回填分支仅在
		// invariants 缺失（旧格式）时进入，导出值本就不可信；统一用
		// len(postList)-1 覆盖，避免旧文件 replyCount 漂移导致 reply_count >
		// post_count 的矛盾（PR #160 复审 🟡）。
		if len(postList) > 1 {
			updates["reply_count"] = len(postList) - 1
		}
		// posters 缺失时按实际发帖人重建（首帖作者 + 回复作者，话题作者置前）
		if len(topic.Posters) == 0 {
			postersJSON, _ := json.Marshal(rebuildPosters(topic.UserId, postList))
			updates["posters"] = string(postersJSON)
		}
		if lastTime != nil {
			updates["last_posted_at"] = *lastTime
		}
		if err := db.Model(&topics.Entity{}).Where("id = ?", topic.Id).Updates(updates).Error; err != nil {
			return fmt.Errorf("rebuildTopicInvariants: 更新话题 %d invariants 失败: %w", topic.Id, err)
		}
		// 参与者统计：导入文件带 topicUserStat 时保留导出值，缺失时从 posts 重建
		if err := ensureTopicUserStats(db, topic.Id); err != nil {
			return err
		}
		if err := ensureTopicCategoryIndexes(db, topic.Id); err != nil {
			return err
		}
	}
	return nil
}

// ensureTopicUserStats 话题参与者统计缺失（count==0）时从 posts 重建。
// 导入文件显式携带 topicUserStat 时保留导出值（count>0 不重建）。
func ensureTopicUserStats(db *gorm.DB, topicID uint64) error {
	var count int64
	if err := db.Model(&topicUserStat.Entity{}).Where("topic_id = ?", topicID).Count(&count).Error; err != nil {
		return fmt.Errorf("ensureTopicUserStats: 统计话题 %d 失败: %w", topicID, err)
	}
	if count == 0 {
		return rebuildTopicUserStats(db, topicID)
	}
	return nil
}

// rebuildPosters 从 posts 重建话题参与者列表：话题作者置前，其余按出现顺序。
func rebuildPosters(topicUserID uint64, postList []*posts.Entity) []topics.Poster {
	seen := map[uint64]bool{}
	var result []topics.Poster
	if topicUserID != 0 {
		seen[topicUserID] = true
		result = append(result, topics.Poster{UserID: topicUserID})
	}
	for _, p := range postList {
		if seen[p.UserId] {
			continue
		}
		seen[p.UserId] = true
		result = append(result, topics.Poster{UserID: p.UserId})
	}
	return result
}

// ensureTopicCategoryIndexes 为话题补齐分类索引行（issue #135）。
func ensureTopicCategoryIndexes(db *gorm.DB, topicID uint64) error {
	var topic topics.Entity
	if err := db.First(&topic, topicID).Error; err != nil {
		return fmt.Errorf("ensureTopicCategoryIndexes: 加载话题 %d 失败: %w", topicID, err)
	}
	for _, cid := range topic.CategoryIds {
		if cid == 0 {
			continue
		}
		var count int64
		if err := db.Model(&topicCategoryIndex.Entity{}).
			Where("topic_id = ? AND category_id = ?", topicID, cid).
			Count(&count).Error; err != nil {
			return fmt.Errorf("ensureTopicCategoryIndexes: 统计话题 %d 分类 %d 失败: %w", topicID, cid, err)
		}
		if count > 0 {
			continue
		}
		if err := db.Create(&topicCategoryIndex.Entity{
			TopicId:    topicID,
			CategoryId: cid,
			Effective:  1,
		}).Error; err != nil {
			return fmt.Errorf("ensureTopicCategoryIndexes: 创建话题 %d 分类 %d 索引失败: %w", topicID, cid, err)
		}
	}
	return nil
}

// rebuildTopicUserStats 从 posts 重建话题参与者统计（reply_count/last_reply_at）。
// 与 postservice.RebuildTopicPostStats 语义一致：仅统计 visibility_status=ACTIVE
// 的回复（PR #160 review, suggestion 7）。
func rebuildTopicUserStats(db *gorm.DB, topicID uint64) error {
	var postList []*posts.Entity
	if err := db.Where("topic_id = ?", topicID).
		Order("post_no asc").Order("id asc").
		Find(&postList).Error; err != nil {
		return fmt.Errorf("rebuildTopicUserStats: 加载话题 %d posts 失败: %w", topicID, err)
	}
	byUser := map[uint64]uint32{}
	lastByUser := map[uint64]time.Time{}
	for _, p := range postList {
		if p.PostNo == 1 || p.VisibilityStatus != posts.VisibilityActive {
			continue
		}
		byUser[p.UserId]++
		if t, ok := lastByUser[p.UserId]; !ok || p.CreatedAt.After(t) {
			lastByUser[p.UserId] = p.CreatedAt
		}
	}
	for userID, count := range byUser {
		last := lastByUser[userID]
		if err := db.Create(&topicUserStat.Entity{
			TopicId:     topicID,
			UserId:      userID,
			ReplyCount:  count,
			LastReplyAt: last,
		}).Error; err != nil {
			return fmt.Errorf("rebuildTopicUserStats: 创建话题 %d 用户 %d 统计失败: %w", topicID, userID, err)
		}
	}
	return nil
}

// importPostRevisions 导入帖子版本快照（首楼编辑 PR 新增的表）。
// 依赖 posts 存在：postId 对应的帖子必须已导入，否则跳过该行并报错。
// 与导出顺序一致，在 posts 之后调用；已存在记录跳过（幂等）。
func importPostRevisions(db *gorm.DB, rows []map[string]any, report *ImportReport) {
	for i, row := range rows {
		line := i + 1
		report.Total++
		id := rowUint64(row, "id")
		postID := rowUint64(row, "postId")
		version := rowUint64(row, "version")
		if id == 0 || postID == 0 {
			report.Failed++
			report.Errors = append(report.Errors, ImportError{Line: line, Table: "postRevisions", Reason: "id/postId 必填"})
			continue
		}
		if version == 0 {
			report.Failed++
			report.Errors = append(report.Errors, ImportError{Line: line, Table: "postRevisions", Reason: "version 必填且 >= 1"})
			continue
		}
		// 帖子必须已存在（外键一致性，与 importTopicCategoryIndexes 同语义）
		var post posts.Entity
		if err := db.First(&post, postID).Error; err != nil {
			report.Failed++
			report.Errors = append(report.Errors, ImportError{Line: line, Table: "postRevisions", Reason: "postId 不存在"})
			continue
		}
		var existing postRevisions.Entity
		if err := db.First(&existing, id).Error; err == nil && existing.Id > 0 {
			report.Skipped++
			continue
		}
		// (post_id, version) 必须唯一：表无唯一索引，合并导入时不同 id 的
		// 相同 post_id+version 会产生重复版本行，DB 不会拦截（review 发现）。
		var dup postRevisions.Entity
		if err := db.Where("post_id = ? AND version = ?", postID, version).First(&dup).Error; err == nil && dup.Id > 0 {
			report.Skipped++
			continue
		}
		createdAt, _ := time.Parse(time.RFC3339, rowString(row, "createdAt"))
		entity := postRevisions.Entity{
			Id:            id,
			PostId:        postID,
			Version:       version,
			EditorId:      rowUint64(row, "editorId"),
			Content:       rowString(row, "content"),
			RenderedHTML:  rowString(row, "renderedHTML"),
			ProcessStatus: int8(rowInt64(row, "processStatus")),
			CreatedAt:     createdAt,
		}
		if err := db.Create(&entity).Error; err != nil {
			report.Failed++
			report.Errors = append(report.Errors, ImportError{Line: line, Table: "postRevisions", Reason: err.Error()})
			continue
		}
		report.Success++
	}
}

func importPosts(db *gorm.DB, rows []map[string]any, report *ImportReport) {
	for i, row := range rows {
		line := i + 1
		report.Total++
		id := rowUint64(row, "id")
		if id == 0 {
			report.Failed++
			report.Errors = append(report.Errors, ImportError{Line: line, Table: "posts", Reason: "id 必填"})
			continue
		}
		topicID := rowUint64(row, "topicId")
		userID := rowUint64(row, "userId")
		if topicID == 0 || userID == 0 {
			report.Failed++
			report.Errors = append(report.Errors, ImportError{Line: line, Table: "posts", Reason: "topicId/userId 必填"})
			continue
		}
		var topic topics.Entity
		if err := db.First(&topic, topicID).Error; err != nil {
			report.Failed++
			report.Errors = append(report.Errors, ImportError{Line: line, Table: "posts", Reason: "topicId 不存在"})
			continue
		}
		var user users.EntityComplete
		if err := db.First(&user, userID).Error; err != nil {
			report.Failed++
			report.Errors = append(report.Errors, ImportError{Line: line, Table: "posts", Reason: "userId 不存在"})
			continue
		}
		content := rowString(row, "content")
		// 删除生命周期状态随导出恢复：墓碑态删除帖（USER_DELETED）保留正文、
		// 隐私擦除帖（ACCOUNT_ANONYMIZED）正文为空。只有 ACTIVE 帖强制要求正文，
		// 否则 round-trip 会把删除帖复活为 ACTIVE 公开可见（PR #217 review 发现）。
		visibility := rowString(row, "visibilityStatus")
		if visibility == "" {
			visibility = posts.VisibilityActive
		}
		if strings.TrimSpace(content) == "" && visibility == posts.VisibilityActive {
			report.Failed++
			report.Errors = append(report.Errors, ImportError{Line: line, Table: "posts", Reason: "content 必填"})
			continue
		}
		var existing posts.Entity
		if err := db.First(&existing, id).Error; err == nil && existing.Id > 0 {
			report.Skipped++
			continue
		}
		post := posts.Entity{
			Id:               id,
			TopicId:          topicID,
			PostNo:           rowUint64(row, "postNo"),
			UserId:           userID,
			ReplyToPostId:    rowUint64(row, "replyToPostId"),
			Content:          content,
			ProcessStatus:    int8(rowInt64(row, "processStatus")),
			LastEditorId:     rowUint64(row, "lastEditorId"),
			VisibilityStatus: visibility,
			RetentionStatus:  rowString(row, "retentionStatus"),
			DeletedBy:        rowUint64(row, "deletedBy"),
			DeleteReason:     rowString(row, "deleteReason"),
		}
		if post.RetentionStatus == "" {
			post.RetentionStatus = posts.RetentionNormal
		}
		if dt := rowString(row, "deletedAt"); dt != "" {
			if t, err := time.Parse(time.RFC3339, dt); err == nil {
				post.DeletedAt = gorm.DeletedAt{Time: t, Valid: true}
			}
		}
		// 最后编辑者/时间随导出恢复（首楼编辑 PR 新增字段，缺失时为 0/空）。
		if le := rowString(row, "lastEditedAt"); le != "" {
			if t, err := time.Parse(time.RFC3339Nano, le); err == nil {
				post.LastEditedAt = &t
			}
		}
		if err := db.Create(&post).Error; err != nil {
			report.Failed++
			report.Errors = append(report.Errors, ImportError{Line: line, Table: "posts", Reason: err.Error()})
			continue
		}
		report.Success++
	}
}

func rowString(row map[string]any, key string) string {
	if v, ok := row[key].(string); ok {
		return v
	}
	return ""
}

func rowInt64(row map[string]any, key string) int64 {
	switch v := row[key].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case json.Number:
		n, _ := v.Int64()
		return n
	}
	return 0
}

func rowUint64(row map[string]any, key string) uint64 {
	return uint64(rowInt64(row, key))
}

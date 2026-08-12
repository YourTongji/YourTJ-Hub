package dataservice

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
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
}

// ImportData 导入 JSON 数据（仅支持 JSON 格式）。
// 支持两种结构：数组 `[{...}]` 或对象 `{"users":[...],"topics":[...],"posts":[...]}`。
// 按 users → topics → posts 顺序导入；已存在记录跳过（幂等）。
// 导入完成后重建话题 invariants（首末帖指针、计数、post_seq、参与者统计、
// 分类索引），保证 round-trip 后结构与源库一致且可继续回复（issue #135）。
func ImportData(_ context.Context, data []byte, format string) (*ImportReport, error) {
	if format != "json" {
		return nil, fmt.Errorf("导入仅支持 JSON 格式")
	}
	parsed, err := parseImportJSON(data)
	if err != nil {
		return nil, err
	}
	report := &ImportReport{
		Errors:   []ImportError{},
		Imported: []string{},
	}
	importUsers(parsed["users"], report)
	importTopics(parsed["topics"], report)
	importPosts(parsed["posts"], report)
	importTopicCategoryIndexes(parsed["topicCategoryIndex"], report)
	importTopicUserStats(parsed["topicUserStat"], report)
	// 话题 invariants 需要 posts 全部落库后才能推导，必须在 posts 导入之后执行。
	rebuildTopicInvariants()
	// 显式主键写入不会推进 PostgreSQL sequence，导入后需手动推进，
	// 否则下一次自动插入可能复用已导入 ID 触发主键冲突。
	resetPostgresSequences()
	for _, t := range []string{"users", "topics", "posts", "topicCategoryIndex", "topicUserStat"} {
		if _, ok := parsed[t]; ok {
			report.Imported = append(report.Imported, t)
		}
	}
	return report, nil
}

// resetPostgresSequences 导入显式主键后推进 users/topics/posts 的 sequence。
// SQLite 无 sequence 概念，无需处理。
func resetPostgresSequences() {
	if dbconnect.IsSqlite() {
		return
	}
	db := dbconnect.Connect()
	for _, table := range []string{"users", "topics", "posts"} {
		db.Exec(fmt.Sprintf(
			`SELECT setval(pg_get_serial_sequence('%s', 'id'), GREATEST((SELECT COALESCE(MAX(id), 1) FROM %s), 1), true)`,
			table, table,
		))
	}
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
		for _, t := range []string{"users", "topics", "posts", "topicCategoryIndex", "topicUserStat"} {
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

func importUsers(rows []map[string]any, report *ImportReport) {
	db := dbconnect.Connect()
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

func importTopics(rows []map[string]any, report *ImportReport) {
	db := dbconnect.Connect()
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
			if category.Get(cid).Id == 0 {
				report.Failed++
				report.Errors = append(report.Errors, ImportError{Line: line, Table: "topics", Reason: fmt.Sprintf("分类 %d 不存在", cid)})
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

func importTopicCategoryIndexes(rows []map[string]any, report *ImportReport) {
	db := dbconnect.Connect()
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

func importTopicUserStats(rows []map[string]any, report *ImportReport) {
	db := dbconnect.Connect()
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
		var existing topicUserStat.Entity
		if err := db.First(&existing, id).Error; err == nil && existing.Id > 0 {
			report.Skipped++
			continue
		}
		lastReplyAt := time.Now()
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
//   - topic_category_index 缺失时按 topics.categoryIds 补齐，避免分类/搜索关联丢失。
//
// 仅当话题缺失关键字段时补算，避免覆盖新导出格式中导出的精确值。
func rebuildTopicInvariants() {
	db := dbconnect.Connect()
	var list []topics.Entity
	if err := db.Find(&list).Error; err != nil {
		return
	}
	for _, topic := range list {
		// 判断该话题是否需要按 posts 回填：invariants 缺失（旧格式导出，
		// postSeq/firstPostId/lastPostId/lastPostedAt 全 0 或空）即回填。
		needsBackfill := topic.PostSeq == 0 || topic.FirstPostId == 0 || topic.LastPostId == 0 || topic.LastPostedAt == nil
		if !needsBackfill {
			// 分类索引由导出文件显式导入；仅当缺失时按 topics.categoryIds 补齐
			ensureTopicCategoryIndexes(db, topic.Id)
			continue
		}
		var postList []*posts.Entity
		if err := db.Where("topic_id = ?", topic.Id).
			Order("post_no asc").Order("id asc").
			Find(&postList).Error; err != nil {
			continue
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
			"post_seq":     maxPostNo,
			"first_post_id": firstPostID,
			"last_post_id":  lastPostID,
			"post_count":    len(postList),
		}
		// reply_count = post_count - 1（首帖不计回复）；若导出已带正确值则保留
		if topic.ReplyCount == 0 && len(postList) > 1 {
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
			continue
		}
		// 参与者统计：导入文件带 topicUserStat 时保留导出值，缺失时从 posts 重建
		var count int64
		db.Model(&topicUserStat.Entity{}).Where("topic_id = ?", topic.Id).Count(&count)
		if count == 0 {
			rebuildTopicUserStats(db, topic.Id)
		}
		ensureTopicCategoryIndexes(db, topic.Id)
	}
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
func ensureTopicCategoryIndexes(db *gorm.DB, topicID uint64) {
	var topic topics.Entity
	if err := db.First(&topic, topicID).Error; err != nil {
		return
	}
	for _, cid := range topic.CategoryIds {
		if cid == 0 {
			continue
		}
		var count int64
		db.Model(&topicCategoryIndex.Entity{}).
			Where("topic_id = ? AND category_id = ?", topicID, cid).
			Count(&count)
		if count > 0 {
			continue
		}
		_ = db.Create(&topicCategoryIndex.Entity{
			TopicId:    topicID,
			CategoryId: cid,
			Effective:  1,
		}).Error
	}
}

// rebuildTopicUserStats 从 posts 重建话题参与者统计（reply_count/last_reply_at）。
func rebuildTopicUserStats(db *gorm.DB, topicID uint64) {
	var postList []*posts.Entity
	if err := db.Where("topic_id = ?", topicID).
		Order("post_no asc").Order("id asc").
		Find(&postList).Error; err != nil {
		return
	}
	byUser := map[uint64]uint32{}
	lastByUser := map[uint64]time.Time{}
	for _, p := range postList {
		if p.PostNo == 1 {
			continue
		}
		byUser[p.UserId]++
		if t, ok := lastByUser[p.UserId]; !ok || p.CreatedAt.After(t) {
			lastByUser[p.UserId] = p.CreatedAt
		}
	}
	for userID, count := range byUser {
		last := lastByUser[userID]
		_ = db.Create(&topicUserStat.Entity{
			TopicId:     topicID,
			UserId:      userID,
			ReplyCount:  count,
			LastReplyAt: last,
		}).Error
	}
}

func importPosts(rows []map[string]any, report *ImportReport) {
	db := dbconnect.Connect()
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
		if strings.TrimSpace(content) == "" {
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
			Id:            id,
			TopicId:       topicID,
			PostNo:        rowUint64(row, "postNo"),
			UserId:        userID,
			ReplyToPostId: rowUint64(row, "replyToPostId"),
			Content:       content,
			ProcessStatus: int8(rowInt64(row, "processStatus")),
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

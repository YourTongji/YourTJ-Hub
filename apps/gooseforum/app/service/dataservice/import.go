package dataservice

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/category"
	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
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
	// 显式主键写入不会推进 PostgreSQL sequence，导入后需手动推进，
	// 否则下一次自动插入可能复用已导入 ID 触发主键冲突。
	resetPostgresSequences()
	for _, t := range []string{"users", "topics", "posts"} {
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
		for _, t := range []string{"users", "topics", "posts"} {
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
		if err := db.Create(&topic).Error; err != nil {
			report.Failed++
			report.Errors = append(report.Errors, ImportError{Line: line, Table: "topics", Reason: err.Error()})
			continue
		}
		report.Success++
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

package moderationDecision

import (
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"gorm.io/gorm"
)

func builder() *gorm.DB {
	return db.Connect().Table(tableName)
}

func Create(entity *Entity) error {
	return builder().Create(entity).Error
}

func Get(id uint64) (entity Entity) {
	builder().Where(queryopt.Eq("id", id)).Limit(1).Find(&entity)
	return
}

// LatestForSubjects 批量返回每个主体最近一次 AI 决策（审核队列展示与人工结论回写用）。
func LatestForSubjects(subjectType string, subjectIDs []uint64) map[uint64]Entity {
	result := make(map[uint64]Entity, len(subjectIDs))
	if len(subjectIDs) == 0 {
		return result
	}
	var list []Entity
	builder().
		Where(queryopt.Eq("subject_type", subjectType)).
		Where(queryopt.In("subject_id", subjectIDs)).
		Order("id DESC").
		Find(&list)
	for _, item := range list {
		if _, ok := result[item.SubjectId]; !ok {
			result[item.SubjectId] = item
		}
	}
	return result
}

// SetHumanAction 写入人工结论（审核/标注）；只更新人工字段，不改动 AI 原始信号。
func SetHumanAction(id uint64, action string, actorID uint64, at time.Time) error {
	return builder().Where(queryopt.Eq("id", id)).Updates(map[string]any{
		"human_action":   action,
		"human_actor_id": actorID,
		"human_at":       at,
	}).Error
}

// ListQuery 管理端分页筛选。
type ListQuery struct {
	Page, PageSize int
	FinalAction    string
	HumanAction    string // "" 不限；"none" 仅未标注
	Mode           string
}

func Page(q ListQuery) (list []Entity, total int64) {
	b := builder()
	if q.FinalAction != "" {
		b = b.Where(queryopt.Eq("final_action", q.FinalAction))
	}
	if q.Mode != "" {
		b = b.Where(queryopt.Eq("mode", q.Mode))
	}
	switch q.HumanAction {
	case "":
	case "none":
		b = b.Where(queryopt.Eq("human_action", ""))
	default:
		b = b.Where(queryopt.Eq("human_action", q.HumanAction))
	}
	b.Count(&total)
	b.Order("id DESC").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&list)
	return
}

// ListLabeled 返回带人工结论的决策（离线阈值回放样本），按 id 倒序、上限 limit。
func ListLabeled(limit int) (list []Entity) {
	builder().
		Where(queryopt.Ne("human_action", "")).
		Order("id DESC").
		Limit(limit).
		Find(&list)
	return
}

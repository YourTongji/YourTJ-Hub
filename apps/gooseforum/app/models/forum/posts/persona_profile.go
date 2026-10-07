package posts

import (
	"gorm.io/gorm"
)

// PublicPersonaPosts excludes hidden parents, deleted first posts and retained evidence.
func PublicPersonaPosts(conn *gorm.DB, uid string, page int) ([]Entity, int64, int64, error) {
	base := func() *gorm.DB {
		return conn.Model(&Entity{}).Where("posts.persona_uid = ? AND posts.process_status = 0 AND posts.visibility_status = 'ACTIVE' AND posts.retention_status <> 'PURGED'", uid).Where("EXISTS (SELECT 1 FROM topics t JOIN posts f ON f.id=t.first_post_id AND f.topic_id=t.id WHERE t.id=posts.topic_id AND t.status=1 AND t.process_status=0 AND t.visibility_status='ACTIVE' AND t.retention_status<>'PURGED' AND t.topic_type=0 AND t.deleted_at IS NULL AND f.process_status=0 AND f.visibility_status='ACTIVE' AND f.retention_status<>'PURGED' AND f.deleted_at IS NULL)")
	}
	var topics, replies int64
	if err := base().Where("posts.post_no = 1").Count(&topics).Error; err != nil {
		return nil, 0, 0, err
	}
	if err := base().Where("posts.post_no > 1").Count(&replies).Error; err != nil {
		return nil, 0, 0, err
	}
	var rows []Entity
	err := base().Where("posts.post_no > 1").Order("posts.id DESC").Offset((max(page, 1) - 1) * 20).Limit(20).Find(&rows).Error
	return rows, topics, replies, err
}

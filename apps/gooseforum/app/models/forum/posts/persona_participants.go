package posts

import "gorm.io/gorm"

type PersonaParticipant struct {
	TopicID    uint64
	PersonaUID string
}

// PersonaParticipants bounds each visible topic's public participant projection.
// Neither the selected columns nor the result contains the private owner.
func PersonaParticipants(conn *gorm.DB, topicIDs []uint64) ([]PersonaParticipant, error) {
	if len(topicIDs) == 0 {
		return []PersonaParticipant{}, nil
	}
	var rows []PersonaParticipant
	err := conn.Raw(`SELECT topic_id, persona_uid FROM (
 SELECT topic_id, persona_uid, ROW_NUMBER() OVER (PARTITION BY topic_id ORDER BY MIN(id)) AS position
 FROM posts WHERE topic_id IN ? AND persona_uid <> '' AND deleted_at IS NULL
 AND retention_status <> 'PURGED' AND visibility_status = ? AND process_status = ? GROUP BY topic_id, persona_uid
 ) participants WHERE position <= 12 ORDER BY topic_id, position`, topicIDs, VisibilityActive, ProcessStatusNormal).Scan(&rows).Error
	return rows, err
}

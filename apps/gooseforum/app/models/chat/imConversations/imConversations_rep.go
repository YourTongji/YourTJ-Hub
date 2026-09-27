package imConversations

import (
	"strings"
	"time"

	"github.com/samber/lo"
)

// MessagePreview fits PostgreSQL's 255-character summary column without changing
// the message body. Keep Unicode characters and complete sticker tokens intact.
func MessagePreview(content string) string {
	runes := []rune(content)
	if len(runes) <= 255 {
		return content
	}
	preview := string(runes[:254])
	if start := strings.LastIndex(preview, "[:sticker:"); start >= 0 && !strings.Contains(preview[start:], ":]") {
		preview = preview[:start]
	}
	return preview + "…"
}

func create(entity *Entity) int64 {
	result := builder().Create(entity)
	return result.RowsAffected
}

func save(entity *Entity) int64 {
	result := builder().Save(entity)
	return result.RowsAffected
}

func SaveOrCreateById(entity *Entity) int64 {
	if entity.Id == 0 {
		return create(entity)
	} else {
		return save(entity)
	}
}

func UpdateLastMsg(id uint64, content string) {
	builder().Where("id = ?", id).Updates(map[string]any{
		"last_msg_content": MessagePreview(content),
		"last_msg_time":    time.Now(),
	})
}

func GetByIds(ids []uint64) (entities []*Entity) {
	if len(ids) == 0 {
		return
	}
	builder().Where("id IN ?", ids).Find(&entities)
	return
}

func GetMapByIds(ids []uint64) map[uint64]*Entity {
	return lo.KeyBy(GetByIds(ids), func(v *Entity) uint64 {
		return v.Id
	})
}

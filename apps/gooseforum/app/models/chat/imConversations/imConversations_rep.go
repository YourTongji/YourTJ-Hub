package imConversations

import (
	"strings"
	"time"

	"github.com/samber/lo"
)

// replyQuotePrefix starts the plain-text quote line a chat reply prepends
// ("> @user: excerpt"); the reply body follows after a blank line.
const replyQuotePrefix = "> "

// stripReplyQuote drops a leading reply quote so a conversation preview shows
// the reply itself instead of the quoted excerpt, which would otherwise consume
// the bounded preview. Content without a quote, and a quote without a body,
// stay untouched.
func stripReplyQuote(content string) string {
	if !strings.HasPrefix(content, replyQuotePrefix) {
		return content
	}
	separator := strings.Index(content, "\n\n")
	if separator < 0 {
		return content
	}
	if body := content[separator+2:]; body != "" {
		return body
	}
	return content
}

// MessagePreview fits PostgreSQL's 255-character summary column without changing
// the message body. A leading reply quote is dropped first, and Unicode
// characters and complete sticker tokens stay intact.
func MessagePreview(content string) string {
	content = stripReplyQuote(content)
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

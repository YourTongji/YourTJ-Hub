package users

import (
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"gorm.io/gorm"
	"strings"
)

// MentionTarget is deliberately limited to fields the editor may display.
type MentionTarget struct {
	UserID    uint64 `gorm:"column:id" json:"userId"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	AvatarURL string `gorm:"column:avatar_url" json:"avatarUrl"`
	ActorType int8   `json:"actorType"`
}

func GetInteractionUserTx(tx *gorm.DB, id uint64) (EntityComplete, error) {
	var u EntityComplete
	err := tx.Where("id = ? AND deleted_at IS NULL", id).Take(&u).Error
	return u, err
}
func MentionTargetsTx(tx *gorm.DB, names []string) (map[string]MentionTarget, error) {
	result := map[string]MentionTarget{}
	for start := 0; start < len(names); start += 500 {
		var rows []MentionTarget
		err := tx.Model(&EntityComplete{}).Select("id, username, nickname, avatar_url, actor_type").Where("username IN ? AND is_frozen = ? AND deleted_at IS NULL", names[start:min(start+500, len(names))], StatusNormal).Where("actor_type IN ?", []int8{ActorTypeHuman, ActorTypeBot}).Find(&rows).Error
		if err != nil {
			return nil, err
		}
		botIDs := []uint64{}
		for _, r := range rows {
			if r.ActorType == ActorTypeBot {
				botIDs = append(botIDs, r.UserID)
			}
		}
		enabledSet := map[uint64]bool{}
		if len(botIDs) > 0 {
			enabled, err := agents.EnabledIDsTx(tx)
			if err != nil {
				return nil, err
			}
			for _, id := range enabled {
				enabledSet[id] = true
			}
		}
		for _, r := range rows {
			if r.ActorType == ActorTypeHuman || enabledSet[r.UserID] {
				result[r.Username] = r
			}
		}
	}
	return result, nil
}
func ResolveMentionIDsTx(tx *gorm.DB, content string, exclude uint64, limit int) ([]uint64, error) {
	names := markdown2html.ExtractMentions(content)
	targets, err := MentionTargetsTx(tx, names)
	if err != nil {
		return nil, err
	}
	ids := make([]uint64, 0)
	seen := map[uint64]bool{}
	for _, name := range names {
		id := targets[name].UserID
		if id == 0 || id == exclude || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
		if limit > 0 && len(ids) >= limit {
			break
		}
	}
	return ids, nil
}
func ListMentionTargets(q string, limit int) ([]MentionTarget, error) {
	if limit < 1 {
		limit = 10
	}
	if limit > 20 {
		limit = 20
	}
	q = strings.TrimSpace(q)
	rows := make([]MentionTarget, 0)
	enabled, err := agents.EnabledIDsTx(db.Connect())
	if err != nil {
		return nil, err
	}
	if len(enabled) == 0 {
		enabled = []uint64{0}
	}
	err = db.Connect().Model(&EntityComplete{}).Select("id, username, nickname, avatar_url, actor_type").Where("is_frozen = ? AND deleted_at IS NULL", StatusNormal).Where("actor_type = ? OR (actor_type = ? AND id IN ?)", ActorTypeHuman, ActorTypeBot, enabled).Where("LOWER(username) LIKE ? OR LOWER(nickname) LIKE ?", strings.ToLower(q)+"%", strings.ToLower(q)+"%").Order("username").Limit(limit).Find(&rows).Error
	return rows, err
}
func FilterHumanRecipientsTx(tx *gorm.DB, ids []uint64) ([]uint64, error) {
	var result []uint64
	if len(ids) == 0 {
		return result, nil
	}
	err := tx.Model(&EntityComplete{}).Where("id IN ? AND actor_type = ? AND deleted_at IS NULL AND is_frozen = ?", ids, ActorTypeHuman, StatusNormal).Pluck("id", &result).Error
	return result, err
}

// ExcludeBotRecipientsTx leaves the legacy handling of nonexistent recipients
// unchanged while routing every known bot to the dedicated Agent event stream.
func ExcludeBotRecipientsTx(tx *gorm.DB, ids []uint64) ([]uint64, error) {
	if len(ids) == 0 {
		return ids, nil
	}
	var bots []uint64
	if err := tx.Model(&EntityComplete{}).Where("id IN ? AND actor_type = ?", ids, ActorTypeBot).Pluck("id", &bots).Error; err != nil {
		return nil, err
	}
	excluded := map[uint64]bool{}
	for _, id := range bots {
		excluded[id] = true
	}
	out := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if !excluded[id] {
			out = append(out, id)
		}
	}
	return out, nil
}

func PublicMentionTargetIDs(names []string) map[string]uint64 {
	targets, err := MentionTargetsTx(db.Connect(), names)
	result := map[string]uint64{}
	if err != nil {
		return result
	}
	for name, target := range targets {
		result[name] = target.UserID
	}
	return result
}

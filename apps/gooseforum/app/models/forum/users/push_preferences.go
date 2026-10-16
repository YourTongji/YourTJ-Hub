package users

import (
	"encoding/json"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func GetPushPreferences(userID uint64) (map[string]bool, error) {
	var user EntityComplete
	if err := builder().Select("id", "push_preferences").First(&user, userID).Error; err != nil {
		return nil, err
	}
	values := map[string]bool{}
	if user.PushPreferences != nil {
		if err := json.Unmarshal([]byte(*user.PushPreferences), &values); err != nil {
			return nil, err
		}
	}
	return values, nil
}

// PatchPushPreferences merges only named groups under the account row lock.
func PatchPushPreferences(userID uint64, changes map[string]*bool) error {
	return builder().Transaction(func(tx *gorm.DB) error {
		var user EntityComplete
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id", "push_preferences").First(&user, userID).Error; err != nil {
			return err
		}
		values := map[string]bool{}
		if user.PushPreferences != nil {
			if err := json.Unmarshal([]byte(*user.PushPreferences), &values); err != nil {
				return err
			}
		}
		for key, value := range changes {
			if value != nil {
				values[key] = *value
			}
		}
		raw, err := json.Marshal(values)
		if err != nil {
			return err
		}
		return tx.Model(&EntityComplete{}).Where("id = ?", userID).Update("push_preferences", string(raw)).Error
	})
}

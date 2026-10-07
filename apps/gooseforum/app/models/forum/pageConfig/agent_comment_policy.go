package pageConfig

import (
	"encoding/json"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// LockAgentCommentPolicyTx also materializes the default row so a concurrent
// first admin save cannot bypass a missing-row lock. All reads use the caller's
// connection, including on single-connection SQLite deployments.
func LockAgentCommentPolicyTx(tx *gorm.DB) (AgentCommentPolicyConfig, error) {
	defaultRow := Entity{PageType: AgentCommentPolicy, Config: `{"allowAgentComments":true}`}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&defaultRow).Error; err != nil {
		return AgentCommentPolicyConfig{}, err
	}
	var row Entity
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("page_type = ?", AgentCommentPolicy).Take(&row).Error; err != nil {
		return AgentCommentPolicyConfig{}, err
	}
	var config AgentCommentPolicyConfig
	err := json.Unmarshal([]byte(row.Config), &config)
	return config, err
}

func SaveAgentCommentPolicyTx(tx *gorm.DB, config AgentCommentPolicyConfig) error {
	encoded, err := json.Marshal(config)
	if err != nil {
		return err
	}
	row := Entity{PageType: AgentCommentPolicy, Config: string(encoded)}
	return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "page_type"}}, DoUpdates: clause.AssignmentColumns([]string{"config", "updated_at"})}).Create(&row).Error
}

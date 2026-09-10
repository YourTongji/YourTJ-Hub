package userTotpChallenges

import (
	"gorm.io/gorm"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
)

func builder() *gorm.DB {
	return db.Connect().Table(tableName)
}

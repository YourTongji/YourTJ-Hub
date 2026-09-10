package course

import (
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"gorm.io/gorm"
)

func courseBuilder() *gorm.DB {
	return db.Connect().Table(tableName)
}

func aliasBuilder() *gorm.DB {
	return db.Connect().Table(aliasTableName)
}

func termBuilder() *gorm.DB {
	return db.Connect().Table(termTableName)
}

func offeringBuilder() *gorm.DB {
	return db.Connect().Table(offeringTableName)
}

func instructorBuilder() *gorm.DB {
	return db.Connect().Table(instructorTableName)
}

func offeringInstructorBuilder() *gorm.DB {
	return db.Connect().Table(offeringInstructorTableName)
}

func reviewBuilder() *gorm.DB {
	return db.Connect().Table(reviewTableName)
}

func helpfulBuilder() *gorm.DB {
	return db.Connect().Table(helpfulTableName)
}

func dislikeBuilder() *gorm.DB {
	return db.Connect().Table(dislikeTableName)
}

func courseStatsBuilder() *gorm.DB {
	return db.Connect().Table(courseStatsTableName)
}

func offeringStatsBuilder() *gorm.DB {
	return db.Connect().Table(offeringStatsTableName)
}

func importRunBuilder() *gorm.DB {
	return db.Connect().Table(importRunTableName)
}

func sourceRefBuilder() *gorm.DB {
	return db.Connect().Table(sourceRefTableName)
}

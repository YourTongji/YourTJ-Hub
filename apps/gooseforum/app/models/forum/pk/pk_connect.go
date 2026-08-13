package pk

import (
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"gorm.io/gorm"
)

// 仅保留被引用的 builder；其余查找表/关联表通过 tx 或 db.Connect().Table(...) 直查，
// 不引入未使用代码（见 pk_rep.go 的 ListFacultiesTx / ListTeacherTimeslotSource）。

func calendarBuilder() *gorm.DB {
	return db.Connect().Table(calendarTableName)
}

func courseDetailBuilder() *gorm.DB {
	return db.Connect().Table(courseDetailTableName)
}

func teacherBuilder() *gorm.DB {
	return db.Connect().Table(teacherTableName)
}

func fetchLogBuilder() *gorm.DB {
	return db.Connect().Table(fetchLogTableName)
}

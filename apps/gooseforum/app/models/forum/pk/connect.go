package pk

import (
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"gorm.io/gorm"
)

func calendarBuilder() *gorm.DB {
	return db.Connect().Table(calendarTableName)
}

func campusBuilder() *gorm.DB {
	return db.Connect().Table(campusTableName)
}

func facultyBuilder() *gorm.DB {
	return db.Connect().Table(facultyTableName)
}

func languageBuilder() *gorm.DB {
	return db.Connect().Table(languageTableName)
}

func assessmentBuilder() *gorm.DB {
	return db.Connect().Table(assessmentTableName)
}

func courseNatureBuilder() *gorm.DB {
	return db.Connect().Table(courseNatureTableName)
}

func majorBuilder() *gorm.DB {
	return db.Connect().Table(majorTableName)
}

func majorCourseBuilder() *gorm.DB {
	return db.Connect().Table(majorCourseTableName)
}

func courseDetailBuilder() *gorm.DB {
	return db.Connect().Table(courseDetailTableName)
}

func teacherBuilder() *gorm.DB {
	return db.Connect().Table(teacherTableName)
}

func teacherTimeslotBuilder() *gorm.DB {
	return db.Connect().Table(teacherTimeslotTableName)
}

func fetchLogBuilder() *gorm.DB {
	return db.Connect().Table(fetchLogTableName)
}

func settingBuilder() *gorm.DB {
	return db.Connect().Table(settingTableName)
}

package pkservice

import (
	"context"
	"fmt"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"gorm.io/gorm"
)

// rebuildTimeslots 重建指定学期（们）的 teacher_timeslots：解析 pk_teacher.arrange_info_text
// 生成每天×节次时间片，单事务内清空重插。同一 (day, section) 去重。
func rebuildTimeslots(ctx context.Context, calendarIds []uint64) (int, error) {
	if len(calendarIds) == 0 {
		return 0, nil
	}
	source, err := pk.ListTeacherTimeslotSource(calendarIds)
	if err != nil {
		return 0, err
	}

	rows, err := buildTimeslotRows(source)
	if err != nil {
		return 0, err
	}
	if err := db.Connect().Transaction(func(tx *gorm.DB) error {
		return pk.ReplaceTeacherTimeslotsTx(tx, calendarIds, rows)
	}); err != nil {
		return 0, err
	}
	return len(rows), nil
}

func buildTimeslotRows(source []pk.TeacherTimeslotSourceRow) ([]pk.TeacherTimeslotEntity, error) {
	seen := map[string]bool{}
	var rows []pk.TeacherTimeslotEntity
	for _, r := range source {
		if r.CalendarId == 0 || r.TeachingClassId == 0 {
			continue
		}
		for _, line := range splitEndline(r.ArrangeInfoText) {
			info := arrangementTextToObj(line)
			if info.OccupyDay <= 0 || len(info.OccupyTime) == 0 {
				continue
			}
			for _, section := range info.OccupyTime {
				if section <= 0 {
					continue
				}
				key := fmt.Sprintf("%d|%d|%d|%d|%s|%s", r.CalendarId, r.TeachingClassId, info.OccupyDay, section, r.TeacherCode, r.TeacherName)
				if seen[key] {
					continue
				}
				seen[key] = true
				rows = append(rows, pk.TeacherTimeslotEntity{
					CalendarId:      r.CalendarId,
					TeachingClassId: r.TeachingClassId,
					OccupyDay:       info.OccupyDay,
					OccupySection:   section,
					TeacherCode:     r.TeacherCode,
					TeacherName:     r.TeacherName,
				})
			}
		}
	}
	return rows, nil
}

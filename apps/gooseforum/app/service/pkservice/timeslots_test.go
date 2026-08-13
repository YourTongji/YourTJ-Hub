package pkservice

import (
	"context"
	"strconv"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
)

func TestBuildTimeslotRows(t *testing.T) {
	source := []pk.TeacherTimeslotSourceRow{
		{CalendarId: 1, TeachingClassId: 10, TeacherCode: "T001", TeacherName: "张三",
			ArrangeInfoText: "张三(T001) 星期一1-2节[1-16周]\n张三(T001) 星期三5-6节[1-16周]"},
		{CalendarId: 1, TeachingClassId: 11, TeacherCode: "T002", TeacherName: "李四",
			ArrangeInfoText: "李四(T002) 星期五9-10节[1-8周]"},
		{CalendarId: 1, TeachingClassId: 10, TeacherCode: "T001", TeacherName: "张三",
			ArrangeInfoText: "张三(T001) 星期一1-2节[1-16周]"}, // 重复行应去重
	}
	rows, err := buildTimeslotRows(source)
	if err != nil {
		t.Fatalf("buildTimeslotRows: %v", err)
	}
	// 张三周一1/2 + 周三5/6 = 4；李四周五9/10 = 2；重复的周一1-2 已去重 → 共 6。
	if len(rows) != 6 {
		t.Fatalf("rows = %d, want 6 (got %+v)", len(rows), rows)
	}
	want := map[string]bool{
		"1|10|1|1|T001|张三":  true,
		"1|10|1|2|T001|张三":  true,
		"1|10|3|5|T001|张三":  true,
		"1|10|3|6|T001|张三":  true,
		"1|11|5|9|T002|李四":  true,
		"1|11|5|10|T002|李四": true,
	}
	for _, r := range rows {
		key := strconv.FormatUint(r.CalendarId, 10) + "|" + strconv.FormatUint(r.TeachingClassId, 10) + "|" + strconv.Itoa(r.OccupyDay) + "|" + strconv.Itoa(r.OccupySection) + "|" + r.TeacherCode + "|" + r.TeacherName
		if !want[key] {
			t.Errorf("unexpected row %q", key)
		}
	}
}

func TestRebuildTimeslotsReplacesForCalendar(t *testing.T) {
	migratePkTables(t)
	conn := db.Connect()
	if err := conn.Create(&pk.CourseDetailEntity{Id: 10, CalendarId: 1, CourseCode: "A001", CourseName: "课程A"}).Error; err != nil {
		t.Fatalf("create course detail: %v", err)
	}
	if err := conn.Create(&pk.TeacherEntity{
		Id: 100, TeachingClassId: 10, TeacherCode: "T001", TeacherName: "张三",
		ArrangeInfoText: "张三(T001) 星期一1-2节[1-16周]\n张三(T001) 星期三3-4节[1-16周]",
	}).Error; err != nil {
		t.Fatalf("create teacher: %v", err)
	}

	n, err := rebuildTimeslots(context.Background(), []uint64{1})
	if err != nil {
		t.Fatalf("rebuildTimeslots: %v", err)
	}
	if n != 4 {
		t.Fatalf("rebuilt = %d, want 4", n)
	}
	var count int64
	if err := conn.Model(&pk.TeacherTimeslotEntity{}).Where("calendar_id = ?", 1).Count(&count).Error; err != nil {
		t.Fatalf("count timeslots: %v", err)
	}
	if count != 4 {
		t.Errorf("timeslots count = %d, want 4", count)
	}

	// 再次重建（幂等）：仍为 4，且 calendar 2 不受影响。
	if err := conn.Create(&pk.CourseDetailEntity{Id: 20, CalendarId: 2, CourseCode: "B001", CourseName: "课程B"}).Error; err != nil {
		t.Fatalf("create course detail 2: %v", err)
	}
	if err := conn.Create(&pk.TeacherEntity{Id: 200, TeachingClassId: 20, TeacherCode: "T002", TeacherName: "李四", ArrangeInfoText: "李四(T002) 星期二3-4节[1-8周]"}).Error; err != nil {
		t.Fatalf("create teacher 2: %v", err)
	}
	if _, err := rebuildTimeslots(context.Background(), []uint64{1, 2}); err != nil {
		t.Fatalf("second rebuild: %v", err)
	}
	var total int64
	if err := conn.Model(&pk.TeacherTimeslotEntity{}).Count(&total).Error; err != nil {
		t.Fatalf("count total: %v", err)
	}
	if total != 6 { // calendar1 4 + calendar2 2
		t.Errorf("total timeslots = %d, want 6", total)
	}
}

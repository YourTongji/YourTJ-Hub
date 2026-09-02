package courseservice

import (
	"context"
	"fmt"
	"strings"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
)

func migrateMaterializeTables(t *testing.T) {
	t.Helper()
	models := []any{
		&course.Entity{},
		&course.AliasEntity{},
		&course.InstructorEntity{},
		&course.OfferingEntity{},
		&course.OfferingInstructorEntity{},
		&course.TermEntity{},
		&course.RelationEntity{},
		&course.SourceRefEntity{},
		&course.ImportRunEntity{},
		&taskQueue.Entity{},
		&pk.CourseDetailEntity{},
		&pk.TeacherEntity{},
		&pk.FacultyEntity{},
		&pk.CalendarEntity{},
		&pk.LanguageEntity{},
		&pk.CourseNatureEntity{},
		&pk.CourseNatureByCalendarEntity{},
		&pk.AssessmentEntity{},
		&pk.CampusEntity{},
		&pk.MajorEntity{},
		&pk.MajorCourseEntity{},
		&pk.TeacherTimeslotEntity{},
		&pk.FetchLogEntity{},
	}
	conn := db.Connect()
	if err := conn.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for _, m := range models {
		if err := conn.Unscoped().Where("1 = 1").Delete(m).Error; err != nil {
			t.Fatalf("clean: %v", err)
		}
	}
}

func seedPkForMaterialize(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	credit := 5.0
	if err := conn.Create(&pk.CourseDetailEntity{
		Id: 1, CalendarId: 1, CourseCode: "A001", CourseName: "高等数学(A)上", Code: "A00101",
		NewCourseCode: "A001N", NewCode: "A001N01", Credit: &credit, Faculty: "数学科学学院",
	}).Error; err != nil {
		t.Fatalf("seed course detail: %v", err)
	}
	if err := conn.Create(&pk.TeacherEntity{Id: 100, TeachingClassId: 1, TeacherCode: "T001", TeacherName: "张三"}).Error; err != nil {
		t.Fatalf("seed teacher: %v", err)
	}
	if err := conn.Create(&pk.CalendarEntity{CalendarId: 1}).Error; err != nil {
		t.Fatalf("seed calendar: %v", err)
	}
	if err := conn.Create(&pk.FacultyEntity{Faculty: "数学科学学院", FacultyI18n: "数学科学学院"}).Error; err != nil {
		t.Fatalf("seed faculty: %v", err)
	}
}

func TestMaterializeFromPkCreatesCatalog(t *testing.T) {
	migrateMaterializeTables(t)
	seedPkForMaterialize(t)

	report, err := MaterializeFromPk(context.Background(), []uint64{1})
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if report.CoursesInserted != 1 {
		t.Errorf("coursesInserted = %d, want 1", report.CoursesInserted)
	}
	if report.InstructorsInserted != 1 {
		t.Errorf("instructorsInserted = %d, want 1", report.InstructorsInserted)
	}

	conn := db.Connect()
	var courses []course.Entity
	if err := conn.Where("primary_code = ?", "A001").Find(&courses).Error; err != nil {
		t.Fatalf("find course: %v", err)
	}
	if len(courses) != 1 {
		t.Fatalf("courses = %d, want 1", len(courses))
	}
	if courses[0].CreditX10 != 50 {
		t.Errorf("credit_x10 = %d, want 50", courses[0].CreditX10)
	}
	if courses[0].Department != "数学科学学院" {
		t.Errorf("department = %q", courses[0].Department)
	}

	var aliases []course.AliasEntity
	if err := conn.Where("source = ?", materializePkSource).Find(&aliases).Error; err != nil {
		t.Fatalf("find aliases: %v", err)
	}
	// 期望 courseCode/code/newCourseCode/newCode 四条 code 别名。
	if len(aliases) != 4 {
		t.Errorf("aliases = %d, want 4: %+v", len(aliases), aliases)
	}
	var instructors []course.InstructorEntity
	if err := conn.Where("name = ?", "张三").Find(&instructors).Error; err != nil {
		t.Fatalf("find instructor: %v", err)
	}
	if len(instructors) != 1 {
		t.Errorf("instructors = %d, want 1", len(instructors))
	}
}

func TestMaterializeFromPkIdempotent(t *testing.T) {
	migrateMaterializeTables(t)
	seedPkForMaterialize(t)

	first, err := MaterializeFromPk(context.Background(), []uint64{1})
	if err != nil {
		t.Fatalf("first materialize: %v", err)
	}
	second, err := MaterializeFromPk(context.Background(), []uint64{1})
	if err != nil {
		t.Fatalf("second materialize: %v", err)
	}
	if second.CoursesInserted != 0 {
		t.Errorf("second coursesInserted = %d, want 0", second.CoursesInserted)
	}
	if second.InstructorsInserted != 0 {
		t.Errorf("second instructorsInserted = %d, want 0", second.InstructorsInserted)
	}
	conn := db.Connect()
	var courseCount int64
	if err := conn.Model(&course.Entity{}).Where("primary_code = ?", "A001").Count(&courseCount).Error; err != nil {
		t.Fatalf("count courses: %v", err)
	}
	if courseCount != 1 {
		t.Errorf("courses after 2nd run = %d, want 1", courseCount)
	}
	_ = first
}

// TestMaterializeFromPkInstructorNormalizedNameIdempotent 回归 #199：教师名含空格/中点/全角字符时
// 归一化前后不同（如 "John Smith"→"johnsmith"），此前第二次物化仍按原始姓名查找 normalized_name
// 导致必然 miss、重复创建教师。断言第二次 InstructorsInserted == 0。
func TestMaterializeFromPkInstructorNormalizedNameIdempotent(t *testing.T) {
	migrateMaterializeTables(t)
	conn := db.Connect()
	credit := 3.0
	if err := conn.Create(&pk.CourseDetailEntity{
		Id: 1, CalendarId: 1, CourseCode: "B101", CourseName: "数据科学导论", Code: "B10101",
		NewCourseCode: "B101N", NewCode: "B101N01", Credit: &credit, Faculty: "数学科学学院",
	}).Error; err != nil {
		t.Fatalf("seed course detail: %v", err)
	}
	// 含空格、中点、全角字符的教师名：归一化值分别 johnsmith / 爱丽丝史密斯 / john，与原始姓名不同。
	for i, raw := range []string{"John Smith", "爱丽丝·史密斯", "Ｊｏｈｎ"} {
		teacher := pk.TeacherEntity{
			Id: uint64(200 + i), TeachingClassId: 1, TeacherCode: fmt.Sprintf("T2%02d", i),
			TeacherName: raw,
		}
		if err := conn.Create(&teacher).Error; err != nil {
			t.Fatalf("seed teacher %q: %v", raw, err)
		}
	}
	if err := conn.Create(&pk.FacultyEntity{Faculty: "数学科学学院", FacultyI18n: "数学科学学院"}).Error; err != nil {
		t.Fatalf("seed faculty: %v", err)
	}
	if err := conn.Create(&pk.CalendarEntity{CalendarId: 1}).Error; err != nil {
		t.Fatalf("seed calendar: %v", err)
	}

	first, err := MaterializeFromPk(context.Background(), []uint64{1})
	if err != nil {
		t.Fatalf("first materialize: %v", err)
	}
	if first.InstructorsInserted != 3 {
		t.Fatalf("first instructorsInserted = %d, want 3", first.InstructorsInserted)
	}

	second, err := MaterializeFromPk(context.Background(), []uint64{1})
	if err != nil {
		t.Fatalf("second materialize: %v", err)
	}
	if second.InstructorsInserted != 0 {
		t.Errorf("second instructorsInserted = %d, want 0（归一化姓名未命中导致重复创建）", second.InstructorsInserted)
	}

	var total int64
	if err := conn.Model(&course.InstructorEntity{}).Count(&total).Error; err != nil {
		t.Fatalf("count instructors: %v", err)
	}
	if total != 3 {
		t.Errorf("instructors after 2 runs = %d, want 3", total)
	}
}

// TestMaterializeFromPkSplitsByIdentityTeacher 回归 review：同一 courseCode 不同教师的
// 教学班必须物化为独立课程卡（(code, teacher) 复合身份），不能按 code 合并成一张卡
// 后只取 TeacherNames[0]（否则漏卡且选中教师依赖查询顺序）。
func TestMaterializeFromPkSplitsByIdentityTeacher(t *testing.T) {
	migrateMaterializeTables(t)
	conn := db.Connect()
	credit := 5.0
	if err := conn.Create(&pk.CourseDetailEntity{
		Id: 1, CalendarId: 1, CourseCode: "A001", CourseName: "高等数学(A)上", Code: "A00101",
		NewCourseCode: "A001N", NewCode: "A001N01", Credit: &credit, Faculty: "数学科学学院",
	}).Error; err != nil {
		t.Fatalf("seed course detail 1: %v", err)
	}
	if err := conn.Create(&pk.CourseDetailEntity{
		Id: 2, CalendarId: 1, CourseCode: "A001", CourseName: "高等数学(A)上", Code: "A00102",
		NewCourseCode: "A001N", NewCode: "A001N02", Credit: &credit, Faculty: "数学科学学院",
	}).Error; err != nil {
		t.Fatalf("seed course detail 2: %v", err)
	}
	if err := conn.Create(&pk.TeacherEntity{Id: 100, TeachingClassId: 1, TeacherCode: "T001", TeacherName: "张三"}).Error; err != nil {
		t.Fatalf("seed teacher 张三: %v", err)
	}
	if err := conn.Create(&pk.TeacherEntity{Id: 101, TeachingClassId: 2, TeacherCode: "T002", TeacherName: "李四"}).Error; err != nil {
		t.Fatalf("seed teacher 李四: %v", err)
	}
	if err := conn.Create(&pk.FacultyEntity{Faculty: "数学科学学院", FacultyI18n: "数学科学学院"}).Error; err != nil {
		t.Fatalf("seed faculty: %v", err)
	}
	if err := conn.Create(&pk.CalendarEntity{CalendarId: 1}).Error; err != nil {
		t.Fatalf("seed calendar: %v", err)
	}

	report, err := MaterializeFromPk(context.Background(), []uint64{1})
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if report.CoursesInserted != 2 {
		t.Fatalf("coursesInserted = %d, want 2（同 code 双教师应拆成两张卡）", report.CoursesInserted)
	}

	var courses []course.Entity
	if err := conn.Where("primary_code = ?", "A001").Find(&courses).Error; err != nil {
		t.Fatalf("find courses: %v", err)
	}
	if len(courses) != 2 {
		t.Fatalf("courses = %d, want 2", len(courses))
	}
	distinct := map[uint64]bool{}
	for _, c := range courses {
		if c.TeacherId != 0 {
			distinct[c.TeacherId] = true
		}
	}
	if len(distinct) != 2 {
		t.Fatalf("cards must map to 2 distinct identity teachers, got %v", distinct)
	}
	var instructors []course.InstructorEntity
	if err := conn.Find(&instructors).Error; err != nil {
		t.Fatalf("find instructors: %v", err)
	}
	for _, ins := range instructors {
		if !distinct[ins.Id] {
			t.Fatalf("instructor %d not referenced by any card", ins.Id)
		}
	}
}

func TestMaterializeFromPkCreatesOfferingAtClassGranularity(t *testing.T) {
	migrateMaterializeTables(t)
	seedPkForMaterialize(t)
	conn := db.Connect()
	// 学期映射：calendarId=1 → calendar_id_i18n "2025-2026-1" → course_term（缺则自动创建）。
	// seedPkForMaterialize 已建 calendarId=1（i18n 空），此处补学期码。
	if err := conn.Model(&pk.CalendarEntity{}).Where("calendar_id = ?", 1).
		Update("calendar_id_i18n", "2025-2026-1").Error; err != nil {
		t.Fatalf("set calendar i18n: %v", err)
	}

	report, err := MaterializeFromPk(context.Background(), []uint64{1})
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if report.OfferingsInserted != 1 {
		t.Fatalf("offeringsInserted = %d, want 1", report.OfferingsInserted)
	}

	var offerings []course.OfferingEntity
	if err := conn.Where("teaching_class_id = ?", 1).Find(&offerings).Error; err != nil {
		t.Fatalf("find offerings: %v", err)
	}
	if len(offerings) != 1 {
		t.Fatalf("offerings = %d, want 1", len(offerings))
	}
	o := offerings[0]
	if o.ClassCode != "A00101" {
		t.Errorf("class_code = %q, want A00101", o.ClassCode)
	}
	if o.Status != course.OfferingStatusVisible {
		t.Errorf("status = %d, want visible", o.Status)
	}
	// term 映射：course_term 应已按 "2025-2026-1" 创建并挂到 offering。
	var term course.TermEntity
	if err := conn.Where("code = ?", "2025-2026-1").First(&term).Error; err != nil {
		t.Fatalf("term not created: %v", err)
	}
	if o.TermId != term.Id {
		t.Errorf("term_id = %d, want %d", o.TermId, term.Id)
	}
	// offering_instructor：该班全量教师（张三）。
	var links []course.OfferingInstructorEntity
	if err := conn.Where("offering_id = ?", o.Id).Find(&links).Error; err != nil {
		t.Fatalf("find offering instructors: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("offering instructors = %d, want 1", len(links))
	}
	// 课程卡身份教师 = 教学班首位教师（张三）。
	var courses []course.Entity
	if err := conn.Where("primary_code = ?", "A001").Find(&courses).Error; err != nil {
		t.Fatalf("find courses: %v", err)
	}
	if len(courses) != 1 || courses[0].TeacherId == 0 {
		t.Fatalf("identity teacher not linked: %+v", courses)
	}
}

func TestMaterializeFromPkOfferingIdempotent(t *testing.T) {
	migrateMaterializeTables(t)
	seedPkForMaterialize(t)

	first, err := MaterializeFromPk(context.Background(), []uint64{1})
	if err != nil {
		t.Fatalf("first materialize: %v", err)
	}
	if first.OfferingsInserted != 1 {
		t.Fatalf("first offeringsInserted = %d, want 1", first.OfferingsInserted)
	}
	second, err := MaterializeFromPk(context.Background(), []uint64{1})
	if err != nil {
		t.Fatalf("second materialize: %v", err)
	}
	if second.OfferingsInserted != 0 {
		t.Errorf("second offeringsInserted = %d, want 0", second.OfferingsInserted)
	}
	if second.OfferingsUpdated != 1 {
		t.Errorf("second offeringsUpdated = %d, want 1", second.OfferingsUpdated)
	}
	conn := db.Connect()
	var count int64
	if err := conn.Model(&course.OfferingEntity{}).Count(&count).Error; err != nil {
		t.Fatalf("count offerings: %v", err)
	}
	if count != 1 {
		t.Errorf("offerings after 2 runs = %d, want 1", count)
	}
}

func TestMaterializeFromPkDoesNotResurrectHiddenOffering(t *testing.T) {
	migrateMaterializeTables(t)
	seedPkForMaterialize(t)
	conn := db.Connect()

	// 先物化得到 offering，再由管理员隐藏；再次物化不得复活 status。
	if _, err := MaterializeFromPk(context.Background(), []uint64{1}); err != nil {
		t.Fatalf("first materialize: %v", err)
	}
	var offering course.OfferingEntity
	if err := conn.Where("teaching_class_id = ?", 1).First(&offering).Error; err != nil {
		t.Fatalf("find offering: %v", err)
	}
	if err := conn.Model(&course.OfferingEntity{}).Where("id = ?", offering.Id).
		Update("status", course.OfferingStatusHidden).Error; err != nil {
		t.Fatalf("hide offering: %v", err)
	}

	if _, err := MaterializeFromPk(context.Background(), []uint64{1}); err != nil {
		t.Fatalf("second materialize: %v", err)
	}
	var after course.OfferingEntity
	if err := conn.Where("teaching_class_id = ?", 1).First(&after).Error; err != nil {
		t.Fatalf("find offering after: %v", err)
	}
	if after.Status != course.OfferingStatusHidden {
		t.Errorf("status = %d, want hidden（物化不得复活管理员隐藏的 offering）", after.Status)
	}
}

func TestMaterializeFromPkBackfillsTeacherCode(t *testing.T) {
	migrateMaterializeTables(t)
	seedPkForMaterialize(t)
	conn := db.Connect()

	// 预置同名同院系教师（teacher_code 为空）：物化应回填工号而非新建。
	preexisting := course.InstructorEntity{
		Name:           "张三",
		NormalizedName: Normalize("张三"),
		Department:     "数学科学学院",
		Status:         0,
	}
	if err := conn.Create(&preexisting).Error; err != nil {
		t.Fatalf("seed instructor: %v", err)
	}

	report, err := MaterializeFromPk(context.Background(), []uint64{1})
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if report.InstructorsInserted != 0 {
		t.Errorf("instructorsInserted = %d, want 0（已存在教师不应重复创建）", report.InstructorsInserted)
	}
	var updated course.InstructorEntity
	if err := conn.First(&updated, preexisting.Id).Error; err != nil {
		t.Fatalf("find instructor: %v", err)
	}
	if updated.TeacherCode != "T001" {
		t.Errorf("teacher_code = %q, want T001（身份主锚回填）", updated.TeacherCode)
	}
}

// TestMaterializeFromPkRedirectsOfferingToMergedTarget 回归 review：课程卡已被确认合并
// （hidden + status=merged EQUIVALENT/RENAMED_FROM 指向可见卡）时，offering 物化写入
// 必须重定向到合并目标卡，不得把 offering 写回 hidden 旧卡逆转已确认的合并。
func TestMaterializeFromPkRedirectsOfferingToMergedTarget(t *testing.T) {
	migrateMaterializeTables(t)
	seedPkForMaterialize(t)
	conn := db.Connect()

	if _, err := MaterializeFromPk(context.Background(), []uint64{1}); err != nil {
		t.Fatalf("first materialize: %v", err)
	}
	var oldCard course.Entity
	if err := conn.Where("primary_code = ?", "A001").First(&oldCard).Error; err != nil {
		t.Fatalf("find old card: %v", err)
	}
	var offering course.OfferingEntity
	if err := conn.Where("teaching_class_id = ?", 1).First(&offering).Error; err != nil {
		t.Fatalf("find offering: %v", err)
	}
	// 构造一次已确认的合并：目标卡（另一教师身份）可见；from 卡隐藏；offering 迁往目标。
	ins := course.InstructorEntity{Name: "李四", NormalizedName: "lisi", Department: "数学科学学院", TeacherCode: "T002"}
	if err := conn.Create(&ins).Error; err != nil {
		t.Fatalf("create target instructor: %v", err)
	}
	targetCard := course.Entity{PrimaryCode: "A001", TeacherId: ins.Id, Name: "高等数学(A)上", Department: "数学科学学院", Status: course.StatusVisible}
	if err := conn.Create(&targetCard).Error; err != nil {
		t.Fatalf("create target card: %v", err)
	}
	if err := conn.Model(&course.OfferingEntity{}).Where("id = ?", offering.Id).
		Update("course_id", targetCard.Id).Error; err != nil {
		t.Fatalf("move offering to target: %v", err)
	}
	if err := conn.Model(&course.Entity{}).Where("id = ?", oldCard.Id).
		Update("status", course.StatusHidden).Error; err != nil {
		t.Fatalf("hide old card: %v", err)
	}
	relation := course.RelationEntity{
		FromCourseId: oldCard.Id, ToCourseId: targetCard.Id,
		RelationType: string(course.RelationEquivalent), Status: string(course.RelationStatusMerged),
	}
	if err := conn.Create(&relation).Error; err != nil {
		t.Fatalf("create merged relation: %v", err)
	}

	// 再次物化：offering 必须重定向到可见的目标卡，不写回 hidden 旧卡。
	if _, err := MaterializeFromPk(context.Background(), []uint64{1}); err != nil {
		t.Fatalf("second materialize: %v", err)
	}
	var after course.OfferingEntity
	if err := conn.Where("teaching_class_id = ?", 1).First(&after).Error; err != nil {
		t.Fatalf("find offering after: %v", err)
	}
	if after.CourseId != targetCard.Id {
		t.Errorf("offering course_id = %d, want %d（merged 目标卡）；物化把 offering 写回了 hidden 旧卡", after.CourseId, targetCard.Id)
	}
	var oldAfter course.Entity
	if err := conn.First(&oldAfter, oldCard.Id).Error; err != nil {
		t.Fatalf("load old card: %v", err)
	}
	if oldAfter.Status != course.StatusHidden {
		t.Errorf("old card status = %d, want hidden（物化不得复活已合并旧卡）", oldAfter.Status)
	}
}

// TestMaterializeFromPkTeacherCodeKeyedDedupe 回归 review：教师按 teacher_code（身份主锚）
// 键控去重。同一工号跨课程只物化一个教师行，二次运行零新建。
func TestMaterializeFromPkTeacherCodeKeyedDedupe(t *testing.T) {
	migrateMaterializeTables(t)
	conn := db.Connect()
	credit := 3.0
	for i, code := range []string{"A001", "B002"} {
		if err := conn.Create(&pk.CourseDetailEntity{
			Id: uint64(1 + i), CalendarId: 1, CourseCode: code,
			CourseName: "课程" + code, Code: code + "01",
			NewCourseCode: code + "N", NewCode: code + "N01",
			Credit: &credit, Faculty: "数学科学学院",
		}).Error; err != nil {
			t.Fatalf("seed course detail %s: %v", code, err)
		}
		if err := conn.Create(&pk.TeacherEntity{
			Id: uint64(100 + i), TeachingClassId: uint64(1 + i),
			TeacherCode: "T001", TeacherName: "张三",
		}).Error; err != nil {
			t.Fatalf("seed teacher: %v", err)
		}
	}
	if err := conn.Create(&pk.CalendarEntity{CalendarId: 1}).Error; err != nil {
		t.Fatalf("seed calendar: %v", err)
	}
	if err := conn.Create(&pk.FacultyEntity{Faculty: "数学科学学院", FacultyI18n: "数学科学学院"}).Error; err != nil {
		t.Fatalf("seed faculty: %v", err)
	}

	first, err := MaterializeFromPk(context.Background(), []uint64{1})
	if err != nil {
		t.Fatalf("first materialize: %v", err)
	}
	if first.InstructorsInserted != 1 {
		t.Fatalf("instructorsInserted = %d, want 1（同 teacher_code 只建一个教师行）", first.InstructorsInserted)
	}
	if first.CoursesInserted != 2 {
		t.Fatalf("coursesInserted = %d, want 2", first.CoursesInserted)
	}
	var instructors []course.InstructorEntity
	if err := conn.Find(&instructors).Error; err != nil {
		t.Fatalf("find instructors: %v", err)
	}
	if len(instructors) != 1 || instructors[0].TeacherCode != "T001" {
		t.Fatalf("instructors = %+v, want single row with teacher_code T001", instructors)
	}
	var courses []course.Entity
	if err := conn.Where("primary_code IN ?", []string{"A001", "B002"}).Find(&courses).Error; err != nil {
		t.Fatalf("find courses: %v", err)
	}
	for _, c := range courses {
		if c.TeacherId != instructors[0].Id {
			t.Fatalf("course %s teacher_id = %d, want %d", c.PrimaryCode, c.TeacherId, instructors[0].Id)
		}
	}
	second, err := MaterializeFromPk(context.Background(), []uint64{1})
	if err != nil {
		t.Fatalf("second materialize: %v", err)
	}
	if second.InstructorsInserted != 0 {
		t.Errorf("second instructorsInserted = %d, want 0（code 键命中缓存/查询）", second.InstructorsInserted)
	}
}

// TestMaterializeFromPkMissingCalendarReturnsError 回归 review：term 解析错误不再静默吞掉。
// calendarId > 0 但 pk_calendar 无该行时整个物化事务必须失败回滚（此前静默写 term_id=0）。
func TestMaterializeFromPkMissingCalendarReturnsError(t *testing.T) {
	migrateMaterializeTables(t)
	conn := db.Connect()
	credit := 3.0
	if err := conn.Create(&pk.CourseDetailEntity{
		Id: 1, CalendarId: 99, CourseCode: "C101", CourseName: "缺失学期课程", Code: "C10101",
		NewCourseCode: "C101N", NewCode: "C101N01", Credit: &credit, Faculty: "数学科学学院",
	}).Error; err != nil {
		t.Fatalf("seed course detail: %v", err)
	}
	if err := conn.Create(&pk.TeacherEntity{Id: 100, TeachingClassId: 1, TeacherCode: "T001", TeacherName: "张三"}).Error; err != nil {
		t.Fatalf("seed teacher: %v", err)
	}
	_, err := MaterializeFromPk(context.Background(), []uint64{99})
	if err == nil {
		t.Fatal("expected error for missing calendar row")
	}
	if !strings.Contains(err.Error(), "lookup calendar") {
		t.Fatalf("unexpected error: %v", err)
	}
	// 整个物化事务回滚：课程卡与 offering 都不落库。
	var courseCount int64
	if err := conn.Model(&course.Entity{}).Count(&courseCount).Error; err != nil {
		t.Fatalf("count courses: %v", err)
	}
	if courseCount != 0 {
		t.Errorf("courses = %d, want 0（事务应整体回滚）", courseCount)
	}
	var offeringCount int64
	if err := conn.Model(&course.OfferingEntity{}).Count(&offeringCount).Error; err != nil {
		t.Fatalf("count offerings: %v", err)
	}
	if offeringCount != 0 {
		t.Errorf("offerings = %d, want 0", offeringCount)
	}
}

// TestMaterializeFromPkEmptyCalendarI18nKeepsZeroTerm calendar 存在但无学期码（i18n 空）是
// 合法"无学期码"情形：物化成功且 term_id=0，不报错也不创建 course_term。
func TestMaterializeFromPkEmptyCalendarI18nKeepsZeroTerm(t *testing.T) {
	migrateMaterializeTables(t)
	seedPkForMaterialize(t)
	conn := db.Connect()
	report, err := MaterializeFromPk(context.Background(), []uint64{1})
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if report.OfferingsInserted != 1 {
		t.Fatalf("offeringsInserted = %d, want 1", report.OfferingsInserted)
	}
	var offering course.OfferingEntity
	if err := conn.Where("teaching_class_id = ?", 1).First(&offering).Error; err != nil {
		t.Fatalf("find offering: %v", err)
	}
	if offering.TermId != 0 {
		t.Errorf("term_id = %d, want 0（calendar 无学期码）", offering.TermId)
	}
	var termCount int64
	if err := conn.Model(&course.TermEntity{}).Count(&termCount).Error; err != nil {
		t.Fatalf("count terms: %v", err)
	}
	if termCount != 0 {
		t.Errorf("terms = %d, want 0", termCount)
	}
}

// TestImportReusesMaterializedOffering 回归 review：导入包无 offering source_ref 但
// teaching_class_id 已由物化链先行写入时，导入必须认领复用物化行（更新非 status 字段 +
// 写 source_ref），不得插入第二行撞 uniq_course_offering_teaching_class 唯一索引。
func TestImportReusesMaterializedOffering(t *testing.T) {
	migrateMaterializeTables(t)
	seedPkForMaterialize(t)
	conn := db.Connect()
	if _, err := MaterializeFromPk(context.Background(), []uint64{1}); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	var materializedOffering course.OfferingEntity
	if err := conn.Where("teaching_class_id = ?", 1).First(&materializedOffering).Error; err != nil {
		t.Fatalf("find materialized offering: %v", err)
	}

	manifestPath := writeManifestFixture(t, map[string]string{
		"courses.jsonl":     `{"id":"c1","code":"A001","name":"高等数学(A)上","department":"数学科学学院","credit":5,"teacher_code":"i1"}` + "\n",
		"instructors.jsonl": `{"id":"i1","name":"张三","department":"数学科学学院"}` + "\n",
		"offerings.jsonl":   `{"id":"o1","course_id":"c1","term":"2025-2026-1","campus":"四平路校区","faculty":"数学科学学院","teaching_class_id":"1","class_code":"A00101","class_name":"01班","instructor_ids":["i1"]}` + "\n",
	})
	report, err := ImportCatalog(context.Background(), manifestPath, false)
	if err != nil {
		t.Fatalf("import catalog: %v", err)
	}
	if report.Updated != 3 {
		t.Fatalf("updated = %d, want 3（instructor+course+offering 认领复用）", report.Updated)
	}
	if report.Inserted != 0 {
		t.Fatalf("inserted = %d, want 0（复用物化行，不插入新行）", report.Inserted)
	}
	// 只有物化链那一行 offering（唯一索引未被撞）。
	var offerings []course.OfferingEntity
	if err := conn.Find(&offerings).Error; err != nil {
		t.Fatalf("find offerings: %v", err)
	}
	if len(offerings) != 1 {
		t.Fatalf("offerings = %d, want 1（导入不得创建同 teaching_class_id 的第二行）", len(offerings))
	}
	if offerings[0].Id != materializedOffering.Id {
		t.Fatalf("offering id = %d, want %d（应复用物化行）", offerings[0].Id, materializedOffering.Id)
	}
	// 非 status 字段被导入行刷新；status 未被导入触碰（物化行可见）。
	if offerings[0].Campus != "四平路校区" || offerings[0].ClassCode != "A00101" || offerings[0].ClassName != "01班" {
		t.Errorf("offering 字段未按导入行更新: %+v", offerings[0])
	}
	if offerings[0].Status != course.OfferingStatusVisible {
		t.Errorf("status = %d, want visible", offerings[0].Status)
	}
	// source_ref 认领指向物化行。
	var ref course.SourceRefEntity
	if err := conn.Where("source = ? AND entity_type = ? AND external_id = ?", "test-fixture", course.EntityTypeOffering, "o1").
		First(&ref).Error; err != nil {
		t.Fatalf("find offering source ref: %v", err)
	}
	if ref.LocalId != materializedOffering.Id {
		t.Errorf("source_ref local_id = %d, want %d", ref.LocalId, materializedOffering.Id)
	}
}

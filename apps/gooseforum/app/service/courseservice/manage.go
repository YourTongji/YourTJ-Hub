package courseservice

import (
	"errors"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/searchservice"
	"gorm.io/gorm"
)

// 管理端课程操作稳定 sentinel：控制器据此映射语义 HTTP 状态。
var (
	// ErrCourseCodeRequired 新增课程缺少主课号。
	ErrCourseCodeRequired = errors.New("course primary code required")
	// ErrCourseNameRequired 新增课程缺少课程名。
	ErrCourseNameRequired = errors.New("course name required")
	// ErrCourseCodeConflict 主课号已被其它课程占用。
	ErrCourseCodeConflict = errors.New("course primary code already exists")
	// ErrCourseCreditInvalid 学分越界（credit_x10 必须 >= 0）。
	ErrCourseCreditInvalid = errors.New("course credit must be non-negative")
)

// AdminCourseItem 管理端课程列表项：含隐藏课程与统计投影。
type AdminCourseItem struct {
	Id          uint64   `json:"id"`
	PrimaryCode string   `json:"primaryCode"`
	Name        string   `json:"name"`
	Department  string   `json:"department"`
	CreditX10   int      `json:"creditX10"`
	Status      int8     `json:"status"` // 0 可见 / 1 隐藏
	Aliases     []string `json:"aliases"`
	Instructors []string `json:"instructors"`
	ReviewCount int      `json:"reviewCount"`
	RatingAvg   *float64 `json:"ratingAvg,omitempty"`
	CreatedAt   string   `json:"createdAt"`
}

// AdminCourseQuery 管理端课程检索条件。
type AdminCourseQuery struct {
	Keyword    string
	Department string
	Page       int
	Size       int
}

// AdminCoursePage 管理端课程分页结果。
type AdminCoursePage struct {
	List    []AdminCourseItem `json:"list"`
	Page    int               `json:"page"`
	Size    int               `json:"size"`
	Total   int64             `json:"total"`
	HasNext bool              `json:"hasNext"`
}

// CourseCreateInput 新增课程请求。
type CourseCreateInput struct {
	PrimaryCode string   `json:"primaryCode"`
	Name        string   `json:"name"`
	Department  string   `json:"department"`
	CreditX10   int      `json:"creditX10"`
	Aliases     []string `json:"aliases"`
	Instructors []string `json:"instructors"`
}

// CourseUpdateInput 编辑课程请求：指针区分“缺省（保留原值）”与“显式置空”。
type CourseUpdateInput struct {
	PrimaryCode *string   `json:"primaryCode"`
	Name        *string   `json:"name"`
	Department  *string   `json:"department"`
	CreditX10   *int      `json:"creditX10"`
	ReviewScope *string   `json:"reviewScope"`
	TeamKey     *string   `json:"teamKey"`
	Aliases     *[]string `json:"aliases"`
	Instructors *[]string `json:"instructors"`
}

// ValidateReviewScope 校验 review_scope 取值（teacher/team/course），返回归一化值或错误。
func ValidateReviewScope(v string) (string, error) {
	switch v {
	case ReviewScopeTeacher, ReviewScopeTeam, ReviewScopeCourse:
		return v, nil
	default:
		return "", ErrReviewScopeInvalid
	}
}

// DeletedCourseInfo 级联删除后的课程快照（供控制器写审计日志）。
type DeletedCourseInfo struct {
	Id            uint64 `json:"id"`
	PrimaryCode   string `json:"primaryCode"`
	Name          string `json:"name"`
	OfferingCount int    `json:"offeringCount"`
	ReviewCount   int    `json:"reviewCount"`
}

// AdminCourseList 返回管理端课程分页（含隐藏课程）。
func AdminCourseList(q AdminCourseQuery) (AdminCoursePage, error) {
	page := q.Page
	if page <= 0 {
		page = 1
	}
	size := q.Size
	if size <= 0 {
		size = 20
	}
	if size > 50 {
		size = 50
	}
	entities, total, err := course.ListCourses(course.ListCourseQuery{
		Keyword:       Normalize(q.Keyword),
		Department:    normalizeMulti([]string{q.Department}),
		Page:          page,
		Size:          size,
		IncludeHidden: true,
	})
	if err != nil {
		return AdminCoursePage{}, err
	}
	if len(entities) == 0 {
		return AdminCoursePage{List: []AdminCourseItem{}, Page: page, Size: size, Total: total}, nil
	}
	items, err := buildAdminCourseItems(entities)
	if err != nil {
		return AdminCoursePage{}, err
	}
	return AdminCoursePage{
		List:    items,
		Page:    page,
		Size:    size,
		Total:   total,
		HasNext: int64(page)*int64(size) < total,
	}, nil
}

// buildAdminCourseItems 复用目录 summary 构建，再合并 status 与统计投影。
func buildAdminCourseItems(entities []course.Entity) ([]AdminCourseItem, error) {
	summaries, err := buildSummaries(entities)
	if err != nil {
		return nil, err
	}
	ids := make([]uint64, 0, len(entities))
	for _, e := range entities {
		ids = append(ids, e.Id)
	}
	stats, err := course.GetCourseStatsMap(ids)
	if err != nil {
		return nil, err
	}
	items := make([]AdminCourseItem, 0, len(entities))
	for i, e := range entities {
		item := AdminCourseItem{
			Id:          e.Id,
			PrimaryCode: e.PrimaryCode,
			Name:        e.Name,
			Department:  e.Department,
			CreditX10:   e.CreditX10,
			Status:      e.Status,
			Aliases:     summaries[i].Aliases,
			Instructors: summaries[i].Instructors,
			CreatedAt:   e.CreatedAt.Format(time.RFC3339),
		}
		if st, ok := stats[e.Id]; ok {
			item.ReviewCount = st.ReviewCount
			if st.RatingCount > 0 {
				avg := float64(st.RatingSum) / float64(st.RatingCount)
				item.RatingAvg = &avg
			}
		}
		items = append(items, item)
	}
	return items, nil
}

// CreateCourse 新增 canonical 课程：创建实体 + 别名 + 教师，同事务入队搜索同步。
// 手动新增的课程暂不含 offering（offering 由 course-import CLI 灌入），评价须在导入 offering 后产生。
func CreateCourse(input CourseCreateInput) (AdminCourseItem, error) {
	code := strings.TrimSpace(input.PrimaryCode)
	name := strings.TrimSpace(input.Name)
	if code == "" {
		return AdminCourseItem{}, ErrCourseCodeRequired
	}
	if name == "" {
		return AdminCourseItem{}, ErrCourseNameRequired
	}
	if input.CreditX10 < 0 {
		return AdminCourseItem{}, ErrCourseCreditInvalid
	}
	department := strings.TrimSpace(input.Department)
	var item AdminCourseItem
	err := dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		// 手动新增的课程无教师（teacher_id=0），冲突检查按 (code, 无教师) 复合身份。
		existing, err := course.GetCourseByCodeTeacherTx(tx, code, 0)
		if err == nil && existing.Id > 0 {
			return ErrCourseCodeConflict
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		pinyin, initials := searchservice.PinyinFields(name)
		entity := course.Entity{
			PrimaryCode:    code,
			Name:           name,
			Department:     department,
			CreditX10:      input.CreditX10,
			NormalizedName: Normalize(name),
			NamePinyin:     pinyin,
			NameInitials:   initials,
			Status:         course.StatusVisible,
		}
		if err := tx.Model(&course.Entity{}).Create(&entity).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return ErrCourseCodeConflict
			}
			return err
		}
		if err := reconcileCourseAliasesTx(tx, entity.Id, input.Aliases); err != nil {
			return err
		}
		if err := reconcileCourseInstructorsTx(tx, entity.Id, department, input.Instructors); err != nil {
			return err
		}
		if err := searchservice.EnqueueCourseSearchTask(tx, entity.Id); err != nil {
			return err
		}
		item = AdminCourseItem{
			Id:          entity.Id,
			PrimaryCode: entity.PrimaryCode,
			Name:        entity.Name,
			Department:  entity.Department,
			CreditX10:   entity.CreditX10,
			Status:      entity.Status,
			Aliases:     cleanStrings(input.Aliases),
			Instructors: cleanStrings(input.Instructors),
			CreatedAt:   entity.CreatedAt.Format(time.RFC3339),
		}
		return nil
	})
	if err != nil {
		return AdminCourseItem{}, err
	}
	return item, nil
}

// UpdateCourse 编辑课程（PATCH 部分更新）：改名同步 normalized/pinyin/initials 并同事务入队搜索同步。
// 更新路径不写 status——管理员隐藏的课程不会被编辑静默复活（与 importer 语义一致）。
func UpdateCourse(courseId uint64, input CourseUpdateInput) (AdminCourseItem, error) {
	if input.CreditX10 != nil && *input.CreditX10 < 0 {
		return AdminCourseItem{}, ErrCourseCreditInvalid
	}
	err := dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		entity := course.GetCourseByIdTx(tx, courseId)
		if entity.Id == 0 {
			return ErrCourseNotFound
		}
		updates := map[string]any{}
		if input.PrimaryCode != nil {
			code := strings.TrimSpace(*input.PrimaryCode)
			if code == "" {
				return ErrCourseCodeRequired
			}
			// 改课号冲突检查按 (新 code, 当前教师) 复合身份。
			existing, err := course.GetCourseByCodeTeacherTx(tx, code, entity.TeacherId)
			if err == nil && existing.Id > 0 && existing.Id != courseId {
				return ErrCourseCodeConflict
			}
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			updates["primary_code"] = code
		}
		if input.Name != nil {
			name := strings.TrimSpace(*input.Name)
			if name == "" {
				return ErrCourseNameRequired
			}
			pinyin, initials := searchservice.PinyinFields(name)
			updates["name"] = name
			updates["normalized_name"] = Normalize(name)
			updates["name_pinyin"] = pinyin
			updates["name_initials"] = initials
		}
		if input.Department != nil {
			updates["department"] = strings.TrimSpace(*input.Department)
		}
		if input.CreditX10 != nil {
			updates["credit_x10"] = *input.CreditX10
		}
		if input.ReviewScope != nil {
			scope, err := ValidateReviewScope(strings.TrimSpace(*input.ReviewScope))
			if err != nil {
				return err
			}
			updates["review_scope"] = scope
		}
		if input.TeamKey != nil {
			updates["team_key"] = strings.TrimSpace(*input.TeamKey)
		}
		if len(updates) > 0 {
			if err := tx.Model(&course.Entity{}).Where("id = ?", courseId).Updates(updates).Error; err != nil {
				if errors.Is(err, gorm.ErrDuplicatedKey) {
					return ErrCourseCodeConflict
				}
				return err
			}
		}
		if input.Aliases != nil {
			if err := reconcileCourseAliasesTx(tx, courseId, *input.Aliases); err != nil {
				return err
			}
		}
		if input.Instructors != nil {
			department := entity.Department
			if input.Department != nil {
				department = strings.TrimSpace(*input.Department)
			}
			if err := reconcileCourseInstructorsTx(tx, courseId, department, *input.Instructors); err != nil {
				return err
			}
		}
		// 编辑路径同事务入队搜索同步（改名/改别名/改教师都会改变搜索文档）。
		return searchservice.EnqueueCourseSearchTask(tx, courseId)
	})
	if err != nil {
		return AdminCourseItem{}, err
	}
	// 事务提交后再回填别名/教师/统计：adminCourseItemForSingle 走独立连接，
	// 在 SQLite 单连接测试环境下事务内调用会死锁（与 CreateReview 回填作者名同理）。
	return adminCourseItemForSingle(course.GetCourse(courseId)), nil
}

// adminCourseItemForSingle 读取单门课程的别名/教师/统计构造管理端列表项。
func adminCourseItemForSingle(entity course.Entity) AdminCourseItem {
	aliases, _ := course.ListAliasesByCourse(entity.Id)
	aliasValues := make([]string, 0, len(aliases))
	for _, a := range aliases {
		aliasValues = append(aliasValues, a.Value)
	}
	summaries, _ := buildSummaries([]course.Entity{entity})
	instructors := []string{}
	if len(summaries) > 0 {
		instructors = summaries[0].Instructors
	}
	item := AdminCourseItem{
		Id:          entity.Id,
		PrimaryCode: entity.PrimaryCode,
		Name:        entity.Name,
		Department:  entity.Department,
		CreditX10:   entity.CreditX10,
		Status:      entity.Status,
		Aliases:     aliasValues,
		Instructors: instructors,
		CreatedAt:   entity.CreatedAt.Format(time.RFC3339),
	}
	if st, err := course.GetCourseStats(entity.Id); err == nil {
		item.ReviewCount = st.ReviewCount
		if st.RatingCount > 0 {
			avg := float64(st.RatingSum) / float64(st.RatingCount)
			item.RatingAvg = &avg
		}
	}
	return item
}

// DeleteCourse 级联删除课程：物理删除课程 + offerings + 教师关联 + 别名 + 评价 + helpful + 统计投影 + 来源映射，
// 同事务入队搜索删除任务（worker 因 GetCourse 未命中而删除索引文档）。
func DeleteCourse(courseId uint64) (DeletedCourseInfo, error) {
	var info DeletedCourseInfo
	err := dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		entity := course.GetCourseByIdTx(tx, courseId)
		if entity.Id == 0 {
			return ErrCourseNotFound
		}
		offeringIds, err := course.ListOfferingIdsByCourseAllTx(tx, courseId)
		if err != nil {
			return err
		}
		var reviewCount int64
		if len(offeringIds) > 0 {
			// 评价关联的 helpful/dislike 标记先清理（物理删除，避免悬挂）。
			if err := tx.Unscoped().Table((&course.HelpfulEntity{}).TableName()).
				Where("review_id IN (SELECT id FROM course_review WHERE offering_id IN ? AND deleted_at IS NULL)", offeringIds).
				Delete(&course.HelpfulEntity{}).Error; err != nil {
				return err
			}
			if err := tx.Unscoped().Table((&course.DislikeEntity{}).TableName()).
				Where("review_id IN (SELECT id FROM course_review WHERE offering_id IN ? AND deleted_at IS NULL)", offeringIds).
				Delete(&course.DislikeEntity{}).Error; err != nil {
				return err
			}
			if err := tx.Model(&course.ReviewEntity{}).
				Where("offering_id IN ?", offeringIds).
				Count(&reviewCount).Error; err != nil {
				return err
			}
			// 评价物理删除（状态机用 status 表达隔离窗口，物理删除彻底移除）。
			// 先清理 review 级来源映射（否则重导 checksum 相同会静默不重建评价）。
			if err := tx.Unscoped().Table((&course.SourceRefEntity{}).TableName()).
				Where("entity_type = ? AND local_id IN (SELECT id FROM course_review WHERE offering_id IN ?)", course.EntityTypeReview, offeringIds).
				Delete(&course.SourceRefEntity{}).Error; err != nil {
				return err
			}
			if err := tx.Unscoped().Table((&course.ReviewEntity{}).TableName()).
				Where("offering_id IN ?", offeringIds).
				Delete(&course.ReviewEntity{}).Error; err != nil {
				return err
			}
			// offering 级统计投影物理删除。
			if err := tx.Unscoped().Table((&course.OfferingStatsEntity{}).TableName()).
				Where("offering_id IN ?", offeringIds).
				Delete(&course.OfferingStatsEntity{}).Error; err != nil {
				return err
			}
			// 教师关联（无软删，直接物理删除）。
			if err := tx.Model(&course.OfferingInstructorEntity{}).
				Where("offering_id IN ?", offeringIds).
				Delete(&course.OfferingInstructorEntity{}).Error; err != nil {
				return err
			}
			// offering 物理删除。
			if err := tx.Unscoped().Table((&course.OfferingEntity{}).TableName()).
				Where("id IN ?", offeringIds).
				Delete(&course.OfferingEntity{}).Error; err != nil {
				return err
			}
			// 来源映射清理（否则重导时 source_ref 指向已删除实体，无法重建）。
			if err := tx.Unscoped().Table((&course.SourceRefEntity{}).TableName()).
				Where("entity_type = ? AND local_id IN ?", course.EntityTypeOffering, offeringIds).
				Delete(&course.SourceRefEntity{}).Error; err != nil {
				return err
			}
		}
		// 课程级统计投影物理删除。
		if err := tx.Unscoped().Table((&course.CourseStatsEntity{}).TableName()).
			Where("course_id = ?", courseId).
			Delete(&course.CourseStatsEntity{}).Error; err != nil {
			return err
		}
		// 别名物理删除（释放 (kind, normalized_value) 唯一索引）。
		if err := tx.Unscoped().Table((&course.AliasEntity{}).TableName()).
			Where("course_id = ?", courseId).
			Delete(&course.AliasEntity{}).Error; err != nil {
			return err
		}
		// 课程来源映射清理。
		if err := tx.Unscoped().Table((&course.SourceRefEntity{}).TableName()).
			Where("entity_type = ? AND local_id = ?", course.EntityTypeCourse, courseId).
			Delete(&course.SourceRefEntity{}).Error; err != nil {
			return err
		}
		// 课程物理删除。
		if err := tx.Unscoped().Table((&course.Entity{}).TableName()).
			Where("id = ?", courseId).
			Delete(&course.Entity{}).Error; err != nil {
			return err
		}
		// 同事务入队搜索删除（worker 删除索引文档）。
		if err := searchservice.EnqueueCourseSearchTask(tx, courseId); err != nil {
			return err
		}
		info = DeletedCourseInfo{
			Id:            entity.Id,
			PrimaryCode:   entity.PrimaryCode,
			Name:          entity.Name,
			OfferingCount: len(offeringIds),
			ReviewCount:   int(reviewCount),
		}
		return nil
	})
	if err != nil {
		return DeletedCourseInfo{}, err
	}
	return info, nil
}

// reconcileCourseAliasesTx 事务内对齐课程的 admin 管理别名（kind=name，source=admin）。
// import 管理的别名（source=外部 ID）不会被删除；admin 别名可增删，冲突（别名已属其它课程）时跳过。
func reconcileCourseAliasesTx(tx *gorm.DB, courseId uint64, aliases []string) error {
	normSet := make(map[string]struct{}, len(aliases))
	for _, a := range aliases {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		norm := Normalize(a)
		if norm == "" {
			continue
		}
		normSet[norm] = struct{}{}
		_, err := course.GetAliasByNormalizedValueTx(tx, course.AliasKindName, norm)
		switch {
		case err == nil:
			// 已存在（import 或 admin 管理）：不同课程则跳过（冲突），同课程保留。
			continue
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return err
		default:
			entity := course.AliasEntity{
				CourseId:        courseId,
				Kind:            course.AliasKindName,
				Value:           a,
				NormalizedValue: norm,
				Source:          "admin",
			}
			if err := tx.Model(&course.AliasEntity{}).Create(&entity).Error; err != nil {
				if !errors.Is(err, gorm.ErrDuplicatedKey) {
					return err
				}
			}
		}
	}
	reconcile := tx.Unscoped().Model(&course.AliasEntity{}).
		Where("course_id = ? AND kind = ? AND source = ?", courseId, course.AliasKindName, "admin")
	if len(normSet) > 0 {
		keys := make([]string, 0, len(normSet))
		for k := range normSet {
			keys = append(keys, k)
		}
		reconcile = reconcile.Where("normalized_value NOT IN ?", keys)
	}
	return reconcile.Delete(&course.AliasEntity{}).Error
}

// reconcileCourseInstructorsTx 事务内把课程级教师名单对齐到 instructor 实体并挂到可见 offering。
// 手动编辑“教师”语义为课程级名单：对课程全部 offering 统一替换，保证课程搜索文档 instructors 一致；
// 每学期差异化教师由 course-import 在下次导入时按 manifest 重新断言。
func reconcileCourseInstructorsTx(tx *gorm.DB, courseId uint64, department string, names []string) error {
	instructorIDs := make([]uint64, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		ins, err := findOrCreateInstructorTx(tx, name, department)
		if err != nil {
			return err
		}
		instructorIDs = append(instructorIDs, ins.Id)
	}
	offeringIds, err := course.ListOfferingIdsByCourseAllTx(tx, courseId)
	if err != nil {
		return err
	}
	for _, oid := range offeringIds {
		if err := replaceOfferingInstructorsTx(tx, oid, instructorIDs); err != nil {
			return err
		}
	}
	return nil
}

// findOrCreateInstructorTx 事务内按 (normalized_name, department) 查找教师，不存在则创建。
func findOrCreateInstructorTx(tx *gorm.DB, name, department string) (course.InstructorEntity, error) {
	norm := Normalize(name)
	existing, err := course.FindInstructorByNameDeptTx(tx, norm, department)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return existing, err
	}
	pinyin, initials := searchservice.PinyinFields(name)
	entity := course.InstructorEntity{
		Name:           name,
		NormalizedName: norm,
		NamePinyin:     pinyin,
		NameInitials:   initials,
		Department:     department,
		Status:         0,
	}
	if err := tx.Model(&course.InstructorEntity{}).Create(&entity).Error; err != nil {
		return existing, err
	}
	return entity, nil
}

// cleanStrings 去空白并去空项，保证空 slice 序列化为 [] 而非 null。
func cleanStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	if out == nil {
		out = []string{}
	}
	return out
}

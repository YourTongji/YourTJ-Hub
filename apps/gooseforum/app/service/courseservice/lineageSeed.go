package courseservice

import (
	"context"
	"fmt"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice/lineage"
	"gorm.io/gorm"
)

// SeedLineageOptions course-lineage-seed 落库选项。
// 默认（全部 false）为 dry-run：只装配卡级候选并报告数量，不写库。
type SeedLineageOptions struct {
	// Write 为 true 时把 EQUIVALENT 候选写入 course_relations（status=pending，source=rule，
	// 含证据快照），供管理端「课程沿革」审核面板人工确认合并。
	Write bool
	// WriteFamily 为 true 时额外写入 SPLIT_FROM / RELATED 候选（家族变体/学分巨变标注），
	// 经管理端 approve 后在详情页沿革区块展示；默认只报告不落库。
	WriteFamily bool
}

// SeedLineageReport course-lineage-seed 装配/落库报告。
type SeedLineageReport struct {
	CardsLoaded       int `json:"cardsLoaded"`       // 参与装配的可见课程卡数
	EquivCandidates   int `json:"equivCandidates"`   // E1 EQUIVALENT 候选数
	SplitCandidates   int `json:"splitCandidates"`   // E2 SPLIT_FROM 候选数
	RelatedCandidates int `json:"relatedCandidates"` // E3 RELATED 候选数
	EquivInserted     int `json:"equivInserted"`     // 本次新写入 EQUIVALENT pending 行数
	EquivSkipped      int `json:"equivSkipped"`      // 已存在（同 from/to/type）跳过的 EQUIVALENT 数
	FamilyInserted    int `json:"familyInserted"`    // 本次新写入 SPLIT/RELATED pending 行数
	FamilySkipped     int `json:"familySkipped"`     // 已存在跳过的 SPLIT/RELATED 数
}

// courseBatch 卡级种子装配的分页/批量步长（分块查询避免 IN 超限，与搜索重建同风格）。
const courseBatch = 500

// SeedLineage 装配课程目录全部可见课程卡为卡级沿革输入，运行 lineage.EvaluateCards，
// 按选项把候选写入 course_relations（幂等，同 (from,to,type) 已存在则跳过）。
//
// 只读路径（dry-run）无任何副作用；写路径在单个事务内完成，候选先查重再插入，
// 中途失败整体回滚，不产生半套 pending 候选（对齐 course-materialize 先例）。
// 返回报告、候选与装配摘要（CLI 渲染 from/to 名称用）。
func SeedLineage(_ context.Context, opt SeedLineageOptions) (*SeedLineageReport, []lineage.CardCandidate, []lineage.CardSummary, error) {
	summaries, err := loadCardSummaries()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("course-lineage-seed: 装配课程卡: %w", err)
	}
	candidates := lineage.EvaluateCards(summaries)
	report := &SeedLineageReport{CardsLoaded: len(summaries)}
	for _, c := range candidates {
		switch c.RelationType {
		case lineage.RelationEquivalent:
			report.EquivCandidates++
		case lineage.RelationSplitFrom:
			report.SplitCandidates++
		case lineage.RelationRelated:
			report.RelatedCandidates++
		}
	}

	if opt.Write || opt.WriteFamily {
		if err := seedWriteRelations(candidates, opt, report); err != nil {
			return nil, nil, nil, fmt.Errorf("course-lineage-seed: 写入候选: %w", err)
		}
	}
	return report, candidates, summaries, nil
}

// loadCardSummaries 分页读取全部可见课程卡，装配卡级沿革输入：
//   - offering（可见）经 teaching_class_id 关联 pk_course_detail，取一系统
//     course_code / new_course_code 作为冗余卡共享课程码证据；
//   - term_id → course_term.code 作为开课学期证据；
//   - teacher_id → course_instructor.teacher_code/name 作为同教师配对键。
//
// 只装配可见卡（status=visible，软删由 ListAllCourses 模型过滤）：隐藏卡已被
// 合并或下架，不应再成为候选来源或目标。
func loadCardSummaries() ([]lineage.CardSummary, error) {
	var entities []course.Entity
	offset := 0
	for {
		batch, err := course.ListAllCourses(courseBatch, offset)
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		for _, e := range batch {
			if e.Status == course.StatusVisible {
				entities = append(entities, e)
			}
		}
		if len(batch) < courseBatch {
			break
		}
		offset += courseBatch
	}
	if len(entities) == 0 {
		return nil, nil
	}

	ids := make([]uint64, 0, len(entities))
	for _, e := range entities {
		ids = append(ids, e.Id)
	}

	// offering（仅可见）→ 按 course 分组；收集 teaching_class_id / term_id。
	offerings, err := course.ListOfferingsByCourses(ids)
	if err != nil {
		return nil, fmt.Errorf("读取开课实例: %w", err)
	}
	offerByCourse := make(map[uint64][]course.OfferingEntity, len(entities))
	var classIDs, termIDs []uint64
	seenClass, seenTerm := map[uint64]bool{}, map[uint64]bool{}
	for _, o := range offerings {
		offerByCourse[o.CourseId] = append(offerByCourse[o.CourseId], o)
		if o.TeachingClassId > 0 && !seenClass[o.TeachingClassId] {
			seenClass[o.TeachingClassId] = true
			classIDs = append(classIDs, o.TeachingClassId)
		}
		if o.TermId > 0 && !seenTerm[o.TermId] {
			seenTerm[o.TermId] = true
			termIDs = append(termIDs, o.TermId)
		}
	}

	// 学期码映射。
	termByID := map[uint64]string{}
	if len(termIDs) > 0 {
		terms, err := course.ListTermsByIDs(termIDs)
		if err != nil {
			return nil, fmt.Errorf("读取学期: %w", err)
		}
		for _, t := range terms {
			termByID[t.Id] = t.Code
		}
	}

	// 一系统课程码映射（教学班 → course_code / new_course_code）。
	pkByID := map[uint64]pk.CourseDetailEntity{}
	if len(classIDs) > 0 {
		details, err := pk.ListCourseDetailsByIDs(classIDs)
		if err != nil {
			return nil, fmt.Errorf("读取一系统教学班: %w", err)
		}
		for _, d := range details {
			pkByID[d.Id] = d
		}
	}

	// 教师映射（teacher_id → teacher_code/name）。
	instructorByID := map[uint64]course.InstructorEntity{}
	var teacherIDs []uint64
	seenTeacher := map[uint64]bool{}
	for _, e := range entities {
		if e.TeacherId != 0 && !seenTeacher[e.TeacherId] {
			seenTeacher[e.TeacherId] = true
			teacherIDs = append(teacherIDs, e.TeacherId)
		}
	}
	if len(teacherIDs) > 0 {
		instructors, err := course.ListInstructorsByIDs(teacherIDs)
		if err != nil {
			return nil, fmt.Errorf("读取教师: %w", err)
		}
		for _, ins := range instructors {
			instructorByID[ins.Id] = ins
		}
	}

	summaries := make([]lineage.CardSummary, 0, len(entities))
	for _, e := range entities {
		s := lineage.CardSummary{
			ID:          e.Id,
			PrimaryCode: e.PrimaryCode,
			Name:        e.Name,
			CreditX10:   e.CreditX10,
			CreatedAt:   e.CreatedAt,
		}
		if ins, ok := instructorByID[e.TeacherId]; ok {
			s.TeacherCode = ins.TeacherCode
			s.TeacherName = ins.Name
		}
		for _, o := range offerByCourse[e.Id] {
			if code, ok := termByID[o.TermId]; ok && code != "" {
				s.Terms = appendUniqueString(s.Terms, code)
			}
			if d, ok := pkByID[o.TeachingClassId]; ok {
				if d.CourseCode != "" {
					s.PkCourseCode = appendUniqueString(s.PkCourseCode, d.CourseCode)
				}
				if d.NewCourseCode != "" {
					s.PkNewCode = appendUniqueString(s.PkNewCode, d.NewCourseCode)
				}
			}
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}

// seedWriteRelations 事务内把候选写入 course_relations。EQUIVALENT 仅在 opt.Write
// 时写入；SPLIT_FROM/RELATED 仅在 opt.WriteFamily 时写入（双轨可控，避免未经
// 确认的展示标注污染线上）。CreateRelationTx 幂等：同 (from,to,type) 已存在
// （任意状态，含已 approved/ignored/merged）返回既有行，不复活、不重复。
func seedWriteRelations(candidates []lineage.CardCandidate, opt SeedLineageOptions, report *SeedLineageReport) error {
	return db.Connect().Transaction(func(tx *gorm.DB) error {
		for _, c := range candidates {
			writeFamily := c.RelationType == lineage.RelationSplitFrom || c.RelationType == lineage.RelationRelated
			if c.RelationType == lineage.RelationEquivalent && !opt.Write {
				continue
			}
			if writeFamily && !opt.WriteFamily {
				continue
			}
			entity := course.RelationEntity{
				FromCourseId: c.FromCardID,
				ToCourseId:   c.ToCardID,
				RelationType: c.RelationType,
				Source:       course.RelationSourceRule,
				Confidence:   c.Confidence,
				EvidenceJson: c.Evidence,
				Status:       string(course.RelationStatusPending),
			}
			existing, err := course.CreateRelationTx(tx, &entity)
			if err != nil {
				return fmt.Errorf("写入沿革候选 (%d→%d %s): %w", c.FromCardID, c.ToCardID, c.RelationType, err)
			}
			inserted := existing.Id == entity.Id
			if c.RelationType == lineage.RelationEquivalent {
				if inserted {
					report.EquivInserted++
				} else {
					report.EquivSkipped++
				}
			} else {
				if inserted {
					report.FamilyInserted++
				} else {
					report.FamilySkipped++
				}
			}
		}
		return nil
	})
}

// appendUniqueString 向切片追加非空且不重复的值。
func appendUniqueString(s []string, v string) []string {
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}

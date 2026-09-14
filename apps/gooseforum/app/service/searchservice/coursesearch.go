package searchservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/meiliconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/meilisearch/meilisearch-go"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

// CourseIndex is the Meilisearch index for canonical course documents.
const CourseIndex = "courses"

// CourseSearchDocument 课程搜索文档：一门课程只出现一次（canonical course）。
// 教师/学期/校区作为数组字段；不索引评价正文。
// (code, teacher) 复合身份模型下 TeacherId/TeacherName 为卡片身份教师
// （teacher_id=0 无教师时为空串），Instructors 保留 offering 级教师并集。
type CourseSearchDocument struct {
	ProjectionVersion int      `json:"_projectionVersion"`
	ID                uint64   `json:"id"`
	PrimaryCode       string   `json:"primaryCode"`
	Name              string   `json:"name"`
	NormalizedName    string   `json:"normalizedName"`
	NamePinyin        string   `json:"namePinyin"`
	NameInitials      string   `json:"nameInitials"`
	Department        string   `json:"department"`
	CreditX10         int      `json:"creditX10"`
	Aliases           []string `json:"aliases"`
	TeacherId         uint64   `json:"teacherId"`
	TeacherName       string   `json:"teacherName"`
	Instructors       []string `json:"instructors"`
	ClassCodes        []string `json:"classCodes"`
	InstructorSearch  []string `json:"instructorSearch"`
	Terms             []string `json:"terms"`
	Campus            []string `json:"campus"`
	Status            int8     `json:"status"`
	CreatedAt         int64    `json:"createdAt"`
	UpdatedAt         int64    `json:"updatedAt"`
}

// convertCourseToSearchDocument 从 canonical course + 关联构建搜索文档。
func convertCourseToSearchDocument(entity course.Entity) (CourseSearchDocument, error) {
	doc := CourseSearchDocument{
		ProjectionVersion: courseProjectionVersion,
		ID:                entity.Id,
		PrimaryCode:       entity.PrimaryCode,
		Name:              entity.Name,
		NormalizedName:    entity.NormalizedName,
		NamePinyin:        entity.NamePinyin,
		NameInitials:      entity.NameInitials,
		Department:        entity.Department,
		CreditX10:         entity.CreditX10,
		Aliases:           []string{},
		TeacherId:         entity.TeacherId,
		Instructors:       []string{},
		Terms:             []string{},
		Campus:            []string{},
		Status:            entity.Status,
		CreatedAt:         entity.CreatedAt.Unix(),
		UpdatedAt:         entity.UpdatedAt.Unix(),
	}
	if entity.TeacherId != 0 {
		if teachers, err := course.ListInstructorsByIDs([]uint64{entity.TeacherId}); err != nil {
			return doc, err
		} else if len(teachers) > 0 {
			doc.TeacherName = teachers[0].Name
			doc.InstructorSearch = append(doc.InstructorSearch, teachers[0].NormalizedName, teachers[0].NamePinyin, teachers[0].NameInitials)
		}
	}
	aliases, err := course.ListAliasesByCourse(entity.Id)
	if err != nil {
		return doc, err
	}
	for _, a := range aliases {
		doc.Aliases = append(doc.Aliases, a.Value)
	}
	offerings, err := course.ListOfferingsByCourse(entity.Id)
	if err != nil {
		return doc, err
	}
	if len(offerings) == 0 {
		return doc, nil
	}
	offeringIds := make([]uint64, 0, len(offerings))
	for _, o := range offerings {
		offeringIds = append(offeringIds, o.Id)
	}
	links, err := course.ListOfferingInstructorLinks(offeringIds)
	if err != nil {
		return doc, err
	}
	instructors, err := course.ListInstructorsByOfferings(offeringIds)
	if err != nil {
		return doc, err
	}
	instructorByID := make(map[uint64]string, len(instructors))
	for _, ins := range instructors {
		doc.InstructorSearch = append(doc.InstructorSearch, ins.NormalizedName, ins.NamePinyin, ins.NameInitials)
		instructorByID[ins.Id] = ins.Name
	}
	seenInstructors := make(map[string]struct{})
	for _, link := range links {
		if name, ok := instructorByID[link.InstructorId]; ok {
			if _, dup := seenInstructors[name]; !dup {
				seenInstructors[name] = struct{}{}
				doc.Instructors = append(doc.Instructors, name)
			}
		}
	}
	termIds := make([]uint64, 0, len(offerings))
	for _, o := range offerings {
		termIds = append(termIds, o.TermId)
	}
	terms, err := course.ListTermsByIDs(termIds)
	if err != nil {
		return doc, err
	}
	termByID := make(map[uint64]course.TermEntity, len(terms))
	for _, t := range terms {
		termByID[t.Id] = t
	}
	seenTerms := make(map[string]struct{})
	seenCampus := make(map[string]struct{})
	for _, o := range offerings {
		if o.ClassCode != "" {
			doc.ClassCodes = append(doc.ClassCodes, o.ClassCode)
		}
		if t, ok := termByID[o.TermId]; ok {
			if _, dup := seenTerms[t.Code]; !dup {
				seenTerms[t.Code] = struct{}{}
				doc.Terms = append(doc.Terms, t.Code)
			}
		}
		if o.Campus != "" {
			if _, dup := seenCampus[o.Campus]; !dup {
				seenCampus[o.Campus] = struct{}{}
				doc.Campus = append(doc.Campus, o.Campus)
			}
		}
	}
	return doc, nil
}

// convertCoursesToSearchDocuments 批量构建搜索文档（rebuild 用）。
// 关联数据（别名/开课/教师/学期）按批一次性查询，避免每门课 5 条 N+1
// （batch=200 时从 ~1000 次查询降到常数次）。
func convertCoursesToSearchDocuments(entities []course.Entity) ([]CourseSearchDocument, error) {
	docs := make([]CourseSearchDocument, 0, len(entities))
	if len(entities) == 0 {
		return docs, nil
	}
	courseIds := make([]uint64, 0, len(entities))
	for _, e := range entities {
		courseIds = append(courseIds, e.Id)
	}
	aliases, err := course.ListAliasesByCourses(courseIds)
	if err != nil {
		return nil, err
	}
	aliasByCourse := make(map[uint64][]string, len(courseIds))
	for _, a := range aliases {
		aliasByCourse[a.CourseId] = append(aliasByCourse[a.CourseId], a.Value)
	}
	offerings, err := course.ListOfferingsByCourses(courseIds)
	if err != nil {
		return nil, err
	}
	offeringByCourse := make(map[uint64][]course.OfferingEntity, len(courseIds))
	offeringIds := make([]uint64, 0, len(offerings))
	for _, o := range offerings {
		offeringByCourse[o.CourseId] = append(offeringByCourse[o.CourseId], o)
		offeringIds = append(offeringIds, o.Id)
	}
	instructorByID := make(map[uint64]string)
	instructorSearchByID := make(map[uint64][]string)
	instructorSearchByOffering := make(map[uint64][]string)
	instructorByOffering := make(map[uint64][]string, len(offeringIds))
	termByID := make(map[uint64]course.TermEntity)
	// 身份教师（course.teacher_id → 姓名）独立于 offering 解析：rebuild 批次里
	// 可能包含无可见 offering 的课程卡（如纯评价卡），此时 teacherName 仍必须
	// 填充，否则这些卡无法按教师搜索（与增量单卡转换不一致）。
	teacherNameByID := make(map[uint64]string)
	teacherIds := make([]uint64, 0, len(entities))
	for _, e := range entities {
		if e.TeacherId != 0 {
			teacherIds = append(teacherIds, e.TeacherId)
		}
	}
	if len(teacherIds) > 0 {
		teachers, err := course.ListInstructorsByIDs(teacherIds)
		if err != nil {
			return nil, err
		}
		for _, t := range teachers {
			teacherNameByID[t.Id] = t.Name
			instructorSearchByID[t.Id] = []string{t.NormalizedName, t.NamePinyin, t.NameInitials}
		}
	}
	if len(offeringIds) > 0 {
		links, err := course.ListOfferingInstructorLinks(offeringIds)
		if err != nil {
			return nil, err
		}
		instructors, err := course.ListInstructorsByOfferings(offeringIds)
		if err != nil {
			return nil, err
		}
		for _, ins := range instructors {
			instructorSearchByID[ins.Id] = []string{ins.NormalizedName, ins.NamePinyin, ins.NameInitials}
			instructorByID[ins.Id] = ins.Name
		}
		for _, link := range links {
			if name, ok := instructorByID[link.InstructorId]; ok {
				instructorByOffering[link.OfferingId] = append(instructorByOffering[link.OfferingId], name)
				instructorSearchByOffering[link.OfferingId] = append(instructorSearchByOffering[link.OfferingId], instructorSearchByID[link.InstructorId]...)
			}
		}
		termIds := make([]uint64, 0, len(offerings))
		for _, o := range offerings {
			termIds = append(termIds, o.TermId)
		}
		terms, err := course.ListTermsByIDs(termIds)
		if err != nil {
			return nil, err
		}
		for _, t := range terms {
			termByID[t.Id] = t
		}
	}
	for _, e := range entities {
		doc := CourseSearchDocument{
			ProjectionVersion: courseProjectionVersion,
			ID:                e.Id,
			PrimaryCode:       e.PrimaryCode,
			Name:              e.Name,
			NormalizedName:    e.NormalizedName,
			NamePinyin:        e.NamePinyin,
			NameInitials:      e.NameInitials,
			Department:        e.Department,
			CreditX10:         e.CreditX10,
			Aliases:           []string{},
			TeacherId:         e.TeacherId,
			Instructors:       []string{},
			Terms:             []string{},
			Campus:            []string{},
			Status:            e.Status,
			CreatedAt:         e.CreatedAt.Unix(),
			UpdatedAt:         e.UpdatedAt.Unix(),
		}
		if e.TeacherId != 0 {
			// 身份教师批量解析在循环外统一做（teacherNameByID 预填充）。
			doc.TeacherName = teacherNameByID[e.TeacherId]
			doc.InstructorSearch = append(doc.InstructorSearch, instructorSearchByID[e.TeacherId]...)
		}
		doc.Aliases = append(doc.Aliases, aliasByCourse[e.Id]...)
		seenInstructors := make(map[string]struct{})
		seenTerms := make(map[string]struct{})
		seenCampus := make(map[string]struct{})
		for _, o := range offeringByCourse[e.Id] {
			if o.ClassCode != "" {
				doc.ClassCodes = append(doc.ClassCodes, o.ClassCode)
			}
			doc.InstructorSearch = append(doc.InstructorSearch, instructorSearchByOffering[o.Id]...)
			for _, name := range instructorByOffering[o.Id] {
				if _, dup := seenInstructors[name]; !dup {
					seenInstructors[name] = struct{}{}
					doc.Instructors = append(doc.Instructors, name)
				}
			}
			if t, ok := termByID[o.TermId]; ok {
				if _, dup := seenTerms[t.Code]; !dup {
					seenTerms[t.Code] = struct{}{}
					doc.Terms = append(doc.Terms, t.Code)
				}
			}
			if o.Campus != "" {
				if _, dup := seenCampus[o.Campus]; !dup {
					seenCampus[o.Campus] = struct{}{}
					doc.Campus = append(doc.Campus, o.Campus)
				}
			}
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

// shouldIndexCourse 课程可见才入索引（hidden 或 soft-deleted 不入）。
func shouldIndexCourse(entity course.Entity) bool {
	return entity.Id != 0 && entity.Status == course.StatusVisible
}

// waitForTaskChecked 等待 Meili 任务完成并校验终态：
// WaitForTask 在任务终态为 failed 时可能返回 (task, nil)（Go error 为空），
// 只检查 error 会把索引拒绝/内部失败当作成功，导致 outbox 永不重试。
// 这里显式检查 task.Status 与 task.Error，失败即返回错误。
func waitForTaskChecked(client meilisearch.TaskReader, taskUID int64, timeout time.Duration) error {
	return waitForTaskCheckedContext(context.Background(), client, taskUID, timeout)
}

func waitForTaskCheckedContext(ctx context.Context, client meilisearch.TaskReader, taskUID int64, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	// The SDK argument is a polling interval, not an operation timeout.
	task, err := client.WaitForTaskWithContext(ctx, taskUID, 100*time.Millisecond)
	if err != nil {
		// The SDK's communication error can hide context cancellation from
		// errors.Is. Preserve the caller's cancellation/deadline semantics.
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
	if task != nil && (task.Status == meilisearch.TaskStatusFailed || task.Status == meilisearch.TaskStatusCanceled) {
		msg := task.Error.Message
		if msg == "" {
			msg = task.Error.Code
		}
		return fmt.Errorf("meilisearch task %d failed: %s", taskUID, msg)
	}
	return nil
}

// BuildSingleCourseSearchDocument 单课程 upsert/delete 到 Meili（worker 与事件共用）。
func BuildSingleCourseSearchDocument(entity course.Entity) error {
	if !meiliconnect.IsAvailable() {
		return errors.New("meilisearch 服务不可用")
	}
	client := meiliconnect.GetClient()
	index := client.Index(CourseIndex)
	pk := "id"
	if !shouldIndexCourse(entity) {
		// 删除分支同样等待任务终态：仅发起请求不等待，
		// 崩溃/失败时无法确认删除是否生效，outbox 会误判成功。
		task, err := index.DeleteDocument(cast.ToString(entity.Id), nil)
		if err != nil {
			return err
		}
		return waitForTaskChecked(client, task.TaskUID, 30*time.Second)
	}
	doc, err := convertCourseToSearchDocument(entity)
	if err != nil {
		return err
	}
	task, err := index.AddDocuments([]CourseSearchDocument{doc}, &meilisearch.DocumentOptions{PrimaryKey: &pk})
	if err != nil {
		return err
	}
	return waitForTaskChecked(client, task.TaskUID, 30*time.Second)
}

// BuildCourseIndex 全量重建课程索引（rebuild-course-search 命令与 e2e 使用）。
// 先清空索引内全部文档（等待 delete-all 任务完成），再按 PG 真值逐批写入：
// 保证已删除/隐藏课程不会残留在索引中，投影可恢复到 PG 事实源。
func BuildCourseIndex(ctx context.Context) (*IndexBuildResult, error) {
	return buildCourseIndex(ctx, true)
}

// RefreshCourseIndex backfills projection fields without emptying the live index.
// Visibility is always rechecked in PostgreSQL; stale deleted documents can be
// removed with the existing full rebuild/reconciliation workflow.
func RefreshCourseIndex(ctx context.Context) (*IndexBuildResult, error) {
	return buildCourseIndex(ctx, false)
}

func buildCourseIndex(ctx context.Context, clearExisting bool) (*IndexBuildResult, error) {
	if !meiliconnect.IsAvailable() {
		return nil, errors.New("meilisearch 服务不可用，请检查配置或连接状态")
	}
	client := meiliconnect.GetClient()
	index := client.Index(CourseIndex)
	pk := "id"
	if err := configureCourseIndex(ctx, index); err != nil {
		return nil, fmt.Errorf("配置课程索引失败: %w", err)
	}
	// 清空旧文档并等待任务终态：避免 rebuild 只做 AddDocuments 叠加，
	// 使 PG 中已不存在/隐藏的课程永久残留索引。
	if clearExisting {
		cleanTask, err := index.DeleteAllDocumentsWithContext(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("清空课程索引失败: %w", err)
		}
		if err := waitForTaskCheckedContext(ctx, client, cleanTask.TaskUID, 60*time.Second); err != nil {
			return nil, fmt.Errorf("清空课程索引任务失败: %w", err)
		}
	}
	return buildCourseIndexPages(ctx,
		course.ListCoursesAfterID,
		convertCoursesToSearchDocuments,
		func(docs []CourseSearchDocument) error {
			task, err := index.AddDocumentsWithContext(ctx, docs, &meilisearch.DocumentOptions{PrimaryKey: &pk})
			if err != nil {
				return err
			}
			return waitForTaskCheckedContext(ctx, client, task.TaskUID, 60*time.Second)
		})
}

// buildCourseIndexPages 分页读取课程并写入索引（依赖注入便于单测失败路径）。
// 按主键游标扫描，避免并发删除使 OFFSET 位移而漏行；刷新与全量重建共用。
// 任一页转换/写入失败必须整体返回错误，避免 CLI 将部分索引误报为成功。
func buildCourseIndexPages(ctx context.Context,
	listCourses func(ctx context.Context, afterID uint64, limit int) ([]course.Entity, error),
	convert func(entities []course.Entity) ([]CourseSearchDocument, error),
	addDocs func(docs []CourseSearchDocument) error,
) (*IndexBuildResult, error) {
	result := &IndexBuildResult{IndexName: CourseIndex}
	var afterID uint64
	const batch = 200
	for {
		if ctx != nil {
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			default:
			}
		}
		entities, err := listCourses(ctx, afterID, batch)
		if err != nil {
			return result, err
		}
		if len(entities) == 0 {
			break
		}
		batchEntities := make([]course.Entity, 0, len(entities))
		for _, e := range entities {
			if shouldIndexCourse(e) {
				batchEntities = append(batchEntities, e)
			}
		}
		docs, err := convert(batchEntities)
		if err != nil {
			return result, fmt.Errorf("convert course search docs batch %d: %w", result.TotalBatches, err)
		}
		if len(docs) > 0 {
			if err := addDocs(docs); err != nil {
				return result, err
			}
		}
		result.ProcessedCount += len(entities)
		result.TotalBatches++
		// Advance through every source row, including an entirely hidden page.
		afterID = entities[len(entities)-1].Id
	}
	return result, nil
}

// configureCourseIndex 设置课程索引的 searchable/filterable/sortable/displayed 属性。
func configureCourseIndex(ctx context.Context, index meilisearch.IndexManager) error {
	return applyManagedSettings(ctx, index, CourseIndex)
}

// TaskTypeCourseSearch 是 course-search outbox worker 的任务类型前缀。
const TaskTypeCourseSearch = "course-search."

// CourseSearchTask 是 enqueue 到 taskQueue 的任务负载。
type CourseSearchTask struct {
	CourseId uint64 `json:"courseId"`
}

// EnqueueCourseSearchTask 在业务事务内入队课程搜索同步任务（transaction-bound outbox）。
// 入队前去重：同一事务内（或此前已提交）存在相同 courseId 的 pending/retrying
// 任务时跳过，避免 reviews 导入等路径对同一课程重复入队 N 条相同任务
// （worker 会重复构建同一文档）。
func EnqueueCourseSearchTask(tx *gorm.DB, courseId uint64) error {
	payload, err := json.Marshal(CourseSearchTask{CourseId: courseId})
	if err != nil {
		return err
	}
	var count int64
	if err := tx.Table((&taskQueue.Entity{}).TableName()).
		Where("type LIKE ?", TaskTypeCourseSearch+"%").
		Where("status IN ?", []int{taskQueue.StatusPending, taskQueue.StatusRetrying}).
		Where("task_json = ?", string(payload)).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil // 已有同课程待处理任务，合并
	}
	return taskQueue.CreateTx(tx, &taskQueue.Entity{
		Type:     TaskTypeCourseSearch + "upsert",
		TaskJson: string(payload),
	})
}

// RunCourseSearchTask worker 处理：始终从 PG 读取最新状态生成投影。
// 注意：GetCourse 未命中时返回零值 entity，此时按 payload.CourseId 删除
// 对应文档（零值 Id 是 0，删除 id="0" 会让真实文档残留索引）。
func RunCourseSearchTask(ctx context.Context, task *taskQueue.Entity) error {
	var payload CourseSearchTask
	if err := json.Unmarshal([]byte(task.TaskJson), &payload); err != nil {
		return err
	}
	entity := course.GetCourse(payload.CourseId)
	if entity.Id == 0 {
		if !meiliconnect.IsAvailable() {
			return errors.New("meilisearch 服务不可用")
		}
		client := meiliconnect.GetClient()
		delTask, err := client.Index(CourseIndex).DeleteDocument(cast.ToString(payload.CourseId), nil)
		if err != nil {
			return err
		}
		// 等待删除任务终态：仅发起请求不等待时，失败/崩溃无法确认删除生效。
		return waitForTaskChecked(client, delTask.TaskUID, 30*time.Second)
	}
	return BuildSingleCourseSearchDocument(entity)
}

// CourseReconcileResult reconcile-course-search 命令的输出。
type CourseReconcileResult struct {
	IndexedDocs int `json:"indexedDocs"`
	PGCourses   int `json:"pgCourses"`
	DriftCount  int `json:"driftCount"`
}

// ReconcileCourseIndex 对比 PG 可见课程数与 Meili 索引文档数，报告漂移。
// 一期只做计数对账（dry-run 语义）；精确到文档级别的 diff 由后续 slice 增强。
func ReconcileCourseIndex(ctx context.Context) (*CourseReconcileResult, error) {
	result := &CourseReconcileResult{}
	var total int64
	_, total, err := course.ListCourses(course.ListCourseQuery{Size: 1})
	if err != nil {
		return nil, err
	}
	result.PGCourses = int(total)
	if !meiliconnect.IsAvailable() {
		result.DriftCount = result.PGCourses // 索引不可用视为全部漂移
		return result, nil
	}
	client := meiliconnect.GetClient()
	index := client.Index(CourseIndex)
	stats, err := index.GetStats(&meilisearch.StatsParams{})
	if err != nil {
		return nil, fmt.Errorf("get course index stats: %w", err)
	}
	result.IndexedDocs = int(stats.NumberOfDocuments)
	diff := result.PGCourses - result.IndexedDocs
	if diff < 0 {
		diff = -diff
	}
	result.DriftCount = diff
	return result, nil
}

// RecoverStaleTasks 启动时恢复课程搜索 worker 类型前缀下崩溃遗留的 Running 任务。
func RecoverStaleTasks() error {
	return taskQueue.RecoverStaleRunning(TaskTypeCourseSearch, taskQueue.LeaseDuration)
}

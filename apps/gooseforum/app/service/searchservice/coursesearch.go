package searchservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/connect/meiliconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/course"
	"github.com/leancodebox/GooseForum/app/models/forum/taskQueue"
	"github.com/meilisearch/meilisearch-go"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

// CourseIndex is the Meilisearch index for canonical course documents.
const CourseIndex = "courses"

// CourseSearchDocument 课程搜索文档：一门课程只出现一次（canonical course）。
// 教师/学期/校区作为数组字段；不索引评价正文。
type CourseSearchDocument struct {
	ID             uint64   `json:"id"`
	PrimaryCode    string   `json:"primaryCode"`
	Name           string   `json:"name"`
	NormalizedName string   `json:"normalizedName"`
	NamePinyin     string   `json:"namePinyin"`
	NameInitials   string   `json:"nameInitials"`
	Department     string   `json:"department"`
	CreditX10      int      `json:"creditX10"`
	Aliases        []string `json:"aliases"`
	Instructors    []string `json:"instructors"`
	Terms          []string `json:"terms"`
	Campus         []string `json:"campus"`
	Status         int8     `json:"status"`
	CreatedAt      int64    `json:"createdAt"`
	UpdatedAt      int64    `json:"updatedAt"`
}

// convertCourseToSearchDocument 从 canonical course + 关联构建搜索文档。
func convertCourseToSearchDocument(entity course.Entity) (CourseSearchDocument, error) {
	doc := CourseSearchDocument{
		ID:             entity.Id,
		PrimaryCode:    entity.PrimaryCode,
		Name:           entity.Name,
		NormalizedName: entity.NormalizedName,
		NamePinyin:     entity.NamePinyin,
		NameInitials:   entity.NameInitials,
		Department:     entity.Department,
		CreditX10:      entity.CreditX10,
		Aliases:        []string{},
		Instructors:    []string{},
		Terms:          []string{},
		Campus:         []string{},
		Status:         entity.Status,
		CreatedAt:      entity.CreatedAt.Unix(),
		UpdatedAt:      entity.UpdatedAt.Unix(),
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

// shouldIndexCourse 课程可见才入索引（hidden 或 soft-deleted 不入）。
func shouldIndexCourse(entity course.Entity) bool {
	return entity.Id != 0 && entity.Status == course.StatusVisible
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
		_, err := index.DeleteDocument(cast.ToString(entity.Id), nil)
		return err
	}
	doc, err := convertCourseToSearchDocument(entity)
	if err != nil {
		return err
	}
	task, err := index.AddDocuments([]CourseSearchDocument{doc}, &meilisearch.DocumentOptions{PrimaryKey: &pk})
	if err != nil {
		return err
	}
	if _, err := client.WaitForTask(task.TaskUID, 30*time.Second); err != nil {
		return err
	}
	return nil
}

// BuildCourseIndex 全量重建课程索引（rebuild-course-search 命令与 e2e 使用）。
func BuildCourseIndex(ctx context.Context) (*IndexBuildResult, error) {
	if !meiliconnect.IsAvailable() {
		return nil, errors.New("meilisearch 服务不可用，请检查配置或连接状态")
	}
	client := meiliconnect.GetClient()
	index := client.Index(CourseIndex)
	pk := "id"
	if err := configureCourseIndex(index); err != nil {
		return nil, fmt.Errorf("配置课程索引失败: %w", err)
	}
	result := &IndexBuildResult{IndexName: CourseIndex}
	offset := 0
	const batch = 200
	for {
		if ctx != nil {
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			default:
			}
		}
		entities, err := course.ListAllCourses(batch, offset)
		if err != nil {
			return result, err
		}
		if len(entities) == 0 {
			break
		}
		docs := make([]CourseSearchDocument, 0, len(entities))
		for _, e := range entities {
			if !shouldIndexCourse(e) {
				continue
			}
			doc, err := convertCourseToSearchDocument(e)
			if err != nil {
				slog.Error("course search doc conversion failed", "courseId", e.Id, "err", err)
				result.FailedCount++
				continue
			}
			docs = append(docs, doc)
		}
		if len(docs) > 0 {
			task, err := index.AddDocuments(docs, &meilisearch.DocumentOptions{PrimaryKey: &pk})
			if err != nil {
				return result, err
			}
			if _, err := client.WaitForTask(task.TaskUID, 60*time.Second); err != nil {
				return result, err
			}
		}
		result.ProcessedCount += len(entities)
		result.TotalBatches++
		offset += batch
	}
	return result, nil
}

// configureCourseIndex 设置课程索引的 searchable/filterable/sortable/displayed 属性。
func configureCourseIndex(index meilisearch.IndexManager) error {
	settings := &meilisearch.Settings{
		SearchableAttributes: []string{
			"name", "normalizedName", "primaryCode", "aliases", "instructors", "namePinyin", "nameInitials",
		},
		FilterableAttributes: []string{
			"department", "terms", "campus", "status",
		},
		SortableAttributes: []string{
			"createdAt", "updatedAt",
		},
		DisplayedAttributes: []string{
			"id", "primaryCode", "name", "department", "creditX10", "aliases", "instructors", "terms", "campus", "status",
		},
	}
	_, err := index.UpdateSettings(settings)
	return err
}

// TaskTypeCourseSearch 是 course-search outbox worker 的任务类型前缀。
const TaskTypeCourseSearch = "course-search."

// CourseSearchTask 是 enqueue 到 taskQueue 的任务负载。
type CourseSearchTask struct {
	CourseId uint64 `json:"courseId"`
}

// EnqueueCourseSearchTask 在业务事务内入队课程搜索同步任务（transaction-bound outbox）。
func EnqueueCourseSearchTask(tx *gorm.DB, courseId uint64) error {
	payload, err := json.Marshal(CourseSearchTask{CourseId: courseId})
	if err != nil {
		return err
	}
	return taskQueue.CreateTx(tx, &taskQueue.Entity{
		Type:     TaskTypeCourseSearch + "upsert",
		TaskJson: string(payload),
	})
}

// RunCourseSearchTask worker 处理：始终从 PG 读取最新状态生成投影。
func RunCourseSearchTask(ctx context.Context, task *taskQueue.Entity) error {
	var payload CourseSearchTask
	if err := json.Unmarshal([]byte(task.TaskJson), &payload); err != nil {
		return err
	}
	entity := course.GetCourse(payload.CourseId)
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

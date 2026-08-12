package searchservice

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/meiliconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/meilisearch/meilisearch-go"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

// CategorySearchDocument 分类搜索文档结构（只含公开可搜字段 + 拼音辅助字段）
type CategorySearchDocument struct {
	ID           uint64 `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Desc         string `json:"desc"`
	NamePinyin   string `json:"namePinyin"`
	NameInitials string `json:"nameInitials"`
}

// convertCategoryToSearchDocument maps a category to its search document.
func convertCategoryToSearchDocument(entity *category.Entity) CategorySearchDocument {
	namePinyin, nameInitials := CategoryPinyinFields(entity.Name)
	return CategorySearchDocument{
		ID:           entity.Id,
		Name:         entity.Name,
		Slug:         entity.Slug,
		Desc:         entity.Desc,
		NamePinyin:   namePinyin,
		NameInitials: nameInitials,
	}
}

// BuildSingleCategorySearchDocument upserts a category document.
func BuildSingleCategorySearchDocument(entity *category.Entity) (*meilisearch.TaskInfo, error) {
	if !meiliconnect.IsAvailable() {
		return nil, nil
	}
	if entity == nil || entity.Id == 0 {
		return nil, nil
	}
	client := meiliconnect.GetClient()
	index := client.Index(CategoryIndex)
	pk := "id"
	doc := convertCategoryToSearchDocument(entity)
	task, err := index.AddDocuments(doc, &meilisearch.DocumentOptions{PrimaryKey: &pk})
	if err != nil {
		slog.Warn(fmt.Sprintf("Meilisearch 处理分类 ID:%v 失败: %v\n", doc.ID, err))
		return nil, fmt.Errorf("add category search document: %w", err)
	}
	slog.Info(fmt.Sprintf("处理分类 ID:%v, TaskUID: %v\n", doc.ID, getTaskUID(task)))
	return task, nil
}

// DeleteCategorySearchDocument removes a category document by id.
func DeleteCategorySearchDocument(categoryID uint64) (*meilisearch.TaskInfo, error) {
	if !meiliconnect.IsAvailable() {
		return nil, nil
	}
	client := meiliconnect.GetClient()
	index := client.Index(CategoryIndex)
	task, err := index.DeleteDocument(cast.ToString(categoryID), nil)
	if err != nil {
		slog.Warn(fmt.Sprintf("Meilisearch 删除分类文档失败: %v, Error: %v\n", categoryID, err))
		return nil, fmt.Errorf("delete category search document: %w", err)
	}
	slog.Info(fmt.Sprintf("删除分类 ID:%v, TaskUID: %v\n", categoryID, getTaskUID(task)))
	return task, nil
}

// BuildCategoryIndex rebuilds the whole Meilisearch category index.
func BuildCategoryIndex() (*IndexBuildResult, error) {
	if !meiliconnect.IsAvailable() {
		return nil, fmt.Errorf("meilisearch 服务不可用，请检查配置或连接状态")
	}
	fmt.Println("开始构建 Meilisearch 分类索引...")
	client := meiliconnect.GetClient()
	index := client.Index(CategoryIndex)
	if err := configureCategoryIndex(index); err != nil {
		return nil, fmt.Errorf("配置分类索引失败: %w", err)
	}

	processedCount := 0
	failedCount := 0
	expectedIDs := make(map[string]struct{})
	categoryList := category.All()
	for _, entity := range categoryList {
		// 先登记应存在于索引的文档 ID：即使本次写入失败，幽灵清理也不得
		// 删除数据库仍要求保留的文档（写入失败由 failedCount 暴露）。
		expectedIDs[cast.ToString(entity.Id)] = struct{}{}
		if _, err := BuildSingleCategorySearchDocument(entity); err != nil {
			failedCount++
			slog.Warn("failed to build category search document", "categoryId", entity.Id, "err", err)
			continue
		}
		processedCount++
	}

	// 幽灵清理删除候选在入队前按数据库最新状态复核，跳过 snapshot 之后
	// 新增的分类（PR #151 review P1 竞态）。
	revalidateCategoryGhost := func(id string) (bool, error) {
		categoryID := cast.ToUint64(id)
		if categoryID == 0 {
			return false, nil
		}
		entity, err := category.GetWithError(categoryID)
		if err != nil {
			// 记录不存在 → 确实是幽灵；其他错误（如 DB 瞬时故障）→ 保守保留。
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return false, nil
			}
			return true, err
		}
		return entity.Id != 0, nil
	}
	deletedIDs, err := cleanupGhostDocuments(index, expectedIDs, revalidateCategoryGhost)
	if err != nil {
		return nil, fmt.Errorf("清理分类索引幽灵文档失败: %w", err)
	}
	ghostRemoved := len(deletedIDs)

	// replay：删除任务入队后、执行前新增的分类重新入队 upsert，
	// 排在 delete 之后执行，确保有效文档最终不丢失。
	replayedCount := 0
	for _, id := range deletedIDs {
		categoryID := cast.ToUint64(id)
		if categoryID == 0 {
			continue
		}
		entity := category.Get(categoryID)
		if entity.Id == 0 {
			continue
		}
		if _, err := BuildSingleCategorySearchDocument(&entity); err != nil {
			failedCount++
			slog.Warn("failed to restore category search document after ghost cleanup", "categoryId", entity.Id, "err", err)
			continue
		}
		replayedCount++
	}

	result := &IndexBuildResult{
		ProcessedCount: processedCount,
		FailedCount:    failedCount,
		TotalBatches:   1,
		IndexName:      CategoryIndex,
		GhostRemoved:   ghostRemoved,
	}
	fmt.Printf("\n=== Meilisearch 分类索引构建完成 ===\n")
	fmt.Printf("成功索引: %d 个分类\n", result.ProcessedCount)
	fmt.Printf("失败数量: %d 个分类\n", result.FailedCount)
	fmt.Printf("提交幽灵文档删除任务: %d 个\n", result.GhostRemoved)
	fmt.Printf("清理期间恢复索引文档: %d 个\n", replayedCount)
	return result, nil
}

// configureCategoryIndex applies searchable and displayed attributes to the category index.
func configureCategoryIndex(index meilisearch.IndexManager) error {
	searchableAttributes := []string{
		"name",
		"desc",
		"slug",
		"namePinyin",
		"nameInitials",
	}
	if _, err := index.UpdateSearchableAttributes(&searchableAttributes); err != nil {
		return fmt.Errorf("设置分类可搜索字段失败: %w", err)
	}
	displayedAttributes := []string{"id", "name", "slug"}
	if _, err := index.UpdateDisplayedAttributes(&displayedAttributes); err != nil {
		return fmt.Errorf("设置分类显示字段失败: %w", err)
	}
	fmt.Println("分类索引配置完成")
	return nil
}

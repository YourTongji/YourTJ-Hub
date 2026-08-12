package searchservice

import (
	"fmt"
	"log/slog"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/meiliconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/meilisearch/meilisearch-go"
	"github.com/spf13/cast"
)

// UserSearchDocument 用户搜索文档结构（只含公开可搜字段 + 拼音辅助字段）
type UserSearchDocument struct {
	ID               uint64 `json:"id"`
	Username         string `json:"username"`
	Nickname         string `json:"nickname"`
	Bio              string `json:"bio"`
	UsernamePinyin   string `json:"usernamePinyin"`
	UsernameInitials string `json:"usernameInitials"`
	NicknamePinyin   string `json:"nicknamePinyin"`
	NicknameInitials string `json:"nicknameInitials"`
}

// convertUserToSearchDocument maps a user to its search document.
func convertUserToSearchDocument(user *users.EntityComplete) UserSearchDocument {
	usernamePinyin, usernameInitials, nicknamePinyin, nicknameInitials := UserPinyinFields(user.Username, user.Nickname)
	return UserSearchDocument{
		ID:               user.Id,
		Username:         user.Username,
		Nickname:         user.Nickname,
		Bio:              user.Bio,
		UsernamePinyin:   usernamePinyin,
		UsernameInitials: usernameInitials,
		NicknamePinyin:   nicknamePinyin,
		NicknameInitials: nicknameInitials,
	}
}

func shouldIndexUser(user *users.EntityComplete) bool {
	return user != nil && user.Id > 0 && !user.DeletedAt.Valid && !user.IsBot()
}

// BuildSingleUserSearchDocument upserts a human user document, or deletes it
// when the user is missing from the public index, soft-deleted, or a bot.
func BuildSingleUserSearchDocument(user *users.EntityComplete) (*meilisearch.TaskInfo, error) {
	if !meiliconnect.IsAvailable() {
		return nil, nil
	}
	if user == nil {
		return nil, nil
	}
	client := meiliconnect.GetClient()
	index := client.Index(UserIndex)
	pk := "id"
	if shouldIndexUser(user) {
		doc := convertUserToSearchDocument(user)
		task, err := index.AddDocuments(doc, &meilisearch.DocumentOptions{PrimaryKey: &pk})
		if err != nil {
			slog.Warn(fmt.Sprintf("Meilisearch 处理用户 ID:%v 失败: %v\n", doc.ID, err))
			return nil, fmt.Errorf("add user search document: %w", err)
		}
		slog.Info(fmt.Sprintf("处理用户 ID:%v, TaskUID: %v\n", doc.ID, getTaskUID(task)))
		return task, nil
	}
	task, err := index.DeleteDocument(cast.ToString(user.Id), nil)
	if err != nil {
		slog.Warn(fmt.Sprintf("Meilisearch 删除用户文档失败: %v, Error: %v\n", user.Id, err))
		return nil, fmt.Errorf("delete user search document: %w", err)
	}
	slog.Info(fmt.Sprintf("删除用户 ID:%v, TaskUID: %v\n", user.Id, getTaskUID(task)))
	return task, nil
}

// DeleteUserSearchDocument removes a user document by id.
func DeleteUserSearchDocument(userID uint64) (*meilisearch.TaskInfo, error) {
	if !meiliconnect.IsAvailable() {
		return nil, nil
	}
	client := meiliconnect.GetClient()
	index := client.Index(UserIndex)
	task, err := index.DeleteDocument(cast.ToString(userID), nil)
	if err != nil {
		slog.Warn(fmt.Sprintf("Meilisearch 删除用户文档失败: %v, Error: %v\n", userID, err))
		return nil, fmt.Errorf("delete user search document: %w", err)
	}
	slog.Info(fmt.Sprintf("删除用户 ID:%v, TaskUID: %v\n", userID, getTaskUID(task)))
	return task, nil
}

// BuildUserIndex rebuilds the whole Meilisearch user index.
func BuildUserIndex() (*IndexBuildResult, error) {
	if !meiliconnect.IsAvailable() {
		return nil, fmt.Errorf("meilisearch 服务不可用，请检查配置或连接状态")
	}
	fmt.Println("开始构建 Meilisearch 用户索引...")
	client := meiliconnect.GetClient()
	index := client.Index(UserIndex)
	if err := configureUserIndex(index); err != nil {
		return nil, fmt.Errorf("配置用户索引失败: %w", err)
	}

	processedCount := 0
	failedCount := 0
	expectedIDs := make(map[string]struct{})
	userList := users.All()
	for _, user := range userList {
		// 先登记应存在于索引的文档 ID：即使本次写入失败，幽灵清理也不得
		// 删除数据库仍要求保留的文档（写入失败由 failedCount 暴露）。
		if shouldIndexUser(user) {
			expectedIDs[cast.ToString(user.Id)] = struct{}{}
		}
		if _, err := BuildSingleUserSearchDocument(user); err != nil {
			failedCount++
			slog.Warn("failed to build user search document", "userId", user.Id, "err", err)
			continue
		}
		if shouldIndexUser(user) {
			processedCount++
		}
	}

	ghostRemoved, err := cleanupGhostDocuments(index, expectedIDs)
	if err != nil {
		return nil, fmt.Errorf("清理用户索引幽灵文档失败: %w", err)
	}

	result := &IndexBuildResult{
		ProcessedCount: processedCount,
		FailedCount:    failedCount,
		TotalBatches:   1,
		IndexName:      UserIndex,
		GhostRemoved:   ghostRemoved,
	}
	fmt.Printf("\n=== Meilisearch 用户索引构建完成 ===\n")
	fmt.Printf("成功索引: %d 个用户\n", result.ProcessedCount)
	fmt.Printf("失败数量: %d 个用户\n", result.FailedCount)
	fmt.Printf("删除幽灵文档: %d 个\n", result.GhostRemoved)
	return result, nil
}

// configureUserIndex applies searchable and displayed attributes to the user index.
func configureUserIndex(index meilisearch.IndexManager) error {
	searchableAttributes := []string{
		"username",
		"nickname",
		"bio",
		"usernamePinyin",
		"usernameInitials",
		"nicknamePinyin",
		"nicknameInitials",
	}
	if _, err := index.UpdateSearchableAttributes(&searchableAttributes); err != nil {
		return fmt.Errorf("设置用户可搜索字段失败: %w", err)
	}
	displayedAttributes := []string{"id", "username", "nickname", "bio"}
	if _, err := index.UpdateDisplayedAttributes(&displayedAttributes); err != nil {
		return fmt.Errorf("设置用户显示字段失败: %w", err)
	}
	fmt.Println("用户索引配置完成")
	return nil
}

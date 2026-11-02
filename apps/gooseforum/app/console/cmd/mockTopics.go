package cmd

import (
	"errors"
	"fmt"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "mock-topics",
		Short: "Mock topics for testing",
		RunE:  runMockTopics,
	}
	appendCommand(cmd)
}

// runMockTopics 批量创建 mock 帖子。任一失败使命令返回非零退出状态,
// 不能用正常退出掩盖部分或全部失败(issue #1284)。
func runMockTopics(cmd *cobra.Command, args []string) error {
	fmt.Println("Starting to mock topics...")

	allUsers := users.All()
	if len(allUsers) == 0 {
		return errors.New("no users found, please create users first")
	}

	totalTopics := 50
	created, failed := 0, 0
	for i := 1; i <= totalTopics; i++ {
		// Pick a user randomly or sequentially
		user := allUsers[i%len(allUsers)]

		req := component.BetterRequest[api.WriteTopicReq]{
			UserId: user.Id,
			Params: api.WriteTopicReq{
				Title:       fmt.Sprintf("测试文章 %03d - %s", i, time.Now().Format("15:04:05")),
				Content:     fmt.Sprintf("这是第 %d 篇自动生成的测试文章内容。\n\n生成时间: %s\n作者: %s", i, time.Now().Format(time.RFC3339), user.Username),
				CategoryId:  []uint64{1}, // Default to first category
				TopicStatus: 1,           // 已发布：mock 数据应进入公开列表（issue #645）
			},
		}

		resp := api.WriteTopic(req)
		if resp.Data.Code == component.SUCCESS {
			created++
			fmt.Printf("[%d/%d] Created topic: %s (User: %s)\n", i, totalTopics, req.Params.Title, user.Username)
		} else {
			failed++
			fmt.Printf("[%d/%d] Failed to create topic: %s\n", i, totalTopics, resp.Data.MessageCode)
		}

		// Small delay to ensure sequence and timestamp difference
		time.Sleep(10 * time.Millisecond)
	}

	fmt.Printf("Mock topics generation done: created=%d failed=%d total=%d\n", created, failed, totalTopics)
	if failed > 0 {
		return fmt.Errorf("%d/%d mock topics failed to create", failed, totalTopics)
	}
	return nil
}

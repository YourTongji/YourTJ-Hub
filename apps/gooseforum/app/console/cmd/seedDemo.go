package cmd

import (
	"fmt"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "seed-demo",
		Short: "Seed demo users, topics and nested replies for local preview",
		Run:   runSeedDemo,
	}
	appendCommand(cmd)

	tokenCmd := &cobra.Command{
		Use:   "token-demo",
		Short: "Print a demo JWT for user id 1",
		Run:   runTokenDemo,
	}
	appendCommand(tokenCmd)
}

func runTokenDemo(cmd *cobra.Command, args []string) {
	token, err := jwtopt.CreateNewTokenDefaultWithVersion(1, 1)
	if err != nil {
		fmt.Println("token error:", err)
		return
	}
	fmt.Println(token)
}

type seedPostRow struct {
	Id uint64
}

func runSeedDemo(cmd *cobra.Command, args []string) {
	fmt.Println("Seeding demo data...")

	if len(users.All()) > 0 {
		fmt.Println("Users already exist, skipping seed.")
		return
	}

	// 1. Create demo users (activated, no email verification needed)
	demoUsers := []struct{ name, email string }{
		{"demo1", "demo1@example.com"},
		{"demo2", "demo2@example.com"},
		{"demo3", "demo3@example.com"},
	}
	var userIds []uint64
	for _, du := range demoUsers {
		userEntity, err := userservice.CreateUser(du.name, "demo123456", du.email, false, "zh-CN")
		if err != nil {
			fmt.Printf("Failed to create user %s: %v\n", du.name, err)
			return
		}
		userIds = append(userIds, userEntity.Id)
		fmt.Printf("Created user: %s (id=%d)\n", du.name, userEntity.Id)
	}

	// 2. Backdate created_at so new-user posting cooldown does not block seeding
	db.Connect().Table("users").
		Where("id IN ?", userIds).
		Updates(map[string]any{
			"created_at": time.Now().Add(-48 * time.Hour),
			"updated_at": time.Now().Add(-48 * time.Hour),
		})

	// 3. Topics
	topicTitles := []string{
		"【水帖】今天也在为期末周头秃，大家有什么摸鱼妙招？",
		"Vue 3 组合式 API 的组件设计心得分享",
	}
	contents := []string{
		"## 开个楼\n\n期末周要来了，图书馆座位比春运还难抢 (｡•́︿•̀｡)。\n\n大家来说说自己的 **摸鱼** 或者 *高效复习* 的小妙招吧：\n\n- 番茄钟 25+5\n- 把手机锁进柜子\n- 组队学习互相监督\n\n> 附：学校图书馆 10 楼靠窗的位置风景很好，适合背书。\n\n```text\n早睡早起，拒绝熬夜复习\n```\n",
		"最近在用 **组合式 API** 重构老项目，分享几个体会：\n\n1. 按**关注点**组织代码，而不是按选项块\n2. `computed` 依赖追踪比 watch 好用的多\n3. 抽 composable 时注意命名：`useXxx`\n\n代码示例：\n\n```ts\nconst { loading, data, refresh } = useTopicList()\n```\n\n欢迎交流～",
	}

	for i, title := range topicTitles {
		resp := api.WriteTopic(component.BetterRequest[api.WriteTopicReq]{
			UserId: userIds[i%len(userIds)],
			Params: api.WriteTopicReq{
				Title:      title,
				Content:    contents[i],
				CategoryId: []uint64{1},
			},
		})
		if resp.Data.Code != component.SUCCESS {
			fmt.Printf("Failed to create topic %q: %s\n", title, resp.Data.MessageCode)
			return
		}
		fmt.Printf("Created topic: %s\n", title)
	}

	// 4. Nested replies on topic 1: OP gets 5 direct children (to test expand),
	//    one of the children gets a reply-of-reply (to test quote chain).
	postContents := []string{
		"我投番茄钟一票，25 分钟真的能进入状态。",
		"回复楼上：关键是番茄之间的 5 分钟别刷手机，不然就回不来了。",
		"图书馆 10 楼已经改成预约制了，别白跑。",
		"组队学习！找个搭子效率翻倍。",
		"摸鱼时间也不全是坏事，偶尔发发呆思路反而通了。",
	}
	var postIds []uint64
	for i, content := range postContents {
		resp := api.CreatePost(component.BetterRequest[api.CreatePostReq]{
			UserId: userIds[i%len(userIds)],
			Params: api.CreatePostReq{
				TopicId:       1,
				Content:       content,
				ReplyToPostId: 1, // 回复楼主（第一楼）
			},
		})
		if resp.Data.Code != component.SUCCESS {
			fmt.Printf("Failed to create post %d: %s\n", i, resp.Data.MessageCode)
			return
		}
		var row seedPostRow
		db.Connect().Table("posts").
			Select("id").
			Where("topic_id = ?", 1).
			Order("id desc").
			Limit(1).
			Scan(&row)
		postIds = append(postIds, row.Id)
	}
	fmt.Printf("Created %d direct replies\n", len(postIds))

	// 回复第 2 条回复（回复中的回复，验证引用链）
	resp := api.CreatePost(component.BetterRequest[api.CreatePostReq]{
		UserId: userIds[2],
		Params: api.CreatePostReq{TopicId: 1, Content: "确实，一刷手机 25 分钟就没了……", ReplyToPostId: postIds[1]},
	})
	if resp.Data.Code != component.SUCCESS {
		fmt.Printf("Failed to create nested reply: %s\n", resp.Data.MessageCode)
		return
	}
	fmt.Println("Created reply-of-reply")

	// 5. Topic 2 追加一条回复
	resp = api.CreatePost(component.BetterRequest[api.CreatePostReq]{
		UserId: userIds[1],
		Params: api.CreatePostReq{TopicId: 2, Content: "学到了，composable 命名规范很重要。"},
	})
	if resp.Data.Code != component.SUCCESS {
		fmt.Printf("Failed to create topic2 reply: %s\n", resp.Data.MessageCode)
		return
	}
	fmt.Println("Created topic2 reply")

	fmt.Println("Seed demo data completed.")
}

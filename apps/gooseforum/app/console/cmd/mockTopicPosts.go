package cmd

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "mock-posts",
		Short: "Mock topic posts",
		RunE:  runMockTopicPosts,
	}
	cmd.Flags().Uint64("topic-id", 0, "Only mock posts for the specified topic")
	cmd.Flags().Int("count", 0, "Post count for --topic-id mode")
	cmd.Flags().Int("topics", 1000, "How many latest topics to scan when --topic-id is empty")
	cmd.Flags().Int("max-per-topic", 5, "Random upper bound per topic when --topic-id is empty")
	appendCommand(cmd)
}

// runMockTopicPosts 批量创建 mock 楼层。任一 CreatePost 失败使命令返回非零退出状态,
// 不能用正常退出掩盖部分或全部失败(issue #1284)。
func runMockTopicPosts(cmd *cobra.Command, args []string) error {
	fmt.Println("Starting to mock posts...")
	topicID, _ := cmd.Flags().GetUint64("topic-id")
	count, _ := cmd.Flags().GetInt("count")
	topicLimit, _ := cmd.Flags().GetInt("topics")
	maxPerTopic, _ := cmd.Flags().GetInt("max-per-topic")

	if topicLimit <= 0 {
		topicLimit = 1000
	}
	if maxPerTopic <= 0 {
		maxPerTopic = 5
	}

	// 1. Get the latest published topics.
	topicList, err := topics.GetLatestPublished(topicLimit)
	if err != nil {
		return fmt.Errorf("load topics: %w", err)
	}
	if topicID > 0 {
		topic := topics.Get(topicID)
		if topic.Id == 0 {
			return fmt.Errorf("topic %d not found", topicID)
		}
		topicList = []*topics.Entity{&topic}
		if count <= 0 {
			count = 60
		}
	}
	fmt.Printf("Found %d topics\n", len(topicList))

	if len(topicList) == 0 {
		return errors.New("no topics found")
	}

	// 2. Get all users
	userList := users.All()
	userIds := lo.Map(userList, func(u *users.EntityComplete, _ int) uint64 {
		return u.Id
	})

	if len(userIds) == 0 {
		return errors.New("no users found, please create users first")
	}
	fmt.Printf("Found %d users\n", len(userIds))

	// 3. Loop topics and generate posts.
	created, failed := 0, 0
	for i, topic := range topicList {
		postCount := count
		if topicID == 0 {
			// rand.Intn 可能为 0；至少建一条，否则 --max-per-topic 1 时整条
			// 命令一条也不建却声称成功（issue #1284）。
			postCount = rand.Intn(maxPerTopic) + 1
		}

		for j := range postCount {
			// Pick random user
			userId := lo.Sample(userIds)

			req := component.BetterRequest[api.CreatePostReq]{
				Params: api.CreatePostReq{
					TopicId: topic.Id,
					Content: fmt.Sprintf("Mock post content %d-%d for topic %d. This is generated for local topic timeline testing.", i, j, topic.Id),
				},
				UserId: userId,
			}
			resp := api.CreatePost(req)

			if resp.Data.Code != component.SUCCESS {
				failed++
				fmt.Printf("Error creating post for topic %d: %s\n", topic.Id, resp.Data.MessageCode)
			} else {
				created++
			}
		}
	}

	fmt.Printf("Mock posts generation done: created=%d failed=%d topics=%d\n", created, failed, len(topicList))
	if created == 0 {
		// 未指定 --max-per-topic 时每条话题至少建一条，故 0 条只可能来自全部失败。
		return fmt.Errorf("created no mock posts (per-topic upper bound %d)", maxPerTopic)
	}
	if failed > 0 {
		return fmt.Errorf("%d mock posts failed to create", failed)
	}
	return nil
}

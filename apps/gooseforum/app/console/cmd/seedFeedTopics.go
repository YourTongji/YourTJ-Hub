package cmd

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/spf13/cobra"
)

var feedImagePool = []string{
	"/file/img/2026/08/06/c8583c27-9bba-47c4-97e0-a4d02ebee28e.webp",
	"/file/img/2026/08/06/92a4b0ed-258a-4d63-be1e-6dc7ca1a42c2.webp",
	"/file/img/2026/08/06/63814598-8d5b-4741-8192-0d22ea91c12d.webp",
}

var feedTitlePool = []string{
	"大家平时都用什么工具管理知识碎片？",
	"从零搭建个人博客的踩坑记录",
	"周末爬山拍的风景，分享一下",
	"推荐一个超好用的命令行工具",
	"上班摸鱼时都在看什么网站？",
	"分享一下你的桌面配置",
	"新买的机械键盘开箱",
	"如何高效阅读技术文档？",
	"你们公司的代码规范是怎样的？",
	"第一次参加开源项目的感受",
	"咖啡重度患者的日常",
	"最近在追什么剧？安利一下",
	"装机配置求点评",
	"程序员应该学点设计吗？",
	"远程办公一年后的感想",
	"今天遇到的最离谱的 bug",
	"有什么适合通勤的耳机推荐？",
	"写论文的时候你们怎么保持专注？",
	"晒晒你最近拍的照片",
	"Vim 还是 Emacs，来辩！",
	"你们如何备份自己的数据？",
	"周末一个人会做什么？",
	"推荐一本最近读的好书",
	"代码重构的必要性和时机",
	"你的第一台电脑是什么？",
	"如何优雅地拒绝需求？",
	"健身房打卡第 30 天",
	"前端工程化的一些思考",
	"大家的内存条都多大？",
	"分享一个冷知识",
	"你理想中的工作环境",
	"独立开发者的日常开销",
	"面试官问过最刁钻的问题",
	"把博客迁移到静态站点了",
	"深夜写代码时的音乐推荐",
	"猫和狗你选哪个？",
	"你的键盘灯光是怎么配的？",
	"用了十年电脑的经验之谈",
	"如何快速学习一门新语言？",
	"最近入坑了摄影",
	"你们的项目怎么管理技术债？",
	"通勤时间你们都在做什么？",
	"今天食堂的菜还不错",
	"聊聊你所在城市的程序员圈子",
	"如何保持代码整洁？",
	"显示器支架值不值得买？",
	"你们团队的周会开多久？",
	"介绍一个你离不开的小工具",
	"年终总结怎么写才有亮点？",
	"多屏办公的布局分享",
}

var feedBodyPool = []string{
	"最近一直在琢磨这件事，想听听大家的看法。\n\n先说下我的观点：工具只是辅助，关键还是养成习惯。用了很多年才发现，适合自己的才是最好的。\n\n欢迎评论区交流～",
	"踩了无数坑之后终于跑通了，简单记录一下过程。\n\n有几个细节特别容易出错：环境变量、权限配置、还有版本兼容性。遇到问题先查日志，往往能少走很多弯路。",
	"趁着周末天气不错出去走了一圈，随手拍了几张。\n\n山上的空气是真的好，视野也开阔。平时写代码写多了，出来透透气感觉整个人都清爽了。",
	"想安利给所有人！这个工具太方便了，之前一直用别的方案，换过来之后效率提升明显。\n\n配置起来也很简单，几分钟就能上手，强烈推荐试试。",
	"今天摸鱼的时候刷到一个很有意思的网站，内容质量意外地高。\n\n收藏夹里又多了几个常去的站点，摸鱼也要有品位嘛。",
	"折腾了好几天终于把桌面整理成理想中的样子。\n\n简洁清爽才是王道，东西少了心情也好。顺便问下大家有什么收纳小技巧？",
	"刚到的快递，迫不及待开箱了。\n\n手感比想象中好，键帽质感也不错。办公室打字应该会很舒服，晚上回去好好调教一下。",
	"很多人读技术文档都是从头读到尾，其实效率并不高。\n\n我的方法是先看目录和示例，遇到问题再深入查细节，有目的地阅读效率高很多。",
	"最近和同事聊起代码规范的话题，发现每家公司差别还挺大的。\n\n好的规范应该能让新人快速上手，而不是束缚老手。大家有什么想法？",
	"第一次给开源项目提 PR 还有点紧张，不过维护者人很好。\n\n整个过程学到了不少东西：代码风格、提交规范、还有沟通方式。推荐大家都去试试。",
}

func init() {
	cmd := &cobra.Command{
		Use:   "seed-feed-topics",
		Short: "Seed 80 feed topics with random image layouts for feed testing",
		Run:   runSeedFeedTopics,
	}
	appendCommand(cmd)
}

// 随机图片布局：约 30% 无图、35% 单图、35% 多图（2-3 张）
func pickImageCount(rng *rand.Rand) int {
	roll := rng.Intn(100)
	if roll < 30 {
		return 0
	}
	if roll < 65 {
		return 1
	}
	return 2 + rng.Intn(2)
}

func buildFeedContent(rng *rand.Rand, imageCount int) string {
	body := feedBodyPool[rng.Intn(len(feedBodyPool))]
	if imageCount == 0 {
		return body
	}
	// 从图池随机选 N 张（顺序随机），以 markdown 图片行追加到正文
	pool := append([]string(nil), feedImagePool...)
	rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	content := body + "\n\n"
	for i := 0; i < imageCount; i++ {
		content += fmt.Sprintf("![img](%s)\n\n", pool[i%len(pool)])
	}
	return content
}

func runSeedFeedTopics(cmd *cobra.Command, args []string) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	userPool := []uint64{1, 2, 3, 4}
	created := 0

	for i := 0; i < 80; i++ {
		imageCount := pickImageCount(rng)
		content := buildFeedContent(rng, imageCount)
		title := feedTitlePool[rng.Intn(len(feedTitlePool))]
		userID := userPool[rng.Intn(len(userPool))]

		// 最近 30 天内随机时间
		createdAt := time.Now().Add(-time.Duration(rng.Intn(30*24)) * time.Hour).Add(-time.Duration(rng.Intn(3600)) * time.Minute)
		lastPostedAt := createdAt.Add(time.Duration(rng.Intn(48)) * time.Hour)

		// 首楼帖子
		post := posts.Entity{
			TopicId:         0, // Create 后回填
			PostNo:          1,
			UserId:          userID,
			Content:         content,
			RenderedHTML:    markdown2html.PostMarkdownToHTML(content),
			RenderedVersion: markdown2html.GetPostVersion(),
			CreatedAt:       createdAt,
			UpdatedAt:       createdAt,
		}
		if err := posts.Create(&post); err != nil {
			fmt.Printf("post create failed #%d: %v\n", i, err)
			return
		}

		// 主题
		excerpt := markdown2html.ExtractDescription(content, 200)
		topic := topics.Entity{
			Title:          title,
			CategoryIds:    []uint64{1},
			UserId:         userID,
			Status:         1,
			ProcessStatus:  0,
			ViewCount:      uint64(rng.Intn(2000)),
			PostCount:      1,
			ReplyCount:     uint64(rng.Intn(20)),
			PostSeq:        1,
			FirstPostId:    post.Id,
			LastPostId:     post.Id,
			LastPostedAt:   &lastPostedAt,
			Excerpt:        excerpt,
			FirstImageURL:  markdown2html.ExtractFirstImageURL(content),
			ImageUrls:      markdown2html.ExtractImageURLs(content),
			Posters:        []topics.Poster{{UserID: userID}},
			CreatedAt:      createdAt,
			UpdatedAt:      lastPostedAt,
		}
		if err := topics.Create(&topic); err != nil {
			fmt.Printf("topic create failed #%d: %v\n", i, err)
			return
		}
		post.TopicId = topic.Id
		if err := posts.Save(&post); err != nil {
			fmt.Printf("post topic ref failed #%d: %v\n", i, err)
			return
		}
		created++
	}

	fmt.Printf("seeded %d feed topics (restart server to refresh list cache)\n", created)
}

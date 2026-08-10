package llmsservice

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/localcache"
	"github.com/leancodebox/GooseForum/app/cacheconfig"
	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/leancodebox/GooseForum/app/models/hotdataserve"
	"github.com/leancodebox/GooseForum/app/service/urlconfig"
	"gorm.io/gorm"
)

var (
	ErrUnavailable  = errors.New("llms export unavailable")
	ErrTopicMissing = errors.New("llms topic missing")

	// 内部 sentinel：构建已达上限/超时。调用方应截断并成功返回（保证结果能被缓存，
	// 否则每次请求都会无界重建，反而放大 DoS）。
	errTopicLimit  = errors.New("llms: topic count limit reached")
	errOutputLimit = errors.New("llms: output size limit reached")
	errBuildBudget = errors.New("llms: build budget exceeded")
)

const (
	topicBatchSize = 200
	postBatchSize  = 500

	// buildFull 的硬上限，防止单次请求无界扫描全部主题/回复。
	// 主题数对齐 sitemap 的 GetLatestPublished(5000)。
	fullMaxTopics   = 5000
	fullMaxBytes    = 8 << 20 // 8 MiB
	fullBuildBudget = 30 * time.Second
)

var (
	projectionCache    = localcache.Cache[string]{MaxEntries: cacheconfig.Current().SEOXML}
	projectionCacheTTL = 10 * time.Second
)

func BuildIndex(host string) (string, error) {
	settings := hotdataserve.GetPostingSettingsConfigCache().LLMS
	if !settings.Enabled {
		return "", ErrUnavailable
	}
	baseURL := normalizeBaseURL(host)
	return cached("index:"+baseURL, baseURL != "", func() (string, error) {
		return buildIndex(baseURL, settings.Files)
	})
}

func BuildFull(host string) (string, error) {
	settings := hotdataserve.GetPostingSettingsConfigCache().LLMS
	if !settings.Enabled || !settings.FullText {
		return "", ErrUnavailable
	}
	baseURL := normalizeBaseURL(host)
	return cached("full:"+baseURL, baseURL != "", func() (string, error) {
		return buildFull(baseURL)
	})
}

func BuildTopic(host string, topicID uint64) (string, error) {
	settings := hotdataserve.GetPostingSettingsConfigCache().LLMS
	if !settings.Enabled || !settings.Files || topicID == 0 {
		return "", ErrUnavailable
	}
	baseURL := normalizeBaseURL(host)
	key := fmt.Sprintf("topic:%d:%s", topicID, baseURL)
	return cached(key, baseURL != "", func() (string, error) {
		topic, err := topics.GetPublished(topicID)
		if errors.Is(err, gorm.ErrRecordNotFound) || topic.Id == 0 {
			return "", ErrTopicMissing
		}
		if err != nil {
			return "", err
		}
		var builder strings.Builder
		if err := appendTopicDocument(&builder, baseURL, &topic, 1, fullMaxBytes); err != nil {
			if errors.Is(err, errOutputLimit) {
				writeTruncationNote(&builder, 1, fullMaxBytes)
				return builder.String(), nil
			}
			return "", err
		}
		return builder.String(), nil
	})
}

func ClearCache() {
	projectionCache.Clear()
}

func cached(key string, enabled bool, loader func() (string, error)) (string, error) {
	if !enabled {
		return loader()
	}
	return projectionCache.GetOrLoadE(key, loader, projectionCacheTTL)
}

func buildIndex(baseURL string, filesEnabled bool) (string, error) {
	var builder strings.Builder
	writeSiteHeader(&builder)
	builder.WriteString("## Topics\n\n")
	// 主题数同样受 fullMaxTopics 约束，超限静默截断（与 sitemap 的 5000 行为对齐）。
	err := forEachPublishedTopic(fullMaxTopics, func(topic *topics.Entity) error {
		path := urlconfig.PostDetail(topic.Id)
		if filesEnabled {
			path = urlconfig.PostMarkdown(topic.Id)
		}
		builder.WriteString("- [")
		builder.WriteString(escapeLinkLabel(topic.Title))
		builder.WriteString("](")
		builder.WriteString(baseURL + path)
		builder.WriteString(")")
		if description := topicDescription(topic); description != "" {
			builder.WriteString(": ")
			builder.WriteString(description)
		}
		builder.WriteByte('\n')
		return nil
	})
	if err != nil && !errors.Is(err, errTopicLimit) {
		return "", err
	}
	return builder.String(), nil
}

func buildFull(baseURL string) (string, error) {
	return buildFullWithLimits(baseURL, fullMaxTopics, fullMaxBytes, fullBuildBudget)
}

// buildFullWithLimits 以主题数/字节/时长三重上限构建全文导出；maxBytes<=0 表示不设字节上限。
// 达到任一上限时以「成功 + 截断注释」返回（结果可被 10s 缓存吸收，singleflight 合并并发重建），
// 避免单次无界扫描打满 DB/CPU/内存，也避免把局部数据问题放大为全站不可用。
func buildFullWithLimits(baseURL string, maxTopics int, maxBytes int64, budget time.Duration) (string, error) {
	var builder strings.Builder
	writeSiteHeader(&builder)
	deadline := time.Now().Add(budget)
	var truncatedReason string
	err := forEachPublishedTopic(maxTopics, func(topic *topics.Entity) error {
		if !time.Now().Before(deadline) {
			truncatedReason = "build budget"
			return errBuildBudget
		}
		if maxBytes > 0 && int64(builder.Len()) >= maxBytes {
			truncatedReason = "output size"
			return errOutputLimit
		}
		err := appendTopicDocument(&builder, baseURL, topic, 2, maxBytes)
		if errors.Is(err, ErrTopicMissing) {
			// 单个首帖异常的主题只影响自身：跳过并继续，而不是让整份 llms-full.txt 失败。
			return nil
		}
		if errors.Is(err, errOutputLimit) {
			truncatedReason = "output size"
			return errOutputLimit
		}
		return err
	})
	if err != nil && !errors.Is(err, errTopicLimit) && !errors.Is(err, errOutputLimit) && !errors.Is(err, errBuildBudget) {
		return "", err
	}
	if truncatedReason == "" {
		switch {
		case errors.Is(err, errTopicLimit):
			truncatedReason = "topic count"
		case errors.Is(err, errOutputLimit):
			truncatedReason = "output size"
		case errors.Is(err, errBuildBudget):
			truncatedReason = "build budget"
		}
	}
	if truncatedReason != "" {
		writeTruncationNote(&builder, maxTopics, maxBytes)
	}
	return builder.String(), nil
}

func writeTruncationNote(builder *strings.Builder, maxTopics int, maxBytes int64) {
	fmt.Fprintf(builder, "\n\n<!-- llms-full.txt truncated: export limited to %d topics and %d bytes. -->\n", maxTopics, maxBytes)
}

// appendTopicDocument 把单个主题及其正常回复写入 builder。maxBytes<=0 表示不设字节上限。
// 正文用 markdown 代码围栏包裹：降低作者注入标题/伪 Source 块污染文档结构的影响，
// 同时保留文本原样（text/plain|markdown 下无脚本执行风险）。
func appendTopicDocument(builder *strings.Builder, baseURL string, topic *topics.Entity, topicHeadingLevel int, maxBytes int64) error {
	postsBatch, err := posts.GetNormalByTopicPostNoAfter(topic.Id, 0, postBatchSize)
	if err != nil {
		return err
	}
	if len(postsBatch) == 0 || postsBatch[0].PostNo != 1 {
		return ErrTopicMissing
	}

	heading := strings.Repeat("#", topicHeadingLevel)
	subheading := heading + "#"
	replyHeading := subheading + "#"
	builder.WriteString(heading + " " + singleLine(topic.Title) + "\n\n")
	builder.WriteString("Source: [View topic](" + baseURL + urlconfig.PostDetail(topic.Id) + ")\n\n")
	if categories := categoryNames(topic.CategoryIds); len(categories) > 0 {
		builder.WriteString("Categories: " + strings.Join(categories, ", ") + "\n\n")
	}

	afterPostNo := uint64(0)
	wroteRepliesHeading := false
	for {
		for _, post := range postsBatch {
			if post == nil {
				continue
			}
			// 逐条回复写前检查字节上限：单个超大主题也能在导出中途被截断，避免内存被打满。
			if maxBytes > 0 && int64(builder.Len())+int64(len(post.Content))+16 > maxBytes {
				return errOutputLimit
			}
			if post.PostNo == 1 {
				builder.WriteString(subheading + " Original post\n\n")
			} else {
				if !wroteRepliesHeading {
					builder.WriteString(subheading + " Replies\n\n")
					wroteRepliesHeading = true
				}
				builder.WriteString(fmt.Sprintf("%s Reply %d\n\n", replyHeading, post.PostNo))
			}
			builder.WriteString("```markdown\n")
			builder.WriteString(strings.TrimSpace(post.Content))
			builder.WriteString("\n```\n\n")
			afterPostNo = post.PostNo
		}
		if len(postsBatch) < postBatchSize {
			break
		}
		postsBatch, err = posts.GetNormalByTopicPostNoAfter(topic.Id, afterPostNo, postBatchSize)
		if err != nil {
			return err
		}
	}
	builder.WriteString("---\n\n")
	return nil
}

func forEachPublishedTopic(limit int, visit func(*topics.Entity) error) error {
	afterID := uint64(0)
	visited := 0
	for {
		batch, err := topics.GetPublishedAfterID(afterID, topicBatchSize)
		if err != nil {
			return err
		}
		for _, topic := range batch {
			if topic == nil {
				continue
			}
			if limit > 0 && visited >= limit {
				return errTopicLimit
			}
			visited++
			if err := visit(topic); err != nil {
				return err
			}
			afterID = topic.Id
		}
		if len(batch) < topicBatchSize {
			return nil
		}
	}
}

func writeSiteHeader(builder *strings.Builder) {
	settings := hotdataserve.GetSiteSettingsConfigCache()
	name := singleLine(settings.SiteName)
	if name == "" {
		name = "YourTJHub"
	}
	builder.WriteString("# " + name + "\n\n")
	if description := singleLine(settings.SiteDescription); description != "" {
		builder.WriteString("> " + description + "\n\n")
	}
}

func topicDescription(topic *topics.Entity) string {
	parts := make([]string, 0, 2)
	if categories := categoryNames(topic.CategoryIds); len(categories) > 0 {
		parts = append(parts, "Categories: "+strings.Join(categories, ", "))
	}
	if excerpt := truncateRunes(singleLine(topic.Excerpt), 200); excerpt != "" {
		parts = append(parts, excerpt)
	}
	return strings.Join(parts, ". ")
}

func categoryNames(ids []uint64) []string {
	categoryMap := hotdataserve.CategoryMap()
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		if category := categoryMap[id]; category != nil {
			if name := singleLine(category.Name); name != "" {
				names = append(names, name)
			}
		}
	}
	return names
}

func normalizeBaseURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}

func singleLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func escapeLinkLabel(value string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "[", "\\[", "]", "\\]")
	return replacer.Replace(singleLine(value))
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if limit <= 0 || len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "..."
}

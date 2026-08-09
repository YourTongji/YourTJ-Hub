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
)

const (
	topicBatchSize = 200
	postBatchSize  = 500
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
		if err := appendTopicDocument(&builder, baseURL, &topic, 1); err != nil {
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
	err := forEachPublishedTopic(func(topic *topics.Entity) error {
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
	if err != nil {
		return "", err
	}
	return builder.String(), nil
}

func buildFull(baseURL string) (string, error) {
	var builder strings.Builder
	writeSiteHeader(&builder)
	err := forEachPublishedTopic(func(topic *topics.Entity) error {
		return appendTopicDocument(&builder, baseURL, topic, 2)
	})
	if err != nil {
		return "", err
	}
	return builder.String(), nil
}

func appendTopicDocument(builder *strings.Builder, baseURL string, topic *topics.Entity, topicHeadingLevel int) error {
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
			if post.PostNo == 1 {
				builder.WriteString(subheading + " Original post\n\n")
			} else {
				if !wroteRepliesHeading {
					builder.WriteString(subheading + " Replies\n\n")
					wroteRepliesHeading = true
				}
				builder.WriteString(fmt.Sprintf("%s Reply %d\n\n", replyHeading, post.PostNo))
			}
			builder.WriteString(strings.TrimSpace(post.Content))
			builder.WriteString("\n\n")
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

func forEachPublishedTopic(visit func(*topics.Entity) error) error {
	afterID := uint64(0)
	for {
		batch, err := topics.GetPublishedAfterID(afterID, topicBatchSize)
		if err != nil {
			return err
		}
		for _, topic := range batch {
			if topic == nil {
				continue
			}
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

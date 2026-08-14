package searchservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/meilisearch/meilisearch-go"
)

const (
	// ghostFetchBatchSize 是拉取索引文档 ID 的分页大小。
	ghostFetchBatchSize = 1000
	// ghostDeleteBatchSize 是批量删除幽灵文档的批大小。
	ghostDeleteBatchSize = 1000
)

// ghostIndex 是幽灵文档清理所需的 Meilisearch 索引操作子集，便于单测注入。
type ghostIndex interface {
	GetDocuments(param *meilisearch.DocumentsQuery, resp *meilisearch.DocumentsResult) error
	DeleteDocuments(identifiers []string, opts *meilisearch.DocumentOptions) (*meilisearch.TaskInfo, error)
}

// ghostRevalidator 在删除幽灵文档前复核索引文档 ID 在数据库中的最新状态，
// 用于关闭 rebuild snapshot 与线上事件处理器之间的竞态（PR #151 review P1）：
// 返回 keep=true 表示该文档当前仍应存在于索引，不得删除。
// 复核遇到错误时应返回 keep=true（宁可不删，也不误删有效文档）。
type ghostRevalidator func(id string) (keep bool, err error)

// cleanupGhostDocuments 删除索引中数据库已不存在的文档（幽灵文档），
// 使 rebuild 后索引与数据库集合一致。want 为 rebuild snapshot 阶段应存在于
// 索引中的文档 ID（字符串形式）集合；revalidate 非 nil 时，每个删除候选在
// 入队删除前都会按数据库最新状态复核，跳过 snapshot 之后新创建或恢复为
// 可索引的文档（这些文档的事件处理器 upsert 可能晚于 snapshot 到达）。
// 返回实际入队删除的文档 ID 列表；调用方应在其后对列表执行 replay 复核，
// 恢复删除入队期间重新变为可索引的文档（见各 Build*Index 的实现）。
func cleanupGhostDocuments(index ghostIndex, want map[string]struct{}, revalidate ghostRevalidator) ([]string, error) {
	indexed, err := fetchIndexDocumentIDs(index)
	if err != nil {
		return nil, err
	}
	ghosts := diffGhostIDs(indexed, want)
	var deleted []string
	for start := 0; start < len(ghosts); start += ghostDeleteBatchSize {
		end := start + ghostDeleteBatchSize
		if end > len(ghosts) {
			end = len(ghosts)
		}
		batch := ghosts[start:end]
		if revalidate != nil {
			kept := batch[:0]
			for _, id := range batch {
				keep, err := revalidate(id)
				if err != nil {
					return nil, fmt.Errorf("revalidate ghost document %s: %w", id, err)
				}
				if keep {
					continue
				}
				kept = append(kept, id)
			}
			batch = kept
		}
		if len(batch) == 0 {
			continue
		}
		if _, err := index.DeleteDocuments(batch, nil); err != nil {
			return nil, fmt.Errorf("delete ghost documents: %w", err)
		}
		deleted = append(deleted, batch...)
	}
	return deleted, nil
}

// fetchIndexDocumentIDs 分页拉取索引中的全部文档 ID（字符串形式）。
// 索引不存在时返回空列表（无文档可清理），避免空库重建失败。
func fetchIndexDocumentIDs(index ghostIndex) ([]string, error) {
	var ids []string
	offset := int64(0)
	for {
		resp := &meilisearch.DocumentsResult{}
		if err := index.GetDocuments(&meilisearch.DocumentsQuery{
			Offset: offset,
			Limit:  ghostFetchBatchSize,
			Fields: []string{"id"},
		}, resp); err != nil {
			var meiliErr *meilisearch.Error
			if errors.As(err, &meiliErr) && meiliErr.StatusCode == http.StatusNotFound {
				return nil, nil
			}
			return nil, fmt.Errorf("list index documents: %w", err)
		}
		for _, hit := range resp.Results {
			id, err := hitDocumentID(hit)
			if err != nil {
				return nil, err
			}
			ids = append(ids, id)
		}
		// 以"短页"作为终止条件，不依赖服务端 Total 字段（更健壮）。
		if len(resp.Results) < ghostFetchBatchSize {
			break
		}
		offset += int64(len(resp.Results))
	}
	return ids, nil
}

// hitDocumentID 从文档命中中读取 id 字段（兼容数字与字符串两种形式）。
func hitDocumentID(hit meilisearch.Hit) (string, error) {
	raw, ok := hit["id"]
	if !ok || len(raw) == 0 {
		return "", fmt.Errorf("index document missing id field")
	}
	var num uint64
	if err := json.Unmarshal(raw, &num); err == nil {
		return strconv.FormatUint(num, 10), nil
	}
	var str string
	if err := json.Unmarshal(raw, &str); err != nil {
		return "", fmt.Errorf("decode index document id %s: %w", raw, err)
	}
	return str, nil
}

// diffGhostIDs 返回 indexed 中存在、但不在 want 中的文档 ID（幽灵文档）。
func diffGhostIDs(indexed []string, want map[string]struct{}) []string {
	ghosts := make([]string, 0, len(indexed))
	for _, id := range indexed {
		if _, ok := want[id]; !ok {
			ghosts = append(ghosts, id)
		}
	}
	return ghosts
}

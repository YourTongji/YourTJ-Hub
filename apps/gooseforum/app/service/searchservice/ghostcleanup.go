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

// cleanupGhostDocuments 删除索引中数据库已不存在的文档（幽灵文档），
// 使 rebuild 后索引与数据库集合一致。want 为数据库当前应存在于
// 索引中的文档 ID（字符串形式）集合；返回删除的幽灵文档数量。
func cleanupGhostDocuments(index ghostIndex, want map[string]struct{}) (int, error) {
	indexed, err := fetchIndexDocumentIDs(index)
	if err != nil {
		return 0, err
	}
	ghosts := diffGhostIDs(indexed, want)
	for start := 0; start < len(ghosts); start += ghostDeleteBatchSize {
		end := start + ghostDeleteBatchSize
		if end > len(ghosts) {
			end = len(ghosts)
		}
		if _, err := index.DeleteDocuments(ghosts[start:end], nil); err != nil {
			return 0, fmt.Errorf("delete ghost documents: %w", err)
		}
	}
	return len(ghosts), nil
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

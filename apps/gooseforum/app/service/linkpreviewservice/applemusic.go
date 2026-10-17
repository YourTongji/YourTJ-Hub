package linkpreviewservice

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

// Apple Music 走固定端点、只取白名单字段的结构化解析，不抓 music.apple.com 的
// 页面：Lookup 返回的 trackName / artistName / artworkUrl100 正是卡片要的「名称 /
// 艺术家 / 封面」，不需要从本地化文案里猜。
//
// 端点来自代码常量，绝不来自用户输入或页面里的 discovery URL；返回数据只解析白名单
// 字段，畸形或超大的响应由 safefetch 的响应体上限与 json 解析一并挡住。
const (
	appleMusicLookupEndpoint = "https://itunes.apple.com/lookup"
	appleMusicLookupHost     = "itunes.apple.com"
)

// appleMusicEntry 是 Lookup 结果里的单条记录，只保留卡片用得到的白名单字段。
type appleMusicEntry struct {
	WrapperType    string `json:"wrapperType"`
	Kind           string `json:"kind"`
	CollectionType string `json:"collectionType"`
	ArtistName     string `json:"artistName"`
	TrackName      string `json:"trackName"`
	CollectionName string `json:"collectionName"`
	ArtworkURL100  string `json:"artworkUrl100"`
}

type appleMusicLookup struct {
	ResultCount int               `json:"resultCount"`
	Results     []appleMusicEntry `json:"results"`
}

// resolveAppleMusic 返回第二个值为 false 时表示平台解析没能给出卡片：调用方继续
// 走通用 HTML 卡片，失败再回退原链接。
func (r *Resolver) resolveAppleMusic(ctx context.Context, target platformTarget, base Preview) (Preview, bool) {
	requestURL := appleMusicLookupEndpoint + "?id=" + url.QueryEscape(target.id)
	if target.storefront != "" {
		requestURL += "&country=" + url.QueryEscape(target.storefront)
	}
	result, err := r.fetcher.FetchJSON(ctx, requestURL)
	if err != nil || result.StatusCode < http.StatusOK || result.StatusCode >= http.StatusMultipleChoices {
		return Preview{}, false
	}
	// 端点是我们自己拼的，但跳转仍可能把它带离允许主机：落地不是 Apple 的 Lookup
	// 主机就当解析失败，通用卡片接手。
	if result.FinalURL == nil || normalizePlatformHost(result.FinalURL.Hostname()) != appleMusicLookupHost {
		return Preview{}, false
	}
	var payload appleMusicLookup
	if err := json.Unmarshal(result.Body, &payload); err != nil || payload.ResultCount < 1 {
		return Preview{}, false
	}
	entry, ok := selectAppleMusicEntry(payload.Results, target.resource)
	if !ok {
		return Preview{}, false
	}
	name := entry.CollectionName
	if target.resource == resourceSong {
		name = entry.TrackName
	}
	if name = cleanText(name, 200); name == "" {
		return Preview{}, false
	}
	preview := base
	preview.Status = StatusReady
	preview.Platform = string(platformAppleMusic)
	preview.ResourceType = string(target.resource)
	preview.Title = name
	preview.Author = cleanText(entry.ArtistName, 100)
	preview.ImageURL = appleArtworkURL(entry.ArtworkURL100)
	if target.resource == resourceSong {
		// 歌曲卡片把所属专辑放在描述行：加值信息且同属白名单字段。
		preview.Description = cleanText(entry.CollectionName, 400)
	}
	preview.FetchedAt = r.now().UTC()
	return preview, true
}

// selectAppleMusicEntry 只认与链接语义一致的记录：歌曲要 track/song，专辑要
// collection/Album。类型对不上就当作解析失败，绝不把专辑标成歌曲。
func selectAppleMusicEntry(entries []appleMusicEntry, resource resourceKind) (appleMusicEntry, bool) {
	for _, entry := range entries {
		switch resource {
		case resourceSong:
			if entry.WrapperType == "track" && entry.Kind == "song" {
				return entry, true
			}
		case resourceAlbum:
			if entry.WrapperType == "collection" && entry.CollectionType == "Album" {
				return entry, true
			}
		}
	}
	return appleMusicEntry{}, false
}

// appleArtworkURL 把 Lookup 的 100×100 封面换成 600×600（Apple 的尺寸段格式是
// {w}x{h}bb，其余部分不变）。换不掉就保留原图；不落在 mzstatic.com 的地址直接丢弃。
func appleArtworkURL(artwork string) string {
	upgraded := strings.Replace(artwork, "/100x100bb.", "/600x600bb.", 1)
	if !assetHostAllowed(upgraded, "mzstatic.com") {
		return ""
	}
	return upgraded
}

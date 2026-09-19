package campusservice

import (
	"context"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type MessageSummary struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Publisher   string `json:"publisher"`
	PublishedAt string `json:"publishedAt"`
}

type MessageLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type MessageDetail struct {
	MessageSummary
	Content string        `json:"content"`
	Links   []MessageLink `json:"links"`
}

func validMessageID(id string) bool {
	n, err := strconv.ParseUint(id, 10, 64)
	return err == nil && n > 0 && strconv.FormatUint(n, 10) == id
}

func messageID(value any) string {
	id := str(value)
	if n, ok := value.(float64); ok {
		if n > 1<<53 || math.Trunc(n) != n {
			return ""
		}
		id = strconv.FormatFloat(n, 'f', 0, 64)
	}
	if !validMessageID(id) {
		return ""
	}
	return id
}

func messageTime(value string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02"} {
		if v, err := time.ParseInLocation(layout, value, time.FixedZone("Asia/Shanghai", 8*3600)); err == nil {
			return v
		}
	}
	return time.Time{}
}

func messageSummaries(value any) []MessageSummary {
	result := []MessageSummary{}
	for _, r := range objects(obj(value)["list"]) {
		published := first(r, "publishTime", "createTime")
		if t := messageTime(published); !t.IsZero() {
			published = t.Format(time.RFC3339)
		}
		result = append(result, MessageSummary{messageID(r["id"]), str(r["title"]), str(r["createUser"]), published})
	}
	sort.SliceStable(result, func(i, j int) bool {
		return messageTime(result[i].PublishedAt).After(messageTime(result[j].PublishedAt))
	})
	return result
}

// Check the user's current list before fetching a detail: guessed IDs must not
// turn a school gateway's broad application permission into an enumeration API.
func (s *Service) Message(ctx context.Context, userID uint64, id string) (MessageDetail, error) {
	if !validMessageID(id) {
		return MessageDetail{}, ErrMessageNotFound
	}
	value, err := s.readPrivate(ctx, userID, func(c Credentials) (any, error) {
		list, e := s.provider.Data(ctx, "messages", c.Access)
		if e != nil {
			return nil, e
		}
		var summary *MessageSummary
		for _, item := range messageSummaries(list) {
			if item.ID == id {
				summary = &item
				break
			}
		}
		if summary == nil {
			return nil, ErrMessageNotFound
		}
		raw, e := s.provider.Message(ctx, id, c.Access)
		if e != nil {
			return nil, e
		}
		m, ok := raw.(map[string]any)
		if !ok {
			return nil, ErrUpstream
		}
		if m["id"] != nil && messageID(m["id"]) != id {
			return nil, ErrUpstream
		}
		content, links := messageContent(str(m["content"]))
		return MessageDetail{*summary, content, links}, nil
	})
	if err != nil {
		return MessageDetail{}, err
	}
	return value.(MessageDetail), nil
}

// School HTML is projected to readable text and explicit links. Rendering a
// message never executes its HTML or automatically loads third-party resources.
func messageContent(raw string) (string, []MessageLink) {
	links := []MessageLink{}
	root, err := html.Parse(strings.NewReader(raw))
	if err != nil {
		return "", links
	}
	var text strings.Builder
	seen := map[string]bool{}
	addLink := func(label, address string) {
		u, e := url.Parse(address)
		if e != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil || len(address) > 2048 || seen[address] || len(links) >= 20 {
			return
		}
		seen[address] = true
		label = strings.Join(strings.Fields(label), " ")
		if label == "" {
			label = u.Hostname()
		}
		links = append(links, MessageLink{label, u.String()})
	}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if text.Len() > 256<<10 {
			return
		}
		if n.Type == html.TextNode {
			text.WriteString(n.Data)
			return
		}
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "iframe", "object", "template", "noscript":
				return
			}
			if n.Data == "br" {
				text.WriteByte('\n')
			}
			if n.Data == "img" {
				alt, src := "查看原文图片", ""
				for _, a := range n.Attr {
					if a.Key == "src" {
						src = a.Val
					}
					if a.Key == "alt" && strings.TrimSpace(a.Val) != "" {
						alt = a.Val
					}
				}
				addLink(alt, src)
			}
		}
		start := text.Len()
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
		if n.Type == html.ElementNode {
			if n.Data == "a" {
				for _, a := range n.Attr {
					if a.Key == "href" {
						addLink(text.String()[start:], a.Val)
					}
				}
			}
			switch n.Data {
			case "p", "div", "li", "tr", "h1", "h2", "h3", "h4", "section", "blockquote":
				text.WriteByte('\n')
			}
		}
	}
	walk(root)
	lines := []string{}
	for _, line := range strings.Split(text.String(), "\n") {
		if line = strings.Join(strings.Fields(line), " "); line != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n\n"), links
}

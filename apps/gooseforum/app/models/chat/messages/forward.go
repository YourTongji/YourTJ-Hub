package messages

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const ForwardType int8 = 4
const MaxForwardMessages = 50
const MaxForwardBytes = 64 << 10
const MaxForwardDepth = 4

// ForwardedBundle is an immutable copy, not a grant to the source conversation.
// It deliberately carries no conversation/message IDs or private notes.
type ForwardedBundle struct {
	Version  int              `json:"version"`
	Messages []ForwardedEntry `json:"messages"`
}
type ForwardedEntry struct {
	SenderName string           `json:"senderName"`
	AvatarURL  string           `json:"avatarUrl,omitempty"`
	Content    string           `json:"content"`
	CreatedAt  string           `json:"createdAt"`
	MsgType    int8             `json:"msgType"`
	Forwarded  *ForwardedBundle `json:"forwarded,omitempty"`
}

func ParseForward(content string) *ForwardedBundle {
	if len(content) > MaxForwardBytes {
		return nil
	}
	var value ForwardedBundle
	count := 0
	if json.Unmarshal([]byte(content), &value) != nil || !validForward(&value, 1, &count) {
		return nil
	}
	return &value
}

// Bound the entire copied tree, including history cards, before any rendering
// or encoding. No nested node points back to a private source conversation.
func validForward(bundle *ForwardedBundle, depth int, count *int) bool {
	if bundle == nil || bundle.Version != 1 || depth > MaxForwardDepth || len(bundle.Messages) == 0 {
		return false
	}
	*count += len(bundle.Messages)
	if *count > MaxForwardMessages {
		return false
	}
	for _, entry := range bundle.Messages {
		if entry.MsgType == ForwardType {
			if !validForward(entry.Forwarded, depth+1, count) {
				return false
			}
		} else if entry.MsgType < 1 || entry.MsgType > 3 || entry.Forwarded != nil {
			return false
		}
	}
	return true
}

func (bundle *ForwardedBundle) Encode() (string, error) {
	count := 0
	if !validForward(bundle, 1, &count) {
		return "", errors.New("invalid or oversized forwarded messages")
	}
	raw, err := json.Marshal(bundle)
	if err != nil {
		return "", err
	}
	if len(raw) > MaxForwardBytes {
		return "", errors.New("invalid or oversized forwarded messages")
	}
	return string(raw), nil
}
func (bundle *ForwardedBundle) Text() string {
	var text strings.Builder
	text.WriteString("[Chat history]")
	for _, entry := range bundle.Messages {
		content := entry.Content
		if entry.Forwarded != nil {
			content = entry.Forwarded.Text()
		}
		fmt.Fprintf(&text, "\n%s: %s", entry.SenderName, content)
	}
	return text.String()
}
func DisplayContent(content string, msgType int8) string {
	if msgType != ForwardType {
		return content
	}
	if bundle := ParseForward(content); bundle != nil {
		return bundle.Text()
	}
	return "[Chat history unavailable]"
}

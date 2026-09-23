package postservice

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

// PostMention maps one accepted occurrence to a current, active numeric identity.
// A client can link this range without duplicating the server's Markdown parser.
type PostMention struct {
	markdown2html.MentionToken
	UserID uint64 `json:"userId"`
}

// ResolvePostMentions batches identity resolution across visible payload bodies.
// Callers must redact unavailable content before passing it here.
func ResolvePostMentions(contents []string) [][]PostMention {
	tokens := make([][]markdown2html.MentionToken, len(contents))
	names := make([]string, 0)
	seen := map[string]bool{}
	for i, content := range contents {
		tokens[i] = markdown2html.ExtractMentionTokens(content)
		for _, token := range tokens[i] {
			if !seen[token.Username] {
				seen[token.Username] = true
				names = append(names, token.Username)
			}
		}
	}
	targets := users.GetMentionTargetIds(names)
	result := make([][]PostMention, len(contents))
	for i, list := range tokens {
		result[i] = make([]PostMention, 0, len(list))
		for _, token := range list {
			if id := targets[token.Username]; id != 0 {
				result[i] = append(result[i], PostMention{MentionToken: token, UserID: id})
			}
		}
	}
	return result
}

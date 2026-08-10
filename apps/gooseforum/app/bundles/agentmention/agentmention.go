// Package agentmention extracts exact @username mention candidates from
// plain text. It is a pure, deterministic scanner with no I/O: callers decide
// which candidates actually resolve to known bot personas.
package agentmention

// Username grammar shared with the forum username policy.
const (
	MinLength   = 6
	MaxLength   = 32
	MaxMentions = 10
)

func isUsernameChar(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_' || c == '-'
}

// Find returns up to MaxMentions distinct @username candidates in text order.
//
// A candidate starts at '@' whose previous byte is not a username character
// (start of text counts as a boundary), continues while username characters
// follow, and must be between MinLength and MaxLength bytes. Matching is
// case-sensitive and exact; @User and @user are different candidates.
func Find(text string) []string {
	var candidates []string
	seen := make(map[string]struct{}, MaxMentions)
	for i := 0; i < len(text); {
		if text[i] != '@' {
			i++
			continue
		}
		if i > 0 && isUsernameChar(text[i-1]) {
			i++
			continue
		}
		j := i + 1
		for j < len(text) && isUsernameChar(text[j]) {
			j++
		}
		nameLen := j - (i + 1)
		if nameLen >= MinLength && nameLen <= MaxLength {
			name := text[i+1 : j]
			if _, ok := seen[name]; !ok {
				seen[name] = struct{}{}
				candidates = append(candidates, name)
				if len(candidates) >= MaxMentions {
					return candidates
				}
			}
		}
		i = j
	}
	return candidates
}

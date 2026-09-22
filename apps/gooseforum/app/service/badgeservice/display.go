package badgeservice

import "encoding/json"

const MaxDisplayBadges = 5

// DisplayBadgesFromList resolves only active, owned badges. An unset preference
// preserves existing profiles; an explicit empty JSON array hides every badge.
func DisplayBadgesFromList(owned []UserBadge, preference string) []UserBadge {
	result := make([]UserBadge, 0, MaxDisplayBadges)
	if preference == "" {
		for _, item := range owned {
			if item.IsEnabled {
				result = append(result, item)
			}
			if len(result) == MaxDisplayBadges {
				break
			}
		}
		return result
	}
	var codes []string
	if json.Unmarshal([]byte(preference), &codes) != nil {
		return result
	}
	seen := map[string]bool{}
	for _, code := range codes {
		if seen[code] {
			continue
		}
		seen[code] = true
		for _, badge := range owned {
			if badge.Code == code && badge.IsEnabled {
				result = append(result, badge)
				break
			}
		}
		if len(result) == MaxDisplayBadges {
			break
		}
	}
	return result
}

func ValidDisplayBadges(owned []UserBadge, codes []string) bool {
	if len(codes) > MaxDisplayBadges {
		return false
	}
	allowed := map[string]bool{}
	for _, badge := range owned {
		if badge.IsEnabled {
			allowed[badge.Code] = true
		}
	}
	seen := map[string]bool{}
	for _, code := range codes {
		if code == "" || seen[code] || !allowed[code] {
			return false
		}
		seen[code] = true
	}
	return true
}

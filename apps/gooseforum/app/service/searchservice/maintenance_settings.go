package searchservice

import (
	"context"
	"reflect"
	"slices"
	"time"

	"github.com/meilisearch/meilisearch-go"
)

// managedSettings is shared by CLI/startup/incremental index configuration and
// admin verification. Search-only helper fields and version metadata are never
// displayed by public searches. Custom ranking rules/synonyms stay untouched.
func managedSettings(name string) *meilisearch.Settings {
	switch name {
	case TopicIndex:
		return &meilisearch.Settings{
			SearchableAttributes: []string{"title", "searchContent"},
			FilterableAttributes: []string{"category", "topicType"},
			SortableAttributes:   []string{"createdAt", "updatedAt"},
			DisplayedAttributes:  []string{"id", "title"},
		}
	case UserIndex:
		return &meilisearch.Settings{
			SearchableAttributes: []string{"username", "nickname", "bio", "usernamePinyin", "usernameInitials", "nicknamePinyin", "nicknameInitials"},
			DisplayedAttributes:  []string{"id", "username", "nickname", "bio"},
		}
	case CategoryIndex:
		return &meilisearch.Settings{
			SearchableAttributes: []string{"name", "desc", "slug", "namePinyin", "nameInitials"},
			DisplayedAttributes:  []string{"id", "name", "slug"},
		}
	case CourseIndex:
		return &meilisearch.Settings{
			Pagination:           &meilisearch.Pagination{MaxTotalHits: maxCourseCandidates + 1},
			SearchableAttributes: []string{"name", "normalizedName", "primaryCode", "classCodes", "aliases", "instructors", "teacherName", "namePinyin", "nameInitials", "instructorSearch"},
			FilterableAttributes: []string{"department", "terms", "campus", "status"},
			SortableAttributes:   []string{"createdAt", "updatedAt"},
			DisplayedAttributes:  []string{"id", "primaryCode", "name", "department", "creditX10", "aliases", "teacherId", "teacherName", "instructors", "terms", "campus", "status"},
		}
	case WikiPageIndex:
		return &meilisearch.Settings{
			SearchableAttributes: []string{"title", "heading", "paragraph"},
			FilterableAttributes: []string{"pageId", "namespace", "isPublic"},
			DisplayedAttributes:  []string{"id", "pageId", "isPublic", "path", "title", "namespace", "heading", "anchor", "paragraph", "sortOrder"},
		}
	}
	return nil
}

func applyManagedSettings(ctx context.Context, index meilisearch.IndexManager, name string) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	task, err := index.UpdateSettingsWithContext(ctx, managedSettings(name))
	if err != nil {
		return err
	}
	return waitForTaskCheckedContext(ctx, index, task.TaskUID, 60*time.Second)
}

func equalAttributeSet(a, b []string) bool {
	a, b = slices.Clone(a), slices.Clone(b)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}

func settingsMatch(actual, expected *meilisearch.Settings) bool {
	if actual == nil || expected == nil {
		return false
	}
	// Searchable attribute order influences ranking. Other attribute lists are sets.
	return reflect.DeepEqual(actual.SearchableAttributes, expected.SearchableAttributes) &&
		equalAttributeSet(actual.DisplayedAttributes, expected.DisplayedAttributes) &&
		(expected.FilterableAttributes == nil || equalAttributeSet(actual.FilterableAttributes, expected.FilterableAttributes)) &&
		(expected.SortableAttributes == nil || equalAttributeSet(actual.SortableAttributes, expected.SortableAttributes)) &&
		(expected.Pagination == nil || (actual.Pagination != nil && actual.Pagination.MaxTotalHits >= expected.Pagination.MaxTotalHits))
}

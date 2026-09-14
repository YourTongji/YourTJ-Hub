package searchservice

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"gorm.io/gorm"
)

// Bump whenever a document's projection semantics change. Settings are checked
// independently; unmarked legacy documents have observed version 0.
const (
	topicProjectionVersion    = 1
	userProjectionVersion     = 1
	categoryProjectionVersion = 1
	courseProjectionVersion   = 1
	wikiProjectionVersion     = 1
)

func expectedProjectionVersion(name string) int {
	switch name {
	case TopicIndex:
		return topicProjectionVersion
	case UserIndex:
		return userProjectionVersion
	case CategoryIndex:
		return categoryProjectionVersion
	case CourseIndex:
		return courseProjectionVersion
	case WikiPageIndex:
		return wikiProjectionVersion
	}
	return 0
}

const maintenanceBatch = 100

var managedIndexes = []string{TopicIndex, UserIndex, CategoryIndex, CourseIndex, WikiPageIndex}

func managedIndex(name string) bool {
	for _, index := range managedIndexes {
		if name == index {
			return true
		}
	}
	return false
}

type projectionDocument map[string]json.RawMessage

func projectionJSON(value any) (projectionDocument, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var doc projectionDocument
	err = json.Unmarshal(raw, &doc)
	return doc, err
}

func topicProjection(ctx context.Context, topic *topics.Entity) (any, error) {
	if !isTopicPubliclySearchable(topic) {
		return nil, nil
	}
	post, err := posts.GetWithContext(ctx, topic.FirstPostId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		post, err = posts.GetFirstSearchPost(ctx, topic.Id)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return topicProjectionWithPost(topic, &post), nil
}

func topicProjectionWithPost(topic *topics.Entity, post *posts.Entity) any {
	if !isTopicPubliclySearchable(topic) {
		return nil
	}
	if post.Id == 0 || post.DeletedAt.Valid || post.ProcessStatus != posts.ProcessStatusNormal || post.VisibilityStatus != posts.VisibilityActive {
		return nil
	}
	return convertTopicToSearchDocument(topic, post)
}

func wikiProjection(ctx context.Context, page *wikiPages.Entity) ([]WikiPageDocument, error) {
	topic, err := topics.GetWithContext(ctx, page.TopicId)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !isTopicPubliclySearchable(&topic) {
		return nil, nil
	}
	return wikiPageDocuments(page), nil
}

// projectionPage delegates all SQL to the source owner's API. The cursor is
// always the last source row, including pages containing no visible documents.
func projectionPage(ctx context.Context, index string, cursor uint64) ([]projectionDocument, uint64, error) {
	docs := make([]projectionDocument, 0, maintenanceBatch)
	add := func(value any) error {
		if value == nil {
			return nil
		}
		doc, err := projectionJSON(value)
		if err == nil {
			docs = append(docs, doc)
		}
		return err
	}
	switch index {
	case CourseIndex:
		rows, err := course.ListCoursesAfterID(ctx, cursor, maintenanceBatch)
		if err != nil {
			return nil, cursor, err
		}
		visible := make([]course.Entity, 0, len(rows))
		for _, row := range rows {
			cursor = row.Id
			if shouldIndexCourse(row) {
				visible = append(visible, row)
			}
		}
		batch, err := convertCoursesToSearchDocuments(visible)
		if err != nil {
			return nil, cursor, err
		}
		for _, doc := range batch {
			if err := add(doc); err != nil {
				return nil, cursor, err
			}
		}
	case TopicIndex:
		rows, err := topics.ListSearchProjectionPage(ctx, cursor, maintenanceBatch)
		if err != nil {
			return nil, cursor, err
		}
		ids := make([]uint64, 0, len(rows))
		for _, row := range rows {
			if isTopicPubliclySearchable(&row) {
				ids = append(ids, row.FirstPostId)
			}
		}
		firstPosts, err := posts.GetSearchPosts(ctx, ids)
		if err != nil {
			return nil, cursor, err
		}
		for _, row := range rows {
			cursor = row.Id
			if !isTopicPubliclySearchable(&row) {
				continue
			}
			post, found := firstPosts[row.FirstPostId]
			var doc any
			if found {
				doc = topicProjectionWithPost(&row, &post)
			} else {
				doc, err = topicProjection(ctx, &row)
			}
			if err != nil {
				return nil, cursor, err
			}
			if err := add(doc); err != nil {
				return nil, cursor, err
			}
		}
	case UserIndex:
		rows, err := users.ListSearchProjectionPage(ctx, cursor, maintenanceBatch)
		if err != nil {
			return nil, cursor, err
		}
		for _, row := range rows {
			cursor = row.Id
			if shouldIndexUser(&row) {
				if err := add(convertUserToSearchDocument(&row)); err != nil {
					return nil, cursor, err
				}
			}
		}
	case CategoryIndex:
		rows, err := category.ListSearchProjectionPage(ctx, cursor, maintenanceBatch)
		if err != nil {
			return nil, cursor, err
		}
		for _, row := range rows {
			cursor = row.Id
			if err := add(convertCategoryToSearchDocument(&row)); err != nil {
				return nil, cursor, err
			}
		}
	case WikiPageIndex:
		rows, err := wikiPages.ListSearchProjectionPage(ctx, cursor, maintenanceBatch)
		if err != nil {
			return nil, cursor, err
		}
		ids := make([]uint64, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, row.TopicId)
		}
		visibility, err := topics.GetSearchTopics(ctx, ids)
		if err != nil {
			return nil, cursor, err
		}
		for _, row := range rows {
			cursor = row.Id
			topic, found := visibility[row.TopicId]
			if !found || !isTopicPubliclySearchable(&topic) {
				continue
			}
			batch := wikiPageDocuments(&row)
			for _, doc := range batch {
				if err := add(doc); err != nil {
					return nil, cursor, err
				}
			}
		}
	default:
		return nil, cursor, errors.New("unknown managed index")
	}
	return docs, cursor, nil
}

// currentProjection revalidates a candidate immediately before and after a
// repair/delete. A DB error must never be interpreted as permission to delete.
func currentProjection(ctx context.Context, index, id string) (projectionDocument, error) {
	var value any
	var err error
	idPart := id
	if index == WikiPageIndex {
		idPart = strings.SplitN(id, "-", 2)[0]
	}
	key, parseErr := strconv.ParseUint(idPart, 10, 64)
	if parseErr != nil || key == 0 {
		return nil, nil //nolint:nilerr // Malformed document IDs cannot belong to a source row; they are ghosts.
	}
	if strconv.FormatUint(key, 10) != idPart {
		return nil, nil
	}
	switch index {
	case CourseIndex:
		var row course.Entity
		row, err = course.GetSearchProjection(ctx, key)
		if err == nil && shouldIndexCourse(row) {
			value, err = convertCourseToSearchDocument(row)
		}
	case TopicIndex:
		var row topics.Entity
		row, err = topics.GetWithContext(ctx, key)
		if err == nil {
			value, err = topicProjection(ctx, &row)
		}
	case UserIndex:
		var row users.EntityComplete
		row, err = users.GetWithContext(ctx, key)
		if err == nil && shouldIndexUser(&row) {
			value = convertUserToSearchDocument(&row)
		}
	case CategoryIndex:
		var row category.Entity
		row, err = category.GetWithContext(ctx, key)
		if err == nil {
			value = convertCategoryToSearchDocument(&row)
		}
	case WikiPageIndex:
		var row wikiPages.Entity
		row, err = wikiPages.GetWithContext(ctx, key)
		if err == nil {
			var batch []WikiPageDocument
			batch, err = wikiProjection(ctx, &row)
			for _, doc := range batch {
				if doc.ID == id {
					value = doc
					break
				}
			}
		}
	default:
		return nil, errors.New("unknown managed index")
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil || value == nil {
		return nil, err
	}
	return projectionJSON(value)
}

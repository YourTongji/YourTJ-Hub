package searchservice

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/meilisearch/meilisearch-go"
)

// IndexCheck records observations over a bounded online scan, not a database
// snapshot. No document contents or example IDs leave the service boundary.
type IndexCheck struct {
	Index            string         `json:"index"`
	ExpectedVersion  int            `json:"expectedVersion"`
	ObservedVersions map[string]int `json:"observedVersions"`
	Expected         int            `json:"expected"`
	Indexed          int            `json:"indexed"`
	Missing          int            `json:"missing"`
	Extra            int            `json:"extra"`
	Outdated         int            `json:"outdated"`
	SettingsOK       bool           `json:"settingsOK"`
	Stable           bool           `json:"stable"`
	Complete         bool           `json:"complete"`
	CheckedAt        time.Time      `json:"checkedAt"`
}

type documentDigest struct {
	Hash    [32]byte
	Version string
}

func documentID(doc projectionDocument) (string, error) {
	raw, ok := doc["id"]
	if !ok {
		return "", errors.New("search document has no id")
	}
	var id string
	if json.Unmarshal(raw, &id) == nil {
		return id, nil
	}
	var number json.Number
	if err := json.Unmarshal(raw, &number); err != nil {
		return "", err
	}
	return number.String(), nil
}

// Arrays in these five projections contain unordered facets/aliases; canonical
// sorting prevents DB row order or Meili serialization from creating false drift.
func canonicalValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			v[key] = canonicalValue(item)
		}
	case []any:
		for i, item := range v {
			v[i] = canonicalValue(item)
		}
		slices.SortFunc(v, func(a, b any) int { aa, _ := json.Marshal(a); bb, _ := json.Marshal(b); return bytes.Compare(aa, bb) })
	}
	return value
}

func digestDocument(doc projectionDocument) (documentDigest, error) {
	raw, err := json.Marshal(doc)
	if err != nil {
		return documentDigest{}, err
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return documentDigest{}, err
	}
	raw, err = json.Marshal(canonicalValue(value))
	var version int
	_ = json.Unmarshal(doc["_projectionVersion"], &version)
	return documentDigest{Hash: sha256.Sum256(raw), Version: strconv.Itoa(version)}, err
}

func maintenancePause(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(100 * time.Millisecond):
		return nil
	}
}

type maintenanceProgress func(phase, index string, processed int) error

func scanIndexedDocuments(ctx context.Context, index meilisearch.IndexManager, progress maintenanceProgress, name string) (map[string]documentDigest, error) {
	docs := make(map[string]documentDigest)
	for offset := int64(0); ; {
		var page meilisearch.DocumentsResult
		if err := index.GetDocumentsWithContext(ctx, &meilisearch.DocumentsQuery{Offset: offset, Limit: maintenanceBatch}, &page); err != nil {
			return nil, err
		}
		for _, hit := range page.Results {
			// Decode RawMessage fields to preserve numeric uint64 identifiers.
			var doc projectionDocument
			raw, err := json.Marshal(hit)
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(raw, &doc); err != nil {
				return nil, err
			}
			id, err := documentID(doc)
			if err != nil {
				return nil, err
			}
			digest, err := digestDocument(doc)
			if err != nil {
				return nil, err
			}
			docs[id] = digest
		}
		offset += int64(len(page.Results))
		// Bound memory on this small deployment; never report a truncated scan as complete.
		if offset > 250000 {
			return nil, errors.New("index exceeds maintenance scan limit")
		}
		if err := progress("read_index", name, int(offset)); err != nil {
			return nil, err
		}
		if len(page.Results) == 0 || offset >= page.Total {
			break
		}
		if err := maintenancePause(ctx); err != nil {
			return nil, err
		}
	}
	return docs, nil
}

func checkManagedIndex(ctx context.Context, client meilisearch.ServiceManager, name string, progress maintenanceProgress) (*IndexCheck, error) {
	report := &IndexCheck{Index: name, ExpectedVersion: expectedProjectionVersion(name), ObservedVersions: map[string]int{}}
	index := client.Index(name)
	before, err := client.GetIndexWithContext(ctx, name)
	var apiError *meilisearch.Error
	if errors.As(err, &apiError) && apiError.MeilisearchApiError.Code == "index_not_found" {
		// An absent index is a complete observation of missing documents, not a
		// transport failure that should prevent checking the remaining indexes.
		for cursor := uint64(0); ; {
			batch, next, err := projectionPage(ctx, name, cursor)
			if err != nil {
				return nil, err
			}
			report.Expected += len(batch)
			if report.Expected > 250000 {
				return nil, errors.New("index exceeds maintenance scan limit")
			}
			if err := progress("compare", name, report.Expected); err != nil {
				return nil, err
			}
			if next == cursor {
				break
			}
			cursor = next
			if err := maintenancePause(ctx); err != nil {
				return nil, err
			}
		}
		_, endErr := client.GetIndexWithContext(ctx, name)
		var endAPIError *meilisearch.Error
		report.Stable = errors.As(endErr, &endAPIError) && endAPIError.MeilisearchApiError.Code == "index_not_found"
		if endErr != nil && !report.Stable {
			return nil, endErr
		}
		report.Missing = report.Expected
		report.CheckedAt = time.Now().UTC()
		return report, nil
	}
	if err != nil {
		return nil, err
	}
	beforeStats, err := index.GetStatsWithContext(ctx, &meilisearch.StatsParams{})
	if err != nil {
		return nil, err
	}
	settings, err := index.GetSettingsWithContext(ctx)
	if err != nil {
		return nil, err
	}
	report.SettingsOK = settingsMatch(settings, managedSettings(name))
	docs, err := scanIndexedDocuments(ctx, index, progress, name)
	if err != nil {
		return nil, err
	}
	report.Indexed = len(docs)
	for _, doc := range docs {
		report.ObservedVersions[doc.Version]++
	}
	for cursor := uint64(0); ; {
		batch, next, err := projectionPage(ctx, name, cursor)
		if err != nil {
			return nil, err
		}
		for _, doc := range batch {
			id, err := documentID(doc)
			if err != nil {
				return nil, err
			}
			expected, err := digestDocument(doc)
			if err != nil {
				return nil, err
			}
			report.Expected++
			if report.Expected > 250000 {
				return nil, errors.New("index exceeds maintenance scan limit")
			}
			if actual, ok := docs[id]; !ok {
				report.Missing++
			} else if actual.Hash != expected.Hash {
				report.Outdated++
			}
			delete(docs, id)
		}
		if err := progress("compare", name, report.Expected); err != nil {
			return nil, err
		}
		if next == cursor {
			break
		}
		cursor = next
		if err := maintenancePause(ctx); err != nil {
			return nil, err
		}
	}
	report.Extra = len(docs)
	after, err := client.GetIndexWithContext(ctx, name)
	if err != nil {
		return nil, err
	}
	afterStats, err := index.GetStatsWithContext(ctx, &meilisearch.StatsParams{})
	if err != nil {
		return nil, err
	}
	report.Stable = before.UpdatedAt.Equal(after.UpdatedAt) && !beforeStats.IsIndexing && !afterStats.IsIndexing &&
		beforeStats.NumberOfDocuments == int64(report.Indexed) && afterStats.NumberOfDocuments == int64(report.Indexed)
	report.Complete = report.Stable && report.SettingsOK && report.Missing == 0 && report.Extra == 0 && report.Outdated == 0
	report.CheckedAt = time.Now().UTC()
	return report, nil
}

func writeProjectionBatch(ctx context.Context, index meilisearch.IndexManager, docs []projectionDocument) error {
	if len(docs) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	// AddDocuments replaces complete documents, removing retired fields as well.
	task, err := index.AddDocumentsWithContext(ctx, docs, &meilisearch.DocumentOptions{PrimaryKey: strPtr("id")})
	if err != nil {
		return err
	}
	return waitForTaskCheckedContext(ctx, index, task.TaskUID, 60*time.Second)
}

func rebuildManagedIndex(ctx context.Context, client meilisearch.ServiceManager, name string, progress maintenanceProgress) error {
	index := client.Index(name)
	if err := progress("settings", name, 0); err != nil {
		return err
	}
	if err := applyManagedSettings(ctx, index, name); err != nil {
		return err
	}
	expected := make(map[string]struct{})
	for cursor := uint64(0); ; {
		batch, next, err := projectionPage(ctx, name, cursor)
		if err != nil {
			return err
		}
		// Wiki source rows can contain many paragraphs; still bound every Meili write.
		for start := 0; start < len(batch); start += maintenanceBatch {
			end := min(start+maintenanceBatch, len(batch))
			if err := writeProjectionBatch(ctx, index, batch[start:end]); err != nil {
				return err
			}
			if end < len(batch) {
				if err := maintenancePause(ctx); err != nil {
					return err
				}
			}
		}
		for _, doc := range batch {
			id, err := documentID(doc)
			if err != nil {
				return err
			}
			expected[id] = struct{}{}
		}
		if len(expected) > 250000 {
			return errors.New("index exceeds maintenance scan limit")
		}
		if err := progress("rebuild", name, len(expected)); err != nil {
			return err
		}
		if next == cursor {
			break
		}
		cursor = next
		if err := maintenancePause(ctx); err != nil {
			return err
		}
	}
	// Finish browsing before deleting so our own deletes cannot shift OFFSET.
	indexed, err := scanIndexedDocuments(ctx, index, progress, name)
	if err != nil {
		return err
	}
	removed := 0
	for id := range indexed {
		if _, ok := expected[id]; ok {
			continue
		}
		if err := progress("cleanup", name, removed); err != nil {
			return err
		}
		current, err := currentProjection(ctx, name, id)
		if err != nil {
			return err
		}
		if current != nil {
			continue
		} // inserted/restored after our source scan
		task, err := index.DeleteDocumentWithContext(ctx, id, nil)
		if err != nil {
			return err
		}
		if err := waitForTaskCheckedContext(ctx, index, task.TaskUID, 60*time.Second); err != nil {
			return err
		}
		// A restore concurrent with deletion must be replayed after the delete.
		current, err = currentProjection(ctx, name, id)
		if err != nil {
			return err
		}
		if current != nil {
			if err := writeProjectionBatch(ctx, index, []projectionDocument{current}); err != nil {
				return err
			}
		}
		removed++
		if err := maintenancePause(ctx); err != nil {
			return err
		}
	}
	return nil
}

func runManagedIndex(ctx context.Context, client meilisearch.ServiceManager, name, action string, progress maintenanceProgress) (*IndexCheck, error) {
	if action == "rebuild" {
		if err := rebuildManagedIndex(ctx, client, name, progress); err != nil {
			return nil, fmt.Errorf("rebuild %s: %w", name, err)
		}
	}
	return checkManagedIndex(ctx, client, name, progress)
}

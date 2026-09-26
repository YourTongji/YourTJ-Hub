package stickerservice

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"path"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/imagepolicy"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
)

const ImportMaxZipBytes = 32 << 20
const ImportMaxExpandedBytes = 64 << 20
const ImportMaxEntries = 500

var ErrInvalidZip = errors.New("invalid sticker ZIP")

type ImportResult struct {
	Imported int           `json:"imported"`
	Skipped  int           `json:"skipped"`
	Failed   []ImportIssue `json:"failed"`
}
type ImportIssue struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

func ImportPack(ctx context.Context, userID uint64, archive []byte) (ImportResult, error) {
	result := ImportResult{Failed: []ImportIssue{}}
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return result, ErrInvalidZip
	}
	remaining := int64(ImportMaxExpandedBytes)
	for index, entry := range reader.File {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		fail := func(reason string) {
			result.Failed = append(result.Failed, ImportIssue{Name: path.Base(entry.Name), Reason: reason})
		}
		if index >= ImportMaxEntries {
			result.Failed = append(result.Failed, ImportIssue{Name: "...", Reason: "tooManyFiles"})
			break
		}
		if entry.FileInfo().IsDir() {
			continue
		}
		base := path.Base(entry.Name)
		if strings.HasPrefix(base, ".") || strings.Contains(entry.Name, "__MACOSX") {
			result.Skipped++
			continue
		}
		if _, ok := imagepolicy.ContentTypeForFilename(base); !ok {
			result.Skipped++
			continue
		}
		// Check the advertised size before opening, and meter actual bytes as well.
		if entry.UncompressedSize64 > uint64(remaining) {
			fail("archiveTooLarge")
			break
		}
		if entry.UncompressedSize64 > filedata.MaxFileSize {
			fail("tooLarge")
			continue
		}
		stream, err := entry.Open()
		if err != nil {
			fail("entryOpenFailed")
			continue
		}
		limit := min(int64(filedata.MaxFileSize), remaining)
		data, err := io.ReadAll(io.LimitReader(stream, limit+1))
		_ = stream.Close()
		remaining -= int64(len(data))
		if remaining < 0 {
			fail("archiveTooLarge")
			break
		}
		if err != nil || len(data) > filedata.MaxFileSize {
			fail("tooLarge")
			continue
		}
		_, err = ImportImage(ctx, userID, data, base, StemName(base), "official", 0, false)
		if err != nil {
			reason := strings.SplitN(err.Error(), ":", 2)[0]
			switch reason {
			case "tooLarge", "invalidImage", "unusableName":
			default:
				reason = "saveFailed"
				slog.Warn("import sticker failed", "name", base, "error", err)
			}
			fail(reason)
			continue
		}
		result.Imported++
	}
	return result, nil
}

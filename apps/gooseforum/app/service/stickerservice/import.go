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

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/imagepolicy"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
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

// importEntry stores one image. It reports a duplicate as skipped and a
// failure as a stable reason; an empty reason means the image was added.
type importEntry func(ctx context.Context, name string, data []byte) (skipped bool, reason string)

// isZipArchive checks the local-file or empty-archive signature.
func isZipArchive(data []byte) bool {
	return bytes.HasPrefix(data, []byte("PK\x03\x04")) || bytes.HasPrefix(data, []byte("PK\x05\x06"))
}

// importUpload accepts either a ZIP archive or one image file. Anything else
// fails as an invalid ZIP, the format the endpoint documents.
func importUpload(ctx context.Context, fileName string, data []byte, handle importEntry) (ImportResult, error) {
	if isZipArchive(data) {
		return importArchive(ctx, data, handle)
	}
	base := path.Base(strings.ReplaceAll(fileName, `\`, "/"))
	if _, ok := imagepolicy.ContentTypeForFilename(base); !ok {
		return ImportResult{Failed: []ImportIssue{}}, ErrInvalidZip
	}
	result := ImportResult{Failed: []ImportIssue{}}
	if len(data) > filedata.MaxFileSize {
		result.Failed = append(result.Failed, ImportIssue{Name: base, Reason: "tooLarge"})
		return result, nil
	}
	recordImport(&result, base)(handle(ctx, base, data))
	return result, nil
}

func recordImport(result *ImportResult, name string) func(skipped bool, reason string) {
	return func(skipped bool, reason string) {
		switch {
		case reason != "":
			result.Failed = append(result.Failed, ImportIssue{Name: name, Reason: reason})
		case skipped:
			result.Skipped++
		default:
			result.Imported++
		}
	}
}

func importArchive(ctx context.Context, archive []byte, handle importEntry) (ImportResult, error) {
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
		recordImport(&result, base)(handle(ctx, base, data))
	}
	return result, nil
}

// ImportPack imports a ZIP archive into the default official pack.
func ImportPack(ctx context.Context, userID uint64, archive []byte) (ImportResult, error) {
	if !isZipArchive(archive) {
		return ImportResult{Failed: []ImportIssue{}}, ErrInvalidZip
	}
	return ImportOfficial(ctx, userID, sticker.PackOfficial, "pack.zip", archive)
}

// ImportOfficial adds a ZIP archive or one image to an official pack. Each
// image becomes an enabled sticker named after its file stem, appended after
// the pack's current stickers.
func ImportOfficial(ctx context.Context, userID uint64, pack, fileName string, data []byte) (ImportResult, error) {
	if pack == "" {
		pack = sticker.PackOfficial
	}
	conn := db.ConnectContext(ctx)
	if known, err := knownPackTx(conn, pack); err != nil {
		return ImportResult{Failed: []ImportIssue{}}, err
	} else if !known {
		return ImportResult{Failed: []ImportIssue{}}, ErrPackInvalid
	}
	order, err := sticker.MaxOfficialSortOrder(conn, pack)
	if err != nil {
		return ImportResult{Failed: []ImportIssue{}}, err
	}
	return importUpload(ctx, fileName, data, func(ctx context.Context, base string, data []byte) (bool, string) {
		order++
		if _, err := ImportImage(ctx, userID, data, base, StemName(base), pack, order, false); err != nil {
			reason := strings.SplitN(err.Error(), ":", 2)[0]
			switch reason {
			case "tooLarge", "invalidImage", "unusableName":
			default:
				reason = "saveFailed"
				slog.Warn("import sticker failed", "name", base, "error", err)
			}
			return false, reason
		}
		return false, ""
	})
}

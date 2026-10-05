package feedservice

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/feedconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
	"gorm.io/gorm"
)

var sampleQueue []feed.CandidateSample
var sampleBytes int

func captureCandidateSample(uid uint64, s *snapshot, pool []Candidate, cfg feedconfig.Config) {
	salted := sha256.Sum256([]byte(cfg.Salt + ":samples:" + s.ID))
	if !cfg.Metrics || float64(binary.BigEndian.Uint64(salted[:8])%10000)/10000 >= cfg.SampleRate {
		return
	}
	data, err := encodeServed(pool)
	if err != nil {
		dropped.Add(1)
		return
	}
	if len(data) > 32<<10 {
		dropped.Add(1)
		return
	}
	row := feed.CandidateSample{ID: s.ID, UserID: uid, Hash: s.Hash, Candidates: data, Seed: s.Seed, CreatedAt: s.Created, ExpiresAt: s.Created.Add(time.Duration(cfg.RetentionDays) * 24 * time.Hour)}
	queueMu.Lock()
	defer queueMu.Unlock()
	if sampleBytes+len(data) > 2<<20 {
		dropped.Add(1)
		return
	}
	sampleQueue = append(sampleQueue, row)
	sampleBytes += len(data)
}
func flushSample(ctx context.Context, cfg feedconfig.Config) error {
	queueMu.Lock()
	if !cfg.Metrics {
		sampleQueue = nil
		sampleBytes = 0
		queueMu.Unlock()
		return nil
	}
	if len(sampleQueue) == 0 {
		queueMu.Unlock()
		return nil
	}
	row := sampleQueue[0]
	sampleQueue[0] = feed.CandidateSample{}
	sampleQueue = sampleQueue[1:]
	sampleBytes -= len(row.Candidates)
	queueMu.Unlock()
	return db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := feed.LockOwnerTx(tx, row.UserID); err != nil {
			return err
		}
		return tx.Create(&row).Error
	})
}

type ReplayResult struct {
	Items              []Candidate `json:"items"`
	CompleteCandidates bool        `json:"completeCandidates"`
	Limitation         string      `json:"limitation"`
}

func ReplaySample(ctx context.Context, id string, w feedconfig.Weights) (ReplayResult, error) {
	var row feed.CandidateSample
	if err := db.ConnectContext(ctx).First(&row, "id = ? AND expires_at > ?", id, time.Now()).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return ReplayResult{}, err
		}
		var served feed.Serve
		if e := db.ConnectContext(ctx).First(&served, "id = ? AND expires_at > ?", id, time.Now()).Error; e != nil {
			return ReplayResult{}, e
		}
		items, e := decodeServed(served.Items)
		if e != nil {
			return ReplayResult{}, e
		}
		return ReplayResult{Items: items, CompleteCandidates: false, Limitation: "Only the recorded served page is available; no counterfactual recall or full candidate reranking is possible."}, nil
	}
	pool := []Candidate{}
	data := []byte(row.Candidates)
	if strings.HasPrefix(row.Candidates, "z1:") {
		encoded, e := base64.RawStdEncoding.DecodeString(row.Candidates[3:])
		if e != nil {
			return ReplayResult{}, e
		}
		reader, e := zlib.NewReader(bytes.NewReader(encoded))
		if e != nil {
			return ReplayResult{}, e
		}
		defer func() { _ = reader.Close() }()
		data, e = io.ReadAll(io.LimitReader(reader, 128<<10))
		if e != nil {
			return ReplayResult{}, e
		}
	}
	if err := json.Unmarshal(data, &pool); err != nil {
		return ReplayResult{}, err
	}
	return ReplayResult{Items: SelectCandidates(pool, w, row.Seed), CompleteCandidates: true, Limitation: "Replays the recorded finite candidate pool; historical visibility and causal effects are not recoverable."}, nil
}

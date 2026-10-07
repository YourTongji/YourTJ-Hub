package forum

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/feedconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/transform"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/feedservice"
	"github.com/gin-gonic/gin"
)

type FeedSessionResponse struct {
	ViewerID      uint64                  `json:"viewerId"`
	Topics        []TopicPayload          `json:"topics"`
	RemovedIDs    []uint64                `json:"removedIds"`
	SeenProofs    []feedservice.SeenProof `json:"seenProofs"`
	SnapshotID    string                  `json:"snapshotId"`
	Pagination    *PaginationPayload      `json:"pagination,omitempty"`
	Available     bool                    `json:"available"`
	SeenConfirmed bool                    `json:"seenConfirmed"`
}

func feedSessionCards(ctx context.Context, uid uint64, rows []topics.Entity) []TopicPayload {
	entities := make([]*topics.Entity, len(rows))
	for i := range rows {
		entities[i] = &rows[i]
	}
	return buildTrackedTopicPayloadsContext(ctx, uid, transform.Topics2VoContext(ctx, entities, hotdataserve.CategoryMap()))
}
func feedSessionFailure(c *gin.Context, err error) {
	status := http.StatusServiceUnavailable
	message := component.MessageOperationFailed
	if errors.Is(err, feedservice.ErrInvalidTrace) || errors.Is(err, feedservice.ErrInvalidCursor) {
		status = http.StatusBadRequest
		message = component.MessageRequestInvalidParams
	}
	if errors.Is(err, feedservice.ErrRateLimited) {
		status = http.StatusTooManyRequests
		c.Header("Retry-After", "5")
	}
	c.JSON(status, component.FailDataCode(message, nil))
}

func FeedRefresh(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 32<<10)
	var req struct {
		SeenPatches       []feedservice.SeenPatch `json:"seenPatches"`
		ReplaceSnapshotID string                  `json:"replaceSnapshotId"`
	}
	if c.ShouldBindJSON(&req) != nil || len(req.ReplaceSnapshotID) > 128 {
		feedSessionFailure(c, feedservice.ErrInvalidTrace)
		return
	}
	uid := component.LoginUserId(c)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 200*time.Millisecond)
	defer cancel()
	c.Request = c.Request.WithContext(ctx)
	if _, err := feedservice.CaptureEvents(ctx, uid, nil, req.SeenPatches); err != nil {
		feedSessionFailure(c, err)
		return
	}
	var response []byte
	_, err := feedservice.RefreshForYou(ctx, uid, req.ReplaceSnapshotID, func(page feedservice.Page) error {
		props := HomeProps{Sort: "for_you", Topics: feedSessionCards(ctx, uid, page.Topics)}
		c.Set("feed.contentAt", page.ContentAt)
		c.Set("feed.entryVariant", page.EntryVariant)
		if err := ctx.Err(); err != nil {
			return err
		}
		decorateFeedProps(c, &props, page.Items, page.Hash, page.Config)
		for i := range props.Topics {
			for _, item := range page.Items {
				if item.ID == props.Topics[i].ID {
					props.Topics[i].FeedReason = item.Reason
				}
			}
		}
		pagination := PaginationPayload{Page: 1, NextPage: 2, HasNext: page.NextCursor != ""}
		if pagination.HasNext {
			pagination.NextURL = "/?" + url.Values{"sort": {"for_you"}, "page": {"2"}, "cursor": {page.NextCursor}}.Encode()
		}
		if props.SeenProofs == nil {
			props.SeenProofs = []feedservice.SeenProof{}
		}
		var err error
		response, err = json.Marshal(component.SuccessData(FeedSessionResponse{ViewerID: uid, Topics: props.Topics, RemovedIDs: []uint64{}, SeenProofs: props.SeenProofs, SnapshotID: page.SnapshotID, Pagination: &pagination, Available: true, SeenConfirmed: true}))
		if err != nil {
			return err
		}
		return ctx.Err()
	})
	if err != nil {
		feedSessionFailure(c, err)
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", response)
}

func FeedReconcile(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4<<10)
	var req struct {
		TopicIDs []uint64 `json:"topicIds"`
	}
	if c.ShouldBindJSON(&req) != nil || len(req.TopicIDs) > 120 || len(req.TopicIDs) == 0 {
		feedSessionFailure(c, feedservice.ErrInvalidCursor)
		return
	}
	unique := map[uint64]bool{}
	for _, id := range req.TopicIDs {
		if id == 0 || unique[id] {
			feedSessionFailure(c, feedservice.ErrInvalidCursor)
			return
		}
		unique[id] = true
	}
	uid := component.LoginUserId(c)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 200*time.Millisecond)
	defer cancel()
	c.Request = c.Request.WithContext(ctx)
	// Reconciliation shares the bounded transport quota without recording a serve.
	if _, err := feedservice.CaptureEvents(ctx, uid, nil, nil); err != nil {
		feedSessionFailure(c, err)
		return
	}
	rows, removed, at, err := feedservice.Reconcile(ctx, uid, req.TopicIDs)
	if err != nil {
		feedSessionFailure(c, err)
		return
	}
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.Id)
	}
	var proofs []feedservice.SeenProof
	if feedconfig.Current().Enabled {
		proofs = feedservice.MintSeenProof(uid, ids, at)
	}
	if proofs == nil {
		proofs = []feedservice.SeenProof{}
	}
	cards := feedSessionCards(ctx, uid, rows)
	if ctx.Err() != nil {
		feedSessionFailure(c, ctx.Err())
		return
	}
	c.JSON(http.StatusOK, component.SuccessData(FeedSessionResponse{ViewerID: uid, Topics: cards, RemovedIDs: removed, SeenProofs: proofs, Available: feedconfig.Current().Enabled && feedconfig.RankReady()}))
}

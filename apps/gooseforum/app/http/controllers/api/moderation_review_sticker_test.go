package api

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
)

func TestModeratorsCannotReadOrDecidePersonalStickerReviews(t *testing.T) {
	queue := ModerationReviewQueue(component.BetterRequest[ReviewQueueReq]{Params: ReviewQueueReq{Kind: "sticker"}})
	if queue.Data.Code != component.FAIL || queue.Data.MessageCode != component.MessagePermissionDenied {
		t.Fatalf("sticker queue response = %#v, want permission denied", queue)
	}
	action := ModerationReviewAction(component.BetterRequest[ReviewActionReq]{Params: ReviewActionReq{Kind: "sticker", Id: 1, Approve: true}})
	if action.Data.Code != component.FAIL || action.Data.MessageCode != component.MessagePermissionDenied {
		t.Fatalf("sticker action response = %#v, want permission denied", action)
	}
}

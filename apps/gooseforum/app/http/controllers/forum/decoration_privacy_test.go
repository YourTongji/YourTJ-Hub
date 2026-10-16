package forum

import (
	"context"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/decorationservice"
)

func TestAnonymousDecorationMaskDoesNotDependOnZeroID(t *testing.T) {
	private := &decorationservice.Display{AvatarFrame: &decorationservice.Style{Code: "private"}}
	for _, author := range []TopicAuthorPayload{
		{ID: 42, Kind: "persona", Decoration: private},
		{ID: 42, PublicUID: "public-persona", Decoration: private},
		{ID: 0, Decoration: private},
	} {
		fillAuthorDecorations(context.Background(), []*TopicAuthorPayload{&author})
		if author.Decoration != nil {
			t.Fatalf("anonymous appearance leaked: %+v", author)
		}
	}
}

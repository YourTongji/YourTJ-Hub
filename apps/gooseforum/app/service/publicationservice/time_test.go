package publicationservice

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"testing"
	"time"
)

func TestFirstPublicStartsAtApprovalAndSurvivesRestore(t *testing.T) {
	created := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	approved := created.Add(30 * 24 * time.Hour)
	post := posts.Entity{CreatedAt: created}
	if !stampFirstPublic(&post, approved, false) || !post.FirstPublicAt.Equal(approved) || post.FirstPublicEstimated {
		t.Fatalf("approval clock %+v", post)
	}
	if stampFirstPublic(&post, approved.Add(24*time.Hour), false) || !post.FirstPublicAt.Equal(approved) {
		t.Fatal("restored/edited publication reset clock")
	}
	legacy := posts.Entity{CreatedAt: created}
	if stampFirstPublic(&legacy, approved, true) || !legacy.FirstPublicEstimated || !legacy.FirstPublicAt.Equal(created) {
		t.Fatal("legacy time claimed exact approval")
	}
}

package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
)

func TestPostWindowSkipsHiddenPendingRunsBeforeLimiting(t *testing.T) {
	conn, router := setupForumInteractionContractTest(t)
	author := createHTTPContractUser(t, conn, contractTestID())
	if err := conn.AutoMigrate(&rolePermissionRs.Entity{}); err != nil {
		t.Fatal(err)
	}
	admin := createContractAdmin(t, conn)
	topicID, firstID := contractTestID(), contractTestID()
	createContractPublishedTopic(t, conn, topicID, firstID, author.Id)
	for floor := uint64(2); floor <= 62; floor++ {
		state := posts.ProcessStatusPending
		if floor == 62 {
			state = posts.ProcessStatusNormal
		}
		p := posts.Entity{Id: contractTestID(), TopicId: topicID, PostNo: floor, UserId: author.Id, Content: "private-pending-body", ProcessStatus: state}
		if floor == 62 {
			p.Content = "public-reply"
		}
		if err := conn.Create(&p).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := conn.Model(&topics.Entity{}).Where("id = ?", topicID).Update("post_seq", 62).Error; err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, query, token string
		floors             []uint64
	}{
		{"reader forward", "afterPostNo=1", "", []uint64{62}},
		{"reader reverse", "beforePostNo=62", "", []uint64{1}},
		{"reader initial", "", "", []uint64{1, 62}},
		{"reader anchor", "anchorPostNo=62", "", []uint64{1, 62}},
		{"author", "afterPostNo=1&limit=2", contractSessionToken(t, author), []uint64{2, 3}},
		{"moderator", "afterPostNo=1&limit=2", contractSessionToken(t, admin), []uint64{2, 3}},
		{"author reverse", "beforePostNo=62&limit=2", contractSessionToken(t, author), []uint64{60, 61}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := fmt.Sprintf("/api/forum/posts/window?topicId=%d&%s", topicID, tc.query)
			rec := serveAuthSecurityJSON(router, http.MethodGet, path, "", tc.token)
			response := decodeContractEnvelope(t, rec)
			var window forum.PostWindowPayload
			if err := json.Unmarshal(response.Result, &window); err != nil {
				t.Fatal(err)
			}
			if tc.token == "" && strings.Contains(string(response.Result), "private-pending-body") {
				t.Fatal("pending content leaked")
			}
			if len(window.Posts) != len(tc.floors) {
				t.Fatalf("window floors=%+v cursors=%d/%d flags=%v/%v", window.Posts, window.BeforePostNo, window.AfterPostNo, window.HasBefore, window.HasAfter)
			}
			for i, floor := range tc.floors {
				if window.Posts[i].PostNo != floor {
					t.Fatalf("floor=%d want %d", window.Posts[i].PostNo, floor)
				}
			}
		})
	}
}

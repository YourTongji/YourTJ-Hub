package routes

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

func TestUserBlocksHTTPContract(t *testing.T) {
	conn, router := setupAccountContractTest(t)
	owner := createHTTPContractUser(t, conn, contractTestID())
	target := createHTTPContractUser(t, conn, contractTestID())
	other := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, owner)
	body := fmt.Sprintf(`{"targetUserId":%d,"blocked":true,"ownerId":%d}`, target.Id, other.Id)
	for range 2 {
		rec := serveJSON(router, "/api/user-block", body, token)
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"result":true`) {
			t.Fatalf("save: %d %s", rec.Code, rec.Body)
		}
	}
	for _, user := range []*users.EntityComplete{owner, target, other} {
		rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/user-blocks?ownerId="+fmt.Sprint(owner.Id), "", contractSessionToken(t, user))
		if rec.Code != 200 || rec.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatalf("read: %d %s", rec.Code, rec.Body)
		}
		visible := strings.Contains(rec.Body.String(), fmt.Sprintf(`"targetUserId":%d`, target.Id))
		if visible != (user.Id == owner.Id) {
			t.Fatalf("block list escaped owner: %s", rec.Body)
		}
	}
	assertInteractionUnauthenticated(t, router, "/api/user-block", body, "auth-required.json")
	for _, invalid := range []string{`{}`, `{"targetUserId":1}`, `{"targetUserId":1,"blocked":null}`, fmt.Sprintf(`{"targetUserId":%d,"blocked":true}`, owner.Id)} {
		rec := serveJSON(router, "/api/user-block", invalid, token)
		if rec.Code != 400 {
			t.Fatalf("invalid request accepted: %s => %d %s", invalid, rec.Code, rec.Body)
		}
	}
	rec := serveJSON(router, "/api/user-block", fmt.Sprintf(`{"targetUserId":%d,"blocked":false}`, target.Id), token)
	if rec.Code != 200 {
		t.Fatal(rec.Body.String())
	}
	if blocked, err := users.InteractionBlocked(owner.Id, target.Id); err != nil || blocked {
		t.Fatalf("unblock: %v %v", blocked, err)
	}
	if err := users.SetBlockedUser(owner.Id, target.Id, true); err != nil {
		t.Fatal(err)
	}
	if err := users.CloseAccount(target.Id); err != nil {
		t.Fatal(err)
	}
	if blocked, err := users.InteractionBlocked(owner.Id, target.Id); err != nil || blocked {
		t.Fatalf("closure: %v %v", blocked, err)
	}
}

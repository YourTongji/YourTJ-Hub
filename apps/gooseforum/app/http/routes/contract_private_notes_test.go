package routes

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

func TestPrivateNotesHTTPContract(t *testing.T) {
	conn, router := setupAccountContractTest(t)
	owner := createHTTPContractUser(t, conn, contractTestID())
	target := createHTTPContractUser(t, conn, contractTestID())
	other := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, owner)
	body := fmt.Sprintf(`{"targetUserId":%d,"note":"实验搭档","ownerId":%d}`, target.Id, other.Id)
	rec := serveJSON(router, "/api/user-note", body, token)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"result":true`) {
		t.Fatalf("save: %d %s", rec.Code, rec.Body)
	}
	for _, test := range []struct {
		token   string
		visible bool
	}{{token, true}, {contractSessionToken(t, target), false}, {contractSessionToken(t, other), false}} {
		rec = serveAuthSecurityJSON(router, http.MethodGet, "/api/user-notes?ownerId="+fmt.Sprint(owner.Id), "", test.token)
		if rec.Code != 200 || strings.Contains(rec.Body.String(), "实验搭档") != test.visible {
			t.Fatalf("private read: %d %s", rec.Code, rec.Body)
		}
		if rec.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal("private response can be cached")
		}
	}
	rec = serveAuthSecurityJSON(router, http.MethodGet, "/api/user-notes", "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated read: %d %s", rec.Code, rec.Body)
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "auth-required.json"))
	public := serveAuthSecurityJSON(router, http.MethodGet, fmt.Sprintf("/api/user-card?userId=%d", target.Id), "", token)
	if strings.Contains(public.Body.String(), "实验搭档") {
		t.Fatal("private note leaked into public card")
	}
	for _, invalid := range []string{`{}`, fmt.Sprintf(`{"targetUserId":%d,"note":null}`, target.Id), fmt.Sprintf(`{"targetUserId":%d,"note":"bad\nname"}`, target.Id), fmt.Sprintf(`{"targetUserId":%d,"note":%q}`, target.Id, strings.Repeat("字", 65)), fmt.Sprintf(`{"targetUserId":%d,"note":"self"}`, owner.Id), `{"targetUserId":999999999,"note":"missing"}`} {
		rec = serveJSON(router, "/api/user-note", invalid, token)
		if rec.Code != 400 {
			t.Fatalf("invalid accepted: %s => %d %s", invalid, rec.Code, rec.Body)
		}
	}
	assertInteractionUnauthenticated(t, router, "/api/user-note", body, "auth-required.json")
	rec = serveJSON(router, "/api/user-note", fmt.Sprintf(`{"targetUserId":%d,"note":""}`, target.Id), token)
	if rec.Code != 200 {
		t.Fatal(rec.Body.String())
	}
	notes, err := users.ListPrivateNotes(owner.Id)
	if err != nil || len(notes) != 0 {
		t.Fatalf("clear: %v %+v", err, notes)
	}
	if err = users.SetPrivateNote(owner.Id, target.Id, "A"); err != nil {
		t.Fatal(err)
	}
	if err = users.SetPrivateNote(other.Id, owner.Id, "B"); err != nil {
		t.Fatal(err)
	}
	if err = users.CloseAccount(owner.Id); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err = conn.Model(&users.PrivateNoteEntity{}).Where("owner_id = ? OR target_user_id = ?", owner.Id, owner.Id).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("closure left notes: %d %v", count, err)
	}
	if err = users.SetPrivateNote(other.Id, owner.Id, "resurrect"); err == nil {
		t.Fatal("closed target accepted note")
	}
}

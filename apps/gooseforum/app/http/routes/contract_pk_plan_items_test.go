package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	pkcontroller "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/pk"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
)

func TestPkPlanItemsHTTPContract(t *testing.T) {
	conn, router := setupPkPlansContractTest(t)
	group := router.Group("/api/pk", middleware.CSRFProtection, middleware.JWTAuthCheck)
	group.GET("plan-items", middleware.RateLimit(middleware.RateLimitPkPlans), pkAuthNoReq(pkcontroller.GetPlanItems))
	group.PUT("plan-items", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitPkPlans), pkAuthJsonReq(pkcontroller.PutPlanItem))
	group.DELETE("plan-items", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitPkPlans), pkAuthJsonReq(pkcontroller.DeletePlanItem))
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)
	fixture := pkPlansFixtureOf(t, "pk-plan-item-success.json")
	var item pk.PlanItem
	if err := json.Unmarshal(fixture.Data, &item); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(item.Payload)
	if err != nil {
		t.Fatal(err)
	}
	put := func(base int64) string {
		return fmt.Sprintf(`{"ownerId":999,"plan":%s,"baseRevision":%d}`, payload, base)
	}
	send := func(method, body string, want int) pkPlansEnvelope {
		recorder := serveAuthSecurityJSON(router, method, "/api/pk/plan-items", body, token)
		if recorder.Code != want {
			t.Fatalf("%s = %d: %s", method, recorder.Code, recorder.Body.String())
		}
		return decodePkPlansEnvelope(t, recorder)
	}
	got := send(http.MethodPut, put(0), 200)
	assertPkPlansFixture(t, got, fixture)
	var saved pk.PlanItem
	if err := json.Unmarshal(got.Data, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Revision != 1 || saved.Payload.Id != item.Payload.Id {
		t.Fatalf("%+v", saved)
	}
	assertPkPlansFixture(t, send(http.MethodGet, "", 200), pkPlansFixtureOf(t, "pk-plan-items-success.json"))
	assertPkPlansFixture(t, send(http.MethodPut, put(0), 409), pkPlansFixtureOf(t, "pk-plan-item-conflict.json"))
	send(http.MethodPut, put(1), 200)
	conflict := send(http.MethodPut, put(1), 409)
	if err := json.Unmarshal(conflict.Data, &saved); err != nil || saved.Revision != 2 {
		t.Fatalf("conflict lacks remote revision: %s %v", conflict.Data, err)
	}
	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		recorder := serveAuthSecurityJSON(router, method, "/api/pk/plans", pkPlansPutBody, token)
		if recorder.Code != 410 {
			t.Fatalf("legacy %s must retire: %d %s", method, recorder.Code, recorder.Body.String())
		}
	}
	send(http.MethodDelete, fmt.Sprintf(`{"planId":%q,"baseRevision":1}`, item.Payload.Id), 409)
	send(http.MethodDelete, fmt.Sprintf(`{"planId":%q,"baseRevision":2}`, item.Payload.Id), 200)
	send(http.MethodPut, put(2), 410)
	send(http.MethodPut, `{"plan":{}}`, 400)
	other := createHTTPContractUser(t, conn, contractTestID())
	otherToken := contractSessionToken(t, other)
	response := serveAuthSecurityJSON(router, http.MethodGet, "/api/pk/plan-items", "", otherToken)
	if string(decodePkPlansEnvelope(t, response).Data) != "[]" {
		t.Fatalf("cross-owner data: %s", response.Body.String())
	}
	anonymous := serveAuthSecurityJSON(router, http.MethodGet, "/api/pk/plan-items", "", "")
	if anonymous.Code != 401 {
		t.Fatalf("anonymous = %d", anonymous.Code)
	}
}

func TestPkPlanItemsLegacyNullArraysStayContractCompatible(t *testing.T) {
	conn, router := setupPkPlansContractTest(t)
	group := router.Group("/api/pk", middleware.CSRFProtection, middleware.JWTAuthCheck)
	group.GET("plan-items", pkAuthNoReq(pkcontroller.GetPlanItems))
	group.PUT("plan-items", middleware.CheckWritableAccount, pkAuthJsonReq(pkcontroller.PutPlanItem))
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)
	legacy := pk.ScheduleSnapshotEntity{UserId: user.Id, Plans: pk.PlanList{{Id: "legacy", Name: "Legacy"}}}
	if err := conn.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	for _, request := range []struct {
		method, body string
		status       int
	}{
		{http.MethodGet, "", 200},
		{http.MethodPut, `{"baseRevision":0,"plan":{"id":"legacy","name":"Local"}}`, 409},
	} {
		response := serveAuthSecurityJSON(router, request.method, "/api/pk/plan-items", request.body, token)
		if response.Code != request.status {
			t.Fatalf("status %d: %s", response.Code, response.Body.String())
		}
		envelope := decodePkPlansEnvelope(t, response)
		var plans []pk.PlanItem
		if request.method == http.MethodGet {
			if err := json.Unmarshal(envelope.Data, &plans); err != nil {
				t.Fatal(err)
			}
		} else {
			var item pk.PlanItem
			if err := json.Unmarshal(envelope.Data, &item); err != nil {
				t.Fatal(err)
			}
			plans = []pk.PlanItem{item}
		}
		if len(plans) != 1 || plans[0].Payload.StagedCourses == nil || plans[0].Payload.SelectedCourses == nil || plans[0].Payload.CustomEvents == nil {
			t.Fatalf("%s emitted null arrays: %s", request.method, envelope.Data)
		}
	}
}

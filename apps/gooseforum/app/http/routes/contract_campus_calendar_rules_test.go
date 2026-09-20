package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/ratelimit"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/calendaradjustment"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
)

func TestCampusCalendarRulesAdminLifecycle(t *testing.T) {
	db, router := setupAdminSiteContractTest(t)
	db.Where("page_type = ?", "campusCalendarAdjustments").Delete(&pageConfig.Entity{})
	t.Cleanup(func() { db.Where("page_type = ?", "campusCalendarAdjustments").Delete(&pageConfig.Entity{}) })
	router.Use(middleware.CSRFProtection)
	group := router.Group("/api/admin/campus", middleware.JWTAuthCheck, middleware.CheckWritableAccount, middleware.CheckPermission(permission.SiteManager))
	group.GET("/calendar-rules", api.CampusCalendarRules)
	group.POST("/calendar-rules", api.SaveCampusCalendarRules)
	group.POST("/calendar-rules/parse", api.ParseCampusCalendarRules)
	campusRoutes(router)
	const path = "/api/admin/campus/calendar-rules"
	user := createHTTPContractUser(t, db, contractTestID())
	token := contractSessionToken(t, user)
	for _, tc := range []struct {
		method, path, token string
		status              int
	}{{"GET", path, "", 401}, {"GET", path, token, 403}, {"POST", path, token, 403}, {"POST", path + "/parse", token, 403}} {
		w := serveAuthSecurityJSON(router, tc.method, tc.path, `{}`, tc.token)
		if w.Code != tc.status {
			t.Fatalf("guard %s %s = %d", tc.method, tc.path, w.Code)
		}
	}
	manager := createContractSiteManager(t, db)
	token = contractSessionToken(t, manager)
	settings, err := calendaradjustment.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	settings.Rules = calendaradjustment.Rules{Holidays: []calendaradjustment.Holiday{{Name: "国庆节", StartDate: "2026-10-01", EndDate: "2026-10-07"}}, Moves: []calendaradjustment.Move{{Name: "国庆补课", FromDate: "2026-10-06", ToDate: "2026-09-20"}}}
	body, err := json.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	w := serveAuthSecurityJSON(router, "POST", path, string(body), token)
	if w.Code != 200 {
		t.Fatalf("save %d %s", w.Code, w.Body.String())
	}
	w = serveAuthSecurityJSON(router, "POST", path, string(body), token)
	if w.Code != 409 {
		t.Fatal("stale revision overwrote settings")
	}
	current, err := calendaradjustment.Read(context.Background())
	if err != nil || len(current.Rules.Moves) != 1 {
		t.Fatal("rules were not persisted")
	}
	public := serveAuthSecurityJSON(router, "GET", "/api/campus/calendar-rules", "", contractSessionToken(t, user))
	if public.Code != 200 || public.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal("authenticated read failed")
	}
	var actual struct {
		Result calendaradjustment.Settings `json:"result"`
	}
	if json.Unmarshal(public.Body.Bytes(), &actual) != nil || actual.Result.Revision != current.Revision {
		t.Fatal("public rules differ")
	}
	current.Rules.Moves[0].ToDate = "2026-10-01"
	bad, err := json.Marshal(current)
	if err != nil {
		t.Fatal(err)
	}
	w = serveAuthSecurityJSON(router, "POST", path, string(bad), token)
	if w.Code != 400 {
		t.Fatal("holiday collision was accepted")
	}
	// Cookie-authenticated cross-site writes cannot change published dates.
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(string(body)))
	req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://evil.invalid")
	cross := httptest.NewRecorder()
	router.ServeHTTP(cross, req)
	if cross.Code != 403 {
		t.Fatal("cross-site save allowed")
	}
	// AI is an explicit draft operation: private campus data and config never enter its prompt.
	ratelimit.Default().Reset("ai.summary.global")
	t.Cleanup(func() { ratelimit.Default().Reset("ai.summary.global") })
	llm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if json.NewDecoder(r.Body).Decode(&request) != nil || r.URL.Path != "/chat/completions" || len(request.Messages) != 2 || request.Messages[1].Content != "通知年份：2026\n通知原文：\n国庆节通知" {
			t.Error("unexpected provider input")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"rules\":{\"holidays\":[],\"moves\":[]},\"warnings\":[\"请补充明确日期\"]}"}}]}`))
	}))
	defer llm.Close()
	persistHTTPContractConfig(t, db, pageConfig.AiSummarySettings, map[string]any{"enabled": false, "baseUrl": llm.URL, "model": "test-model", "globalPerMinute": 1})
	hotdataserve.ClearAiSummarySettingsConfigCache()
	w = serveAuthSecurityJSON(router, "POST", path+"/parse", `{"year":2026,"text":"国庆节通知"}`, token)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "请补充明确日期") {
		t.Fatalf("parse %d %s", w.Code, w.Body.String())
	}
	after, err := calendaradjustment.Read(context.Background())
	if err != nil || after.Revision != actual.Result.Revision {
		t.Fatal("parse changed published rules")
	}
	w = serveAuthSecurityJSON(router, "POST", path+"/parse", `{"year":2026,"text":"国庆节通知"}`, token)
	if w.Code != 429 {
		t.Fatal("AI did not share global quota")
	}
}

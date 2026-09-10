package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/gin-gonic/gin"
)

type materializeDeadlineWriter struct {
	*httptest.ResponseRecorder
	deadline time.Time
}

func (w *materializeDeadlineWriter) SetWriteDeadline(deadline time.Time) error {
	w.deadline = deadline
	return nil
}

func TestMaterializePkCalendarExtendsWriteDeadline(t *testing.T) {
	setupPkAdminTest(t)
	w := &materializeDeadlineWriter{ResponseRecorder: httptest.NewRecorder()}
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/admin/pk/materialize-calendar", nil)
	started := time.Now()
	// A missing local calendar keeps this test small; the HTTP execution budget
	// must already be established before touching the catalog.
	MaterializePkCalendar(component.BetterRequest[MaterializePkCalendarReq]{
		GinContext: ctx, Params: MaterializePkCalendarReq{Term: "121"},
	})
	remaining := w.deadline.Sub(started)
	if remaining < 2*time.Minute || remaining > 3*time.Minute {
		t.Fatalf("write deadline budget=%s; want bounded headroom beyond the two-minute transaction", remaining)
	}
}

package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/optlogger"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/pkservice"
)

type MaterializePkCalendarReq struct {
	Term     string `json:"term" validate:"required"`
	Audience string `json:"audience"`
}

type MaterializePkCalendarResult struct {
	CalendarId uint64 `json:"calendarId"`
	courseservice.MaterializeReport
}

const materializePkTimeout = 2 * time.Minute

// MaterializePkCalendar commits one local calendar before returning its report.
// Request cancellation rolls back the transaction; repeating a completed request
// is idempotent. Unlike a sync, it never requires or fetches an upstream cookie.
func MaterializePkCalendar(req component.BetterRequest[MaterializePkCalendarReq]) component.Response {
	audience, err := pkservice.ParseAudience(req.Params.Audience)
	if err != nil {
		return component.FailResponseError(err)
	}
	id, _, err := pkservice.ResolveSyncTermForAudience(audience, req.Params.Term)
	if err != nil {
		return component.FailResponseError(err)
	}
	ctx := context.Background()
	if req.GinContext != nil {
		ctx = req.GinContext.Request.Context()
		// The server defaults to a ten-second write deadline. Leave bounded
		// headroom to return either the committed report or a transaction timeout.
		err := http.NewResponseController(req.GinContext.Writer).SetWriteDeadline(time.Now().Add(materializePkTimeout + 10*time.Second))
		if err != nil && !errors.Is(err, http.ErrNotSupported) {
			return component.FailResponseError(err)
		}
	}
	ctx, cancel := context.WithTimeout(ctx, materializePkTimeout)
	defer cancel()
	report, err := courseservice.MaterializeFromPkForAudience(ctx, audience, []uint64{id})
	if err != nil {
		return component.FailResponseError(err)
	}
	optlogger.UserOptCode(req.UserId, optlogger.MaterializePk, id, "admin.opt.pk.materialized", optlogger.MessageParams{
		"coursesInserted": report.CoursesInserted, "coursesUpdated": report.CoursesUpdated,
		"offeringsInserted": report.OfferingsInserted, "offeringsUpdated": report.OfferingsUpdated,
	})
	return component.SuccessResponse(MaterializePkCalendarResult{CalendarId: id, MaterializeReport: *report})
}

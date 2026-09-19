package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/calendaradjustment"
	"github.com/gin-gonic/gin"
)

func calendarRulesFailure(c *gin.Context, err error) {
	status, code := http.StatusServiceUnavailable, "campus.rulesUnavailable"
	var params component.MessageParams
	switch {
	case errors.Is(err, calendaradjustment.ErrInvalid):
		status, code = http.StatusBadRequest, "campus.rulesInvalid"
		reason := strings.TrimPrefix(err.Error(), calendaradjustment.ErrInvalid.Error()+": ")
		if reason == calendaradjustment.ErrInvalid.Error() {
			reason = "规则信息无效，请检查日期与字段后重试。"
		}
		params = component.MessageParams{"reason": reason}
	case errors.Is(err, calendaradjustment.ErrConflict):
		status, code = http.StatusConflict, "campus.rulesChanged"
	case errors.Is(err, calendaradjustment.ErrAIUnavailable):
		code = "campus.rulesAIUnavailable"
	case errors.Is(err, calendaradjustment.ErrAIOutput):
		status, code = http.StatusBadGateway, "campus.rulesAIOutput"
	case errors.Is(err, calendaradjustment.ErrLimited):
		status, code = http.StatusTooManyRequests, "campus.rulesLimited"
		c.Header("Retry-After", "60")
	}
	c.JSON(status, component.FailDataCode(component.MessageCode(code), params))
}
func CampusCalendarRules(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	settings, err := calendaradjustment.Read(c.Request.Context())
	if err != nil {
		calendarRulesFailure(c, err)
		return
	}
	c.JSON(http.StatusOK, component.SuccessData(settings))
}
func SaveCampusCalendarRules(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 128<<10)
	var req calendaradjustment.Settings
	if c.ShouldBindJSON(&req) != nil {
		calendarRulesFailure(c, calendaradjustment.ErrInvalid)
		return
	}
	settings, err := calendaradjustment.Save(c.Request.Context(), req)
	if err != nil {
		calendarRulesFailure(c, err)
		return
	}
	c.JSON(http.StatusOK, component.SuccessData(settings))
}
func ParseCampusCalendarRules(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10)
	var req calendaradjustment.ParseRequest
	if c.ShouldBindJSON(&req) != nil {
		calendarRulesFailure(c, calendaradjustment.ErrInvalid)
		return
	}
	draft, err := calendaradjustment.Parse(c.Request.Context(), req)
	if err != nil {
		calendarRulesFailure(c, err)
		return
	}
	c.JSON(http.StatusOK, component.SuccessData(draft))
}

package api

import (
	"errors"
	"net/http"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/campus"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/campusservice"
	"github.com/gin-gonic/gin"
)

func campusFailure(c *gin.Context, err error) {
	status := http.StatusServiceUnavailable
	message := "campus.upstreamUnavailable"
	switch {
	case errors.Is(err, campusservice.ErrDisabled):
		message = "campus.disabled"
	case errors.Is(err, campusservice.ErrAuthorization):
		status = http.StatusConflict
		message = "campus.authorizationRequired"
	case errors.Is(err, campusservice.ErrFlow):
		status = http.StatusBadRequest
		message = "campus.authorizationExpired"
	case errors.Is(err, campusservice.ErrPermission):
		status = http.StatusConflict
		message = "campus.messageAuthorizationRequired"
	case errors.Is(err, campusservice.ErrMessageNotFound):
		status = http.StatusNotFound
		message = "campus.messageUnavailable"
	case errors.Is(err, campusservice.ErrConflict):
		status = http.StatusConflict
		message = "campus.identityUnavailable"
	case errors.Is(err, campus.ErrChanged):
		status = http.StatusConflict
		message = "campus.connectionChanged"
	}
	c.JSON(status, component.FailDataCode(component.MessageCode(message), nil))
}
func campusService(c *gin.Context) *campusservice.Service {
	c.Header("Cache-Control", "private, no-store")
	c.Header("Referrer-Policy", "no-referrer")
	s, e := campusservice.Default()
	if e != nil {
		campusFailure(c, e)
		return nil
	}
	return s
}
func CampusStatus(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	s, e := campusservice.Default()
	if errors.Is(e, campusservice.ErrDisabled) {
		c.JSON(200, component.SuccessData(campusservice.Status{}))
		return
	}
	if e != nil {
		campusFailure(c, e)
		return
	}
	v, e := s.Status(c.GetUint64("userId"), c.GetString("currentJti"))
	if e != nil {
		campusFailure(c, e)
		return
	}
	c.JSON(200, component.SuccessData(v))
}
func CampusStart(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	s := campusService(c)
	if s == nil {
		return
	}
	var req struct {
		Mode string `json:"mode"`
	}
	if c.ShouldBindJSON(&req) != nil {
		campusFailure(c, campusservice.ErrFlow)
		return
	}
	u, e := s.Start(c.GetUint64("userId"), c.GetString("currentJti"), req.Mode)
	if e != nil {
		campusFailure(c, e)
		return
	}
	c.JSON(200, component.SuccessData(gin.H{"url": u}))
}
func CampusCallback(c *gin.Context) {
	s := campusService(c)
	if s == nil {
		return
	}
	e := s.Callback(c.Request.Context(), c.GetUint64("userId"), c.GetString("currentJti"), c.Query("state"), c.Query("code"))
	destination := "/campus?authorization=ready"
	if e != nil {
		destination = "/campus?authorization=failed"
	}
	c.Redirect(http.StatusSeeOther, destination)
}
func CampusConfirm(c *gin.Context) {
	s := campusService(c)
	if s == nil {
		return
	}
	if e := s.Confirm(c.GetUint64("userId"), c.GetString("currentJti")); e != nil {
		campusFailure(c, e)
		return
	}
	c.JSON(200, component.SuccessData(nil))
}
func CampusUnbind(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	s := campusService(c)
	if s == nil {
		return
	}
	var req struct {
		Revision string `json:"revision"`
	}
	if c.ShouldBindJSON(&req) != nil || req.Revision == "" {
		campusFailure(c, campusservice.ErrFlow)
		return
	}
	if e := s.Unbind(c.Request.Context(), c.GetUint64("userId"), req.Revision); e != nil {
		campusFailure(c, e)
		return
	}
	c.JSON(200, component.SuccessData(nil))
}
func CampusDataset(c *gin.Context) {
	s := campusService(c)
	if s == nil {
		return
	}
	d, e := s.Dataset(c.Request.Context(), c.GetUint64("userId"), c.Param("dataset"))
	if e != nil {
		campusFailure(c, e)
		return
	}
	c.JSON(200, component.SuccessData(d))
}

func CampusMessage(c *gin.Context) {
	s := campusService(c)
	if s == nil {
		return
	}
	d, e := s.Message(c.Request.Context(), c.GetUint64("userId"), c.Param("messageId"))
	if e != nil {
		campusFailure(c, e)
		return
	}
	c.JSON(200, component.SuccessData(d))
}

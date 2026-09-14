package api

import (
	"net/http"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/statusservice"
	"github.com/gin-gonic/gin"
)

func ServerStatus(c *gin.Context) {
	period := c.DefaultQuery("range", "24h")
	serverPeriod := c.DefaultQuery("serverRange", "1h")
	c.Header("Cache-Control", "no-store")
	if !statusservice.ValidRange(period) || !statusservice.ValidServerRange(serverPeriod) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "result": nil, "messageCode": "status.invalidRange"})
		return
	}
	c.JSON(http.StatusOK, component.SuccessData(statusservice.Public(c.Request.Context(), period, serverPeriod)))
}

package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/profileprivacyservice"
	"github.com/gin-gonic/gin"
)

const maxPrivacySettingsBodyBytes = 4096

// GetPrivacySettings returns the caller's own profile privacy settings. There
// is no way to read another account's raw preferences.
func GetPrivacySettings(req component.BetterRequest[component.Null]) component.Response {
	req.GinContext.Header("Cache-Control", "private, no-store")
	settings, err := profileprivacyservice.Get(req.UserId)
	if err != nil {
		response := component.FailResponseCode(component.MessageOperationFailed, nil)
		response.Code = http.StatusServiceUnavailable
		return response
	}
	return component.SuccessResponse(settings)
}

// UpdatePrivacySettings applies a partial update. The contract declares
// additionalProperties: false, so unknown keys, nulls and non-string values
// are rejected before anything is written.
func UpdatePrivacySettings(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	userID := c.GetUint64("userId")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, component.FailDataCode(component.MessageAuthRequired, nil))
		return
	}
	update, ok := decodePrivacySettingsUpdate(http.MaxBytesReader(c.Writer, c.Request.Body, maxPrivacySettingsBodyBytes))
	if !ok {
		c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageRequestParseFailed, nil))
		return
	}
	settings, err := profileprivacyservice.Update(userID, update)
	switch {
	case errors.Is(err, users.ErrPrivacyAudience):
		c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageRequestInvalidParams, nil))
	case err != nil:
		c.JSON(http.StatusServiceUnavailable, component.FailDataCode(component.MessageOperationFailed, nil))
	default:
		c.JSON(http.StatusOK, component.SuccessData(settings))
	}
}

func decodePrivacySettingsUpdate(body io.Reader) (users.PrivacySettingsUpdate, bool) {
	var update users.PrivacySettingsUpdate
	var raw map[string]json.RawMessage
	decoder := json.NewDecoder(body)
	if err := decoder.Decode(&raw); err != nil || len(raw) == 0 {
		return update, false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return update, false
	}
	targets := map[string]**string{
		"profileTopics":   &update.ProfileTopics,
		"profileActivity": &update.ProfileActivity,
		"profileLikes":    &update.ProfileLikes,
		"followingList":   &update.FollowingList,
		"followersList":   &update.FollowersList,
	}
	for key, value := range raw {
		target, known := targets[key]
		var audience string
		if !known || json.Unmarshal(value, &audience) != nil || string(value) == "null" {
			return update, false
		}
		*target = &audience
	}
	return update, true
}

package api

import (
	"github.com/leancodebox/GooseForum/app/http/controllers/component"
	"github.com/leancodebox/GooseForum/app/models/forum/userSessions"
	"github.com/leancodebox/GooseForum/app/service/sessionservice"
)

// SessionVO is the session item returned to the client. The raw user agent
// is not exposed; the frontend renders a parsed device label instead.
type SessionVO struct {
	Id        uint64 `json:"id"`
	IpMasked  string `json:"ipMasked"`
	UserAgent string `json:"userAgent"`
	CreatedAt int64  `json:"createdAt"`
	ExpiresAt int64  `json:"expiresAt"`
	IsCurrent bool   `json:"isCurrent"`
}

func toSessionVO(entity userSessions.Entity, currentJti string) SessionVO {
	return SessionVO{
		Id:        entity.Id,
		IpMasked:  sessionservice.MaskIP(entity.Ip),
		UserAgent: entity.UserAgent,
		CreatedAt: entity.CreatedAt.UnixMilli(),
		ExpiresAt: entity.ExpiresAt.UnixMilli(),
		IsCurrent: entity.Jti == currentJti,
	}
}

// ListSessions 获取当前用户的登录会话列表（最新在前，标记当前会话）
func ListSessions(req component.BetterRequest[component.Null]) component.Response {
	userID := req.UserId
	entities, err := sessionservice.List(userID)
	if err != nil {
		return component.FailResponseCode(component.MessageSessionListFailed, nil)
	}
	currentJti := req.GinContext.GetString("currentJti")
	result := make([]SessionVO, 0, len(entities))
	for _, entity := range entities {
		result = append(result, toSessionVO(entity, currentJti))
	}
	return component.SuccessResponse(result)
}

type RevokeSessionReq struct {
	// binding:"required" only applies during Gin's binding phase; the route binds
	// non-strictly (UpButterReq), so validate:"required" is what actually rejects
	// malformed JSON / missing id / id 0 before the handler sees a zero value.
	Id uint64 `json:"id" binding:"required" validate:"required"`
}

// RevokeSession 吊销指定会话（当前会话不可吊销，避免自锁）
func RevokeSession(req component.BetterRequest[RevokeSessionReq]) component.Response {
	userID := req.UserId
	sessionID := req.Params.Id
	currentJti := req.GinContext.GetString("currentJti")

	entities, err := sessionservice.List(userID)
	if err != nil {
		return component.FailResponseCode(component.MessageSessionRevokeFailed, nil)
	}
	for _, entity := range entities {
		if entity.Id != sessionID {
			continue
		}
		if entity.Jti == currentJti {
			return component.FailResponseCode(component.MessageSessionCurrentNotRevocable, nil)
		}
		if err := sessionservice.RevokeByID(userID, sessionID); err != nil {
			return component.FailResponseCode(component.MessageSessionRevokeFailed, nil)
		}
		return component.SuccessResponseCode("会话已吊销", component.MessageSessionRevokeSuccess, nil)
	}
	return component.FailResponseCode(component.MessageSessionNotFound, nil)
}

// RevokeAllSessions 吊销该用户全部会话（含当前），并自增 TokenVersion 双保险
func RevokeAllSessions(req component.BetterRequest[component.Null]) component.Response {
	userID := req.UserId
	if err := sessionservice.RevokeAllAndInvalidate(userID); err != nil {
		return component.FailResponseCode(component.MessageSessionRevokeFailed, nil)
	}
	return component.SuccessResponseCode("已退出所有设备", component.MessageSessionRevokeAllSuccess, nil)
}

package view

import (
	"context"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/decorationservice"
	"time"
)

// UserInfoShow is the authenticated user summary exposed to pages and APIs.
type UserInfoShow struct {
	Decoration          *decorationservice.Display `json:"decoration,omitempty"`
	UserId              uint64                     `json:"userId,omitempty"`
	Username            string                     `json:"username"`
	Email               string                     `json:"email"`
	Nickname            string                     `json:"nickname"`
	Bio                 string                     `json:"bio"`
	Signature           string                     `json:"Signature"`
	Prestige            int64                      `json:"prestige"`
	AvatarUrl           string                     `json:"avatarUrl"`
	UserPoint           int64                      `json:"userPoint"`
	CreateTime          time.Time                  `json:"createTime"`
	CanAccessAdmin      bool                       `json:"canAccessAdmin"`
	IsActivated         int8                       `json:"isActivated"`
	ExternalInformation users.ExternalInformation  `json:"externalInformation"`
}

// BuildUserShow maps a user entity to the authenticated user summary payload.
func BuildUserShow(user users.EntityComplete) *UserInfoShow {
	return &UserInfoShow{
		Decoration:          decorationservice.Effective(context.Background(), user.Id),
		UserId:              user.Id,
		Username:            user.Username,
		Email:               user.Email,
		Nickname:            user.Nickname,
		Bio:                 user.Bio,
		Signature:           user.Signature,
		Prestige:            user.Prestige,
		AvatarUrl:           user.GetWebAvatarUrl(),
		CreateTime:          user.CreatedAt,
		CanAccessAdmin:      user.RoleId > 0,
		IsActivated:         user.IsActivated,
		ExternalInformation: user.ExternalInformation,
	}
}

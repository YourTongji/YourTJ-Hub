package api

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/buildinfo"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/eventbus"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jsonopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/llmprovider"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/randopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/ratelimit"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/securestore"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/datastruct"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/defaultconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/badges"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/dailyStats"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderators"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/optRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/role"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicCategoryIndex"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userActivities"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userBadges"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userStatistics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/badgeservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/contentdeleteservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/dataservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/eventhandlers"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/filemigrateservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/llmsservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/mailservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/optlogger"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/searchservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/storageservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/themeservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

type TrafficOverviewReq struct {
	StartDate string `json:"startDate"` // YYYY-MM-DD
	EndDate   string `json:"endDate"`   // YYYY-MM-DD
}

type DailyTraffic struct {
	Date       string `json:"date"`
	RegCount   int64  `json:"regCount"`
	TopicCount int64  `json:"topicCount"`
	ReplyCount int64  `json:"replyCount"`
}

func GetTrafficOverview(req component.BetterRequest[TrafficOverviewReq]) component.Response {
	startDate := req.Params.StartDate
	endDate := req.Params.EndDate

	if startDate == "" {
		startDate = time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}

	keys := []dailyStats.StatType{
		dailyStats.StatTypeRegCount,
		dailyStats.StatTypeTopicCount,
		dailyStats.StatTypeReplyCount,
	}

	stats, err := dailyStats.GetStatsInRange(keys, startDate, endDate)
	if err != nil {
		return component.FailResponseCode(component.MessageAdminStatsFetchFailed, nil)
	}

	// 按日期分组
	dailyMap := make(map[string]*DailyTraffic)

	// 初始化日期范围内的每一天
	curr, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)
	for !curr.After(end) {
		d := curr.Format("2006-01-02")
		dailyMap[d] = &DailyTraffic{Date: d}
		curr = curr.AddDate(0, 0, 1)
	}

	for _, s := range stats {
		dateStr := s.StatDate.Format("2006-01-02")
		if item, ok := dailyMap[dateStr]; ok {
			switch dailyStats.StatType(s.StatKey) {
			case dailyStats.StatTypeRegCount:
				item.RegCount = s.StatValue
			case dailyStats.StatTypeTopicCount:
				item.TopicCount = s.StatValue
			case dailyStats.StatTypeReplyCount:
				item.ReplyCount = s.StatValue
			}
		}
	}

	// 转换为数组并排序
	var result []*DailyTraffic
	curr, _ = time.Parse("2006-01-02", startDate)
	for !curr.After(end) {
		d := curr.Format("2006-01-02")
		result = append(result, dailyMap[d])
		curr = curr.AddDate(0, 0, 1)
	}

	return component.SuccessResponse(result)
}

type UserListReq struct {
	Username string `json:"username"`
	UserId   uint64 `json:"userId"`
	Email    string `json:"email"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
}

type UserItem struct {
	UserId         uint64                              `json:"userId"`
	Username       string                              `json:"username"`
	AvatarUrl      string                              `json:"avatarUrl"`
	Email          string                              `json:"email"`
	Status         int8                                `json:"status"`
	ActorType      int8                                `json:"actorType"`
	Validate       int8                                `json:"validate"`
	Prestige       int64                               `json:"prestige"`
	RoleList       []datastruct.Option[string, uint64] `json:"roleList"`
	RoleId         uint64                              `json:"roleId,omitempty"`
	CreateTime     string                              `json:"createTime"`
	LastActiveTime string                              `json:"lastActiveTime"`
	Badges         []badgeservice.UserBadge            `json:"badges"`
}

func UserList(req component.BetterRequest[UserListReq]) component.Response {
	var pageData = users.Page(users.PageQuery{
		Page:     req.Params.Page,
		PageSize: req.Params.PageSize,
		Username: req.Params.Username,
		UserId:   req.Params.UserId,
		Email:    req.Params.Email,
	})

	userIds := lo.Map(pageData.Data, func(item users.EntityComplete, _ int) uint64 {
		return item.Id
	})
	usList := userStatistics.GetByUserIds(userIds)
	usMap := lo.KeyBy(usList, func(v *userStatistics.Entity) uint64 {
		return v.UserId
	})
	roleEntityList := role.AllEffective()
	roleMap := lo.KeyBy(roleEntityList, func(v *role.Entity) uint64 {
		return v.Id
	})
	list := lo.Map(pageData.Data, func(t users.EntityComplete, _ int) UserItem {
		var roleList []datastruct.Option[string, uint64]
		if roleEntity, ok := roleMap[t.RoleId]; ok {
			roleList = append(roleList, datastruct.Option[string, uint64]{
				Name:  roleEntity.RoleName,
				Value: roleEntity.Id,
			})
		}
		LastActiveTime := t.CreatedAt.Format(time.RFC3339)
		if usItem, ok := usMap[t.Id]; ok {
			LastActiveTime = usItem.LastActiveTime.Format(time.RFC3339)
		}
		return UserItem{
			UserId:         t.Id,
			AvatarUrl:      t.GetWebAvatarUrl(),
			Username:       t.Username,
			Email:          t.Email,
			ActorType:      t.ActorType,
			Status:         t.IsFrozen,
			Validate:       t.IsActivated,
			Prestige:       t.Prestige,
			RoleList:       roleList,
			RoleId:         t.RoleId,
			CreateTime:     t.CreatedAt.Format(time.RFC3339),
			LastActiveTime: LastActiveTime,
			Badges:         badgeservice.GetUserBadges(t.Id),
		}
	})
	return component.SuccessPage(
		list,
		pageData.Page,
		pageData.PageSize,
		pageData.Total,
	)
}

type BadgeSaveReq = badgeservice.Badge

func BadgeList(req component.BetterRequest[component.Null]) component.Response {
	return component.SuccessResponse(badgeservice.AllForAdmin())
}

func generateCustomBadgeCode() string {
	for range 16 {
		code := "custom_" + strings.ToLower(randopt.RandomString(10))
		if badges.GetByCode(code).Id == 0 {
			return code
		}
	}
	return fmt.Sprintf("custom_%x", time.Now().UnixNano())
}

func SaveBadge(req component.BetterRequest[BadgeSaveReq]) component.Response {
	params := req.Params
	params.Code = strings.TrimSpace(params.Code)
	params.Name = strings.TrimSpace(params.Name)
	if params.Name == "" {
		return component.FailResponseCode(component.MessageAdminBadgeNameRequired, nil)
	}
	if params.Type == "" {
		params.Type = badges.TypeCustom
	}
	if params.Type != badges.TypeSystem && params.Type != badges.TypeCustom {
		return component.FailResponseCode(component.MessageAdminBadgeTypeInvalid, nil)
	}
	if params.Code == "" {
		if params.Type == badges.TypeSystem {
			return component.FailResponseCode(component.MessageAdminBadgeCodeRequired, nil)
		}
		params.Code = generateCustomBadgeCode()
	}
	if params.GrantMode == "" {
		params.GrantMode = badges.GrantModeManual
	}
	if params.GrantMode != badges.GrantModeAuto && params.GrantMode != badges.GrantModeManual {
		return component.FailResponseCode(component.MessageAdminBadgeGrantModeInvalid, nil)
	}
	if params.IconType == "" {
		params.IconType = badges.IconTypeAsset
	}
	if params.Type == badges.TypeSystem {
		systemBadge := badgeservice.ResolveOne(params.Code)
		if systemBadge.Code == "" || systemBadge.Type != badges.TypeSystem {
			return component.FailResponseCode(component.MessageAdminBadgeSystemNotFound, nil)
		}
		params.GrantMode = systemBadge.GrantMode
	}

	entity := badges.GetByCode(params.Code)
	if entity.Id == 0 {
		entity.Code = params.Code
	}
	entity.Type = params.Type
	entity.GrantMode = params.GrantMode
	entity.Name = params.Name
	entity.Description = params.Description
	entity.IconType = params.IconType
	entity.IconKey = params.IconKey
	entity.IconURL = params.IconURL
	entity.Color = params.Color
	entity.Level = params.Level
	entity.IsEnabled = params.IsEnabled
	entity.IsWearable = params.IsWearable
	entity.SortOrder = params.SortOrder
	if err := badges.Save(&entity); err != nil {
		return component.FailResponseCode(component.MessageAdminBadgeSaveFailed, nil)
	}
	badgeservice.InvalidateDefinitions()
	userservice.ClearUserPublicProfileCache()
	return component.SuccessResponseCode("success", component.MessageOperationSuccess, nil)
}

type BadgeDeleteReq struct {
	Code string `json:"code"`
}

func DeleteBadge(req component.BetterRequest[BadgeDeleteReq]) component.Response {
	code := strings.TrimSpace(req.Params.Code)
	if code == "" {
		return component.FailResponseCode(component.MessageAdminBadgeCodeRequired, nil)
	}
	badge := badgeservice.ResolveOne(code)
	if badge.Type == badges.TypeSystem {
		return component.FailResponseCode(component.MessageAdminBadgeSystemDeleteBlock, nil)
	}
	if err := badges.DeleteByCode(code); err != nil {
		return component.FailResponseCode(component.MessageAdminBadgeDeleteFailed, nil)
	}
	badgeservice.InvalidateDefinitions()
	userservice.ClearUserPublicProfileCache()
	return component.SuccessResponseCode("success", component.MessageOperationSuccess, nil)
}

type UserBadgeOptionsReq struct {
	UserId uint64 `json:"userId"`
}

type UserBadgeOptionsResp struct {
	Options []badgeservice.Badge     `json:"options"`
	Active  []badgeservice.UserBadge `json:"active"`
}

func UserBadgeOptions(req component.BetterRequest[UserBadgeOptionsReq]) component.Response {
	return component.SuccessResponse(UserBadgeOptionsResp{
		Options: badgeservice.ManualGrantBadgesForAdmin(),
		Active:  badgeservice.GetUserBadges(req.Params.UserId),
	})
}

type SaveUserBadgesReq struct {
	UserId     uint64   `json:"userId"`
	BadgeCodes []string `json:"badgeCodes"`
}

func SaveUserBadges(req component.BetterRequest[SaveUserBadgesReq]) component.Response {
	userID := req.Params.UserId
	if userID == 0 {
		return component.FailResponseCode(component.MessageUserNotFound, nil)
	}
	allowed := lo.KeyBy(badgeservice.ManualGrantBadgesForAdmin(), func(item badgeservice.Badge) string { return item.Code })
	nextCodes := lo.Uniq(req.Params.BadgeCodes)
	nextSet := map[string]bool{}
	for _, code := range nextCodes {
		if _, ok := allowed[code]; !ok {
			continue
		}
		nextSet[code] = true
		_, _ = badgeservice.Grant(userID, code, userBadges.SourceManual, "管理员手动授予", req.UserId)
	}
	for _, active := range badgeservice.GetUserBadges(userID) {
		if active.Source != userBadges.SourceManual {
			continue
		}
		if !nextSet[active.Code] {
			_ = userBadges.Revoke(userID, active.Code)
		}
	}
	userservice.InvalidateUserPublicProfileCache(userID)
	return component.SuccessResponseCode("success", component.MessageOperationSuccess, nil)
}

type EditUserReq struct {
	UserId   uint64 `json:"userId"`
	Status   int8   `json:"status"`
	Validate int8   `json:"validate"`
	RoleId   uint64 `json:"roleId"`
}

func EditUser(req component.BetterRequest[EditUserReq]) component.Response {
	params := req.Params
	user, err := users.Get(params.UserId)
	if err != nil || user.Id == 0 {
		return component.FailResponseCode(component.MessageAdminTargetUserFetchFailed, nil)
	}
	// 机器人（Agent）账号不允许被授予任何角色（管理/版主等）。
	if user.IsBot() && params.RoleId != 0 {
		return component.FailResponseCode(component.MessageAdminAgentRoleNotAllowed, nil)
	}
	opt := false
	changes := make([]string, 0, 3)
	oldFrozen := user.IsFrozen
	oldActivated := user.IsActivated
	oldRoleID := user.RoleId
	if user.IsFrozen != params.Status {
		changes = append(changes, "status")
		user.IsFrozen = params.Status
		opt = true
	}
	if user.IsActivated != params.Validate {
		changes = append(changes, "activation")
		user.IsActivated = params.Validate
		opt = true
	}
	if user.RoleId != params.RoleId {
		changes = append(changes, "role")
		user.RoleId = params.RoleId
		opt = true
	}
	if opt {
		if err := userservice.SaveUser(&user); err != nil {
			return component.FailResponseCode(component.MessageUserUpdateFailed, nil)
		}
		optlogger.UserOptCode(req.UserId, optlogger.EditUser, user.Id, "admin.opt.user.updated", optlogger.MessageParams{
			"userId":        user.Id,
			"changes":       changes,
			"oldFrozen":     oldFrozen,
			"newFrozen":     user.IsFrozen,
			"oldActivated":  oldActivated,
			"newActivated":  user.IsActivated,
			"oldRoleId":     oldRoleID,
			"newRoleId":     user.RoleId,
			"changedFields": strings.Join(changes, ", "),
		})
	}
	return component.SuccessResponseCode("success", component.MessageOperationSuccess, nil)
}

type TopicsListReq struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	Search   string `form:"search"`
	UserId   uint64 `form:"userId"`
}

type TopicAdminBaseVo struct {
	Id            uint64   `json:"id"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	CategoryId    []uint64 `json:"categoryId"`
	UserId        uint64   `json:"userId"`
	TopicStatus   int8     `json:"topicStatus"`
	ProcessStatus int8     `json:"processStatus"`
	CreatedAt     string   `json:"createdAt"`
	UpdatedAt     string   `json:"updatedAt"`
}

type TopicInfoAdminVo struct {
	TopicAdminBaseVo
	Username      string `json:"username"`
	UserAvatarUrl string `json:"userAvatarUrl"`
	ViewCount     uint64 `json:"viewCount"`
	ReplyCount    uint64 `json:"replyCount"`
	LikeCount     uint64 `json:"likeCount"`
	PinWeight     int    `json:"pinWeight"`
}

type TopicSourceReq struct {
	TopicId uint64 `json:"topicId" validate:"required"`
}

type TopicSourceVo struct {
	TopicAdminBaseVo
	Content string `json:"content"`
}

func TopicsList(req component.BetterRequest[TopicsListReq]) component.Response {
	param := req.Params
	pageData := topics.PageForAdmin(topics.AdminPageQuery{Page: max(param.Page, 1), PageSize: param.PageSize, Search: param.Search, UserId: param.UserId})
	userIds := lo.Map(pageData.Data, func(t topics.Entity, _ int) uint64 {
		return t.UserId
	})
	userMap := users.GetMapByIds(userIds)
	return component.SuccessResponse(component.Page[TopicInfoAdminVo]{
		List: lo.Map(pageData.Data, func(t topics.Entity, _ int) TopicInfoAdminVo {
			username := ""
			userAvatarUrl := ""
			if user := userMap[t.UserId]; user != nil {
				username = user.Username
				userAvatarUrl = user.GetWebAvatarUrl()
			}
			return TopicInfoAdminVo{
				TopicAdminBaseVo: TopicAdminBaseVo{
					Id:            t.Id,
					Title:         t.Title,
					Description:   t.Excerpt,
					CategoryId:    t.CategoryIds,
					UserId:        t.UserId,
					TopicStatus:   t.Status,
					ProcessStatus: t.ProcessStatus,
					CreatedAt:     t.CreatedAt.Format(time.RFC3339),
					UpdatedAt:     t.UpdatedAt.Format(time.RFC3339),
				},
				Username:      username,
				UserAvatarUrl: userAvatarUrl,
				ViewCount:     t.ViewCount,
				ReplyCount:    t.ReplyCount,
				LikeCount:     t.LikeCount,
				PinWeight:     t.PinWeight,
			}
		}),
		Page:    pageData.Page,
		Size:    pageData.PageSize,
		HasNext: pageData.HasNext,
	})
}

func TopicSource(req component.BetterRequest[TopicSourceReq]) component.Response {
	topic := topics.Get(req.Params.TopicId)
	if topic.Id == 0 {
		return component.FailResponseCode(component.MessageTopicNotFound, nil)
	}
	firstPost := posts.Get(topic.FirstPostId)
	if firstPost.Id == 0 {
		firstPost, _ = posts.GetByTopicPostNoAtOrAfter(topic.Id, 1)
	}

	return component.SuccessResponse(TopicSourceVo{
		TopicAdminBaseVo: TopicAdminBaseVo{
			Id:            topic.Id,
			Title:         topic.Title,
			Description:   topic.Excerpt,
			CategoryId:    topic.CategoryIds,
			UserId:        topic.UserId,
			TopicStatus:   topic.Status,
			ProcessStatus: topic.ProcessStatus,
			CreatedAt:     topic.CreatedAt.Format(time.RFC3339),
			UpdatedAt:     topic.UpdatedAt.Format(time.RFC3339),
		},
		Content: firstPost.Content,
	})
}

type EditTopicReq struct {
	TopicId       uint64 `json:"topicId" validate:"required"`
	ProcessStatus int8   `json:"processStatus" validate:"oneof=0 1"` // 0正常 1封禁
}

type EditTopicPinReq struct {
	TopicId   uint64 `json:"topicId" validate:"required"`
	PinWeight int    `json:"pinWeight" validate:"min=0,max=1000000"`
}

type EditTopicCategoriesReq struct {
	TopicId    uint64   `json:"topicId" validate:"required"`
	CategoryId []uint64 `json:"categoryId" validate:"min=1,max=3"`
}

type DeleteTopicReq struct {
	TopicId uint64 `json:"topicId" validate:"required"`
	Reason  string `json:"reason" validate:"required,min=1,max=500"`
}

// DeletePostAsModeratorReq 管理端删除单个回复的请求。
type DeletePostAsModeratorReq struct {
	PostId uint64 `json:"postId" validate:"required"`
	Reason string `json:"reason" validate:"required,min=1,max=500"`
}

// DeletePostAsModerator 管理端治理删除单个回复：作者不可自行恢复，
// 记录审计日志与删除原因，并同步清理搜索/缓存/通知/附件。
func DeletePostAsModerator(req component.BetterRequest[DeletePostAsModeratorReq]) component.Response {
	if err := contentdeleteservice.DeletePostAsModerator(req.UserId, req.Params.PostId, req.Params.Reason); err != nil {
		return component.FailResponseError(err)
	}
	return component.SuccessResponseCode("操作成功", component.MessageOperationSuccess, nil)
}

// EditTopic updates topic moderation status.
func EditTopic(req component.BetterRequest[EditTopicReq]) component.Response {
	topic := topics.Get(req.Params.TopicId)
	if topic.Id == 0 {
		return component.FailResponseCode(component.MessageTopicNotFound, nil)
	}

	if topic.ProcessStatus == req.Params.ProcessStatus {
		return component.SuccessResponseCode("操作成功", component.MessageOperationSuccess, nil)
	}

	if err := db.ConnectContext(requestContext(req.GinContext)).Transaction(func(tx *gorm.DB) error {
		if err := topics.UpdateProcessStatusTx(tx, topic.Id, req.Params.ProcessStatus); err != nil {
			return err
		}
		return searchservice.EnqueueTopicSearchTask(tx, topic.Id)
	}); err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	topic.ProcessStatus = req.Params.ProcessStatus

	// 记录操作日志
	statusCode := "unblocked"
	if req.Params.ProcessStatus == 1 {
		statusCode = "blocked"
	}
	optlogger.UserOptCode(req.UserId, optlogger.EditTopic, topic.Id, "admin.opt.topic.statusChanged", optlogger.MessageParams{
		"title":  topic.Title,
		"status": statusCode,
	})
	moderationservice.TopicStatusChanged(req.UserId, topic.Id, topic.Title, req.Params.ProcessStatus == 1)
	hotdataserve.InvalidateTopicListCacheForCategories(topic.CategoryIds...)
	// 封禁/解封不发布 Topic*Event，需同步清理 LLMS 公开投影缓存，避免封禁内容在 10s 窗口内继续导出。
	llmsservice.ClearCache()
	return component.SuccessResponseCode("操作成功", component.MessageOperationSuccess, nil)
}

// RestoreTopicReq 管理端恢复被治理删除话题的请求。
type RestoreTopicReq struct {
	TopicId uint64 `json:"topicId" validate:"required"`
}

// RestoreTopic 管理端恢复被治理删除（MODERATOR_REMOVED）的话题（review MEDIUM-2）。
// 管理端是治理删除的唯一恢复通道：作者不可恢复管理端删除；恢复后重建搜索索引、
// 清缓存、恢复附件可见性并写审计日志与埋点。
func RestoreTopic(req component.BetterRequest[RestoreTopicReq]) component.Response {
	if err := contentdeleteservice.RestoreTopicAsModerator(req.UserId, req.Params.TopicId); err != nil {
		return component.FailResponseError(err)
	}
	return component.SuccessResponseCode("操作成功", component.MessageContentRestoreSuccess, nil)
}

func DeleteTopic(req component.BetterRequest[DeleteTopicReq]) component.Response {
	// 用 UnscopedGet 读取：被管理端删除的话题 deleted_at 已置位，
	// 软删过滤的 Get 会返回空行，导致下方的幂等分支永远不可达（死代码）。
	// 必须先读到已删除行才能判断"重复删除直接成功"。
	topic := topics.UnscopedGet(req.Params.TopicId)
	if topic.Id == 0 {
		return component.FailResponseCode(component.MessageTopicNotFound, nil)
	}
	// wiki 分站页面话题由 wiki 修订审核流程管理，禁止经论坛管理端删除，
	// 避免软删话题后残留 wiki_pages/wiki_page_revisions 孤儿页面。
	if topic.TopicType == topics.TopicTypeWiki {
		return component.FailResponseCode(component.MessageTopicOperationDenied, nil)
	}
	// 幂等：已处于管理端删除状态时直接成功，避免重复删除重置 deleted_at / 重复广播。
	if topic.VisibilityStatus == topics.VisibilityModeratorRemoved {
		return component.SuccessResponseCode("操作成功", component.MessageOperationSuccess, nil)
	}

	reason := strings.TrimSpace(req.Params.Reason)
	if reason == "" {
		return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
	}
	// 管理端治理删除：双状态机 MODERATOR_REMOVED + ContentDeletedEvent。
	// 不硬删 topic_category_index：版主日志/举报的按分类作用域查询依赖该索引定位话题，
	// 且公开列表已按 visibility_status=ACTIVE 过滤，删除话题不会因此出现在分类页。
	if err := contentdeleteservice.DeleteTopicAs(topic, req.UserId, topics.VisibilityModeratorRemoved, reason); err != nil {
		return component.FailResponseError(err)
	}
	return component.SuccessResponseCode("操作成功", component.MessageOperationSuccess, nil)
}

func EditTopicPin(req component.BetterRequest[EditTopicPinReq]) component.Response {
	topic := topics.Get(req.Params.TopicId)
	if topic.Id == 0 {
		return component.FailResponseCode(component.MessageTopicNotFound, nil)
	}
	if topic.PinWeight == req.Params.PinWeight {
		return component.SuccessResponseCode("操作成功", component.MessageOperationSuccess, nil)
	}
	oldPinWeight := topic.PinWeight
	if err := topics.UpdatePinWeight(topic.Id, req.Params.PinWeight); err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	hotdataserve.InvalidateTopicListCacheForCategories(topic.CategoryIds...)
	optlogger.UserOptCode(req.UserId, optlogger.EditTopic, topic.Id, "admin.opt.topic.pinWeightChanged", optlogger.MessageParams{
		"title":        topic.Title,
		"oldPinWeight": oldPinWeight,
		"pinWeight":    req.Params.PinWeight,
	})
	return component.SuccessResponseCode("操作成功", component.MessageOperationSuccess, nil)
}

// EditTopicCategories updates topic categories.
func EditTopicCategories(req component.BetterRequest[EditTopicCategoriesReq]) component.Response {
	categoryIds := lo.Uniq(req.Params.CategoryId)
	if len(categoryIds) == 0 {
		return component.FailResponseCode(component.MessageAdminTopicCategoryRequired, nil)
	}
	if len(categoryIds) > 3 {
		return component.FailResponseCode(component.MessageAdminTopicCategoryTooMany, nil)
	}
	for _, categoryId := range categoryIds {
		if categoryId == 0 || category.Get(categoryId).Id == 0 {
			return component.FailResponseCode(component.MessageAdminCategoryNotFound, nil)
		}
	}

	topic := topics.Get(req.Params.TopicId)
	if topic.Id == 0 {
		return component.FailResponseCode(component.MessageTopicNotFound, nil)
	}

	oldCategoryIds := append([]uint64(nil), topic.CategoryIds...)
	topic.CategoryIds = categoryIds
	if err := db.ConnectContext(requestContext(req.GinContext)).Transaction(func(tx *gorm.DB) error {
		if err := topics.UpdateCategoryIDsTx(tx, topic.Id, categoryIds); err != nil {
			return err
		}
		if err := topicCategoryIndex.ReplaceTopicCategoriesTx(tx, topic.Id, categoryIds); err != nil {
			return err
		}
		return searchservice.EnqueueTopicSearchTask(tx, topic.Id)
	}); err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	optlogger.UserOptCode(req.UserId, optlogger.EditTopic, topic.Id, "admin.opt.topic.categoriesChanged", optlogger.MessageParams{
		"title":          topic.Title,
		"oldCategoryIds": oldCategoryIds,
		"categoryIds":    categoryIds,
	})
	hotdataserve.InvalidateTopicListCacheForCategories(append(oldCategoryIds, categoryIds...)...)
	// 分类变更不发布事件，同步清理 LLMS 投影缓存（投影内嵌 Categories 列表）。
	llmsservice.ClearCache()
	return component.SuccessResponseCode("操作成功", component.MessageOperationSuccess, nil)
}

type PermissionListReq struct {
}

func GetPermissionList(req component.BetterRequest[PermissionListReq]) component.Response {
	res := permission.BuildOptions(component.RequestLang(req.GinContext))
	return component.SuccessResponse(res)
}

type GetAllRoleItemReq struct {
}

func GetAllRoleItem(req component.BetterRequest[GetAllRoleItemReq]) component.Response {
	res := lo.Map(role.AllEffective(), func(t *role.Entity, _ int) datastruct.Option[string, uint64] {
		return datastruct.Option[string, uint64]{Name: t.RoleName, Label: t.RoleName, Value: t.Id}
	})
	return component.SuccessResponse(res)
}

type RoleListReq struct {
}

type RoleItem struct {
	RoleId      uint64           `json:"roleId"`
	RoleName    string           `json:"roleName"`
	Effective   int              `json:"effective"`
	Permissions []PermissionItem `json:"permissions"`
	CreateTime  string           `json:"createTime"`
}

type PermissionItem struct {
	Id   uint64 `json:"id"`
	Name string `json:"name"`
}

func RoleList(req component.BetterRequest[RoleListReq]) component.Response {
	lang := component.RequestLang(req.GinContext)
	pageData := role.Page(role.PageQuery{})
	roleIds := lo.Map(pageData.Data, func(t role.Entity, _ int) uint64 {
		return t.Id
	})
	rpGroup := make(map[uint64][]uint64)
	if len(roleIds) > 0 {
		rpGroup = rolePermissionRs.GetRsGroupByRoleIds(roleIds)
	}
	list := lo.Map(pageData.Data, func(t role.Entity, _ int) RoleItem {
		pList, ok := rpGroup[t.Id]
		permissionItemList := make([]PermissionItem, 0)
		if ok {
			permissionItemList = lo.Map(pList, func(t uint64, _ int) PermissionItem {
				p := permission.Enum(t)
				return PermissionItem{Id: p.Id(), Name: p.LocalizedName(lang)}
			})
		}
		return RoleItem{
			RoleId:      t.Id,
			RoleName:    t.RoleName,
			Effective:   t.Effective,
			Permissions: permissionItemList,
			CreateTime:  t.CreatedAt.Format(time.RFC3339),
		}
	})

	return component.SuccessPage(
		list,
		pageData.Page,
		pageData.PageSize,
		pageData.Total,
	)
}

type RoleSaveReq struct {
	Id          uint     `json:"id"`
	RoleName    string   `json:"roleName" validate:"required"`
	Permissions []uint64 `json:"permissions" validate:"required,min=1,max=100"`
}

func RoleSave(req component.BetterRequest[RoleSaveReq]) component.Response {
	var roleEntity role.Entity
	if req.Params.Id > 0 {
		roleEntity = role.Get(req.Params.Id)
	} else {
		roleEntity = role.Entity{
			Effective: 1,
		}
	}
	roleEntity.RoleName = req.Params.RoleName
	if err := role.SaveOrCreateById(&roleEntity); err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}

	rsList := rolePermissionRs.GetRsByRoleId(roleEntity.Id)
	canUpdateMap := lo.SliceToMap(req.Params.Permissions, func(id uint64) (uint64, bool) {
		return id, true
	})

	// 更新数据
	for _, item := range rsList {
		item.Effective = 0
		if _, ok := canUpdateMap[item.PermissionId]; ok {
			item.Effective = 1
			// 如果已经存在，从 map 中删除，避免重复插入
			delete(canUpdateMap, item.PermissionId)
		}
		rolePermissionRs.SaveOrCreateById(item)
	}
	// 插入新的条目
	for id := range canUpdateMap {
		rsItem := rolePermissionRs.Entity{
			RoleId:       roleEntity.Id,
			PermissionId: id,
			Effective:    1,
		}
		rolePermissionRs.SaveOrCreateById(&rsItem)
	}
	permission.InvalidateRole(roleEntity.Id)

	return component.SuccessResponse(true)
}

type RoleSaveDel struct {
	Id uint `json:"id"`
}

func RoleDel(req component.BetterRequest[RoleSaveDel]) component.Response {
	roleEntity := role.Get(req.Params.Id)
	if roleEntity.Id == 0 {
		return component.FailResponseCode(component.MessageAdminRoleNotFound, nil)
	}
	rsList := rolePermissionRs.GetRsByRoleId(roleEntity.Id)
	// 删除
	lo.ForEach(rsList, func(item *rolePermissionRs.Entity, _ int) {
		rolePermissionRs.DeleteEntity(item)
	})
	role.DeleteEntity(&roleEntity)
	permission.InvalidateRole(roleEntity.Id)
	return component.SuccessResponse(true)
}

type OptRecordPageReq struct {
	Page       int    `json:"page"`
	PageSize   int    `json:"pageSize"`
	OptUserId  uint64 `json:"optUserId"`
	OptType    int    `json:"optType"`
	TargetType int    `json:"targetType"`
	TargetId   int    `json:"targetId"`
}

func OptRecordPage(req component.BetterRequest[OptRecordPageReq]) component.Response {
	pageData := optRecord.Page(optRecord.PageQuery{
		Page:       req.Params.Page,
		PageSize:   component.BoundPageSizeWithRange(req.Params.PageSize, 10, 50),
		OptUserId:  req.Params.OptUserId,
		OptType:    req.Params.OptType,
		TargetType: req.Params.TargetType,
		TargetId:   req.Params.TargetId,
	})
	return component.SuccessPage(
		lo.Map(pageData.Data, func(item optRecord.Entity, _ int) optRecord.Entity {
			return item
		}),
		pageData.Page,
		pageData.PageSize,
		pageData.Total,
	)
}

type FileResourcePageReq struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

type FileResourceItem struct {
	filedata.FileResource
	UploaderUsername string `json:"uploaderUsername"`
}

func FileResourcePage(req component.BetterRequest[FileResourcePageReq]) component.Response {
	pageData := filedata.FileResourcePage(req.Params.Page, component.BoundPageSizeWithRange(req.Params.PageSize, 10, 50))
	userIDs := lo.Map(pageData.List, func(item filedata.FileResource, _ int) uint64 {
		return item.UserId
	})
	userMap := users.GetMapByIds(userIDs)
	return component.SuccessPage(
		lo.Map(pageData.List, func(item filedata.FileResource, _ int) FileResourceItem {
			username := ""
			if user := userMap[item.UserId]; user != nil {
				username = user.Username
			}
			return FileResourceItem{FileResource: item, UploaderUsername: username}
		}),
		pageData.Page,
		pageData.PageSize,
		pageData.MaxId,
	)
}

type CategoryListReq struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

type CategoryItem struct {
	Id         uint64                  `json:"id"`
	Category   string                  `json:"category"`
	Desc       string                  `json:"desc"`
	Icon       string                  `json:"icon"`
	Color      string                  `json:"color"`
	Slug       string                  `json:"slug"`
	Sort       int                     `json:"sort"`
	Moderators []CategoryModeratorItem `json:"moderators"`
}

type CategoryModeratorItem struct {
	Id        uint64 `json:"id"`
	UserId    uint64 `json:"userId"`
	Username  string `json:"username"`
	AvatarUrl string `json:"avatarUrl"`
	Status    int    `json:"status"`
}

// GetCategoryList 获取分类列表
func GetCategoryList(req component.BetterRequest[CategoryListReq]) component.Response {
	categories := category.All()
	categoryIds := lo.Map(categories, func(item *category.Entity, _ int) uint64 {
		return item.Id
	})
	moderatorList := moderators.GetByCategoryIds(categoryIds)
	moderatorUserIds := lo.Uniq(lo.Map(moderatorList, func(item *moderators.Entity, _ int) uint64 {
		return item.UserId
	}))
	userMap := users.GetMapByIds(moderatorUserIds)
	moderatorGroup := lo.GroupBy(moderatorList, func(item *moderators.Entity) uint64 {
		return item.ScopeId
	})

	return component.SuccessResponse(lo.Map(categories, func(t *category.Entity, _ int) CategoryItem {
		return CategoryItem{
			Id:         t.Id,
			Category:   t.Name,
			Desc:       t.Desc,
			Icon:       t.Icon,
			Color:      t.Color,
			Slug:       t.Slug,
			Sort:       t.Sort,
			Moderators: buildCategoryModeratorItems(moderatorGroup[t.Id], userMap),
		}
	}))
}

func buildCategoryModeratorItems(moderatorList []*moderators.Entity, userMap map[uint64]*users.EntityComplete) []CategoryModeratorItem {
	return lo.Map(moderatorList, func(item *moderators.Entity, _ int) CategoryModeratorItem {
		user := userMap[item.UserId]
		username := ""
		avatarURL := ""
		if user != nil {
			username = user.Username
			avatarURL = user.GetWebAvatarUrl()
		}
		return CategoryModeratorItem{
			Id:        item.Id,
			UserId:    item.UserId,
			Username:  username,
			AvatarUrl: avatarURL,
			Status:    item.Status,
		}
	})
}

type AddCategoryModeratorReq struct {
	CategoryId uint64 `json:"categoryId" validate:"required"`
	UserId     uint64 `json:"userId"`
	Username   string `json:"username"`
}

type ModeratorUserReq struct {
	UserId   uint64 `json:"userId"`
	Username string `json:"username"`
}

func resolveModeratorUser(params ModeratorUserReq) (users.EntityComplete, bool) {
	if params.UserId != 0 {
		user, err := users.Get(params.UserId)
		return user, err == nil && user.Id != 0
	}
	username := strings.TrimSpace(params.Username)
	if username == "" {
		return users.EntityComplete{}, false
	}
	user, err := users.GetByUsername(username)
	return user, err == nil && user.Id != 0
}

func AddCategoryModerator(req component.BetterRequest[AddCategoryModeratorReq]) component.Response {
	categoryEntity := category.Get(req.Params.CategoryId)
	if categoryEntity.Id == 0 {
		return component.FailResponseCode(component.MessageAdminCategoryNotFound, nil)
	}
	if req.Params.UserId == 0 && strings.TrimSpace(req.Params.Username) == "" {
		return component.FailResponseCode(component.MessageAdminModeratorUserRequired, nil)
	}
	user, ok := resolveModeratorUser(ModeratorUserReq{UserId: req.Params.UserId, Username: req.Params.Username})
	if !ok {
		return component.FailResponseCode(component.MessageAdminModeratorUserNotFound, nil)
	}
	// 机器人（Agent）账号不允许成为版主。
	if user.IsBot() {
		return component.FailResponseCode(component.MessageAdminAgentRoleNotAllowed, nil)
	}

	entity := moderators.GetByUserScope(user.Id, moderators.ScopeCategory, categoryEntity.Id)
	entity.UserId = user.Id
	entity.ScopeType = moderators.ScopeCategory
	entity.ScopeId = categoryEntity.Id
	entity.Status = moderators.StatusEnabled
	if entity.CreatedBy == 0 {
		entity.CreatedBy = req.UserId
	}
	if err := moderators.Save(&entity); err != nil {
		slog.Error("save category moderator failed", "categoryId", categoryEntity.Id, "userId", user.Id, "err", err)
		return component.FailResponse()
	}
	moderationservice.Invalidate()
	optlogger.UserOptCode(req.UserId, optlogger.EditCategory, categoryEntity.Id, "admin.opt.category.moderatorAdded", optlogger.MessageParams{
		"categoryId":   categoryEntity.Id,
		"categoryName": categoryEntity.Name,
		"userId":       user.Id,
		"username":     user.Username,
	})
	return component.SuccessResponse(true)
}

func GetGlobalModeratorList(req component.BetterRequest[struct{}]) component.Response {
	moderatorList := moderators.GetByScope(moderators.ScopeGlobal, 0)
	moderatorUserIds := lo.Uniq(lo.Map(moderatorList, func(item *moderators.Entity, _ int) uint64 {
		return item.UserId
	}))
	userMap := users.GetMapByIds(moderatorUserIds)
	return component.SuccessResponse(buildCategoryModeratorItems(moderatorList, userMap))
}

func AddGlobalModerator(req component.BetterRequest[ModeratorUserReq]) component.Response {
	if req.Params.UserId == 0 && strings.TrimSpace(req.Params.Username) == "" {
		return component.FailResponseCode(component.MessageAdminModeratorUserRequired, nil)
	}
	user, ok := resolveModeratorUser(req.Params)
	if !ok {
		return component.FailResponseCode(component.MessageAdminModeratorUserNotFound, nil)
	}
	// 机器人（Agent）账号不允许成为版主。
	if user.IsBot() {
		return component.FailResponseCode(component.MessageAdminAgentRoleNotAllowed, nil)
	}
	entity := moderators.GetByUserScope(user.Id, moderators.ScopeGlobal, 0)
	entity.UserId = user.Id
	entity.ScopeType = moderators.ScopeGlobal
	entity.ScopeId = 0
	entity.Status = moderators.StatusEnabled
	if entity.CreatedBy == 0 {
		entity.CreatedBy = req.UserId
	}
	if err := moderators.Save(&entity); err != nil {
		slog.Error("save global moderator failed", "userId", user.Id, "err", err)
		return component.FailResponse()
	}
	moderationservice.Invalidate()
	return component.SuccessResponse(true)
}

func DeleteGlobalModerator(req component.BetterRequest[struct {
	Id uint64 `json:"id" validate:"required"`
}]) component.Response {
	entity := moderators.Get(req.Params.Id)
	if entity.Id == 0 || entity.ScopeType != moderators.ScopeGlobal {
		return component.FailResponseCode(component.MessageAdminModeratorNotFound, nil)
	}
	if err := moderators.Delete(&entity); err != nil {
		slog.Error("delete global moderator failed", "moderatorId", entity.Id, "err", err)
		return component.FailResponse()
	}
	moderationservice.Invalidate()
	return component.SuccessResponse(true)
}

func DeleteCategoryModerator(req component.BetterRequest[struct {
	Id uint64 `json:"id" validate:"required"`
}]) component.Response {
	entity := moderators.Get(req.Params.Id)
	if entity.Id == 0 || entity.ScopeType != moderators.ScopeCategory {
		return component.FailResponseCode(component.MessageAdminModeratorNotFound, nil)
	}
	categoryEntity := category.Get(entity.ScopeId)
	if err := moderators.Delete(&entity); err != nil {
		slog.Error("delete category moderator failed", "moderatorId", entity.Id, "err", err)
		return component.FailResponse()
	}
	moderationservice.Invalidate()
	optlogger.UserOptCode(req.UserId, optlogger.EditCategory, entity.ScopeId, "admin.opt.category.moderatorRemoved", optlogger.MessageParams{
		"categoryId":   entity.ScopeId,
		"categoryName": categoryEntity.Name,
		"userId":       entity.UserId,
	})
	return component.SuccessResponse(true)
}

type CategorySaveReq struct {
	Id       uint64 `json:"id"`
	Category string `json:"category" validate:"required"`
	Desc     string `json:"desc"`
	Icon     string `json:"icon"`
	Color    string `json:"color"`
	Slug     string `json:"slug"`
	Sort     int    `json:"sort"`
}

// SaveCategory 保存分类
func SaveCategory(req component.BetterRequest[CategorySaveReq]) component.Response {
	if len(strings.TrimSpace(req.Params.Category)) == 0 {
		return component.FailResponseCode(component.MessageAdminCategoryRequired, nil)
	}

	entity := category.Get(req.Params.Id)
	if req.Params.Id != 0 && entity.Id == 0 {
		return component.FailResponseCode(component.MessageAdminCategoryDataNotFound, nil)
	}
	entity.Name = req.Params.Category
	entity.Desc = req.Params.Desc
	entity.Icon = req.Params.Icon
	entity.Color = req.Params.Color
	entity.Slug = req.Params.Slug
	entity.Sort = req.Params.Sort

	if err := db.ConnectContext(requestContext(req.GinContext)).Transaction(func(tx *gorm.DB) error {
		if err := category.SaveTx(tx, &entity); err != nil {
			return err
		}
		return searchservice.EnqueueCategorySearchTask(tx, entity.Id)
	}); err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	hotdataserve.ClearCategoryCache()
	return component.SuccessResponse(true)
}

// DeleteCategory 删除分类
func DeleteCategory(req component.BetterRequest[struct {
	Id uint64 `json:"id"`
}]) component.Response {
	entity := category.Get(req.Params.Id)
	if entity.Id == 0 {
		return component.FailResponseCode(component.MessageAdminCategoryNotFound, nil)
	}
	if category.Count() == 1 {
		return component.FailResponseCode(component.MessageAdminCategoryKeepOne, nil)
	}
	if topicCategoryIndex.GetOneByCategoryId(entity.Id).Id > 0 {
		return component.FailResponseCode(component.MessageAdminCategoryHasTopics, nil)
	}
	if err := db.ConnectContext(requestContext(req.GinContext)).Transaction(func(tx *gorm.DB) error {
		if err := category.DeleteTx(tx, &entity); err != nil {
			return err
		}
		return searchservice.EnqueueCategorySearchTask(tx, entity.Id)
	}); err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	hotdataserve.ClearCategoryCache()
	return component.SuccessResponse(true)
}

func GetFriendLinks(req component.BetterRequest[component.Null]) component.Response {
	res := pageConfig.GetConfigByPageType(pageConfig.FriendShipLinks, defaultconfig.GetDefaultFriendLinksConfig())
	normalizeFriendLinks(res)
	return component.SuccessResponse(res)
}

type SaveFriendLinksReq struct {
	LinksInfo []pageConfig.FriendLinksGroup `json:"linksInfo"`
}

// SaveFriendLinks 保存友情链接
func SaveFriendLinks(req component.BetterRequest[SaveFriendLinksReq]) component.Response {
	normalizeFriendLinks(req.Params.LinksInfo)
	return savePageConfig(pageConfig.FriendShipLinks, req.Params.LinksInfo, hotdataserve.ClearFriendLinksConfigCache)
}

func normalizeFriendLinks(groups []pageConfig.FriendLinksGroup) {
	for i := range groups {
		if groups[i].Links == nil {
			groups[i].Links = []pageConfig.LinkItem{}
		}
	}
}

// GetSponsors 获取赞助商配置
func GetSponsors(req component.BetterRequest[component.Null]) component.Response {
	res := pageConfig.GetConfigByPageType(pageConfig.SponsorsPage, defaultconfig.GetDefaultSponsorsConfig())
	fillSponsorsConfigDefaults(&res)
	return component.SuccessResponse(res)
}

type SaveSponsorsReq struct {
	SponsorsInfo pageConfig.SponsorsConfig `json:"sponsorsInfo"`
}

// SaveSponsors 保存赞助商配置
func SaveSponsors(req component.BetterRequest[SaveSponsorsReq]) component.Response {
	config := req.Params.SponsorsInfo
	fillSponsorsConfigDefaults(&config)
	return savePageConfig(pageConfig.SponsorsPage, config, hotdataserve.ClearSponsorsConfigCache)
}

func fillSponsorsConfigDefaults(config *pageConfig.SponsorsConfig) {
	defaultConfig := defaultconfig.GetDefaultSponsorsConfig()
	if config.Sponsors.Level0 == nil {
		config.Sponsors.Level0 = []pageConfig.SponsorItem{}
	}
	if config.Sponsors.Level1 == nil {
		config.Sponsors.Level1 = []pageConfig.SponsorItem{}
	}
	if config.Sponsors.Level2 == nil {
		config.Sponsors.Level2 = []pageConfig.SponsorItem{}
	}
	if config.Sponsors.Level3 == nil {
		config.Sponsors.Level3 = []pageConfig.SponsorItem{}
	}
	if config.Content.Title == "" {
		config.Content.Title = defaultConfig.Content.Title
	}
	if config.Content.Description == "" {
		config.Content.Description = defaultConfig.Content.Description
	}
	if config.Contact.Title == "" {
		config.Contact.Title = defaultConfig.Contact.Title
	}
	if config.Contact.Description == "" {
		config.Contact.Description = defaultConfig.Contact.Description
	}
	if config.Contact.ButtonText == "" {
		config.Contact.ButtonText = defaultConfig.Contact.ButtonText
	}
	if config.Contact.ButtonLink == "" {
		config.Contact.ButtonLink = defaultConfig.Contact.ButtonLink
	}
	if config.Rules == nil {
		config.Rules = defaultConfig.Rules
	}
}

// GetSiteSettings 获取站点设置
func GetSiteSettings(req component.BetterRequest[component.Null]) component.Response {
	defaultSettings := defaultconfig.GetDefaultSiteSettingsConfig()
	res := pageConfig.GetConfigByPageType(pageConfig.SiteSettings, defaultSettings)
	return component.SuccessResponse(res)
}

func ServerVersion(req component.BetterRequest[component.Null]) component.Response {
	return component.SuccessResponse(buildinfo.Get())
}

type SaveSiteSettingsReq struct {
	Settings pageConfig.SiteSettingsConfig `json:"settings"`
}

// SaveSiteSettings 保存站点设置
func SaveSiteSettings(req component.BetterRequest[SaveSiteSettingsReq]) component.Response {
	return savePageConfig(pageConfig.SiteSettings, req.Params.Settings, func() {
		hotdataserve.ClearSiteSettingsConfigCache()
		llmsservice.ClearCache()
	})
}

func GetSiteChrome(req component.BetterRequest[component.Null]) component.Response {
	config := pageConfig.GetConfigByPageType(pageConfig.SiteChrome, defaultconfig.GetDefaultSiteChromeConfig())
	return component.SuccessResponse(config)
}

type SaveSiteChromeReq struct {
	Settings pageConfig.SiteChromeConfig `json:"settings"`
}

func SaveSiteChrome(req component.BetterRequest[SaveSiteChromeReq]) component.Response {
	return savePageConfig(pageConfig.SiteChrome, req.Params.Settings, hotdataserve.ClearSiteChromeConfigCache)
}

func GetSiteTheme(req component.BetterRequest[component.Null]) component.Response {
	return component.SuccessResponse(themeservice.LoadConfig())
}

type SaveSiteThemeReq struct {
	Settings pageConfig.SiteThemeConfig `json:"settings" validate:"required"`
}

func SaveSiteTheme(req component.BetterRequest[SaveSiteThemeReq]) component.Response {
	config := themeservice.LoadConfig()
	config.Prepublish = &pageConfig.SiteThemePrepublish{
		Enabled:   req.Params.Settings.Enabled,
		Themes:    themeservice.CloneDefinitions(req.Params.Settings.Themes),
		UpdatedAt: time.Now().Format(time.RFC3339),
	}
	config = themeservice.NormalizeConfig(config)
	savePageConfig(pageConfig.SiteTheme, config, themeservice.ClearCaches)
	return component.SuccessResponse(config)
}

func PublishSiteTheme(req component.BetterRequest[component.Null]) component.Response {
	config := themeservice.LoadConfig()
	if config.Prepublish == nil {
		return component.SuccessResponse(config)
	}
	now := time.Now().Format(time.RFC3339)
	config.Enabled = config.Prepublish.Enabled
	config.Themes = themeservice.CloneDefinitions(config.Prepublish.Themes)
	config.PublishedAt = now
	config.Prepublish = nil
	config = themeservice.NormalizeConfig(config)
	savePageConfig(pageConfig.SiteTheme, config, themeservice.ClearCaches)
	return component.SuccessResponse(config)
}

// GetMailSettings 获取邮件设置：仅回显是否已配置密码，不回显密码明文/密文
// （issue #324 S2，参照 onesystem cookieConfigured 模式）。
func GetMailSettings(req component.BetterRequest[component.Null]) component.Response {
	return component.SuccessResponse(hotdataserve.GetMailSettingsView())
}

type SaveMailSettingsReq struct {
	Settings pageConfig.MailSettingsInput `json:"settings" validate:"required"`
}

// SaveMailSettings 保存邮件设置：smtpPassword 明文仅在请求瞬间存在——非空时
// securestore 加密后落库；为空/掩码时保留已存密文（issue #324 S2）。
func SaveMailSettings(req component.BetterRequest[SaveMailSettingsReq]) component.Response {
	input := req.Params.Settings
	entity := pageConfig.GetByPageType(pageConfig.EmailSettings)
	storage := jsonopt.Decode[pageConfig.MailSettingsStorage](entity.Config)
	storage.EnableMail = input.EnableMail
	storage.SmtpHost = input.SmtpHost
	storage.SmtpPort = input.SmtpPort
	storage.UseSSL = input.UseSSL
	storage.SmtpUsername = input.SmtpUsername
	storage.FromName = input.FromName
	storage.FromEmail = input.FromEmail
	if pwd := strings.TrimSpace(input.SmtpPassword); pwd != "" {
		sealed, err := securestore.EncryptPurpose(pwd, securestore.MailSmtpPasswordPurpose)
		if err != nil {
			return component.FailResponseError(fmt.Errorf("加密 SMTP 密码失败（请确认 app.signingKey 已配置）：%w", err))
		}
		storage.SmtpPasswordEncrypted = sealed
		storage.SmtpPassword = ""
	}
	return savePageConfig(pageConfig.EmailSettings, storage, hotdataserve.ClearMailSettingsConfigCache)
}

type TestMailConnectionReq struct {
	Settings  pageConfig.MailSettingsInput `json:"settings" validate:"required"`
	TestEmail string                       `json:"testEmail" validate:"required,email"`
}

type TestMailConnectionResp struct {
	Success     bool                    `json:"success"`
	MessageCode component.MessageCode   `json:"messageCode"`
	Params      component.MessageParams `json:"params,omitempty"`
}

// TestMailConnection 测试邮件连接
func TestMailConnection(req component.BetterRequest[TestMailConnectionReq]) component.Response {
	if req.Params.TestEmail == "" {
		return component.FailResponseCode(component.MessageAdminTestEmailRequired, nil)
	}

	cfg := req.Params.Settings.ToConfig()
	// 管理端 GET 不再回显密码：测试时密码留空则使用已存密码（issue #324 S2）。
	if cfg.SmtpPassword == "" {
		cfg.SmtpPassword = hotdataserve.GetMailSettingsConfigCache().SmtpPassword
	}
	err := mailservice.SendTestEmailWithConfig(cfg, req.Params.TestEmail)
	if err != nil {
		errText := err.Error()
		return component.SuccessResponse(TestMailConnectionResp{
			Success:     false,
			MessageCode: component.MessageAdminTestEmailFailed,
			Params:      component.MessageParams{"error": errText},
		})
	}

	return component.SuccessResponse(TestMailConnectionResp{
		Success:     true,
		MessageCode: component.MessageAdminTestEmailSuccess,
		Params:      component.MessageParams{"email": req.Params.TestEmail},
	})
}

// GetAnnouncement 获取公告设置
func GetAnnouncement(req component.BetterRequest[component.Null]) component.Response {
	config := pageConfig.GetConfigByPageType(pageConfig.Announcement, defaultconfig.GetDefaultAnnouncementConfig())
	return component.SuccessResponse(config)
}

type SaveAnnouncementReq struct {
	Settings pageConfig.AnnouncementConfig `json:"settings" validate:"required"`
}

// SaveAnnouncement 保存公告设置
func SaveAnnouncement(req component.BetterRequest[SaveAnnouncementReq]) component.Response {
	req.Params.Settings.PublishedAt = time.Now().Format(time.RFC3339)
	return savePageConfig(pageConfig.Announcement, req.Params.Settings, hotdataserve.ClearAnnouncementConfigCache)
}

// GetSecuritySettings 获取安全与注册设置
func GetSecuritySettings(req component.BetterRequest[component.Null]) component.Response {
	defaultSettings := defaultconfig.GetDefaultSecuritySettingsConfig()
	res := pageConfig.GetConfigByPageType(pageConfig.SecuritySettings, defaultSettings)
	return component.SuccessResponse(res)
}

type SaveSecuritySettingsReq struct {
	Settings pageConfig.SecurityAndRegistration `json:"settings" validate:"required"`
}

// SaveSecuritySettings 保存安全与注册设置
func SaveSecuritySettings(req component.BetterRequest[SaveSecuritySettingsReq]) component.Response {
	// 新增/更新的禁用用户名：自动冻结匹配的存量账号（幂等，重复保存不会重复处理）
	current := pageConfig.GetConfigByPageType(pageConfig.SecuritySettings, defaultconfig.GetDefaultSecuritySettingsConfig())
	newBanned := req.Params.Settings.BannedUsernames
	for _, username := range newBanned {
		normalized := strings.ToLower(strings.TrimSpace(username))
		if normalized == "" {
			continue
		}
		already := false
		for _, existing := range current.BannedUsernames {
			if strings.ToLower(strings.TrimSpace(existing)) == normalized {
				already = true
				break
			}
		}
		if !already {
			if err := moderationservice.FreezeUsersByBannedUsername(username, req.UserId); err != nil {
				slog.Warn("freeze users for banned username failed", "username", username, "err", err)
			}
		}
	}
	return savePageConfig(pageConfig.SecuritySettings, req.Params.Settings, hotdataserve.ClearSecuritySettingsConfigCache)
}

// GetPostingSettings 获取发布内容设置
func GetPostingSettings(req component.BetterRequest[component.Null]) component.Response {
	defaultSettings := defaultconfig.GetDefaultPostingSettingsConfig()
	res := pageConfig.GetConfigByPageType(pageConfig.PostingSettings, defaultSettings)
	return component.SuccessResponse(res)
}

type SavePostingSettingsReq struct {
	Settings pageConfig.PostingContent `json:"settings" validate:"required"`
}

// SavePostingSettings 保存发布内容设置
func SavePostingSettings(req component.BetterRequest[SavePostingSettingsReq]) component.Response {
	return savePageConfig(pageConfig.PostingSettings, req.Params.Settings, func() {
		hotdataserve.ClearPostingSettingsConfigCache()
		llmsservice.ClearCache()
	})
}

// GetRateLimitSettings 获取滥用防护（限流）设置
func GetRateLimitSettings(req component.BetterRequest[component.Null]) component.Response {
	defaultSettings := defaultconfig.GetDefaultRateLimitConfig()
	res := pageConfig.GetConfigByPageType(pageConfig.RateLimitSettings, defaultSettings)
	return component.SuccessResponse(res)
}

type SaveRateLimitSettingsReq struct {
	Settings pageConfig.RateLimitConfig `json:"settings" validate:"required"`
}

// SaveRateLimitSettings 保存滥用防护（限流）设置
func SaveRateLimitSettings(req component.BetterRequest[SaveRateLimitSettingsReq]) component.Response {
	res := savePageConfig(pageConfig.RateLimitSettings, req.Params.Settings, hotdataserve.ClearRateLimitConfigCache)
	// 窗口/配额调整需对已存在的计数立即生效：清空内存计数，避免旧窗口的 entry.window
	// 继续作用于新配置（如 3600s → 60s 时旧 key 仍按 3600s 计数）。
	ratelimit.Default().ResetAll()
	return res
}

// GetMCPSettings 获取内置 MCP server 设置
func GetMCPSettings(req component.BetterRequest[component.Null]) component.Response {
	config := pageConfig.GetConfigByPageType(pageConfig.MCPSettings, defaultconfig.GetDefaultMCPSettingsConfig())
	return component.SuccessResponse(config)
}

type SaveMCPSettingsReq struct {
	Settings pageConfig.MCPSettingsConfig `json:"settings" validate:"required"`
}

// SaveMCPSettings 保存内置 MCP server 设置
func SaveMCPSettings(req component.BetterRequest[SaveMCPSettingsReq]) component.Response {
	return savePageConfig(pageConfig.MCPSettings, req.Params.Settings, hotdataserve.ClearMCPSettingsConfigCache)
}

// GetScheduleSettings 获取排课器节次作息表设置（未保存过时回内置默认 12 节作息）
func GetScheduleSettings(req component.BetterRequest[component.Null]) component.Response {
	config := pageConfig.GetConfigByPageType(pageConfig.ScheduleSettings, defaultconfig.GetDefaultScheduleSettingsConfig())
	return component.SuccessResponse(config)
}

type SaveScheduleSettingsReq struct {
	Settings pageConfig.ScheduleSettingsConfig `json:"settings" validate:"required"`
}

// SaveScheduleSettings 保存排课器节次作息表设置：条目需节次 1..12 且起止时间为
// 合法 HH:MM，任一条目非法整单拒绝（invalidParams，避免静默丢弃管理员输入）；
// 合法输入按节次升序排序并去重（同节次保留首个）后落库。
func SaveScheduleSettings(req component.BetterRequest[SaveScheduleSettingsReq]) component.Response {
	sectionTimes, ok := sanitizeScheduleSectionTimes(req.Params.Settings.SectionTimes)
	if !ok {
		return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
	}
	return savePageConfig(pageConfig.ScheduleSettings, pageConfig.ScheduleSettingsConfig{SectionTimes: sectionTimes}, hotdataserve.ClearScheduleSettingsConfigCache)
}

// isValidScheduleClockTime 校验严格 HH:MM（两位小时/分钟 + 冒号）且时钟值合法
// （time.Parse 的 "15" 布局可接受一位小时，故先做显式形状检查）。
func isValidScheduleClockTime(value string) bool {
	if len(value) != 5 || value[2] != ':' {
		return false
	}
	_, err := time.Parse("15:04", value)
	return err == nil
}

// sanitizeScheduleSectionTimes 校验并规范化节次条目：非法返回 false；
// 否则返回按节次升序、去重后的切片。起止时序（start < end，严格 HH:MM
// 下字典序即时间序）同样整单拒绝——管理端 UI 逐行校验时序，API 直连
// 客户端不能绕过该不变量。
func sanitizeScheduleSectionTimes(input []pageConfig.ScheduleSectionTime) ([]pageConfig.ScheduleSectionTime, bool) {
	seen := make(map[int]bool, len(input))
	times := make([]pageConfig.ScheduleSectionTime, 0, len(input))
	for _, item := range input {
		if item.Section < 1 || item.Section > 12 {
			return nil, false
		}
		if !isValidScheduleClockTime(item.Start) || !isValidScheduleClockTime(item.End) {
			return nil, false
		}
		// 严格 HH:MM 下字典序即时间序：start >= end（倒序或零时长）整单拒绝。
		if item.Start >= item.End {
			return nil, false
		}
		if seen[item.Section] {
			continue
		}
		seen[item.Section] = true
		times = append(times, item)
	}
	slices.SortFunc(times, func(a, b pageConfig.ScheduleSectionTime) int {
		return cmp.Compare(a.Section, b.Section)
	})
	return times, true
}

// GetAiSummarySettings 获取 AI 课程总结配置（B7, issue #181）。
// apiKey 仅回显是否已配置（明文/密文均不出现在响应中，issue #324 安全模式）。
func GetAiSummarySettings(req component.BetterRequest[component.Null]) component.Response {
	return component.SuccessResponse(hotdataserve.GetAiSummarySettingsView())
}

type SaveAiSummarySettingsReq struct {
	Settings pageConfig.AiSummarySettingsInput `json:"settings" validate:"required"`
}

// SaveAiSummarySettings 保存 AI 课程总结配置：apiKey 明文仅在请求瞬间存在——
// 非空时 securestore 加密后落库；为空时保留已存密文（issue #324 安全模式）。
func SaveAiSummarySettings(req component.BetterRequest[SaveAiSummarySettingsReq]) component.Response {
	input := req.Params.Settings
	if strings.TrimSpace(input.BaseURL) != "" {
		if !isValidHTTPURL(input.BaseURL) {
			return component.FailResponseCode(component.MessageAdminAiSummarySaveFailed,
				component.MessageParams{"error": "BaseURL 必须是合法的 http(s) URL"})
		}
	}
	entity := pageConfig.GetByPageType(pageConfig.AiSummarySettings)
	storage := jsonopt.Decode[pageConfig.AiSummarySettingsStorage](entity.Config)
	storage.Enabled = input.Enabled
	storage.GlobalPerMinute = input.GlobalPerMinute
	storage.BaseURL = strings.TrimRight(strings.TrimSpace(input.BaseURL), "/")
	storage.Model = strings.TrimSpace(input.Model)
	storage.Temperature = input.Temperature
	storage.MaxTokens = input.MaxTokens
	if key := strings.TrimSpace(input.APIKey); key != "" {
		sealed, err := securestore.EncryptPurpose(key, securestore.AiSummaryAPIKeyPurpose)
		if err != nil {
			return component.FailResponseError(fmt.Errorf("加密 AI 总结 apiKey 失败（请确认 app.signingKey 已配置）：%w", err))
		}
		storage.APIKeyEncrypted = sealed
	}
	// TODO: 尚无清除已存密钥的入口（apiKey 留空 = 保留已存密文）；如需显式
	// 清除需扩展保存契约（新字段或独立操作），另行跟进。
	return savePageConfig(pageConfig.AiSummarySettings, storage, hotdataserve.ClearAiSummarySettingsConfigCache)
}

// isValidHTTPURL 校验 BaseURL 为合法 http(s) 绝对 URL（允许内网/本机端点——
// 自托管 LLM 如 Ollama 常为内网地址，不做私网拒绝，仅防注入协议头）。
func isValidHTTPURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil {
		return false
	}
	return true
}

// ListAiSummaryModelsReq 拉取模型列表请求：支持携带临时 baseUrl/apiKey 以便
// 「先测试再保存」；留空则使用当前已保存配置。
type ListAiSummaryModelsReq struct {
	BaseURL string `json:"baseUrl"`
	APIKey  string `json:"apiKey,omitempty"`
}

// ListAiSummaryModelsResp 模型列表响应。
type ListAiSummaryModelsResp struct {
	Models []llmprovider.ModelInfo `json:"models"`
}

// ListAiSummaryModels 拉取 OpenAI-compatible /models 列表（管理后台自动获取 model）。
// 未实现 /models 的服务返回明确错误，前端允许手动输入 model 兜底。
func ListAiSummaryModels(req component.BetterRequest[ListAiSummaryModelsReq]) component.Response {
	cfg := llmprovider.Config{
		BaseURL: strings.TrimRight(strings.TrimSpace(req.Params.BaseURL), "/"),
		APIKey:  strings.TrimSpace(req.Params.APIKey),
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		stored := hotdataserve.GetAiSummarySettingsConfigCache()
		cfg.BaseURL = strings.TrimRight(stored.BaseURL, "/")
		cfg.APIKey = stored.APIKey
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return component.FailResponseCode(component.MessageAdminAiSummaryModelsFailed,
			component.MessageParams{"error": "请先填写 BaseURL（端点地址）"})
	}
	if !isValidHTTPURL(cfg.BaseURL) {
		return component.FailResponseCode(component.MessageAdminAiSummaryModelsFailed,
			component.MessageParams{"error": "BaseURL 必须是合法的 http(s) URL"})
	}
	ctx, cancel := context.WithTimeout(req.GinContext.Request.Context(), llmprovider.ModelsTimeout)
	defer cancel()
	models, err := cfg.ListModels(ctx)
	if err != nil {
		if errors.Is(err, llmprovider.ErrModelsUnsupported) {
			return component.FailResponseCode(component.MessageAdminAiSummaryModelsUnsupported, nil)
		}
		return component.FailResponseCode(component.MessageAdminAiSummaryModelsFailed,
			component.MessageParams{"error": "拉取模型列表失败"})
	}
	return component.SuccessResponse(ListAiSummaryModelsResp{Models: models})
}

// GetOnesystemSettings 获取一系统同步凭证配置：仅返回是否已配置，不回显密文或明文。
func GetOnesystemSettings(req component.BetterRequest[component.Null]) component.Response {
	config := hotdataserve.GetOnesystemSettingsConfigCache()
	return component.SuccessResponse(map[string]any{
		"cookieConfigured": strings.TrimSpace(config.CookieEncrypted) != "",
	})
}

type SaveOnesystemSettingsReq struct {
	// Cookie 一系统 Cookie header（明文，仅在保存瞬间存在）；留空表示清除已存凭证。
	Cookie string `json:"cookie" validate:"max=4096"`
}

// SaveOnesystemSettings 保存一系统 Cookie：securestore 加密后落库（密文经 OneSystemSettingsStorage
// 持久化，领域结构 json:"-" 防导出泄露），明文不持久化。清除时传空字符串。
func SaveOnesystemSettings(req component.BetterRequest[SaveOnesystemSettingsReq]) component.Response {
	encrypted := ""
	if cookie := strings.TrimSpace(req.Params.Cookie); cookie != "" {
		sealed, err := securestore.EncryptPurpose(cookie, securestore.OneSystemCookiePurpose)
		if err != nil {
			return component.FailResponseError(fmt.Errorf("加密一系统 Cookie 失败（请确认 app.signingKey 已配置）：%w", err))
		}
		encrypted = sealed
	}
	return savePageConfig(pageConfig.OneSystemSettings, pageConfig.OneSystemSettingsStorage{CookieEncrypted: encrypted}, hotdataserve.ClearOnesystemSettingsConfigCache)
}

// GetHttpNotifySettings 获取 HTTP 通知设置：仅回显各端点是否已配置密钥，不回显
// 密钥明文/密文（issue #324 S1）。
func GetHttpNotifySettings(req component.BetterRequest[component.Null]) component.Response {
	return component.SuccessResponse(hotdataserve.GetHttpNotifyView())
}

type SaveHttpNotifySettingsReq struct {
	Settings pageConfig.HttpNotifyConfigInput `json:"settings" validate:"required"`
}

// SaveHttpNotifySettings 保存 HTTP 通知设置：各端点 secret 明文仅在请求瞬间存在——
// 非空时 securestore 加密后落库；为空时按 id（无 id 按 url）保留已存密文/存量明文
// （issue #324 S1）。
func SaveHttpNotifySettings(req component.BetterRequest[SaveHttpNotifySettingsReq]) component.Response {
	input := req.Params.Settings
	entity := pageConfig.GetByPageType(pageConfig.HttpNotify)
	storage := jsonopt.Decode[pageConfig.HttpNotifyStorageConfig](entity.Config)
	existing := make(map[string]pageConfig.HttpNotifyStorageEndpoint, len(storage.Endpoints))
	for _, e := range storage.Endpoints {
		key := e.Id
		if key == "" {
			key = e.URL
		}
		existing[key] = e
	}
	next := make([]pageConfig.HttpNotifyStorageEndpoint, 0, len(input.Endpoints))
	for _, ep := range input.Endpoints {
		key := ep.Id
		if key == "" {
			key = ep.URL
		}
		orig := existing[key]
		sealed := orig.SecretEncrypted
		legacy := orig.Secret
		if secret := strings.TrimSpace(ep.Secret); secret != "" {
			encrypted, err := securestore.EncryptPurpose(secret, securestore.HttpNotifySecretPurpose)
			if err != nil {
				return component.FailResponseError(fmt.Errorf("加密 webhook secret 失败（请确认 app.signingKey 已配置）：%w", err))
			}
			sealed = encrypted
			legacy = ""
		}
		next = append(next, pageConfig.HttpNotifyStorageEndpoint{
			Id:                 ep.Id,
			Name:               ep.Name,
			Enabled:            ep.Enabled,
			URL:                ep.URL,
			Secret:             legacy,
			SecretEncrypted:    sealed,
			Events:             ep.Events,
			TimeoutSeconds:     ep.TimeoutSeconds,
			FailureCount:       ep.FailureCount,
			LastError:          ep.LastError,
			AbnormalTerminated: ep.AbnormalTerminated,
		})
	}
	return savePageConfig(pageConfig.HttpNotify, pageConfig.HttpNotifyStorageConfig{Enabled: input.Enabled, Endpoints: next}, hotdataserve.ClearHttpNotifyConfigCache)
}

// GetStorageSettings 获取存储设置：仅回显是否已配置凭据，不回显凭据明文/密文
// （issue #324 S3）。
func GetStorageSettings(req component.BetterRequest[component.Null]) component.Response {
	return component.SuccessResponse(hotdataserve.GetStorageSettingsView())
}

type SaveStorageSettingsReq struct {
	Settings pageConfig.StorageSettingsInput `json:"settings" validate:"required"`
}

// SaveStorageSettings 保存存储设置：accessKey/secretKey 明文仅在请求瞬间存在——
// 非空时 securestore 加密后落库；为空时保留已存密文（issue #324 S3）。
func SaveStorageSettings(req component.BetterRequest[SaveStorageSettingsReq]) component.Response {
	input := req.Params.Settings
	provider := input.Provider
	if provider == "" {
		provider = storageservice.ProviderLocal
	}
	if provider != storageservice.ProviderLocal && provider != storageservice.ProviderS3 {
		return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
	}
	if provider == storageservice.ProviderS3 && (input.Endpoint == "" || input.Bucket == "") {
		return component.FailResponseCode(component.MessageAdminStorageSaveFailed,
			component.MessageParams{"error": "S3 模式需要填写 Endpoint 与 Bucket"})
	}
	entity := pageConfig.GetByPageType(pageConfig.StorageSettingsPage)
	storage := jsonopt.Decode[pageConfig.StorageSettingsStorage](entity.Config)
	storage.Provider = provider
	storage.Endpoint = input.Endpoint
	storage.Bucket = input.Bucket
	storage.Region = input.Region
	storage.BucketLookup = input.BucketLookup
	storage.Secure = input.Secure
	storage.PublicUrlPrefix = input.PublicUrlPrefix
	if key := strings.TrimSpace(input.AccessKey); key != "" {
		sealed, err := securestore.EncryptPurpose(key, securestore.StorageAccessKeyPurpose)
		if err != nil {
			return component.FailResponseError(fmt.Errorf("加密存储 accessKey 失败（请确认 app.signingKey 已配置）：%w", err))
		}
		storage.AccessKeyEncrypted = sealed
		storage.AccessKey = ""
	}
	if key := strings.TrimSpace(input.SecretKey); key != "" {
		sealed, err := securestore.EncryptPurpose(key, securestore.StorageSecretKeyPurpose)
		if err != nil {
			return component.FailResponseError(fmt.Errorf("加密存储 secretKey 失败（请确认 app.signingKey 已配置）：%w", err))
		}
		storage.SecretKeyEncrypted = sealed
		storage.SecretKey = ""
	}
	return savePageConfig(pageConfig.StorageSettingsPage, storage, hotdataserve.ClearStorageSettingsConfigCache)
}

type TestStorageConnectionReq struct {
	Settings pageConfig.StorageSettingsInput `json:"settings" validate:"required"`
}

type TestStorageConnectionResp struct {
	Success     bool                    `json:"success"`
	MessageCode component.MessageCode   `json:"messageCode"`
	Params      component.MessageParams `json:"params,omitempty"`
}

// TestStorageConnection 测试存储连接（不落库）
func TestStorageConnection(req component.BetterRequest[TestStorageConnectionReq]) component.Response {
	cfg := req.Params.Settings.ToConfig()
	// 管理端 GET 不再回显凭据：测试时凭据留空则使用已存凭据（issue #324 S3）。
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		stored := hotdataserve.GetStorageSettingsConfigCache()
		if cfg.AccessKey == "" {
			cfg.AccessKey = stored.AccessKey
		}
		if cfg.SecretKey == "" {
			cfg.SecretKey = stored.SecretKey
		}
	}
	if cfg.Provider == "" {
		cfg.Provider = storageservice.ProviderLocal
	}
	if cfg.Provider == storageservice.ProviderLocal {
		return component.SuccessResponse(TestStorageConnectionResp{
			Success:     true,
			MessageCode: component.MessageAdminStorageTestSuccess,
		})
	}
	if err := storageservice.TestConnection(requestContext(req.GinContext), cfg); err != nil {
		return component.SuccessResponse(TestStorageConnectionResp{
			Success:     false,
			MessageCode: component.MessageAdminStorageTestFailed,
			Params:      component.MessageParams{"error": err.Error()},
		})
	}
	return component.SuccessResponse(TestStorageConnectionResp{
		Success:     true,
		MessageCode: component.MessageAdminStorageTestSuccess,
	})
}

type CreateStorageMigrateReq struct {
	ClearAfterMigrate bool `json:"clearAfterMigrate"`
}

// CreateStorageMigrateTask 创建文件迁移到对象存储的后台任务
func CreateStorageMigrateTask(req component.BetterRequest[CreateStorageMigrateReq]) component.Response {
	taskID, err := filemigrateservice.CreateMigrateTask(req.Params.ClearAfterMigrate)
	if err != nil {
		if errors.Is(err, errStorageProviderInvalid) {
			return component.FailResponseCode(component.MessageAdminStorageMigrateInvalidProvider, nil)
		}
		return component.FailResponseCode(component.MessageAdminStorageMigrateFailed,
			component.MessageParams{"error": err.Error()})
	}
	return successDataMap("taskId", taskID)
}

// GetStorageMigrateTasks 获取文件迁移任务列表
func GetStorageMigrateTasks(req component.BetterRequest[component.Null]) component.Response {
	tasks, err := filemigrateservice.ListMigrateTasks(20)
	if err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	return component.SuccessResponse(tasks)
}

// errStorageProviderInvalid marks "provider is not s3" without leaking details.
var errStorageProviderInvalid = errors.New("storage provider is not s3")

// 数据管理：导出
type CreateExportTaskReq struct {
	Tables []string `json:"tables" validate:"required"`
	Format string   `json:"format" validate:"required,oneof=json csv"`
}

// CreateExportTask 创建数据导出后台任务（issue #324 S4：操作审计）。
func CreateExportTask(req component.BetterRequest[CreateExportTaskReq]) component.Response {
	taskID, err := dataservice.ExportData(req.Params.Tables, req.Params.Format)
	if err != nil {
		return component.FailResponseCode(component.MessageAdminDataExportFailed,
			component.MessageParams{"error": err.Error()})
	}
	optlogger.UserOptCode(req.UserId, optlogger.ExportData, taskID, "admin.opt.data.exported", optlogger.MessageParams{
		"tables": req.Params.Tables,
		"format": req.Params.Format,
	})
	return successDataMap("taskId", taskID)
}

// ListExportTasks 获取导出任务列表
func ListExportTasks(req component.BetterRequest[component.Null]) component.Response {
	tasks, err := dataservice.ListExportTasks(20)
	if err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	return component.SuccessResponse(tasks)
}

// DownloadExportTask 下载导出文件（issue #324 S4：下载操作审计）。
func DownloadExportTask(c *gin.Context) {
	taskID := c.Param("taskId")
	task, err := taskQueue.GetByID(taskID)
	if err != nil || task.Id == 0 {
		c.JSON(http.StatusNotFound, component.FailDataCode(component.MessageAdminDataTaskNotFound, nil))
		return
	}
	if task.Type != dataservice.TaskTypeExport {
		c.JSON(http.StatusNotFound, component.FailDataCode(component.MessageAdminDataTaskNotFound, nil))
		return
	}
	if task.Status != taskQueue.StatusSuccess {
		c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageAdminDataTaskNotReady, nil))
		return
	}
	path, err := dataservice.ExportFilePath(&task)
	if err != nil {
		c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageAdminDataDownloadDenied, nil))
		return
	}
	fileName := filepath.Base(path)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
	optlogger.UserOptCode(c.GetUint64("userId"), optlogger.ExportData, task.Id, "admin.opt.data.exported.download", optlogger.MessageParams{
		"fileName": fileName,
	})
	c.File(path)
}

// ListImportTasks 获取导入任务状态列表。
func ListImportTasks(req component.BetterRequest[component.Null]) component.Response {
	tasks, err := dataservice.ListImportTasks(20)
	if err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	return component.SuccessResponse(tasks)
}

type ImportTaskReq struct {
	TaskId uint64 `uri:"taskId" validate:"required"`
}

// ReplayImportTask requeues a failed import while preserving its staged body.
func ReplayImportTask(req component.BetterRequest[ImportTaskReq]) component.Response {
	task, err := dataservice.ReplayImportTaskContext(requestContext(req.GinContext), req.Params.TaskId)
	if err != nil {
		return component.FailResponseCode(component.MessageOperationFailed,
			component.MessageParams{"error": "导入任务当前不可重放"})
	}
	return component.SuccessResponse(task)
}

// ImportData 导入 JSON 数据（multipart file 字段）
func ImportData(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, dataservice.MaxImportSize)
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageAdminDataImportFailed,
			component.MessageParams{"error": "未获取到上传文件"}))
		return
	}
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageAdminDataImportFailed,
			component.MessageParams{"error": "读取上传文件失败，请稍后重试"}))
		return
	}
	defer func() { _ = src.Close() }()
	data, err := io.ReadAll(io.LimitReader(src, dataservice.MaxImportSize+1))
	if err != nil {
		c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageAdminDataImportFailed,
			component.MessageParams{"error": "读取上传文件失败，请稍后重试"}))
		return
	}
	if len(data) > dataservice.MaxImportSize {
		c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageAdminDataImportFailed,
			component.MessageParams{"error": "导入文件超过 50MB 限制"}))
		return
	}
	report, err := dataservice.EnqueueImport(c.Request.Context(), data, "json")
	if err != nil {
		if errors.Is(err, dataservice.ErrImportInvalidFormat) {
			c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageAdminDataImportInvalidFormat,
				component.MessageParams{"error": "导入文件格式无效"}))
			return
		}
		slog.Error("admin import enqueue failed", "error", err)
		c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageAdminDataImportFailed,
			component.MessageParams{"error": "导入任务暂存失败，请稍后重试"}))
		return
	}
	c.JSON(http.StatusOK, component.SuccessData(report))
}

// 审核队列：待审内容（ProcessStatus=2）
type ReviewQueueReq struct {
	Kind     string `json:"kind" validate:"required,oneof=topic post"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
}

type ReviewQueueItem struct {
	Id            uint64 `json:"id"`
	Title         string `json:"title"`
	Excerpt       string `json:"excerpt"`
	UserId        uint64 `json:"userId"`
	Username      string `json:"username"`
	ProcessStatus int8   `json:"processStatus"`
	CreatedAt     string `json:"createdAt"`
	TopicId       uint64 `json:"topicId,omitempty"`
	PostNo        uint64 `json:"postNo,omitempty"`
}

// ReviewQueue 列出待审的主题或回复。
func ReviewQueue(req component.BetterRequest[ReviewQueueReq]) component.Response {
	page := req.Params.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.Params.PageSize
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	items := make([]ReviewQueueItem, 0, pageSize)
	var total int64
	if req.Params.Kind == "topic" {
		result := topics.PagePendingReview(page, pageSize)
		total = result.Total
		userIDs := make([]uint64, 0, len(result.Data))
		for _, t := range result.Data {
			userIDs = append(userIDs, t.UserId)
		}
		userMap := users.GetMapByIds(userIDs)
		for _, t := range result.Data {
			username := ""
			if u, ok := userMap[t.UserId]; ok {
				username = u.Username
			}
			excerpt := t.Excerpt
			if excerpt == "" {
				excerpt = t.Title
			}
			items = append(items, ReviewQueueItem{
				Id: t.Id, Title: t.Title, Excerpt: excerpt,
				UserId: t.UserId, Username: username,
				ProcessStatus: t.ProcessStatus,
				CreatedAt:     t.CreatedAt.Format(time.RFC3339),
			})
		}
	} else {
		result := posts.PagePendingReview(page, pageSize)
		total = result.Total
		userIDs := make([]uint64, 0, len(result.Data))
		for _, p := range result.Data {
			userIDs = append(userIDs, p.UserId)
		}
		userMap := users.GetMapByIds(userIDs)
		topicIDs := make([]uint64, 0, len(result.Data))
		for _, p := range result.Data {
			topicIDs = append(topicIDs, p.TopicId)
		}
		topicMap, err := topics.GetMapByIds(topicIDs)
		if err != nil {
			slog.Error("admin review queue: load topics failed", "error", err)
			return component.BuildResponse(http.StatusInternalServerError,
				component.FailDataCode(component.MessageAdminReviewFailed, nil))
		}
		for _, p := range result.Data {
			username := ""
			if u, ok := userMap[p.UserId]; ok {
				username = u.Username
			}
			title := ""
			if t, ok := topicMap[p.TopicId]; ok {
				title = t.Title
			}
			excerpt := p.Content
			if len(excerpt) > 120 {
				excerpt = excerpt[:120]
			}
			items = append(items, ReviewQueueItem{
				Id: p.Id, Title: title, Excerpt: excerpt,
				UserId: p.UserId, Username: username,
				ProcessStatus: p.ProcessStatus,
				CreatedAt:     p.CreatedAt.Format(time.RFC3339),
				TopicId:       p.TopicId, PostNo: p.PostNo,
			})
		}
	}
	return component.SuccessResponse(map[string]any{
		"items": items, "total": total, "page": page, "pageSize": pageSize,
	})
}

type ReviewActionReq struct {
	Kind    string `json:"kind" validate:"required,oneof=topic post"`
	Id      uint64 `json:"id" validate:"required"`
	Approve bool   `json:"approve"`
}

// ReviewAction 审核通过（ProcessStatus=0）或拒绝（ProcessStatus=1）。
func ReviewAction(req component.BetterRequest[ReviewActionReq]) component.Response {
	targetStatus := int8(0)
	if !req.Params.Approve {
		targetStatus = 1
	}
	if req.Params.Kind == "topic" {
		topic := topics.Get(req.Params.Id)
		if topic.Id == 0 {
			return component.FailResponseCode(component.MessageAdminReviewNotFound, nil)
		}
		// wiki 分站内容走 wiki 修订审核流程（review N1）：禁止在论坛审核队列
		// 直接通过/拒绝 wiki 主题，避免绕过 wiki_page_revisions 状态流转。
		if topic.TopicType == topics.TopicTypeWiki {
			return component.FailResponseCode(component.MessageAdminReviewTargetInvalid, nil)
		}
		if topic.ProcessStatus != topics.ProcessStatusPending {
			return component.FailResponseCode(component.MessageAdminReviewProcessed, nil)
		}
		if err := db.ConnectContext(requestContext(req.GinContext)).Transaction(func(tx *gorm.DB) error {
			if err := topics.UpdateProcessStatusTx(tx, topic.Id, targetStatus); err != nil {
				return err
			}
			// 首楼同步状态
			if topic.FirstPostId > 0 {
				if err := posts.UpdateProcessStatusTx(tx, topic.FirstPostId, targetStatus); err != nil {
					return err
				}
			}
			return searchservice.EnqueueTopicSearchTask(tx, topic.Id)
		}); err != nil {
			return component.FailResponseCode(component.MessageAdminReviewFailed,
				component.MessageParams{"error": err.Error()})
		}
		topic.ProcessStatus = targetStatus
		hotdataserve.InvalidateTopicListCacheForCategories(topic.CategoryIds...)
		// 审核后无条件重建搜索索引（issue #132）：拒绝（ProcessStatus→blocked）
		// 时 BuildSingleTopicSearchDocument 会把文档从索引删除，避免被拒话题
		// 残留在公共搜索；批准时 upsert 恢复（下方事件也会重建，幂等）。
		firstPost := posts.Get(topic.FirstPostId)
		// 批准后补发事件：新建主题发完整发布事件（搜索索引/统计/积分/活动/通知），
		// 编辑主题仅重建索引与通知，避免重复积分。
		if req.Params.Approve && topic.Status == 1 {
			if userActivities.HasRecord(userActivities.ActionPost, userActivities.SubjectTopic, topic.Id) {
				eventbus.Publish(detachedRequestContext(req.GinContext), &eventhandlers.TopicUpdatedEvent{Topic: &topic, FirstPost: &firstPost})
			} else {
				userStatistics.WriteTopic(topic.UserId)
				eventbus.Publish(detachedRequestContext(req.GinContext), &eventhandlers.TopicPublishedEvent{Topic: &topic, FirstPost: &firstPost})
			}
		}
		optlogger.UserOptCode(req.UserId, optlogger.EditTopic, topic.Id, "admin.opt.review.topic",
			optlogger.MessageParams{"id": topic.Id, "approve": req.Params.Approve})
	} else {
		post := posts.Get(req.Params.Id)
		if post.Id == 0 {
			return component.FailResponseCode(component.MessageAdminReviewNotFound, nil)
		}
		// 仅 wiki 首楼（post_no<=1）由 wiki 修订审核流程管理，禁止在论坛队列直接审核；
		// wiki 分站评论（post_no>1）仍走论坛审核队列（review N1）。
		if topicEntity := topics.GetSimple(post.TopicId); topicEntity.TopicType == topics.TopicTypeWiki && post.PostNo <= 1 {
			return component.FailResponseCode(component.MessageAdminReviewTargetInvalid, nil)
		}
		if post.ProcessStatus != posts.ProcessStatusPending {
			return component.FailResponseCode(component.MessageAdminReviewProcessed, nil)
		}
		topicEntity := topics.GetSimple(post.TopicId)
		if err := db.ConnectContext(requestContext(req.GinContext)).Transaction(func(tx *gorm.DB) error {
			if err := posts.UpdateProcessStatusTx(tx, post.Id, targetStatus); err != nil {
				return err
			}
			return searchservice.EnqueueTopicSearchTask(tx, topicEntity.Id)
		}); err != nil {
			return component.FailResponseCode(component.MessageAdminReviewFailed,
				component.MessageParams{"error": err.Error()})
		}
		hotdataserve.InvalidateTopicListCacheForCategories(topicEntity.CategoryIds...)
		// 批准后补发事件：仅对新建待审回复补发（编辑场景创建时已发布过事件）。
		if req.Params.Approve && !userActivities.HasRecord(userActivities.ActionComment, userActivities.SubjectPost, post.Id) {
			userStatistics.WriteComment(post.UserId)
			topicEntity := topics.GetSimple(post.TopicId)
			replyToAuthorID := uint64(0)
			if post.ReplyToPostId > 0 {
				if parent := posts.Get(post.ReplyToPostId); parent.Id > 0 {
					replyToAuthorID = parent.UserId
				}
			}
			eventbus.Publish(detachedRequestContext(req.GinContext), &eventhandlers.CommentCreatedEvent{
				TopicId:             post.TopicId,
				PostId:              post.Id,
				PostNo:              post.PostNo,
				UserId:              post.UserId,
				Content:             post.Content,
				TopicAuthorId:       topicEntity.UserId,
				ReplyToPostId:       post.ReplyToPostId,
				ReplyToPostAuthorId: replyToAuthorID,
			})
		}
		optlogger.UserOptCode(req.UserId, optlogger.EditTopic, post.TopicId, "admin.opt.review.post",
			optlogger.MessageParams{"id": post.Id, "topicId": post.TopicId, "approve": req.Params.Approve})
	}
	return component.SuccessResponseCode("success", component.MessageOperationSuccess, nil)
}

// GetTermsOfService 获取服务条款配置
func GetTermsOfService(req component.BetterRequest[component.Null]) component.Response {
	config := pageConfig.GetConfigByPageType(pageConfig.TermsOfService, defaultconfig.GetDefaultTermsOfServiceConfig())
	return component.SuccessResponse(config)
}

// GetPrivacyPolicy 获取隐私政策配置
func GetPrivacyPolicy(req component.BetterRequest[component.Null]) component.Response {
	config := pageConfig.GetConfigByPageType(pageConfig.PrivacyPolicy, defaultconfig.GetDefaultPrivacyPolicyConfig())
	return component.SuccessResponse(config)
}

type SaveTermsOfServiceReq struct {
	Settings pageConfig.TermsOfServiceConfig `json:"settings" validate:"required"`
}

// SaveTermsOfService 保存服务条款配置
func SaveTermsOfService(req component.BetterRequest[SaveTermsOfServiceReq]) component.Response {
	req.Params.Settings.HtmlContent = ""
	return savePageConfig(pageConfig.TermsOfService, req.Params.Settings, hotdataserve.ClearTermsOfServiceConfigCache)
}

type SavePrivacyPolicyReq struct {
	Settings pageConfig.PrivacyPolicyConfig `json:"settings" validate:"required"`
}

// SavePrivacyPolicy 保存隐私政策配置
func SavePrivacyPolicy(req component.BetterRequest[SavePrivacyPolicyReq]) component.Response {
	req.Params.Settings.HtmlContent = ""
	return savePageConfig(pageConfig.PrivacyPolicy, req.Params.Settings, hotdataserve.ClearPrivacyPolicyConfigCache)
}

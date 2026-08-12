package api

import (
	"context"

	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/algorithm"
	"github.com/leancodebox/GooseForum/app/bundles/captchaOpt"
	"github.com/leancodebox/GooseForum/app/bundles/eventbus"
	"github.com/leancodebox/GooseForum/app/bundles/i18n"
	"github.com/leancodebox/GooseForum/app/http/controllers/component"
	"github.com/leancodebox/GooseForum/app/models/filemodel/filedata"
	"github.com/leancodebox/GooseForum/app/models/forum/userFollow"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/models/hotdataserve"
	"github.com/leancodebox/GooseForum/app/service/emailactivationservice"
	"github.com/leancodebox/GooseForum/app/service/eventhandlers"
	"github.com/leancodebox/GooseForum/app/service/fileusageservice"
	"github.com/leancodebox/GooseForum/app/service/mailservice"
	"github.com/leancodebox/GooseForum/app/service/moderationservice"
	"github.com/leancodebox/GooseForum/app/service/oauthservice"
	"github.com/leancodebox/GooseForum/app/service/tokenservice"
	"github.com/leancodebox/GooseForum/app/service/urlconfig"
	"github.com/leancodebox/GooseForum/app/service/userservice"

	"github.com/gin-gonic/gin"
)

func GetCaptcha(req component.BetterRequest[component.Null]) component.Response {
	captchaId, captchaImg := captchaOpt.GenerateCaptcha()
	return component.SuccessResponse(map[string]any{
		"captchaId":  captchaId,
		"captchaImg": captchaImg,
	})
}

type GetUserCardReq struct {
	UserId uint64 `form:"userId" json:"userId" binding:"required"`
}

func GetUserCard(req component.BetterRequest[GetUserCardReq]) component.Response {
	userId := req.Params.UserId
	card, ok := userservice.GetUserCard(userId)
	if !ok {
		return component.FailResponseCode(component.MessageUserNotFound, nil)
	}
	currentUserId := req.UserId
	card.IsSelf = currentUserId == userId
	card.IsFollowing = false
	if currentUserId > 0 && currentUserId != userId {
		card.IsFollowing = userFollow.IsFollowing(currentUserId, userId)
	}

	return component.SuccessResponse(card)
}

// emailChangeCooldown 邮箱变更冷静期：变更后 24 小时内新邮箱不能用于密码重置。
const emailChangeCooldown = 24 * time.Hour

type EditUserEmailReq struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// EditUserEmail updates the current user's email and resets activation state.
// 修改邮箱需要登录密码二次确认（与 ChangePassword 一致），校验失败在触碰数据库前拒绝。
func EditUserEmail(req component.BetterRequest[EditUserEmailReq]) component.Response {
	userEntity, err := req.GetUser()
	if err != nil {
		return component.FailResponseCode(component.MessageUserFetchFailed, nil)
	}

	if err = algorithm.VerifyEncryptPassword(userEntity.Password, req.Params.Password); err != nil {
		if userEntity.Email == "" && oauthservice.HasOAuthBinding(userEntity.Id) {
			return component.FailResponseCode(component.MessageAuthPasswordOAuthRequired, nil)
		}
		return component.FailResponseCode(component.MessageAuthOldPasswordInvalid, nil)
	}

	newEmail := strings.ToLower(strings.TrimSpace(req.GetParams().Email))

	if err := component.ValidateEmailDomain(newEmail); err != nil {
		return component.FailResponseError(err)
	}

	if users.ExistEmail(newEmail) {
		return component.FailResponseCode(component.MessageAuthEmailExists, nil)
	}

	// 保存新邮箱前记录旧邮箱，用于写库成功后向旧地址发送变更通知。
	oldEmail := userEntity.Email
	now := time.Now()
	userEntity.Email = newEmail
	userEntity.IsActivated = users.ActivationPending
	userEntity.ActivatedAt = nil
	userEntity.EmailChangedAt = &now

	err = userservice.SaveUser(&userEntity)
	if err != nil {
		return component.FailResponseCode(component.MessageUserUpdateFailed, nil)
	}

	// 新邮箱：激活邮件
	if err = emailactivationservice.SendActivationEmail(&userEntity); err != nil {
		slog.Info("验证邮件发送失败", "error", err)
	}

	// 旧邮箱：变更通知（失败只记日志，不阻断成功响应；
	// 这是受害者得知邮箱被改的主要途径，失败需以 Error 级暴露以便告警/审计）。
	if err = mailservice.AddToQueue(mailservice.EmailTask{
		To:       oldEmail,
		Username: userEntity.Username,
		NewEmail: newEmail,
		Type:     "email_changed",
		Locale:   userEntity.Locale,
	}); err != nil {
		slog.Error("邮箱变更通知入队失败", "userId", userEntity.Id, "oldEmail", oldEmail, "error", err)
	}

	return component.SuccessResponseCode("更新成功", component.MessageUserUpdateSuccess, nil)
}

func ResendActivationEmail(req component.BetterRequest[component.Null]) component.Response {
	userEntity, err := req.GetUser()
	if err != nil || userEntity.Id == 0 {
		return component.FailResponseCode(component.MessageUserFetchFailed, nil)
	}

	result, err := emailactivationservice.Resend(userEntity)
	if err != nil {
		if errors.Is(err, emailactivationservice.ErrDisabled) {
			return component.FailResponseCode(component.MessageAuthActivationDisabled, nil)
		}
		if errors.Is(err, emailactivationservice.ErrAlreadyVerified) {
			return component.FailResponseCode(component.MessageAuthActivationAlreadyVerified, nil)
		}
		if errors.Is(err, emailactivationservice.ErrCooldown) {
			return component.FailResponseCode(component.MessageAuthActivationResendCooldown, component.MessageParams{
				"retryAfterSeconds": result.RetryAfterSeconds,
			})
		}
		if errors.Is(err, emailactivationservice.ErrDailyLimit) {
			return component.FailResponseCode(component.MessageAuthActivationResendDaily, component.MessageParams{
				"limit": result.DailyLimit,
			})
		}
		slog.Error("resend activation email failed", "userId", userEntity.Id, "err", err)
		return component.FailResponseCode(component.MessageAuthActivationResendFailed, nil)
	}

	return component.SuccessResponseCode(map[string]any{
		"remainingToday": result.RemainingToday,
	}, component.MessageAuthActivationResendSuccess, component.MessageParams{
		"remainingToday": result.RemainingToday,
	})
}

type EditUsernameReq struct {
	Username string `json:"username" validate:"required"`
}

// EditUsername updates the current user's username.
func EditUsername(req component.BetterRequest[EditUsernameReq]) component.Response {
	userEntity, err := req.GetUser()
	if err != nil {
		return component.FailResponseCode(component.MessageUserFetchFailed, nil)
	}
	newUsername := req.GetParams().Username
	if !component.ValidateUsername(newUsername) {
		return component.FailResponseCode(component.MessageAuthUsernameInvalid, nil)
	}
	// 保留/禁用用户名检查
	if _, err := moderationservice.CheckUsernameAllowed(newUsername); err != nil {
		return component.FailResponseError(err)
	}
	if users.ExistUsername(newUsername) {
		return component.FailResponseCode(component.MessageAuthUsernameExists, nil)
	}
	userEntity.Username = newUsername
	err = userservice.SaveUser(&userEntity)
	if err != nil {
		return component.FailResponseCode(component.MessageUserUpdateFailed, nil)
	}

	eventbus.Publish(context.Background(), &eventhandlers.UserSearchIndexUpdatedEvent{UserId: userEntity.Id})

	return component.SuccessResponseCode("更新成功", component.MessageUserUpdateSuccess, nil)
}

type EditUserInfoReq struct {
	Nickname            string                    `json:"nickname"`
	Bio                 string                    `json:"bio"`
	Signature           string                    `json:"signature"`
	Website             string                    `json:"website"`
	WebsiteName         string                    `json:"websiteName"`
	Locale              string                    `json:"locale,omitempty"`
	ExternalInformation users.ExternalInformation `json:"externalInformation"`
}

// EditUserInfo updates the current user's profile fields.
func EditUserInfo(req component.BetterRequest[EditUserInfoReq]) component.Response {
	userEntity, err := req.GetUser()
	if err != nil {
		return component.FailResponseCode(component.MessageUserFetchFailed, nil)
	}

	userEntity.Nickname = req.Params.Nickname
	// 全字段覆盖：bio/signature 允许清空（空字符串也要落库）
	userEntity.Bio = req.Params.Bio
	userEntity.Signature = req.Params.Signature
	userEntity.Website = req.Params.Website
	userEntity.WebsiteName = req.Params.WebsiteName
	if strings.TrimSpace(req.Params.Locale) != "" {
		userEntity.Locale = i18n.Normalize(req.Params.Locale)
	}
	userEntity.ExternalInformation = req.Params.ExternalInformation

	err = userservice.SaveUser(&userEntity)
	if err != nil {
		return component.FailResponseCode(component.MessageUserUpdateFailed, nil)
	}
	eventbus.Publish(context.Background(), &eventhandlers.UserSearchIndexUpdatedEvent{UserId: userEntity.Id})
	return component.SuccessResponseCode("更新成功", component.MessageUserUpdateSuccess, nil)
}

type EditUserProfileCoverReq struct {
	ProfileCoverUrl string `json:"profileCoverUrl"`
}

// EditUserProfileCover updates the current user's profile cover.
func EditUserProfileCover(req component.BetterRequest[EditUserProfileCoverReq]) component.Response {
	userEntity, err := req.GetUser()
	if err != nil {
		return component.FailResponseCode(component.MessageUserFetchFailed, nil)
	}
	if userEntity.RoleId == 0 {
		return component.FailResponseCode(component.MessagePermissionDenied, nil)
	}

	userEntity.ProfileCoverUrl = strings.TrimSpace(req.Params.ProfileCoverUrl)
	err = userservice.SaveUser(&userEntity)
	if err != nil {
		return component.FailResponseCode(component.MessageUserUpdateFailed, nil)
	}
	return component.SuccessResponseCode("更新成功", component.MessageUserUpdateSuccess, nil)
}

type SetPresetAvatarReq struct {
	AvatarUrl string `json:"avatarUrl" validate:"required"`
}

// SetPresetAvatar switches the current user to a built-in avatar.
func SetPresetAvatar(req component.BetterRequest[SetPresetAvatarReq]) component.Response {
	userEntity, err := req.GetUser()
	if err != nil {
		return component.FailResponseCode(component.MessageUserFetchFailed, nil)
	}

	avatarUrl := strings.TrimSpace(req.Params.AvatarUrl)
	if !isAllowedPresetAvatar(avatarUrl) {
		return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
	}

	userEntity.AvatarUrl = avatarUrl
	if err := userservice.SaveUser(&userEntity); err != nil {
		return component.FailResponseCode(component.MessageUserUpdateFailed, nil)
	}
	return component.SuccessResponse(map[string]string{
		"avatarUrl": userEntity.GetWebAvatarUrl(),
	})
}

type WearBadgeReq struct {
	BadgeCode string `json:"badgeCode"`
}

func WearBadge(req component.BetterRequest[WearBadgeReq]) component.Response {
	badgeCode := strings.TrimSpace(req.Params.BadgeCode)
	if !userservice.SetWornBadge(req.UserId, badgeCode) {
		return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
	}
	return component.SuccessResponseCode("更新成功", component.MessageUserUpdateSuccess, nil)
}

func isAllowedPresetAvatar(avatarUrl string) bool {
	for i := 1; i <= 12; i++ {
		if avatarUrl == fmt.Sprintf("/static/pic/%d.webp", i) {
			return true
		}
	}
	return false
}

// UploadAvatar stores a new avatar for the current user.
func UploadAvatar(c *gin.Context) {
	postingConfig := hotdataserve.GetPostingSettingsConfigCache()
	if !postingConfig.UploadControl.AllowAttachments {
		c.JSON(200, component.FailDataCode(component.MessageUploadAttachmentDisabled, nil))
		return
	}

	userId := c.GetUint64("userId")

	if userId == 0 {
		c.JSON(200, component.FailDataCode(component.MessageAuthRequired, nil))
		return
	}

	userEntity, err := users.Get(userId)
	if err != nil {
		c.JSON(200, component.FailDataCode(component.MessageUserFetchFailed, nil))
		return
	}

	if code, err := component.CheckUserPermission(&userEntity, component.PermissionActionUploadAttachment); err != nil {
		c.JSON(code, component.FailDataError(err))
		return
	}

	if postingConfig.UploadControl.NewUserUploadCooldownMinutes > 0 {
		cooldownTime := userEntity.CreatedAt.Add(time.Duration(postingConfig.UploadControl.NewUserUploadCooldownMinutes) * time.Minute)
		if time.Now().Before(cooldownTime) {
			minutes := postingConfig.UploadControl.NewUserUploadCooldownMinutes
			availableAt := cooldownTime.Format("2006-01-02 15:04:05")
			c.JSON(200, component.FailDataCode(
				component.MessageUploadCooldown,

				component.MessageParams{"minutes": minutes, "availableAt": availableAt}))
			return
		}
	}

	files, err := avatarFormFiles(c)
	if err != nil {
		slog.Error(err.Error())
		c.JSON(200, component.FailDataCode(component.MessageUploadFileMissing, nil))
		return
	}

	fileCount := files.Count()
	if postingConfig.UploadControl.MaxDailyUploadsPerUser > 0 {
		count := filedata.CountDailyUploads(userId)
		if count+int64(fileCount) > int64(postingConfig.UploadControl.MaxDailyUploadsPerUser) {
			c.JSON(200, component.FailDataCode(
				component.MessageUploadDailyLimitAvatar,

				component.MessageParams{"count": count, "fileCount": fileCount}))
			return
		}
	}

	maxSize := avatarUploadMaxSize()
	if configMaxSize := int64(postingConfig.UploadControl.MaxAttachmentSizeKb) * 1024; configMaxSize > 0 && configMaxSize < maxSize {
		maxSize = configMaxSize
	}
	allowedExts := postingConfig.UploadControl.AuthorizedExtensions

	mainData, err := readAvatarUploadFile(files.Main, maxSize, allowedExts)
	if err != nil {
		c.JSON(200, component.FailDataError(err))
		return
	}

	var fileEntities []*filedata.Entity
	if files.AvatarMedium == nil {
		fileEntity, err := filedata.SaveAvatar(userId, mainData, files.Main.Filename)
		if err != nil {
			c.JSON(200, component.FailDataCode(
				component.MessageUploadSaveFailed,

				component.MessageParams{"error": err.Error()}))
			return
		}
		fileEntities = []*filedata.Entity{fileEntity}
	} else {
		uploads := make([]filedata.AvatarUpload, 0, 2)
		uploads = append(uploads, filedata.AvatarUpload{
			Filename: files.Main.Filename,
			Data:     mainData,
		})
		fileData, err := readAvatarUploadFile(files.AvatarMedium, maxSize, allowedExts)
		if err != nil {
			c.JSON(200, component.FailDataError(err))
			return
		}
		uploads = append(uploads, filedata.AvatarUpload{
			Filename: files.AvatarMedium.Filename,
			Data:     fileData,
		})

		fileEntities, err = filedata.SaveAvatarSet(userId, uploads)
		if err != nil {
			c.JSON(200, component.FailDataCode(
				component.MessageUploadSaveFailed,

				component.MessageParams{"error": err.Error()}))
			return
		}
	}

	userEntity.AvatarUrl = fileEntities[0].Name
	if err := userservice.SaveUser(&userEntity); err != nil {
		c.JSON(200, component.FailDataCode(component.MessageUserUpdateFailed, nil))
		return
	}

	fileNames := make([]string, 0, len(fileEntities))
	for _, fileEntity := range fileEntities {
		if fileEntity != nil {
			fileNames = append(fileNames, fileEntity.Name)
		}
	}
	fileusageservice.ReplaceAvatar(userId, fileNames)

	response := map[string]string{
		"avatarUrl": urlconfig.FilePath(fileEntities[0].Name),
	}
	if len(fileEntities) > 1 {
		response["avatarMediumUrl"] = urlconfig.FilePath(fileEntities[1].Name)
	}
	c.JSON(200, component.SuccessDataCode(response, component.MessageUploadSuccess, nil))
}

type avatarUploadFiles struct {
	Main         *multipart.FileHeader
	AvatarMedium *multipart.FileHeader
}

func (files avatarUploadFiles) Count() int {
	count := 0
	for _, file := range []*multipart.FileHeader{files.Main, files.AvatarMedium} {
		if file != nil {
			count++
		}
	}
	return count
}

func avatarFormFiles(c *gin.Context) (avatarUploadFiles, error) {
	main, err := c.FormFile("avatar")
	if err != nil {
		return avatarUploadFiles{}, err
	}

	files := avatarUploadFiles{Main: main}
	files.AvatarMedium, _ = c.FormFile("avatarMedium")
	return files, nil
}

func avatarUploadMaxSize() int64 {
	return int64(filedata.MaxFileSize)
}

func readAvatarUploadFile(file *multipart.FileHeader, maxSize int64, allowedExts []string) ([]byte, error) {
	if file.Filename == "" {
		return nil, component.NewMessageError(component.MessageUploadFilenameRequired, "文件名不能为空", nil)
	}
	if file.Size > maxSize {
		return nil, component.NewMessageError(
			component.MessageUploadFileTooLarge,
			fmt.Sprintf("文件大小超过限制，最大允许%dKB", maxSize/1024),
			component.MessageParams{"maxSizeKb": maxSize / 1024},
		)
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if len(allowedExts) > 0 {
		if !isAllowedExtension(ext, allowedExts) {
			extensions := strings.Join(allowedExts, ", ")
			return nil, component.NewMessageError(
				component.MessageUploadUnsupportedExt,
				"不支持的文件格式，允许的格式为: "+extensions,
				component.MessageParams{"extensions": extensions},
			)
		}
	} else if _, err := filedata.CheckImageType(file.Filename); err != nil {
		return nil, component.NewMessageError(component.MessageUploadUnsupportedImage, "不支持的图片格式，仅支持 JPG、PNG、GIF、WebP、BMP 格式", nil)
	}

	src, err := file.Open()
	if err != nil {
		return nil, component.NewMessageError(component.MessageUploadOpenFailed, "打开文件失败", nil)
	}
	defer func() { _ = src.Close() }()

	header := make([]byte, 512)
	n, _ := io.ReadFull(src, header)
	if n > 0 && !isValidImageContent(header[:n]) {
		return nil, component.NewMessageError(component.MessageUploadInvalidImage, "文件内容不是有效的图片格式", nil)
	}

	remainingData, err := io.ReadAll(io.LimitReader(src, maxSize-int64(n)+1))
	if err != nil {
		return nil, component.NewMessageError(component.MessageUploadReadFailed, "读取文件失败", nil)
	}
	fileData := append(bytes.Clone(header[:n]), remainingData...)
	if int64(len(fileData)) > maxSize {
		return nil, component.NewMessageError(
			component.MessageUploadFileTooLarge,
			fmt.Sprintf("文件大小超过限制，最大允许%dKB", maxSize/1024),
			component.MessageParams{"maxSizeKb": maxSize / 1024},
		)
	}
	return fileData, nil
}

// ChangePasswordReq is the password change request.
type ChangePasswordReq struct {
	OldPassword string `json:"oldPassword" validate:"required"`
	NewPassword string `json:"newPassword" validate:"required"`
}

// ChangePassword updates the current user's password.
func ChangePassword(req component.BetterRequest[ChangePasswordReq]) component.Response {
	userEntity, err := req.GetUser()
	if err != nil {
		return component.FailResponseCode(component.MessageUserFetchFailed, nil)
	}
	// 机器人（Agent）账号没有可用密码，也不允许通过改密接口变更。
	if userEntity.IsBot() {
		return component.FailResponseCode(component.MessageAuthOldPasswordInvalid, nil)
	}
	if err = component.ValidatePassword(req.Params.NewPassword, 6); err != nil {
		return component.FailResponseError(err)
	}
	err = algorithm.VerifyEncryptPassword(userEntity.Password, req.Params.OldPassword)
	if err != nil {
		return component.FailResponseCode(component.MessageAuthOldPasswordInvalid, nil)
	}

	userEntity.SetPassword(req.Params.NewPassword)
	err = userservice.SaveUser(&userEntity)
	if err != nil {
		return component.FailResponseCode(component.MessageAuthPasswordUpdateFailed, nil)
	}

	return component.SuccessResponseCode("密码修改成功", component.MessageAuthPasswordUpdateSuccess, nil)
}

// ForgotPasswordReq is the password reset email request.
type ForgotPasswordReq struct {
	Email       string `json:"email" validate:"required,email"`
	CaptchaId   string `json:"captchaId,omitempty"`
	CaptchaCode string `json:"captchaCode,omitempty"`
	Website     string `json:"website,omitempty"` // 蜜罐字段，正常用户不可见
}

// ForgotPassword 忘记密码 - 发送重置邮件
func ForgotPassword(req component.BetterRequest[ForgotPasswordReq]) component.Response {
	// 蜜罐字段：填了即机器，静默拒绝（返回成功但不发邮件）。
	if strings.TrimSpace(req.Params.Website) != "" {
		slog.Warn("honeypot_hit", "action", "forgot-password", "ip", clientIPOf(req.GinContext), "userId", uint64(0))
		return component.SuccessResponseCode("操作成功：如果该邮箱已注册，您将收到密码重置邮件", component.MessageAuthResetMailQueued, nil)
	}

	securityConfig := hotdataserve.GetSecuritySettingsConfigCache()
	if ok, needCaptcha := checkCaptchaForRequest(req.GinContext, req.Params.CaptchaId, req.Params.CaptchaCode, securityConfig.CaptchaRequired, minSubmitSecondsFor(), "forgot-password"); !ok {
		if needCaptcha {
			return component.FailResponseCode(component.MessageCaptchaRequired, component.MessageParams{"action": "forgot-password"})
		}
		return component.FailResponseCode(component.MessageAuthCaptchaInvalid, nil)
	}

	userEntity, err := users.GetByEmail(req.Params.Email)
	if err != nil || userEntity.IsBot() {
		// 为了安全考虑，即使邮箱不存在（或命中机器人账号）也返回统一成功消息；
		// 先执行等量 dummy 工作（HMAC 签名 + 同步 noop 入队）抹平响应时间差。
		return forgotPasswordSilentSuccess(req.Params.Email)
	}

	// 冷静期：邮箱变更后 24 小时内，新邮箱不能用于密码重置。
	// 静默返回成功（与邮箱未注册完全一致，无枚举差异），但绝不入队重置邮件，
	// 防止会话 token 被接管后立刻用新邮箱重置密码。同样先执行等量 dummy 工作对齐耗时。
	if userEntity.EmailChangedAt != nil && time.Since(*userEntity.EmailChangedAt) < emailChangeCooldown {
		return forgotPasswordSilentSuccess(req.Params.Email)
	}
	token, err := tokenservice.GeneratePasswordResetToken(userEntity.Id, userEntity.Email, userEntity.TokenVersion)
	if err != nil {
		return component.FailResponseCode(component.MessageAuthResetTokenCreateFailed, nil)
	}

	err = mailservice.AddToQueue(mailservice.EmailTask{
		To:       userEntity.Email,
		Username: userEntity.Username,
		Token:    token,
		Type:     "reset_password",
		Locale:   userEntity.Locale,
	})
	if err != nil {
		slog.Error("添加密码重置邮件任务到队列失败", "error", err)
		return component.FailResponseCode(component.MessageAuthResetMailSendFailed, nil)
	}

	return component.SuccessResponseCode("操作成功：如果该邮箱已注册，您将收到密码重置邮件", component.MessageAuthResetMailQueued, nil)
}

// dummyTimingUsername 是 forgot-password 等时化 noop 任务中与真实用户名同量的
// 固定占位值（用户名上限 32 字符），使 dummy 任务的序列化与 DB 写入负载与
// 已注册路径的 reset_password 任务一致（review #129 P2）。
const dummyTimingUsername = "timing-dummy-username-0123456789"

// forgotPasswordSilentSuccess 在"未知邮箱/机器人账号/邮箱变更冷静期"路径返回与
// 已注册路径一致的响应：先执行等量 dummy 工作（一次 HMAC 令牌签名 + 一次同步
// task_queue 写入 email.noop 任务，由邮件 worker 静默消费、不发邮件），抹平响应
// 时间差，避免通过响应时间枚举邮箱注册状态（CWE-208，与 #109/#119 的等时化
// 模式一致）。noop 任务携带与真实 reset_password 任务同量的负载（复用刚生成的
// dummy token 填充 Token、Username 用固定占位），确保序列化与 DB 写入负载一致。
// dummy 工作失败时返回与已注册路径相同的失败码（令牌生成失败 →
// auth.passwordReset.tokenCreateFailed、队列写入失败 → auth.passwordReset.mailSendFailed），
// 使两条路径在任何状态下响应逐字节一致，不残留系统级故障窗口内的枚举信号。
func forgotPasswordSilentSuccess(email string) component.Response {
	dummyToken, err := tokenservice.GeneratePasswordResetToken(0, email, 0)
	if err != nil {
		slog.Error("forgot-password 等时化令牌生成失败", "email", email, "error", err)
		return component.FailResponseCode(component.MessageAuthResetTokenCreateFailed, nil)
	}
	if err := mailservice.AddToQueue(mailservice.EmailTask{
		To:       email,
		Username: dummyTimingUsername,
		Token:    dummyToken,
		Type:     "noop",
	}); err != nil {
		slog.Error("forgot-password 等时化队列写入失败", "email", email, "error", err)
		return component.FailResponseCode(component.MessageAuthResetMailSendFailed, nil)
	}
	return component.SuccessResponseCode("操作成功：如果该邮箱已注册，您将收到密码重置邮件", component.MessageAuthResetMailQueued, nil)
}

// ResetPasswordReq is the password reset confirmation request.
type ResetPasswordReq struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"newPassword" validate:"required"`
}

// ResetPassword 重置密码
func ResetPassword(req component.BetterRequest[ResetPasswordReq]) component.Response {
	claims, err := tokenservice.ParsePasswordResetToken(req.Params.Token)
	if err != nil {
		return component.FailResponseCode(component.MessageAuthResetTokenInvalid, nil)
	}

	userEntity, err := users.Get(claims.UserId)
	if err != nil {
		return component.FailResponseCode(component.MessageUserNotFound, nil)
	}
	// 机器人（Agent）账号不参与密码重置流程。
	if userEntity.IsBot() {
		return component.FailResponseCode(component.MessageAuthResetTokenInvalid, nil)
	}

	if userEntity.Email != claims.Email {
		return component.FailResponseCode(component.MessageAuthResetTokenInvalid, nil)
	}

	// 重置令牌绑定签发时的 token_version（issue #106）：密码变更 / 撤销会自增
	// token_version，因此旧的重置链接在账户被重置或恢复后立即失效，无法重放。
	// 仅校验令牌签名与 email 不足以防止伪造链路接管账户。
	if userEntity.TokenVersion != claims.TokenVersion {
		return component.FailResponseCode(component.MessageAuthResetTokenInvalid, nil)
	}

	if userEntity.IsFrozen == users.StatusFrozen {
		return component.FailResponseCode(component.MessagePermissionUserFrozen, component.MessageParams{
			"action":     "写入",
			"actionCode": string(component.PermissionActionWrite),
		})
	}

	if err = component.ValidatePassword(req.Params.NewPassword, 6); err != nil {
		return component.FailResponseError(err)
	}

	userEntity.SetPassword(req.Params.NewPassword)
	err = userservice.SaveUser(&userEntity)
	if err != nil {
		return component.FailResponseCode(component.MessageAuthResetFailed, nil)
	}

	return component.SuccessResponseCode("密码重置成功", component.MessageAuthResetSuccess, nil)
}

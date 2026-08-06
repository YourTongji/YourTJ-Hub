package vo

// RegReq is the user registration request payload.
type RegReq struct {
	Email          string `json:"email" validate:"required,email"`
	Username       string `json:"userName"  validate:"required"`
	Password       string `json:"passWord"  validate:"required"`
	Locale         string `json:"locale,omitempty"`
	InvitationCode string `json:"invitationCode,omitempty"`
	CaptchaId      string `json:"captchaId,omitempty"`
	CaptchaCode    string `json:"captchaCode,omitempty"`
	Website        string `json:"website,omitempty"` // 蜜罐字段，正常用户不可见
}

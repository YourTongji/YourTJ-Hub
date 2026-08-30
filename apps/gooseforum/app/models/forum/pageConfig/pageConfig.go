package pageConfig

import (
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/markdown2html"
)

const tableName = "page_config"
const pid = "id"
const filedPageType = "page_type"

type Entity struct {
	Id        uint64    `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`                              // 主键
	PageType  string    `gorm:"column:page_type;uniqueIndex;type:varchar(128);not null;default:'';" json:"pageType"` // 页面类型
	Config    string    `gorm:"column:config;type:text;" json:"content"`                                             //
	CreatedAt time.Time `gorm:"column:created_at;index;autoCreateTime;<-:create;" json:"createdAt"`                  //
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
}

// func (itself *Entity) BeforeSave(tx *gorm.DB) (err error) {}
// func (itself *Entity) BeforeCreate(tx *gorm.DB) (err error) {}
// func (itself *Entity) AfterCreate(tx *gorm.DB) (err error) {}
// func (itself *Entity) BeforeUpdate(tx *gorm.DB) (err error) {}
// func (itself *Entity) AfterUpdate(tx *gorm.DB) (err error) {}
// func (itself *Entity) AfterSave(tx *gorm.DB) (err error) {}
// func (itself *Entity) BeforeDelete(tx *gorm.DB) (err error) {}
// func (itself *Entity) AfterDelete(tx *gorm.DB) (err error) {}
// func (itself *Entity) AfterFind(tx *gorm.DB) (err error) {}

func (itself *Entity) TableName() string {
	return tableName
}

const (
	FriendShipLinks     = `friendShipLinks`
	SponsorsPage        = `sponsors`
	SiteSettings        = `siteSettings`
	EmailSettings       = `emailSetting`
	Announcement        = `announcement`
	SecuritySettings    = `securitySettings`
	StorageSettingsPage = `storageSettings`
	TermsOfService      = `termsOfService`
	PrivacyPolicy       = `privacyPolicy`
	PostingSettings     = `postingSettings`
	HttpNotify          = `httpNotify`
	SiteTheme           = `siteTheme`
	SiteChrome          = `siteChrome`
	RateLimitSettings   = `rateLimitSettings`
	MCPSettings         = `mcpSettings`
	AiSummarySettings   = `aiSummarySettings`
	OneSystemSettings   = `onesystemSettings`
	WikiSyncSettings    = `wikiSyncSettings`
	ScheduleSettings    = `scheduleSettings`
	Version             = `version`
	Migration           = `migration`
)

type LinkItem struct {
	Name    string `json:"name"`
	Desc    string `json:"desc"`
	Url     string `json:"url"`
	LogoUrl string `json:"logoUrl"`
	Status  int    `json:"status"`
}

type FriendLinksGroup struct {
	Name  string     `json:"name,omitempty"`
	Emoji string     `json:"emoji,omitempty"`
	Color string     `json:"color,omitempty"`
	Links []LinkItem `json:"links"`
}

type FooterItem struct {
	Name string `json:"name"`
	Url  string `json:"url"`
}

type PItem struct {
	Content string `json:"content"`
}

type SponsorItem struct {
	Link      string `json:"link"`
	Message   string `json:"message"`
	AvatarUrl string `json:"avatarUrl"`
	Name      string `json:"name"`
}

type Sponsors struct {
	Level0 []SponsorItem `json:"level0"`
	Level1 []SponsorItem `json:"level1"`
	Level2 []SponsorItem `json:"level2"`
	Level3 []SponsorItem `json:"level3"`
}

type SponsorsConfig struct {
	Sponsors Sponsors          `json:"sponsors"`
	Content  SponsorsPageIntro `json:"content"`
	Contact  SponsorsContact   `json:"contact"`
	Rules    []SponsorsRule    `json:"rules"`
}

type SponsorsPageIntro struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type SponsorsContact struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ButtonText  string `json:"buttonText"`
	ButtonLink  string `json:"buttonLink"`
}

type SponsorsRule struct {
	Content string `json:"content"`
}

// SiteSettingsConfig 站点设置配置
type SiteSettingsConfig struct {
	// 站点基本信息
	SiteName        string `json:"siteName"`
	SiteLogo        string `json:"siteLogo"`
	SiteDescription string `json:"siteDescription"`
	SiteKeywords    string `json:"siteKeywords"`
	SiteUrl         string `json:"siteUrl"`
	SiteEmail       string `json:"siteEmail"`
	ExternalLinks   string `json:"externalLinks"`
}

type FooterInfo struct {
	Primary []PItem      `json:"primary"`
	List    []FooterItem `json:"list"`
}

type SiteChromeConfig struct {
	Header        []ChromeItem  `json:"header"`
	MainMenu      []ChromeItem  `json:"mainMenu"`
	Resources     []ChromeItem  `json:"resources"`
	SidebarGroups []ChromeGroup `json:"sidebarGroups"`
	FooterInfo    FooterInfo    `json:"footerInfo"`
	BrandType     string        `json:"brandType"`
	BrandText     string        `json:"brandText"`
	BrandImage    string        `json:"brandImage"`
}

type ChromeItem struct {
	ID        string `json:"id"`
	Enabled   bool   `json:"enabled"`
	Type      string `json:"type"`
	Label     string `json:"label"`
	I18nLabel string `json:"i18nLabel"`
	URL       string `json:"url"`
}

type ChromeGroup struct {
	ID        string       `json:"id"`
	Title     string       `json:"title"`
	I18nLabel string       `json:"i18nLabel"`
	Items     []ChromeItem `json:"items"`
}

// MailSettingsConfig 邮件设置配置（S2，issue #324）：smtpPassword 只落库密文
// （securestore AES-256-GCM），运行时由 hotdataserve 解密为明文置于 json:"-"
// 字段——明文/密文绝不随 JSON 序列化导出（管理端 GET 仅回显是否已配置）。
// 持久化走 MailSettingsStorage（密文带 json 标签）。
type MailSettingsConfig struct {
	// SMTP服务器设置
	EnableMail   bool   `json:"enableMail"`
	SmtpHost     string `json:"smtpHost"`
	SmtpPort     int    `json:"smtpPort"`
	UseSSL       bool   `json:"useSSL"`
	SmtpUsername string `json:"smtpUsername"`
	SmtpPassword string `json:"-"` // 运行时明文（服务内存）；密文见 MailSettingsStorage
	FromName     string `json:"fromName"`
	FromEmail    string `json:"fromEmail"`
}

// MailSettingsStorage 邮件设置的落库 JSON 形状：与对外 MailSettingsConfig 分离，
// 密文只在持久化序列化时出现，不进入 API 响应/缓存结构。
// SmtpPassword 为 v25 迁移前的存量明文（json:"smtpPassword"，兼容读取，迁移后清空）；
// SmtpPasswordEncrypted 为 securestore 密文（优先于明文）。
type MailSettingsStorage struct {
	EnableMail            bool   `json:"enableMail"`
	SmtpHost              string `json:"smtpHost"`
	SmtpPort              int    `json:"smtpPort"`
	UseSSL                bool   `json:"useSSL"`
	SmtpUsername          string `json:"smtpUsername"`
	SmtpPassword          string `json:"smtpPassword,omitempty"`          // 迁移前存量明文（兼容读取）
	SmtpPasswordEncrypted string `json:"smtpPasswordEncrypted,omitempty"` // 密文（AES-256-GCM）
	FromName              string `json:"fromName"`
	FromEmail             string `json:"fromEmail"`
}

// ToConfig 将落库形状转为领域结构（密码为密文原样拷贝，调用方解密）。
func (s MailSettingsStorage) ToConfig() MailSettingsConfig {
	password := s.SmtpPasswordEncrypted
	if password == "" {
		password = s.SmtpPassword
	}
	return MailSettingsConfig{
		EnableMail:   s.EnableMail,
		SmtpHost:     s.SmtpHost,
		SmtpPort:     s.SmtpPort,
		UseSSL:       s.UseSSL,
		SmtpUsername: s.SmtpUsername,
		SmtpPassword: password,
		FromName:     s.FromName,
		FromEmail:    s.FromEmail,
	}
}

// MailSettingsInput 管理端保存/测试请求的绑定形状：smtpPassword 为明文输入
// （仅请求瞬间存在，绝不落库）。空密码 = 保持已存值（issue #324 S2）。
type MailSettingsInput struct {
	EnableMail   bool   `json:"enableMail"`
	SmtpHost     string `json:"smtpHost"`
	SmtpPort     int    `json:"smtpPort"`
	UseSSL       bool   `json:"useSSL"`
	SmtpUsername string `json:"smtpUsername"`
	SmtpPassword string `json:"smtpPassword"`
	FromName     string `json:"fromName"`
	FromEmail    string `json:"fromEmail"`
}

// ToConfig 将输入形状转为领域结构（含明文密码，供 mailservice 发送使用）。
func (i MailSettingsInput) ToConfig() MailSettingsConfig {
	return MailSettingsConfig(i)
}

// MailSettingsView 管理端 GET 回显形状：不含密码，仅回显是否已配置（issue #324 S2）。
type MailSettingsView struct {
	EnableMail             bool   `json:"enableMail"`
	SmtpHost               string `json:"smtpHost"`
	SmtpPort               int    `json:"smtpPort"`
	UseSSL                 bool   `json:"useSSL"`
	SmtpUsername           string `json:"smtpUsername"`
	SmtpPasswordConfigured bool   `json:"smtpPasswordConfigured"`
	FromName               string `json:"fromName"`
	FromEmail              string `json:"fromEmail"`
}

// ToView 由落库形状生成回显视图。
func (s MailSettingsStorage) ToView() MailSettingsView {
	return MailSettingsView{
		EnableMail:             s.EnableMail,
		SmtpHost:               s.SmtpHost,
		SmtpPort:               s.SmtpPort,
		UseSSL:                 s.UseSSL,
		SmtpUsername:           s.SmtpUsername,
		SmtpPasswordConfigured: strings.TrimSpace(s.SmtpPassword) != "" || strings.TrimSpace(s.SmtpPasswordEncrypted) != "",
		FromName:               s.FromName,
		FromEmail:              s.FromEmail,
	}
}

// ToView 由领域结构生成回显视图（默认配置路径：密码恒为空）。
func (c MailSettingsConfig) ToView() MailSettingsView {
	return MailSettingsView{
		EnableMail:             c.EnableMail,
		SmtpHost:               c.SmtpHost,
		SmtpPort:               c.SmtpPort,
		UseSSL:                 c.UseSSL,
		SmtpUsername:           c.SmtpUsername,
		SmtpPasswordConfigured: strings.TrimSpace(c.SmtpPassword) != "",
		FromName:               c.FromName,
		FromEmail:              c.FromEmail,
	}
}

// AnnouncementItem 单则公告（多则公告模式）
type AnnouncementItem struct {
	ID      string `json:"id"`      // 稳定标识，用于前端轮播 key
	Title   string `json:"title"`   // 公告标题（可空，仅展示时优先）
	Content string `json:"content"` // Markdown 内容
	Enabled bool   `json:"enabled"` // 是否启用
}

// AnnouncementConfig 公告设置配置
type AnnouncementConfig struct {
	Enabled     bool               `json:"enabled"`               // 是否启用公告
	Content     string             `json:"content"`               // 兼容：单则模式公告内容
	PublishedAt string             `json:"publishedAt,omitempty"` // 公告生效时间
	Items       []AnnouncementItem `json:"items,omitempty"`       // 多则公告列表（非空时优先于单则 Content）
	HtmlContent string             `json:"-"`                     // 预渲染后的 HTML，仅服务端使用
}

func (itself *AnnouncementConfig) PrepareHTML() {
	if itself == nil || itself.HtmlContent != "" || itself.Content == "" {
		return
	}
	itself.HtmlContent = markdown2html.MarkdownToHTML(itself.Content)
}

func (itself AnnouncementConfig) GetHtmlContent() string {
	if itself.HtmlContent != "" || itself.Content == "" {
		return itself.HtmlContent
	}
	return markdown2html.MarkdownToHTML(itself.Content)
}

// GetActiveItems 返回启用中的多则公告（含预渲染 HTML），
// 供首页轮播与多则展示使用；未配置多则时返回空。
func (itself AnnouncementConfig) GetActiveItems() []AnnouncementItem {
	items := make([]AnnouncementItem, 0, len(itself.Items))
	for _, item := range itself.Items {
		if !item.Enabled || strings.TrimSpace(item.Content) == "" {
			continue
		}
		items = append(items, AnnouncementItem{
			ID:      item.ID,
			Title:   item.Title,
			Content: item.Content,
			Enabled: true,
		})
	}
	return items
}

type SecurityAndRegistration struct {
	EnableSignup            bool     `json:"enableSignup"`
	EnableEmailVerification bool     `json:"enableEmailVerification"`
	AllowedDomains          []string `json:"allowedDomains"`
	ReservedUsernames       []string `json:"reservedUsernames"` // 保留用户名：注册/改名拒绝
	BannedUsernames         []string `json:"bannedUsernames"`   // 禁用用户名：注册/改名拒绝，存量账号自动冻结
	SensitiveWords          []string `json:"sensitiveWords"`    // 敏感词：命中后按 SensitiveAction 处理
	SensitiveAction         string   `json:"sensitiveAction"`   // block=直接拦截 review=转人工审核
	CaptchaRequired         bool     `json:"captchaRequired"`   // 注册/登录/找回密码是否要求验证码
}

// StorageSettings 存储设置配置（本地 SQLite BLOB 或 S3 兼容对象存储）。
// S3 凭据（issue #324 S3）：accessKey/secretKey 只落库密文（securestore
// AES-256-GCM），运行时由 hotdataserve 解密为明文置于 json:"-" 字段——明文/
// 密文绝不随 JSON 序列化导出（管理端 GET 仅回显是否已配置）。
// 持久化走 StorageSettingsStorage（密文带 json 标签）。
type StorageSettings struct {
	Provider        string `json:"provider"`        // local | s3
	Endpoint        string `json:"endpoint"`        // S3 兼容 endpoint，如 https://cos.ap-shanghai.myqcloud.com
	Bucket          string `json:"bucket"`          // 存储桶
	Region          string `json:"region"`          // 区域（COS/OSS 必须，R2 可忽略）
	BucketLookup    string `json:"bucketLookup"`    // auto | dns | path（COS 需 dns，MinIO/R2 可 auto/path）
	Secure          bool   `json:"secure"`          // 是否使用 HTTPS
	AccessKey       string `json:"-"`               // 运行时明文（服务内存）；密文见 StorageSettingsStorage
	SecretKey       string `json:"-"`               // 运行时明文（服务内存）；密文见 StorageSettingsStorage
	PublicUrlPrefix string `json:"publicUrlPrefix"` // 可选公开访问前缀（CDN），留空则走 /file/img 代理
}

// StorageSettingsStorage 存储设置的落库 JSON 形状：与对外 StorageSettings 分离，
// 密文只在持久化序列化时出现，不进入 API 响应/缓存结构。
// AccessKey/SecretKey 为 v25 迁移前的存量明文（兼容读取，迁移后清空）；
// AccessKeyEncrypted/SecretKeyEncrypted 为 securestore 密文（优先于明文）。
type StorageSettingsStorage struct {
	Provider           string `json:"provider"`
	Endpoint           string `json:"endpoint"`
	Bucket             string `json:"bucket"`
	Region             string `json:"region"`
	BucketLookup       string `json:"bucketLookup"`
	Secure             bool   `json:"secure"`
	AccessKey          string `json:"accessKey,omitempty"`          // 迁移前存量明文（兼容读取）
	SecretKey          string `json:"secretKey,omitempty"`          // 迁移前存量明文（兼容读取）
	AccessKeyEncrypted string `json:"accessKeyEncrypted,omitempty"` // 密文（AES-256-GCM）
	SecretKeyEncrypted string `json:"secretKeyEncrypted,omitempty"` // 密文（AES-256-GCM）
	PublicUrlPrefix    string `json:"publicUrlPrefix"`
}

// ToConfig 将落库形状转为领域结构（凭据为密文或存量明文原样拷贝，调用方解密/识别）。
func (s StorageSettingsStorage) ToConfig() StorageSettings {
	accessKey := s.AccessKeyEncrypted
	if accessKey == "" {
		accessKey = s.AccessKey
	}
	secretKey := s.SecretKeyEncrypted
	if secretKey == "" {
		secretKey = s.SecretKey
	}
	return StorageSettings{
		Provider:        s.Provider,
		Endpoint:        s.Endpoint,
		Bucket:          s.Bucket,
		Region:          s.Region,
		BucketLookup:    s.BucketLookup,
		Secure:          s.Secure,
		AccessKey:       accessKey,
		SecretKey:       secretKey,
		PublicUrlPrefix: s.PublicUrlPrefix,
	}
}

// StorageSettingsView 管理端 GET 回显形状：不含凭据，仅回显是否已配置（issue #324 S3）。
type StorageSettingsView struct {
	Provider            string `json:"provider"`
	Endpoint            string `json:"endpoint"`
	Bucket              string `json:"bucket"`
	Region              string `json:"region"`
	BucketLookup        string `json:"bucketLookup"`
	Secure              bool   `json:"secure"`
	AccessKeyConfigured bool   `json:"accessKeyConfigured"`
	SecretKeyConfigured bool   `json:"secretKeyConfigured"`
	PublicUrlPrefix     string `json:"publicUrlPrefix"`
}

// ToView 由落库形状生成回显视图。
func (s StorageSettingsStorage) ToView() StorageSettingsView {
	return StorageSettingsView{
		Provider:            s.Provider,
		Endpoint:            s.Endpoint,
		Bucket:              s.Bucket,
		Region:              s.Region,
		BucketLookup:        s.BucketLookup,
		Secure:              s.Secure,
		AccessKeyConfigured: strings.TrimSpace(s.AccessKey) != "" || strings.TrimSpace(s.AccessKeyEncrypted) != "",
		SecretKeyConfigured: strings.TrimSpace(s.SecretKey) != "" || strings.TrimSpace(s.SecretKeyEncrypted) != "",
		PublicUrlPrefix:     s.PublicUrlPrefix,
	}
}

// ToView 由领域结构生成回显视图（默认配置路径：凭据恒为空）。
func (c StorageSettings) ToView() StorageSettingsView {
	return StorageSettingsView{
		Provider:            c.Provider,
		Endpoint:            c.Endpoint,
		Bucket:              c.Bucket,
		Region:              c.Region,
		BucketLookup:        c.BucketLookup,
		Secure:              c.Secure,
		AccessKeyConfigured: strings.TrimSpace(c.AccessKey) != "",
		SecretKeyConfigured: strings.TrimSpace(c.SecretKey) != "",
		PublicUrlPrefix:     c.PublicUrlPrefix,
	}
}

// StorageSettingsInput 管理端保存/测试请求的绑定形状：accessKey/secretKey 为明文
// 输入（仅请求瞬间存在，绝不落库）。空值 = 保持已存密文（issue #324 S3）。
type StorageSettingsInput struct {
	Provider        string `json:"provider"`
	Endpoint        string `json:"endpoint"`
	Bucket          string `json:"bucket"`
	Region          string `json:"region"`
	BucketLookup    string `json:"bucketLookup"`
	Secure          bool   `json:"secure"`
	AccessKey       string `json:"accessKey"`
	SecretKey       string `json:"secretKey"`
	PublicUrlPrefix string `json:"publicUrlPrefix"`
}

// ToConfig 将输入形状转为领域结构（含明文凭据，供测试连接等服务使用）。
func (i StorageSettingsInput) ToConfig() StorageSettings {
	return StorageSettings(i)
}

// TermsOfServiceConfig 服务条款配置
type TermsOfServiceConfig struct {
	Enabled     bool   `json:"enabled"` // 是否启用服务条款
	Content     string `json:"content"` // 条款内容（markdown）
	HtmlContent string `json:"-"`       // 预渲染后的 HTML，仅服务端使用
}

func (itself *TermsOfServiceConfig) PrepareHTML() {
	if itself == nil || itself.HtmlContent != "" || itself.Content == "" {
		return
	}
	itself.HtmlContent = markdown2html.MarkdownToHTML(itself.Content)
}

func (itself TermsOfServiceConfig) GetHtmlContent() string {
	if itself.HtmlContent != "" || itself.Content == "" {
		return itself.HtmlContent
	}
	return markdown2html.MarkdownToHTML(itself.Content)
}

// PrivacyPolicyConfig 隐私政策配置（结构与服务条款一致）。
type PrivacyPolicyConfig = TermsOfServiceConfig

type PostingContent struct {
	TextControl struct {
		MinPostLength              int `json:"minPostLength"`
		MaxPostLength              int `json:"maxPostLength"`
		MinTitleLength             int `json:"minTitleLength"`
		MaxTitleLength             int `json:"maxTitleLength"`
		NewUserPostCooldownMinutes int `json:"newUserPostCooldownMinutes"`
	} `json:"textControl"`
	UploadControl struct {
		AllowAttachments             bool     `json:"allowAttachments"`
		AuthorizedExtensions         []string `json:"authorizedExtensions"`
		MaxAttachmentSizeKb          int      `json:"maxAttachmentSizeKb"`
		MaxDailyUploadsPerUser       int      `json:"maxDailyUploadsPerUser"`
		NewUserUploadCooldownMinutes int      `json:"newUserUploadCooldownMinutes"`
	} `json:"uploadControl"`
	LLMS LLMSConfig `json:"llms"`
}

type LLMSConfig struct {
	Enabled  bool `json:"enabled"`
	FullText bool `json:"fullText"`
	Files    bool `json:"files"`
}

// RateLimitRule 单个动作的限流配额。Action 取值见 rateLimitActions。
type RateLimitRule struct {
	Action        string `json:"action"`
	WindowSeconds int    `json:"windowSeconds"`
	LimitPerIp    int    `json:"limitPerIp"`
	LimitPerUser  int    `json:"limitPerUser"`
}

// RateLimitConfig 滥用防护配置，全部数值可在管理面板热修改。
type RateLimitConfig struct {
	Enabled                  bool            `json:"enabled"`                  // 总开关
	SkipAdmin                bool            `json:"skipAdmin"`                // 管理员豁免
	Actions                  []RateLimitRule `json:"actions"`                  // 各动作配额
	NewUserCaptchaAfterPosts int             `json:"newUserCaptchaAfterPosts"` // 新用户窗口内连发 N 帖后要求验证码，0 关闭
	NewUserCaptchaDays       int             `json:"newUserCaptchaDays"`       // 新用户判定窗口（注册 N 天内），0 表示所有用户
	MinSubmitSeconds         int             `json:"minSubmitSeconds"`         // 验证码提交耗时下限（秒），低于判定为机器
}

type HttpNotifyConfig struct {
	Enabled   bool                 `json:"enabled"`
	Endpoints []HttpNotifyEndpoint `json:"endpoints"`
}

// HttpNotifyEndpoint 通知端点（S1，issue #324）：Secret 只落库密文（securestore
// AES-256-GCM），运行时由 hotdataserve 解密为明文置于 json:"-" 字段——明文/
// 密文绝不随 JSON 序列化导出（管理端 GET 仅回显是否已配置）。
// 持久化走 HttpNotifyStorageEndpoint（密文带 json 标签）。
type HttpNotifyEndpoint struct {
	Id                 string   `json:"id"`
	Name               string   `json:"name"`
	Enabled            bool     `json:"enabled"`
	URL                string   `json:"url"`
	Secret             string   `json:"-"` // 运行时明文（服务内存）；密文见 HttpNotifyStorageEndpoint
	Events             []string `json:"events"`
	TimeoutSeconds     int      `json:"timeoutSeconds"`
	FailureCount       int      `json:"failureCount"`
	LastError          string   `json:"lastError"`
	AbnormalTerminated bool     `json:"abnormalTerminated"`
}

// HttpNotifyStorageEndpoint 通知端点的落库 JSON 形状：与对外 HttpNotifyEndpoint
// 分离，密文只在持久化序列化时出现，不进入 API 响应/缓存结构。
// Secret 为 v25 迁移前的存量明文（json:"secret"，兼容读取，迁移后清空）；
// SecretEncrypted 为 securestore 密文（优先于明文）。
type HttpNotifyStorageEndpoint struct {
	Id                 string   `json:"id"`
	Name               string   `json:"name"`
	Enabled            bool     `json:"enabled"`
	URL                string   `json:"url"`
	Secret             string   `json:"secret,omitempty"`          // 迁移前存量明文（兼容读取）
	SecretEncrypted    string   `json:"secretEncrypted,omitempty"` // 密文（AES-256-GCM）
	Events             []string `json:"events"`
	TimeoutSeconds     int      `json:"timeoutSeconds"`
	FailureCount       int      `json:"failureCount"`
	LastError          string   `json:"lastError"`
	AbnormalTerminated bool     `json:"abnormalTerminated"`
}

// ToConfig 将落库形状转为领域结构（密钥为密文或存量明文原样拷贝，调用方解密/识别）。
func (s HttpNotifyStorageEndpoint) ToConfig() HttpNotifyEndpoint {
	secret := s.SecretEncrypted
	if secret == "" {
		secret = s.Secret
	}
	return HttpNotifyEndpoint{
		Id:                 s.Id,
		Name:               s.Name,
		Enabled:            s.Enabled,
		URL:                s.URL,
		Secret:             secret,
		Events:             s.Events,
		TimeoutSeconds:     s.TimeoutSeconds,
		FailureCount:       s.FailureCount,
		LastError:          s.LastError,
		AbnormalTerminated: s.AbnormalTerminated,
	}
}

// HttpNotifyStorageConfig 通知设置的落库 JSON 形状。
type HttpNotifyStorageConfig struct {
	Enabled   bool                        `json:"enabled"`
	Endpoints []HttpNotifyStorageEndpoint `json:"endpoints"`
}

// ToConfig 将落库形状转为领域结构。
func (s HttpNotifyStorageConfig) ToConfig() HttpNotifyConfig {
	endpoints := make([]HttpNotifyEndpoint, 0, len(s.Endpoints))
	for _, e := range s.Endpoints {
		endpoints = append(endpoints, e.ToConfig())
	}
	return HttpNotifyConfig{Enabled: s.Enabled, Endpoints: endpoints}
}

// HttpNotifyEndpointView 管理端 GET 回显形状：不含密钥，仅回显是否已配置（issue #324 S1）。
type HttpNotifyEndpointView struct {
	Id                 string   `json:"id"`
	Name               string   `json:"name"`
	Enabled            bool     `json:"enabled"`
	URL                string   `json:"url"`
	SecretConfigured   bool     `json:"secretConfigured"`
	Events             []string `json:"events"`
	TimeoutSeconds     int      `json:"timeoutSeconds"`
	FailureCount       int      `json:"failureCount"`
	LastError          string   `json:"lastError"`
	AbnormalTerminated bool     `json:"abnormalTerminated"`
}

// HttpNotifyView 管理端 GET 回显形状：不含密钥（issue #324 S1）。
type HttpNotifyView struct {
	Enabled   bool                     `json:"enabled"`
	Endpoints []HttpNotifyEndpointView `json:"endpoints"`
}

// ToView 由落库形状生成回显视图。
func (s HttpNotifyStorageConfig) ToView() HttpNotifyView {
	endpoints := make([]HttpNotifyEndpointView, 0, len(s.Endpoints))
	for _, e := range s.Endpoints {
		endpoints = append(endpoints, HttpNotifyEndpointView{
			Id:                 e.Id,
			Name:               e.Name,
			Enabled:            e.Enabled,
			URL:                e.URL,
			SecretConfigured:   strings.TrimSpace(e.SecretEncrypted) != "" || strings.TrimSpace(e.Secret) != "",
			Events:             e.Events,
			TimeoutSeconds:     e.TimeoutSeconds,
			FailureCount:       e.FailureCount,
			LastError:          e.LastError,
			AbnormalTerminated: e.AbnormalTerminated,
		})
	}
	return HttpNotifyView{Enabled: s.Enabled, Endpoints: endpoints}
}

// ToView 由领域结构生成回显视图（默认配置路径：密钥恒为空）。
func (c HttpNotifyConfig) ToView() HttpNotifyView {
	endpoints := make([]HttpNotifyEndpointView, 0, len(c.Endpoints))
	for _, e := range c.Endpoints {
		endpoints = append(endpoints, HttpNotifyEndpointView{
			Id:                 e.Id,
			Name:               e.Name,
			Enabled:            e.Enabled,
			URL:                e.URL,
			SecretConfigured:   strings.TrimSpace(e.Secret) != "",
			Events:             e.Events,
			TimeoutSeconds:     e.TimeoutSeconds,
			FailureCount:       e.FailureCount,
			LastError:          e.LastError,
			AbnormalTerminated: e.AbnormalTerminated,
		})
	}
	return HttpNotifyView{Enabled: c.Enabled, Endpoints: endpoints}
}

// HttpNotifyEndpointInput 管理端保存请求的绑定形状：secret 为明文输入
// （仅请求瞬间存在，绝不落库）。空值 = 保持同 id 端点的已存密文（issue #324 S1）。
type HttpNotifyEndpointInput struct {
	Id                 string   `json:"id"`
	Name               string   `json:"name"`
	Enabled            bool     `json:"enabled"`
	URL                string   `json:"url"`
	Secret             string   `json:"secret"`
	Events             []string `json:"events"`
	TimeoutSeconds     int      `json:"timeoutSeconds"`
	FailureCount       int      `json:"failureCount"`
	LastError          string   `json:"lastError"`
	AbnormalTerminated bool     `json:"abnormalTerminated"`
}

// HttpNotifyConfigInput 管理端保存请求的绑定形状（issue #324 S1）。
type HttpNotifyConfigInput struct {
	Enabled   bool                      `json:"enabled"`
	Endpoints []HttpNotifyEndpointInput `json:"endpoints"`
}

// MCPSettingsConfig 内置 MCP server 配置，可在管理面板热修改。
type MCPSettingsConfig struct {
	Enabled bool `json:"enabled"` // /mcp 端点总开关
	Writes  bool `json:"writes"`  // 写工具（create_topic / create_post）开关
}

// ScheduleSettingsConfig 排课器节次作息表配置（12 节上课时间），可在管理面板热修改；
// SSR 透传给 /schedule 页面 props，未配置时回退内置默认作息。
type ScheduleSettingsConfig struct {
	SectionTimes []ScheduleSectionTime `json:"sectionTimes"`
}

// ScheduleSectionTime 单个节次的开始/结束时间（HH:MM）。
type ScheduleSectionTime struct {
	Section int    `json:"section"` // 节次 1..12
	Start   string `json:"start"`   // 开始时间 HH:MM
	End     string `json:"end"`     // 结束时间 HH:MM
}

// AiSummaryConfig AI 课程总结配置（B7，issue #181），可在管理面板热修改。
// Enabled 总开关、GlobalPerMinute 全局每分钟 LLM 生成上限（成本护栏）。
// BaseURL/Model/APIKey/Temperature/MaxTokens 为 provider 参数：管理后台配置
// 优先，未配置时回退 config.toml [ai_summary]（向后兼容）。
// APIKey 为运行时明文（securestore 解密），标 json:"-"：明文/密文绝不随
// JSON 序列化导出（遵循 issue #324 安全模式），持久化走 AiSummarySettingsStorage。
type AiSummaryConfig struct {
	Enabled         bool     `json:"enabled"`               // 总开关（关闭时端点返回 status=disabled）
	GlobalPerMinute int      `json:"globalPerMinute"`       // 全局每分钟生成上限（0 = 用默认 5）
	BaseURL         string   `json:"baseUrl"`               // OpenAI-compatible 端点，如 https://api.openai.com/v1
	Model           string   `json:"model"`                 // 模型 ID，如 gpt-4o
	APIKey          string   `json:"-"`                     // 运行时明文（服务内存）；密文见 AiSummarySettingsStorage
	Temperature     *float64 `json:"temperature,omitempty"` // 可选；不配用默认 0.3
	MaxTokens       *int     `json:"maxTokens,omitempty"`   // 可选；不配用默认 1024
}

// AiSummarySettingsStorage AI 总结配置的落库 JSON 形状：与对外 AiSummaryConfig 分离，
// apiKey 密文只在持久化序列化时出现，不进入 API 响应/缓存结构。
type AiSummarySettingsStorage struct {
	Enabled         bool     `json:"enabled"`
	GlobalPerMinute int      `json:"globalPerMinute"`
	BaseURL         string   `json:"baseUrl"`
	Model           string   `json:"model"`
	APIKeyEncrypted string   `json:"apiKeyEncrypted,omitempty"` // 密文（AES-256-GCM）
	Temperature     *float64 `json:"temperature,omitempty"`
	MaxTokens       *int     `json:"maxTokens,omitempty"`
}

// ToConfig 将落库形状转为领域结构（apiKey 密文原样拷贝，调用方解密）。
func (s AiSummarySettingsStorage) ToConfig() AiSummaryConfig {
	return AiSummaryConfig{
		Enabled:         s.Enabled,
		GlobalPerMinute: s.GlobalPerMinute,
		BaseURL:         s.BaseURL,
		Model:           s.Model,
		APIKey:          s.APIKeyEncrypted,
		Temperature:     s.Temperature,
		MaxTokens:       s.MaxTokens,
	}
}

// AiSummarySettingsView 管理端 GET 回显形状：apiKey 仅回显是否已配置
// （明文/密文均不出现在响应中）。
type AiSummarySettingsView struct {
	Enabled          bool     `json:"enabled"`
	GlobalPerMinute  int      `json:"globalPerMinute"`
	BaseURL          string   `json:"baseUrl"`
	Model            string   `json:"model"`
	APIKeyConfigured bool     `json:"apiKeyConfigured"`
	Temperature      *float64 `json:"temperature,omitempty"`
	MaxTokens        *int     `json:"maxTokens,omitempty"`
}

// ToView 由落库形状生成回显视图。
func (s AiSummarySettingsStorage) ToView() AiSummarySettingsView {
	return AiSummarySettingsView{
		Enabled:          s.Enabled,
		GlobalPerMinute:  s.GlobalPerMinute,
		BaseURL:          s.BaseURL,
		Model:            s.Model,
		APIKeyConfigured: strings.TrimSpace(s.APIKeyEncrypted) != "",
		Temperature:      s.Temperature,
		MaxTokens:        s.MaxTokens,
	}
}

// ToView 由领域结构生成回显视图（默认配置路径：apiKey 恒为空）。
func (c AiSummaryConfig) ToView() AiSummarySettingsView {
	return AiSummarySettingsView{
		Enabled:          c.Enabled,
		GlobalPerMinute:  c.GlobalPerMinute,
		BaseURL:          c.BaseURL,
		Model:            c.Model,
		APIKeyConfigured: strings.TrimSpace(c.APIKey) != "",
		Temperature:      c.Temperature,
		MaxTokens:        c.MaxTokens,
	}
}

// AiSummarySettingsInput 管理端保存请求的绑定形状：apiKey 为明文（仅在请求瞬间
// 存在），空串表示保留已存密文。
type AiSummarySettingsInput struct {
	Enabled         bool     `json:"enabled"`
	GlobalPerMinute int      `json:"globalPerMinute"`
	BaseURL         string   `json:"baseUrl"`
	Model           string   `json:"model"`
	APIKey          string   `json:"apiKey,omitempty"`
	Temperature     *float64 `json:"temperature,omitempty"`
	MaxTokens       *int     `json:"maxTokens,omitempty"`
}

// OneSystemSettingsConfig 一系统同步凭证配置：只落库密文（securestore AES-256-GCM），
// 明文仅在保存时短暂出现；读取时由同步服务在内存中解密，管理端 GET 仅回显是否已配置。
// CookieEncrypted 标 json:"-"：密文绝不随 JSON 序列化导出（review MEDIUM），持久化走 OneSystemSettingsStorage。
type OneSystemSettingsConfig struct {
	CookieEncrypted string `json:"-"` // 加密后的一系统 Cookie header
}

// OneSystemSettingsStorage 一系统凭证的落库 JSON 形状：与对外 OneSystemSettingsConfig 分离，
// 密文只在持久化序列化时出现，不进入 API 响应/缓存结构。ToConfig 转回领域结构。
type OneSystemSettingsStorage struct {
	CookieEncrypted string `json:"cookieEncrypted"`
}

// ToConfig 将落库形状转为领域结构（二者当前字段一致，仅为序列化语义隔离）。
func (s OneSystemSettingsStorage) ToConfig() OneSystemSettingsConfig {
	return OneSystemSettingsConfig{CookieEncrypted: s.CookieEncrypted}
}

// WikiAssetCDNSelf 资源由论坛自身提供（/wiki/_assets/ 路由，默认）。
const WikiAssetCDNSelf = "self"

// WikiAssetCDNJsDelivr 资源由 jsDelivr CDN 提供（gh 镜像 GitHub 仓库文件）。
const WikiAssetCDNJsDelivr = "jsDelivr"

// WikiAssetCDNDefault 默认资源 CDN（self：论坛二进制内置服务，无外部依赖）。
const WikiAssetCDNDefault = WikiAssetCDNSelf

// WikiSyncSettingsConfig wiki 同步设置（webhook 验签密钥 + 资源 CDN）：
// 密钥只落库密文（securestore AES-256-GCM），明文仅在保存时短暂出现；
// 读取时由同步服务在内存中解密，管理端 GET 仅回显是否已配置。
// WebhookSecretEncrypted / WebhookSecretCleared 标 json:"-"：
// 密文与清除标记绝不随 JSON 序列化导出，持久化走 WikiSyncSettingsStorage。
type WikiSyncSettingsConfig struct {
	WebhookSecretEncrypted string `json:"-"`
	// WebhookSecretCleared 管理端显式清除过密钥：为 true 时即使 config.toml
	// 存在旧明文 [wiki.git].webhook_secret 也保持禁用（fail-closed），
	// 避免管理员误以为已禁用而旧密钥仍生效。
	WebhookSecretCleared bool `json:"-"`
	// AssetCDN wiki 资源（图片/附件）的对外提供方式：self（默认，走
	// /wiki/_assets/）或 jsDelivr（gh 镜像）。渲染期由同步器据此生成资源 URL。
	AssetCDN string `json:"-"`
}

// WikiSyncSettingsStorage wiki 同步设置的落库 JSON 形状：与对外
// WikiSyncSettingsConfig 分离，密文只在持久化序列化时出现。
type WikiSyncSettingsStorage struct {
	WebhookSecretEncrypted string `json:"webhookSecretEncrypted"`
	WebhookSecretCleared   bool   `json:"webhookSecretCleared"`
	// AssetCDN 空串 = 默认 self（存量配置无此字段）。
	AssetCDN string `json:"assetCDN,omitempty"`
}

// ToConfig 将落库形状转为领域结构（AssetCDN 空串归一为默认 self）。
func (s WikiSyncSettingsStorage) ToConfig() WikiSyncSettingsConfig {
	cfg := WikiSyncSettingsConfig(s)
	if cfg.AssetCDN == "" {
		cfg.AssetCDN = WikiAssetCDNDefault
	}
	return cfg
}

type SiteThemeConfig struct {
	Version     int                   `json:"version"`
	Enabled     bool                  `json:"enabled"`
	Themes      []SiteThemeDefinition `json:"themes"`
	Prepublish  *SiteThemePrepublish  `json:"prepublish,omitempty"`
	PublishedAt string                `json:"publishedAt,omitempty"`
}

type SiteThemeTokens struct {
	ColorBase100          string `json:"color-base-100"`
	ColorBase200          string `json:"color-base-200"`
	ColorBase300          string `json:"color-base-300"`
	ColorBaseContent      string `json:"color-base-content"`
	ColorIconMuted        string `json:"color-icon-muted"`
	ColorLine             string `json:"color-line"`
	ColorPrimary          string `json:"color-primary"`
	ColorPrimaryContent   string `json:"color-primary-content"`
	ColorSecondary        string `json:"color-secondary"`
	ColorSecondaryContent string `json:"color-secondary-content"`
	ColorAccent           string `json:"color-accent"`
	ColorAccentContent    string `json:"color-accent-content"`
	ColorNeutral          string `json:"color-neutral"`
	ColorNeutralContent   string `json:"color-neutral-content"`
	ColorInfo             string `json:"color-info"`
	ColorInfoContent      string `json:"color-info-content"`
	ColorSuccess          string `json:"color-success"`
	ColorSuccessContent   string `json:"color-success-content"`
	ColorWarning          string `json:"color-warning"`
	ColorWarningContent   string `json:"color-warning-content"`
	ColorError            string `json:"color-error"`
	ColorErrorContent     string `json:"color-error-content"`
	RadiusSelector        string `json:"radius-selector"`
	RadiusField           string `json:"radius-field"`
	RadiusBox             string `json:"radius-box"`
	SizeSelector          string `json:"size-selector"`
	SizeField             string `json:"size-field"`
	Border                string `json:"border"`
	Depth                 string `json:"depth"`
}

func (tokens *SiteThemeTokens) NormalizeFrom(defaults SiteThemeTokens) {
	tokens.ColorBase100 = normalizeSiteThemeTokenValue(tokens.ColorBase100, defaults.ColorBase100)
	tokens.ColorBase200 = normalizeSiteThemeTokenValue(tokens.ColorBase200, defaults.ColorBase200)
	tokens.ColorBase300 = normalizeSiteThemeTokenValue(tokens.ColorBase300, defaults.ColorBase300)
	tokens.ColorBaseContent = normalizeSiteThemeTokenValue(tokens.ColorBaseContent, defaults.ColorBaseContent)
	tokens.ColorIconMuted = normalizeSiteThemeTokenValue(tokens.ColorIconMuted, defaults.ColorIconMuted)
	tokens.ColorLine = normalizeSiteThemeTokenValue(tokens.ColorLine, defaults.ColorLine)
	tokens.ColorPrimary = normalizeSiteThemeTokenValue(tokens.ColorPrimary, defaults.ColorPrimary)
	tokens.ColorPrimaryContent = normalizeSiteThemeTokenValue(tokens.ColorPrimaryContent, defaults.ColorPrimaryContent)
	tokens.ColorSecondary = normalizeSiteThemeTokenValue(tokens.ColorSecondary, defaults.ColorSecondary)
	tokens.ColorSecondaryContent = normalizeSiteThemeTokenValue(tokens.ColorSecondaryContent, defaults.ColorSecondaryContent)
	tokens.ColorAccent = normalizeSiteThemeTokenValue(tokens.ColorAccent, defaults.ColorAccent)
	tokens.ColorAccentContent = normalizeSiteThemeTokenValue(tokens.ColorAccentContent, defaults.ColorAccentContent)
	tokens.ColorNeutral = normalizeSiteThemeTokenValue(tokens.ColorNeutral, defaults.ColorNeutral)
	tokens.ColorNeutralContent = normalizeSiteThemeTokenValue(tokens.ColorNeutralContent, defaults.ColorNeutralContent)
	tokens.ColorInfo = normalizeSiteThemeTokenValue(tokens.ColorInfo, defaults.ColorInfo)
	tokens.ColorInfoContent = normalizeSiteThemeTokenValue(tokens.ColorInfoContent, defaults.ColorInfoContent)
	tokens.ColorSuccess = normalizeSiteThemeTokenValue(tokens.ColorSuccess, defaults.ColorSuccess)
	tokens.ColorSuccessContent = normalizeSiteThemeTokenValue(tokens.ColorSuccessContent, defaults.ColorSuccessContent)
	tokens.ColorWarning = normalizeSiteThemeTokenValue(tokens.ColorWarning, defaults.ColorWarning)
	tokens.ColorWarningContent = normalizeSiteThemeTokenValue(tokens.ColorWarningContent, defaults.ColorWarningContent)
	tokens.ColorError = normalizeSiteThemeTokenValue(tokens.ColorError, defaults.ColorError)
	tokens.ColorErrorContent = normalizeSiteThemeTokenValue(tokens.ColorErrorContent, defaults.ColorErrorContent)
	tokens.RadiusSelector = normalizeSiteThemeTokenValue(tokens.RadiusSelector, defaults.RadiusSelector)
	tokens.RadiusField = normalizeLegacyRadiusField(normalizeSiteThemeTokenValue(tokens.RadiusField, defaults.RadiusField))
	tokens.RadiusBox = normalizeSiteThemeTokenValue(tokens.RadiusBox, defaults.RadiusBox)
	tokens.SizeSelector = normalizeSiteThemeTokenValue(tokens.SizeSelector, defaults.SizeSelector)
	tokens.SizeField = normalizeSiteThemeTokenValue(tokens.SizeField, defaults.SizeField)
	tokens.Border = normalizeSiteThemeTokenValue(tokens.Border, defaults.Border)
	tokens.Depth = normalizeSiteThemeTokenValue(tokens.Depth, defaults.Depth)
}

func (tokens SiteThemeTokens) BaseColor() string {
	return sanitizeSiteThemeTokenValue(tokens.ColorBase100)
}

func (tokens SiteThemeTokens) AppendCSSVariables(sb *strings.Builder) {
	appendSiteThemeCSSVar(sb, "color-base-100", tokens.ColorBase100)
	appendSiteThemeCSSVar(sb, "color-base-200", tokens.ColorBase200)
	appendSiteThemeCSSVar(sb, "color-base-300", tokens.ColorBase300)
	appendSiteThemeCSSVar(sb, "color-base-content", tokens.ColorBaseContent)
	appendSiteThemeCSSVar(sb, "color-icon-muted", tokens.ColorIconMuted)
	appendSiteThemeCSSVar(sb, "color-line", tokens.ColorLine)
	appendSiteThemeCSSVar(sb, "color-primary", tokens.ColorPrimary)
	appendSiteThemeCSSVar(sb, "color-primary-content", tokens.ColorPrimaryContent)
	appendSiteThemeCSSVar(sb, "color-secondary", tokens.ColorSecondary)
	appendSiteThemeCSSVar(sb, "color-secondary-content", tokens.ColorSecondaryContent)
	appendSiteThemeCSSVar(sb, "color-accent", tokens.ColorAccent)
	appendSiteThemeCSSVar(sb, "color-accent-content", tokens.ColorAccentContent)
	appendSiteThemeCSSVar(sb, "color-neutral", tokens.ColorNeutral)
	appendSiteThemeCSSVar(sb, "color-neutral-content", tokens.ColorNeutralContent)
	appendSiteThemeCSSVar(sb, "color-info", tokens.ColorInfo)
	appendSiteThemeCSSVar(sb, "color-info-content", tokens.ColorInfoContent)
	appendSiteThemeCSSVar(sb, "color-success", tokens.ColorSuccess)
	appendSiteThemeCSSVar(sb, "color-success-content", tokens.ColorSuccessContent)
	appendSiteThemeCSSVar(sb, "color-warning", tokens.ColorWarning)
	appendSiteThemeCSSVar(sb, "color-warning-content", tokens.ColorWarningContent)
	appendSiteThemeCSSVar(sb, "color-error", tokens.ColorError)
	appendSiteThemeCSSVar(sb, "color-error-content", tokens.ColorErrorContent)
	appendSiteThemeCSSVar(sb, "radius-selector", tokens.RadiusSelector)
	appendSiteThemeCSSVar(sb, "radius-field", tokens.RadiusField)
	appendSiteThemeCSSVar(sb, "radius-box", tokens.RadiusBox)
	appendSiteThemeCSSVar(sb, "size-selector", tokens.SizeSelector)
	appendSiteThemeCSSVar(sb, "size-field", tokens.SizeField)
	appendSiteThemeCSSVar(sb, "border", tokens.Border)
	appendSiteThemeCSSVar(sb, "depth", tokens.Depth)
}

func normalizeSiteThemeTokenValue(value string, defaultValue string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "{};<>") {
		return defaultValue
	}
	return value
}

func normalizeLegacyRadiusField(value string) string {
	switch strings.TrimSpace(value) {
	case "0.375rem", "6px":
		return "0.5rem"
	default:
		return value
	}
}

func sanitizeSiteThemeTokenValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "{};<>") {
		return ""
	}
	return value
}

func appendSiteThemeCSSVar(sb *strings.Builder, name string, value string) {
	value = sanitizeSiteThemeTokenValue(value)
	if value == "" {
		return
	}
	sb.WriteString("--gf-")
	sb.WriteString(name)
	sb.WriteByte(':')
	sb.WriteString(value)
	sb.WriteByte(';')
}

type SiteThemeDefinition struct {
	Name        string          `json:"name"`
	Label       string          `json:"label"`
	ColorScheme string          `json:"colorScheme"`
	Tokens      SiteThemeTokens `json:"tokens"`
}

type SiteThemePrepublish struct {
	Enabled   bool                  `json:"enabled"`
	Themes    []SiteThemeDefinition `json:"themes"`
	UpdatedAt string                `json:"updatedAt,omitempty"`
}

func FirstSiteThemeDefinition(themes []SiteThemeDefinition) SiteThemeDefinition {
	if len(themes) == 0 {
		return SiteThemeDefinition{}
	}
	return themes[0]
}

func NormalizeSiteThemeDefinitions(themes []SiteThemeDefinition, defaults []SiteThemeDefinition, fallback SiteThemeDefinition) []SiteThemeDefinition {
	if len(themes) == 0 {
		return cloneSiteThemeDefinitions(defaults)
	}
	for index := range themes {
		NormalizeSiteThemeDefinition(&themes[index], defaults, fallback)
	}
	return themes
}

func NormalizeSiteThemeDefinition(theme *SiteThemeDefinition, defaults []SiteThemeDefinition, fallback SiteThemeDefinition) {
	defaultTheme := defaultSiteThemeDefinition(theme.Name, defaults, fallback)
	if theme.Name != defaultTheme.Name {
		theme.Name = defaultTheme.Name
	}
	if theme.Label == "" {
		theme.Label = defaultTheme.Label
	}
	if !isSiteThemeColorScheme(theme.ColorScheme) {
		theme.ColorScheme = defaultTheme.ColorScheme
	}
	theme.Tokens.NormalizeFrom(defaultTheme.Tokens)
}

func cloneSiteThemeDefinitions(themes []SiteThemeDefinition) []SiteThemeDefinition {
	cloned := make([]SiteThemeDefinition, len(themes))
	copy(cloned, themes)
	return cloned
}

func defaultSiteThemeDefinition(name string, defaults []SiteThemeDefinition, fallback SiteThemeDefinition) SiteThemeDefinition {
	name = strings.TrimSpace(name)
	for _, theme := range defaults {
		if theme.Name == name {
			return theme
		}
	}
	return fallback
}

func isSiteThemeColorScheme(value string) bool {
	return value == "dark" || value == "light"
}

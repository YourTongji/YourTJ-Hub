package pkservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// 与上游一致的一系统请求常量。
const (
	onesystemManualArrangeURL = "https://1.tongji.edu.cn/api/arrangementservice/manualArrange/page?profile"
	onesystemReferer          = "https://1.tongji.edu.cn/taskResultQuery"
	onesystemUserAgent        = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"
	onesystemPageSize         = 200
	onesystemMaxAttempts      = 5
	onesystemRequestTimeout   = 15 * time.Second
)

var retryableHTTPStatus = map[int]bool{429: true, 500: true, 502: true, 503: true, 504: true}

// CourseRaw 一系统 manualArrange/page 单条教学班原始字段（与上游 sync.ts 字段集对齐）。
type CourseRaw struct {
	CalendarIdI18n       string       `json:"calendarIdI18n"`
	TeachingLanguage     string       `json:"teachingLanguage"`
	TeachingLanguageI18n string       `json:"teachingLanguageI18n"`
	CourseLabelId        *uint64      `json:"courseLabelId"`
	CourseLabelName      string       `json:"courseLabelName"`
	AssessmentMode       string       `json:"assessmentMode"`
	AssessmentModeI18n   string       `json:"assessmentModeI18n"`
	Campus               string       `json:"campus"`
	CampusI18n           string       `json:"campusI18n"`
	Faculty              string       `json:"faculty"`
	FacultyI18n          string       `json:"facultyI18n"`
	MajorList            []string     `json:"majorList"`
	Id                   *uint64      `json:"id"`
	Code                 string       `json:"code"`
	Name                 string       `json:"name"`
	Period               *float64     `json:"period"`
	WeekHour             *float64     `json:"weekHour"`
	Number               *int         `json:"number"`
	ElcNumber            *int         `json:"elcNumber"`
	StartWeek            *int         `json:"startWeek"`
	EndWeek              *int         `json:"endWeek"`
	CourseCode           string       `json:"courseCode"`
	CourseName           string       `json:"courseName"`
	Credits              *float64     `json:"credits"`
	Credit               *float64     `json:"credit"`
	NewCourseCode        string       `json:"newCourseCode"`
	ArrangeInfo          string       `json:"arrangeInfo"`
	TeacherList          []TeacherRaw `json:"teacherList"`
}

// TeacherRaw 一系统教学班教师（teacherList 元素）。
type TeacherRaw struct {
	Id          *uint64 `json:"id"`
	TeacherCode string  `json:"teacherCode"`
	TeacherName string  `json:"teacherName"`
}

type manualArrangePage struct {
	Code onesystemCode `json:"code"` // 0 或 200=成功；其他=业务/鉴权失败（一系统 HTTP 200 信封，真实成功码为 200）
	Msg  string        `json:"msg"`
	Data struct {
		Total_ int         `json:"total_"`
		List   []CourseRaw `json:"list"`
	} `json:"data"`
}

// onesystemCode 解析一系统响应信封的 code 字段：0 或 200 表示成功，其他表示业务/鉴权失败。
// 对照 serverless 上游（courseSync.ts code===200、sync.ts 仅检查 HTTP res.ok），
// 一系统 manualArrange 的成功码是 200；兼容历史假设的 0 一并视为成功。
type onesystemCode int64

// UnmarshalJSON 兼容数字（1）、数字字符串（"1"）、null 与缺失。
// 注意 JSON 字符串值会带引号，须先剥离引号再 ParseInt，否则 "1" 会被误判为成功。
// 显式但非数字的 code 值（如 "success"）fail-closed：返回错误，避免把未知失败形状
// 当作成功页而误删存量数据（review HIGH）。
func (c *onesystemCode) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" {
		*c = 0
		return nil
	}
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
		s = strings.TrimSpace(s)
	}
	if s == "" {
		*c = 0
		return nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("无法解析 code 字段 %q", s)
	}
	*c = onesystemCode(n)
	return nil
}

// onesystemClient 一系统 manualArrange 分页抓取客户端。忽略环境代理（等价上游 trust_env=False），
// 每页超时 15s，可重试状态（429/500/502/503/504）退避重试至多 5 次。
type onesystemClient struct {
	baseURL     string
	httpClient  *http.Client
	maxAttempts int
	backoff     func(attempt int) time.Duration
	sleep       func(time.Duration)
}

// newOnesystemClient 构造默认客户端；测试可通过字段覆盖 baseURL/backoff/sleep。
func newOnesystemClient() *onesystemClient {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil // trust_env=False：不读取 HTTP_PROXY/HTTPS_PROXY
	return &onesystemClient{
		baseURL:     onesystemManualArrangeURL,
		httpClient:  &http.Client{Transport: transport, Timeout: onesystemRequestTimeout},
		maxAttempts: onesystemMaxAttempts,
		backoff:     func(attempt int) time.Duration { return backoffDuration(attempt) },
		sleep:       time.Sleep,
	}
}

// backoffDuration 上游 sleep(min(10s, (1+attempt*2)s))。
func backoffDuration(attempt int) time.Duration {
	sec := 1 + attempt*2
	if sec > 10 {
		sec = 10
	}
	return time.Duration(sec) * time.Second
}

// fetchPage 抓取一页。返回解析后的分页结果；非可重试 HTTP 状态返回带状态码与响应体摘要的错误（AC2）。
func (c *onesystemClient) fetchPage(ctx context.Context, cookie string, calendarId, pageNum, pageSize int) (*manualArrangePage, error) {
	payload := map[string]any{
		"condition": map[string]any{
			"trainingLevel":     "",
			"campus":            "",
			"calendar":          calendarId,
			"college":           "",
			"course":            "",
			"ids":               []any{},
			"isChineseTeaching": nil,
		},
		"pageNum_":  pageNum,
		"pageSize_": pageSize,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("一系统请求序列化失败: %w", err)
	}

	var lastErr error
	attempts := 0
	for attempt := 1; attempt <= c.maxAttempts; attempt++ {
		attempts = attempt
		page, retryable, err := c.tryFetch(ctx, cookie, body)
		if err == nil {
			return page, nil
		}
		lastErr = err
		if !retryable || attempt >= c.maxAttempts {
			break
		}
		c.sleep(c.backoff(attempt))
	}
	return nil, fmt.Errorf("一系统请求失败（尝试 %d 次后失败）: %w", attempts, lastErr)
}

// tryFetch 执行单次 POST；返回 (page, 是否可重试, 错误)。
func (c *onesystemClient) tryFetch(ctx context.Context, cookie string, body []byte) (*manualArrangePage, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, false, fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", onesystemUserAgent)
	req.Header.Set("Referer", onesystemReferer)
	req.Header.Set("Cookie", cookie)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, true, fmt.Errorf("网络请求失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if readErr != nil {
		return nil, false, fmt.Errorf("读取响应失败: %w", readErr)
	}
	if resp.StatusCode == http.StatusOK {
		var page manualArrangePage
		if err := json.Unmarshal(respBody, &page); err != nil {
			return nil, false, fmt.Errorf("一系统返回无法解析: %w", err)
		}
		if code := int64(page.Code); code != 0 && code != 200 {
			// HTTP 200 但业务/鉴权失败（code 非 0/200）：勿当成功页，否则会误删存量数据（review HIGH）。
			// 鉴权失败不可重试，直接以明确错误中止；错误体需脱敏。
			hint := redactCredentials(strings.TrimSpace(page.Msg))
			if hint == "" {
				hint = redactCredentials(strings.TrimSpace(string(respBody)))
			}
			if len(hint) > 200 {
				hint = hint[:200]
			}
			return nil, false, fmt.Errorf("一系统返回业务失败: code=%d %s", code, hint)
		}
		return &page, false, nil
	}
	hint := redactCredentials(strings.TrimSpace(string(respBody)))
	if len(hint) > 200 {
		hint = hint[:200]
	}
	err = fmt.Errorf("一系统请求失败: HTTP %d %s", resp.StatusCode, hint)
	if retryableHTTPStatus[resp.StatusCode] {
		return nil, true, err
	}
	return nil, false, err
}

// redactCredentials 移除响应体提示中的敏感凭证片段（JWTUser/JSESSIONID/sessionid/token/cookie 等），
// 防止上游错误响应把会话凭证带回并持久化到 fetchlog/日志。
var sensitiveKeyPattern = regexp.MustCompile(`(?i)(jwtuser|jsessionid|sessionid|cookie|token|authorization|set-cookie)\s*[=:]\s*[^\s;&,]+`)

func redactCredentials(hint string) string {
	return sensitiveKeyPattern.ReplaceAllString(hint, "$1=***")
}

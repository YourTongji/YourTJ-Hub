package console

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/captchaOpt"
	jwtopt "github.com/leancodebox/GooseForum/app/bundles/jwtopt"
	"github.com/leancodebox/GooseForum/app/bundles/preferences"
	"github.com/leancodebox/GooseForum/app/bundles/ratelimit"
	paniclog "github.com/leancodebox/GooseForum/app/bundles/recovery"
	"github.com/leancodebox/GooseForum/app/bundles/setting"
	"github.com/leancodebox/GooseForum/app/bundles/signalwatch"
	"github.com/leancodebox/GooseForum/app/console/job"
	"github.com/leancodebox/GooseForum/app/http/routes"
	"github.com/leancodebox/GooseForum/app/service/backgroundservice"
	"github.com/leancodebox/GooseForum/app/service/dataservice"
	"github.com/leancodebox/GooseForum/app/service/filemigrateservice"
	"github.com/leancodebox/GooseForum/app/service/mailservice"
	"github.com/leancodebox/GooseForum/app/service/oauthservice"
	"github.com/leancodebox/GooseForum/app/service/oidcservice"
	"github.com/leancodebox/GooseForum/app/service/sessionservice"
	"github.com/spf13/cast"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

// CmdServe represents the available web sub-command.
var CmdServe = &cobra.Command{
	Use:   "serve",
	Short: "Start web server",
	Run:   runWeb,
	Args:  cobra.NoArgs,
}

func runWeb(_ *cobra.Command, _ []string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	slog.Info("GooseForum:start")
	slog.Info(fmt.Sprintf("GooseForum:useMem %d KB", m.Alloc/1024/8))

	warnInsecureServerURL()
	startDebugServices()
	ginServe()
}

// warnInsecureServerURL logs a startup warning when a non-local deployment is
// reachable over plain http (CWE-614). Cookies are already fail-closed by
// setting.CookieSecure(), so this is a deployment hygiene notice, not a fatal
// gate — template defaults are `production` + `http://localhost`, and browsers
// treat `localhost` as a secure context that still accepts Secure cookies,
// so we do not refuse to boot (issue #113).
func warnInsecureServerURL() {
	serverURL := strings.TrimSpace(preferences.GetString("server.url", ""))
	if !shouldWarnInsecureServerURL() {
		return
	}
	slog.Warn(fmt.Sprintf(
		"server.url=%q 在非 local 环境下不是 https，会话 Cookie 已强制 Secure，浏览器不会在明文连接上回传它们；请将 server.url 改为 https:// 反向代理地址或显式配置 HTTPS 终结",
		serverURL,
	))
}

// shouldWarnInsecureServerURL is the pure decision predicate backing
// warnInsecureServerURL. Returns true only when the deployment is non-local
// and `server.url` points at a non-https, non-loopback host.
func shouldWarnInsecureServerURL() bool {
	if setting.IsLocal() {
		return false
	}
	serverURL := strings.TrimSpace(preferences.GetString("server.url", ""))
	if serverURL == "" {
		return false
	}
	u, err := url.Parse(serverURL)
	if err != nil {
		return false
	}
	if strings.ToLower(u.Scheme) == "https" {
		return false
	}
	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	switch host {
	case "localhost", "127.0.0.1", "::1", "":
		return false
	}
	return true
}

func startDebugServices() {
	if !setting.IsDebug() {
		return
	}
	go servePprof()
}

func servePprof() {
	defer paniclog.Recover("pprof_server")
	// go tool pprof http://localhost:19070/debug/pprof/profile
	// go tool pprof -http=:9001 http://localhost:19070/debug/pprof/heap
	// http://127.0.0.1:19070/debug/pprof/
	const addr = "127.0.0.1:19070"
	srv := &http.Server{
		Addr:              addr,
		Handler:           pprofMux(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("debug listen ", "err", err)
	}
}

func pprofMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	return mux
}

func ginServe() {
	// 拒绝使用内置默认签名密钥启动：该密钥公开在源码中，攻击者可据此
	// 伪造 JWT 并解密 TOTP 密钥（见 jwtopt.DefaultSigningKey）。
	// 配置错误必须以非零退出码终止，否则 systemd/docker 会把
	// "配置错误"误判为"正常退出"，重启策略与告警都不会生效。
	if jwtopt.IsSigningKeyDefault() {
		slog.Error("app.signingKey 未配置，仍在使用内置默认密钥。请配置一个随机密钥后重试。")
		os.Exit(1)
	}
	preferences.OpenConfigChangeEvent()
	// 初始化OAuth配置
	oauthservice.InitOAuth()
	oidcservice.InitOIDC()
	captchaOpt.StartCleanup()
	ratelimit.StartCleanup()
	mailservice.StartEmailProcessor()
	// 文件迁移 worker：处理管理面板创建的 file-migrate 任务
	backgroundservice.RunWorker("file_migrate_worker", filemigrateservice.TaskTypeFileMigrate, filemigrateservice.RunMigrateTask)
	// 数据导出 worker：处理管理面板创建的 export 任务
	backgroundservice.RunWorker("data_export_worker", dataservice.TaskTypeExport, dataservice.RunExportTask)
	sessionservice.CleanupExpired()
	oidcservice.CleanupExpired()
	job.Run()

	port := preferences.GetString("server.port", 8080)
	engine := newGinEngine()
	routes.RegisterByGin(engine)
	// local 模式默认只绑定回环地址（安全），配置 server.host 可覆盖为 0.0.0.0 以允许局域网访问
	host := preferences.GetString("server.host", "")
	if host == "" && setting.IsLocal() {
		host = `127.0.0.1`
	}
	address := fmt.Sprintf("%v:%v", host, port)
	srv := &http.Server{
		Addr:           address,
		Handler:        engine,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	quit := make(chan os.Signal, 1)
	signalwatch.ListenSignal(quit)
	go func() {
		defer paniclog.Recover("http_server")
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http serve ", "err", err)
			fmt.Println("http serve ", "err", err)
			quit <- os.Interrupt
		}
	}()

	slog.Info("GooseForum:listen " + port)
	slog.Info("use port:" + port)
	slog.Info("start use:" + cast.ToString(setting.GetUnitTime()))
	fmt.Println("if in local you can http://localhost:" + port)

	data := <-quit
	slog.Info("Shutdown Server ...", "signal", data)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Info("Server Shutdown", "err", err)
	}

	slog.Info("Server exiting")
}

func newGinEngine() *gin.Engine {
	if setting.IsDebug() {
		gin.SetMode(gin.DebugMode)
		return gin.Default()
	} else {
		gin.DisableConsoleColor()
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	// 只信任部署层反向代理（1Panel/openresty → 127.0.0.1），
	// 防止客户端伪造 X-Forwarded-For 绕过按 IP 的限流。
	trustedProxies := preferences.GetStringSlice("server.trusted_proxies")
	if len(trustedProxies) == 0 {
		trustedProxies = []string{"127.0.0.1", "::1"}
	}
	_ = engine.SetTrustedProxies(trustedProxies)
	return engine
}

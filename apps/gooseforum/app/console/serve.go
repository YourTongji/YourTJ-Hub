package console

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/captchaOpt"
	jwtopt "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/ratelimit"
	paniclog "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/recovery"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/setting"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/signalwatch"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/console/job"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/routes"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/migration"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/backgroundservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/dataservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/filemigrateservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/mailservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/oauthservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/oidcservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/searchservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/sessionservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/webpushservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/wikiservice"
	"github.com/spf13/cast"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

// CmdServe represents the available web sub-command.
var CmdServe = &cobra.Command{
	Use:   "serve",
	Short: "Start web server",
	RunE:  runWeb,
	Args:  cobra.NoArgs,
}

func runWeb(_ *cobra.Command, _ []string) error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	slog.Info("GooseForum:start")
	slog.Info(fmt.Sprintf("GooseForum:useMem %d KB", m.Alloc/1024/8))

	warnInsecureServerURL()
	webpushservice.LogConfigStatus()
	startDebugServices()
	return ginServe()
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

func ginServe() error {
	// fail-closed：拒绝在不安全的 JWT 签名密钥下启动。空值、内置公开默认值
	// 与部署模板占位符都可使攻击者伪造密码重置令牌（见 issue #106），因此
	// 配置错误必须以非零退出码终止——否则 systemd/docker 会把"配置错误"
	// 误判为"正常退出"，重启策略与告警都不会生效。
	if reason := jwtopt.SigningKeyProblem(); reason != "" {
		slog.Error("app.signingKey 不可用，拒绝启动", "reason", reason,
			"hint", "请配置一个随机密钥（例如 openssl rand -base64 32）后重试")
		os.Exit(1)
	}
	// prepareServeRuntime 只配置与数据库无关的启动态，可在迁移门闸开启前安全执行。
	prepareServeRuntime()
	port := preferences.GetString("server.port", 8080)
	serverRuntime, err := newServeRuntime(port)
	if err != nil {
		return fmt.Errorf("create serve runtime: %w", err)
	}
	serverRuntime.start()

	slog.Info("GooseForum:listen " + port)
	slog.Info("use port:" + port)
	slog.Info("start use:" + cast.ToString(setting.GetUnitTime()))
	fmt.Println("if in local you can http://localhost:" + port)

	return serverRuntime.wait()
}

// prepareServeRuntime 只配置与数据库无关的启动态，可在迁移门闸开启前安全执行。
func prepareServeRuntime() {
	preferences.OpenConfigChangeEvent()
}

type serveRuntime struct {
	server       *http.Server
	startupGate  *middleware.StartupGate
	listener     net.Listener
	quit         chan os.Signal
	shutdownOnce sync.Once
	// startupDone 在启动 goroutine（迁移）落定后关闭。wait() 收到 quit 信号后
	// 必须先等它再读 fatalErr：外部信号/测试可能抢在 setFatal 之前把信号入队，
	// 若无此屏障会读到 nil，把致命迁移错误吞掉（#370 遗留竞态）。
	startupDone   chan struct{}
	fatalMu       sync.Mutex
	fatalErr      error
	migrate       func() error
	startBusiness func()
}

func newServeRuntime(port string) (*serveRuntime, error) {
	engine := newGinEngine()
	// 启动门闸必须是第一个中间件，且在 RegisterByGin 之前注册：gin 在路由
	// 注册时快照处理器链，gate 注册晚了会静默失效；gate 前置保证迁移期间
	// 没有任何 DB 依赖的中间件/控制器被执行。
	startupGate := middleware.NewStartupGate()
	engine.Use(startupGate.Handler)
	routes.RegisterByGin(engine)
	host := preferences.GetString("server.host", "")
	if host == "" && setting.IsLocal() {
		host = `127.0.0.1`
	}
	address := fmt.Sprintf("%v:%v", host, port)
	srv := newHTTPServer(address, engine)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", address, err)
	}
	return &serveRuntime{
		server:        srv,
		startupGate:   startupGate,
		listener:      listener,
		quit:          make(chan os.Signal, 1),
		startupDone:   make(chan struct{}),
		migrate:       migration.M,
		startBusiness: startBusinessServices,
	}, nil
}

func (r *serveRuntime) start() {
	signalwatch.ListenSignal(r.quit)
	go func() {
		defer paniclog.Recover("http_server")
		if err := r.server.Serve(r.listener); !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http serve ", "err", err)
			fmt.Println("http serve ", "err", err)
			r.requestShutdown()
		}
	}()
	// 迁移完成前监听器已就绪并只返回 503（StartupGate），完成后才启动
	// 业务服务并放行流量；worker/cron 全部在成功路径内启动，避免半迁移
	// 实例处理业务请求。
	go func() {
		// 无论成功/失败/deferred，启动结果落定后关闭屏障，wait() 才能安全读 fatal。
		defer close(r.startupDone)
		if err := r.runStartup(); err != nil {
			r.setFatal(err)
			r.requestShutdown()
		}
	}()
}

// runStartup applies migrations and, on success, boots the business services
// and opens the startup gate. A hard migration error or a panic in the
// migration path (e.g. dbconnect.Connect panics on an unreachable database)
// returns an error so the instance exits non-zero instead of staying on the
// 503 loading page forever.
func (r *serveRuntime) runStartup() (fatalErr error) {
	defer func() {
		if panicValue := recover(); panicValue != nil {
			paniclog.LogPanic("startup_migration", panicValue)
			fatalErr = fmt.Errorf("startup migration panicked: %v", panicValue)
		}
	}()
	err := r.migrate()
	if err != nil {
		if migration.Deferred(err) {
			slog.Warn("startup migration deferred, serving with degraded state", "err", err)
		} else {
			slog.Error("startup migration failed", "err", err)
			return err
		}
	}
	r.startBusiness()
	r.startupGate.Complete()
	return nil
}

// startBusinessServices boots every business-facing component that reads or
// writes the database (or depends on a migrated schema). It runs only after
// migration succeeds or defers via a non-fatal sentinel.
func startBusinessServices() {
	// 初始化OAuth配置
	oauthservice.InitOAuth()
	oidcservice.InitOIDC()
	captchaOpt.StartCleanup()
	ratelimit.StartCleanup()
	mailservice.StartEmailProcessor()
	// 启动时立即回收租约过期的 Running 任务（issue #138）：任务领取采用
	// 原子 CAS + processed_at 租约，worker 执行期间心跳续租，worker 循环内
	// 也会周期性回收过期租约；此处是启动时的即时清扫，让崩溃遗留任务尽快
	// 回到 Pending 重新领取。邮件/文件迁移/导出 worker 共用同一恢复逻辑，
	// 各按类型前缀处理。
	mailservice.RecoverStaleTasks()
	filemigrateservice.RecoverStaleTasks()
	dataservice.RecoverStaleTasks()
	dataservice.RecoverImportTasks()
	searchservice.RecoverStaleTasks()
	searchservice.RecoverTopicSearchTasks()
	searchservice.RecoverUserSearchTasks()
	searchservice.RecoverCategorySearchTasks()
	// 启动时确保话题索引的 filterable 属性配置（topicType 过滤依赖；见
	// EnsureTopicIndexConfigured 注释，review N2：仅手动 rebuild 不覆盖存量部署）。
	searchservice.EnsureTopicIndexConfigured()
	courseservice.RecoverStaleTasks()
	courseservice.RecoverCourseStatsRebuildTasks()
	// 文件迁移 worker：处理管理面板创建的 file-migrate 任务
	backgroundservice.RunWorker("file_migrate_worker", filemigrateservice.TaskTypeFileMigrate, filemigrateservice.RunMigrateTask)
	// 数据导出 worker：处理管理面板创建的 export 任务
	backgroundservice.RunWorker("data_export_worker", dataservice.TaskTypeExport, dataservice.RunExportTask)
	// 数据导入 worker：消费 import 任务，事务化导入暂存文件中的数据。
	backgroundservice.RunWorker("data_import_worker", dataservice.TaskTypeImport, dataservice.RunImportTask)
	// 课程搜索同步 worker：消费 course-search. 前缀 outbox 任务，投影到 Meili
	backgroundservice.RunWorker("course_search_worker", searchservice.TaskTypeCourseSearch, searchservice.RunCourseSearchTask)
	// 主题、用户、分类搜索 worker：消费 transaction-bound outbox，避免业务
	// 请求/事件 consumer 同步等待 Meilisearch。
	backgroundservice.RunWorker("topic_search_worker", searchservice.TaskTypeTopicSearch, searchservice.RunTopicSearchTask)
	backgroundservice.RunWorker("user_search_worker", searchservice.TaskTypeUserSearch, searchservice.RunUserSearchTask)
	backgroundservice.RunWorker("category_search_worker", searchservice.TaskTypeCategorySearch, searchservice.RunCategorySearchTask)
	// 课程统计重建 worker：消费 course-stats. 前缀任务（管理页“重建课程统计”触发）
	backgroundservice.RunWorker("course_stats_worker", courseservice.TaskTypeCourseStatsRebuild, courseservice.RunCourseStatsRebuildTask)
	// 课评删除隔离窗口清理 worker（issue #175 B3 隐私合规）：消费
	// course-review-cleanup 前缀任务，脱敏超窗 deleted 行；失败按 taskQueue
	// 语义重试至多 3 次后 failed 并有日志
	backgroundservice.RunWorker("course_review_cleanup_worker", courseservice.TaskTypeCourseReviewCleanup, courseservice.RunCleanupTask)
	// Web Push 推送 worker：消费 webpush. 前缀 outbox 任务（通知行创建后入队），
	// 向用户浏览器订阅发送系统推送。实例未配置 VAPID 密钥时任务直接 no-op
	// 置 Success（dev 从 main 快照同步的任务行绝不会外发）。
	webpushservice.RecoverStaleTasks()
	backgroundservice.RunWorker("webpush_worker", webpushservice.TaskTypePush, webpushservice.RunPushTask)
	sessionservice.CleanupExpired()
	oidcservice.CleanupExpired()
	job.Run()

	// 启动时异步执行一次 wiki GitHub 同步（D1）：进程启动后立即拉取仓库最新
	// head 并投影到论坛。未配置 [wiki.git].repo 时幂等跳过（Sync 报错仅告警，
	// 不阻塞服务启动）；失败不重试，由每日定时同步兜底。
	// 崩溃恢复（issue #290）：上次进程被杀/重启遗留的 running 运行行在
	// 启动时回收为 failed，管理端手动同步按钮不会被永久禁用。
	wikiservice.ReconcileStaleRuns()
	if wikiservice.LoadGitConfig().Enabled() {
		go func() {
			defer paniclog.Recover("wiki_startup_sync")
			if _, err := wikiservice.Sync("startup"); err != nil && !errors.Is(err, wikiservice.ErrSyncAlreadyRunning) {
				slog.Warn("wiki startup sync failed (scheduled sync will retry)", "error", err)
			}
		}()
	}
}

// setFatal records a startup failure that must be surfaced as a non-zero exit
// code; wait reads it after the graceful shutdown completes. The mutex keeps
// this race-free when an external signal (not requestShutdown) wakes wait.
func (r *serveRuntime) setFatal(err error) {
	r.fatalMu.Lock()
	defer r.fatalMu.Unlock()
	r.fatalErr = err
}

// fatal returns the recorded startup failure, or nil on a normal run.
func (r *serveRuntime) fatal() error {
	r.fatalMu.Lock()
	defer r.fatalMu.Unlock()
	return r.fatalErr
}

// requestShutdown asks the event loop to stop, idempotently and without
// blocking: an external signal may already be queued, and signal.Notify
// panics on a closed channel so the channel is never closed.
func (r *serveRuntime) requestShutdown() {
	r.shutdownOnce.Do(func() {
		select {
		case r.quit <- os.Interrupt:
		default:
		}
	})
}

func (r *serveRuntime) wait() error {
	signal := <-r.quit
	slog.Info("Shutdown Server ...", "signal", signal)
	// 先等启动 goroutine 落定再读 fatal：quit 信号可能先于 setFatal 到达
	// （外部信号或测试直接 requestShutdown），直接读 fatal() 会把致命迁移
	// 错误吞成 nil 导致进程以 0 退出。启动 goroutine 内先 setFatal 再发信号，
	// 屏障等待只有微秒级；外部信号撞上超长迁移时关停会等迁移结束（更安全）。
	<-r.startupDone
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := r.server.Shutdown(shutdownCtx); err != nil {
		slog.Info("Server Shutdown", "err", err)
	}

	slog.Info("Server exiting")
	return r.fatal()
}

func newHTTPServer(address string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
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

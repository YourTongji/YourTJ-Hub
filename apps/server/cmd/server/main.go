package main

import (
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	server "yourtj-hub/server"
	"yourtj-hub/server/internal/config"
)

func main() {
	cfg := config.Load()

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// 健康检查
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API 路由组（M3 起挂载业务 API）
	api := r.Group("/api")
	api.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	// Web 静态资源（go:embed，见 ADR-0001：部署合并单二进制）
	// API 路由优先，其余请求回退到前端 SPA（history 路由）
	webSub, err := fs.Sub(server.WebFS, "webdist")
	if err != nil {
		log.Fatalf("embed webdist: %v", err)
	}
	fileServer := http.FileServer(http.FS(webSub))
	r.NoRoute(func(c *gin.Context) {
		path := strings.TrimPrefix(c.Request.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(webSub, path); err == nil {
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}
		// SPA fallback：非文件路径统一回 index.html
		c.Request.URL.Path = "/"
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	addr := ":" + cfg.Port
	log.Printf("yourtj-hub server listening on %s (env=%s)", addr, cfg.Env)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}

// @title Nexus Core API
// @version 0.0.1
// @description 简易个人订阅制软件管理平台 API
// @host localhost:8080
// @BasePath /
// @schemes http

package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"nexus-core/api"
	_ "nexus-core/docs"
	"nexus-core/global"
	"nexus-core/monitor"
	"nexus-core/persistence/base"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

//
// @Author yfy2001
// @Date 2025/7/21 14 12
//

// openBrowser 根据操作系统打开默认浏览器
// 仅在Windows环境下尝试打开浏览器
func openBrowser(url string) {
	if runtime.GOOS == "windows" {
		_ = execCommand("cmd", "/c", "start", url)
		return
	}
	_ = execCommand("xdg-open", url)
}

// execCommand wraps exec.Command(...).Start
func execCommand(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	return cmd.Start()
}

// main 应用程序入口点
// 初始化配置、注册路由、启动服务器
func main() {
	cfg := global.LoadConfig()
	fmt.Println("Nexus Core starting...")
	base.MainDBManager = base.InitDBManager(cfg.DBConfig)
	base.AutoMigrate(base.MainDBManager.GetDefaultDB())
	r := api.WebEngine

	// register default routes
	api.RegisterDefaultRoutes()

	// start monitor (after DB ready)
	monitor.GlobalMonitor.Start()

	// construct swagger URL based on config.SwaggerURL
	var swaggerUrl string
	if cfg.SwaggerURL == "" {
		cfg.SwaggerURL = "/swagger/index.html"
	}
	// if it's a full URL, use as-is (but allow %d for port formatting)
	if len(cfg.SwaggerURL) >= 7 && (cfg.SwaggerURL[:7] == "http://" || cfg.SwaggerURL[:8] == "https://") {
		// if contains %d, format with port
		if containsPercentD(cfg.SwaggerURL) {
			swaggerUrl = fmt.Sprintf(cfg.SwaggerURL, cfg.Port)
		} else {
			swaggerUrl = cfg.SwaggerURL
		}
	} else if containsPercentD(cfg.SwaggerURL) {
		// not a full URL but has %d - treat as format and prefix with http://localhost:%d
		swaggerUrl = fmt.Sprintf(cfg.SwaggerURL, cfg.Port)
		// if resulting string does not start with http, prefix localhost
		if !(len(swaggerUrl) >= 7 && (swaggerUrl[:7] == "http://" || swaggerUrl[:8] == "https://")) {
			swaggerUrl = fmt.Sprintf("http://localhost:%d%s", cfg.Port, swaggerUrl)
		}
	} else {
		// relative path or simple path: prefix with http://localhost:PORT
		if len(cfg.SwaggerURL) > 0 && cfg.SwaggerURL[0] == '/' {
			swaggerUrl = fmt.Sprintf("http://localhost:%d%s", cfg.Port, cfg.SwaggerURL)
		} else {
			swaggerUrl = fmt.Sprintf("http://localhost:%d/%s", cfg.Port, cfg.SwaggerURL)
		}
	}

	fmt.Println("Swagger UI: ", swaggerUrl)
	// optionally open browser if enabled
	if cfg.AutoOpenBrowser && cfg.SwaggerEnabled {
		go func() {
			time.Sleep(500 * time.Millisecond)
			openBrowser(swaggerUrl)
		}()
	}

	addr := fmt.Sprintf(":%d", cfg.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// run server in background
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	// graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		fmt.Printf("Server forced to shutdown: %v\n", err)
	}

	// stop monitor
	monitor.GlobalMonitor.Stop()

	fmt.Println("Server exiting")
}

// containsPercentD checks if a string contains the %d format verb
func containsPercentD(s string) bool {
	for i := 0; i < len(s)-1; i++ {
		if s[i] == '%' && s[i+1] == 'd' {
			return true
		}
	}
	return false
}

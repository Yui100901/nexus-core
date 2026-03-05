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
	"nexus-core/config"
	_ "nexus-core/docs"
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
	cfg := config.Load()
	fmt.Println("Nexus Core starting...")

	// initialize DB once at startup and verify connectivity
	db := base.Connect()
	if db == nil {
		panic("failed to initialize database")
	}
	// optional lightweight health check: try a simple DB operation
	if err := db.Exec("SELECT 1").Error; err != nil {
		panic(fmt.Sprintf("database health check failed: %v", err))
	}

	r := api.WebEngine

	// register default routes
	api.RegisterDefaultRoutes()

	// start monitor (after DB ready)
	monitor.GlobalMonitor.Start()

	swaggerUrl := fmt.Sprintf("http://localhost:%d/swagger/index.html", cfg.Port)
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

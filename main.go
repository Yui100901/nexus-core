// @title Nexus Core API
// @version 0.0.1
// @description 简易个人订阅制软件管理平台 API
// @host localhost:8080
// @BasePath /
// @schemes http

package main

import (
	"fmt"
	"nexus-core/api"
	"nexus-core/config"
	_ "nexus-core/docs"
	"os/exec"
	"runtime"
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
		_ = exec.Command("cmd", "/c", "start", url).Start()
		return
	}
	_ = exec.Command("xdg-open", url).Start()
}

// main 应用程序入口点
// 初始化配置、注册路由、启动服务器
func main() {
	cfg := config.Load()
	fmt.Println("Nexus Core starting...")
	r := api.WebEngine

	// register default routes
	api.RegisterDefaultRoutes()

	swaggerUrl := fmt.Sprintf("http://localhost:%d/docs/index.html", cfg.Port)
	fmt.Println("Swagger UI: ", swaggerUrl)
	// optionally open browser
	if cfg.AutoOpenBrowser {
		go func() {
			time.Sleep(500 * time.Millisecond)
			openBrowser(swaggerUrl)
		}()
	}

	addr := fmt.Sprintf(":%d", cfg.Port)
	if err := r.Run(addr); err != nil {
		panic(err)
	}
}

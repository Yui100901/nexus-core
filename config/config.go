package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 应用程序配置结构
// 包含服务器端口、数据库路径和其他运行时选项
type Config struct {
	Port            int    `yaml:"port"`              // 服务器监听端口
	DBPath          string `yaml:"db_path"`           // 数据库文件路径
	SwaggerEnabled  bool   `yaml:"swagger_enabled"`   // 是否启用Swagger文档
	AutoOpenBrowser bool   `yaml:"auto_open_browser"` // 是否自动打开浏览器
}

var cfg *Config // 全局配置实例

// Load 加载应用程序配置
// 优先从config-dev.yml文件加载，如果文件不存在则使用默认值
func Load() *Config {
	if cfg != nil {
		return cfg
	}
	// defaults
	cfg = &Config{
		Port:            8080,
		DBPath:          "test.db",
		SwaggerEnabled:  true,
		AutoOpenBrowser: true,
	}
	f, err := os.ReadFile("config-dev.yml")
	if err != nil {
		// no config file, use defaults
		return cfg
	}
	_ = yaml.Unmarshal(f, cfg)
	// sanity checks
	if cfg.Port == 0 {
		cfg.Port = 8080
	}
	if cfg.DBPath == "" {
		cfg.DBPath = "test.db"
	}
	// ensure DBPath timestamp suffix to avoid lock in tests? keep as is
	_ = time.Now()
	return cfg
}

// Get 获取配置实例
// 如果尚未加载配置，则先加载默认配置
func Get() *Config {
	if cfg == nil {
		return Load()
	}
	return cfg
}

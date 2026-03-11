package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 应用程序配置结构
// 包含服务器端口、数据库路径和其他运行时选项
type Config struct {
	Port            int       `yaml:"port"`              // 服务器监听端口
	DBConfig        *DBConfig `yaml:"db_list"`           // 数据库文件路径
	SwaggerEnabled  bool      `yaml:"swagger_enabled"`   // 是否启用Swagger文档
	AutoOpenBrowser bool      `yaml:"auto_open_browser"` // 是否自动打开浏览器
}

type DBConfig struct {
	DefaultDBName string            `yaml:"default_db_name"` // 默认数据库名称
	ConnectList   []DBConnectConfig `yaml:"connect_list"`    // 数据库连接配置列表
}

type DBConnectConfig struct {
	Name                   string `yaml:"name"`
	DBType                 string `yaml:"db_type"`
	DBPath                 string `yaml:"db_path"`
	MaxOpenConns           *int   `yaml:"max_open_conns"`
	MaxIdleConns           *int   `yaml:"max_idle_conns"`
	ConnMaxLifetimeMinutes *int   `yaml:"conn_max_lifetime"`
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
		Port: 8080,
		DBConfig: &DBConfig{
			DefaultDBName: "test",
			ConnectList: []DBConnectConfig{
				{Name: "test", DBType: "sqlite", DBPath: "test.db"},
			},
		},
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
	if cfg.DBConfig.ConnectList == nil || len(cfg.DBConfig.ConnectList) == 0 {
		cfg.DBConfig.ConnectList = []DBConnectConfig{
			{Name: "test", DBType: "sqlite", DBPath: "test.db"},
		}
	}
	// ensure DBList timestamp suffix to avoid lock in tests? keep as is
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

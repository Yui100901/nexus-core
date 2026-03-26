package global

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 应用程序配置结构
// 包含服务器端口、数据库路径和其他运行时选项
type Config struct {
	Port            int       `yaml:"port"`              // 服务器监听端口
	DBConfig        *DBConfig `yaml:"db_list"`           // 数据库配置
	SwaggerEnabled  bool      `yaml:"swagger_enabled"`   // 是否启用Swagger文档
	AutoOpenBrowser bool      `yaml:"auto_open_browser"` // 是否自动打开浏览器
	// SwaggerURL 是用于打印/打开的 Swagger UI 地址或路径。可配置为完整 URL（以 http:// 或 https:// 开头），
	// 或以 "/" 开头的相对路径（会自动加上 http://localhost:PORT 前缀），或者包含端口占位符 %d（例如 http://localhost:%d/swagger/index.html）
	SwaggerURL string `yaml:"swagger_url"`
	// SwaggerDocURL 是 swagger UI 用来加载 swagger JSON 的 URL（默认为 /swagger/doc.json）
	SwaggerDocURL string `yaml:"swagger_doc_url"`
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

// LoadConfig 加载应用程序配置
// 优先从config-dev.yml文件加载，如果文件不存在则使用默认值
func LoadConfig() *Config {
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
		// sensible defaults
		SwaggerURL:    "/swagger/index.html",
		SwaggerDocURL: "/swagger/doc.json",
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

// GetConfig 获取配置实例
// 如果尚未加载配置，则先加载默认配置
func GetConfig() *Config {
	if cfg == nil {
		return LoadConfig()
	}
	return cfg
}

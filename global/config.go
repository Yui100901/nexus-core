package global

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port            int           `yaml:"port"`
	DBConfig        *DBConfig     `yaml:"db_list"`
	SwaggerEnabled  bool          `yaml:"swagger_enabled"`
	AutoOpenBrowser bool          `yaml:"auto_open_browser"`
	SwaggerURL      string        `yaml:"swagger_url"`
	SwaggerDocURL   string        `yaml:"swagger_doc_url"`
	MQTT            MQTTConfig    `yaml:"mqtt"`
	Control         ControlConfig `yaml:"control"`
}

type DBConfig struct {
	DefaultDBName string            `yaml:"default_db_name"`
	ConnectList   []DBConnectConfig `yaml:"connect_list"`
}

type DBConnectConfig struct {
	Name                   string `yaml:"name"`
	DBType                 string `yaml:"db_type"`
	DBPath                 string `yaml:"db_path"`
	MaxOpenConns           *int   `yaml:"max_open_conns"`
	MaxIdleConns           *int   `yaml:"max_idle_conns"`
	ConnMaxLifetimeMinutes *int   `yaml:"conn_max_lifetime"`
}

type MQTTConfig struct {
	BrokerURL             string `yaml:"broker_url"`
	ClientID              string `yaml:"client_id"`
	Username              string `yaml:"username"`
	Password              string `yaml:"password"`
	PublishTimeoutSeconds int    `yaml:"publish_timeout_seconds"`
}

type ControlConfig struct {
	DispatchTimeoutSeconds int `yaml:"dispatch_timeout_seconds"`
	DispatchMaxRetries     int `yaml:"dispatch_max_retries"`
	NodeOnlineTTLSeconds   int `yaml:"node_online_ttl_seconds"`
}

var cfg *Config

func LoadConfig() *Config {
	if cfg != nil {
		return cfg
	}

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
		SwaggerURL:      "/swagger/index.html",
		SwaggerDocURL:   "/swagger/doc.json",
		MQTT: MQTTConfig{
			PublishTimeoutSeconds: 5,
		},
		Control: ControlConfig{
			DispatchTimeoutSeconds: 5,
			DispatchMaxRetries:     0,
			NodeOnlineTTLSeconds:   120,
		},
	}

	f, err := os.ReadFile("config-dev.yml")
	if err != nil {
		return cfg
	}
	_ = yaml.Unmarshal(f, cfg)

	if cfg.Port == 0 {
		cfg.Port = 8080
	}
	if cfg.DBConfig == nil {
		cfg.DBConfig = &DBConfig{}
	}
	if cfg.DBConfig.ConnectList == nil || len(cfg.DBConfig.ConnectList) == 0 {
		cfg.DBConfig.ConnectList = []DBConnectConfig{
			{Name: "test", DBType: "sqlite", DBPath: "test.db"},
		}
	}
	if cfg.MQTT.PublishTimeoutSeconds <= 0 {
		cfg.MQTT.PublishTimeoutSeconds = 5
	}
	if cfg.Control.DispatchTimeoutSeconds <= 0 {
		cfg.Control.DispatchTimeoutSeconds = 5
	}
	if cfg.Control.NodeOnlineTTLSeconds <= 0 {
		cfg.Control.NodeOnlineTTLSeconds = 120
	}

	return cfg
}

func GetConfig() *Config {
	if cfg == nil {
		return LoadConfig()
	}
	return cfg
}

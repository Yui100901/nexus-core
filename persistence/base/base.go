package base

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"nexus-core/config"
	"nexus-core/persistence/model"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

//
// @Author yfy2001
// @Date 2025/7/21 15 26
//

var defaultDBName = config.Get().DBConfig.DefaultDBName

var DefaultDBManager = NewDBManager(defaultDBName)

type DBManager struct {
	mu            sync.Mutex
	dbInstanceMap map[string]*gorm.DB
	defaultName   string
}

func NewDBManager(defaultName string) *DBManager {
	return &DBManager{
		dbInstanceMap: make(map[string]*gorm.DB),
		defaultName:   defaultName,
	}
}

// GetDB 获取指定名称的数据库实例，如果不存在则 panic
func (m *DBManager) GetDB(name string) *gorm.DB {
	m.mu.Lock()
	defer m.mu.Unlock()

	db, ok := m.dbInstanceMap[name]
	if !ok {
		panic(fmt.Sprintf("db instance %s not initialized", name))
	}

	return db
}

// GetDefaultDB 获取默认数据库实例
func (m *DBManager) GetDefaultDB() *gorm.DB {
	return m.GetDB(m.defaultName)
}

func (m *DBManager) Init(cfgs []config.DBConnectConfig) {
	for _, cfg := range cfgs {
		if err := m.InitDB(cfg.Name, cfg.DBType, cfg.DBPath); err != nil {
			panic(fmt.Sprintf("failed to initialize database: %v", err))
		}
	}
}

func (m *DBManager) InitDB(name, dbType, dsn string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 如果已经存在，阻止重复初始化
	if _, ok := m.dbInstanceMap[name]; ok {
		return fmt.Errorf("db instance %s already initialized", name)
	}

	var (
		db  *gorm.DB
		err error
	)

	switch dbType {
	case "sqlite":
		db, err = InitDatabaseSqlite(dsn)
	case "mysql":
		db, err = InitDatabaseMysql(dsn)
	default:
		return fmt.Errorf("unsupported database type: %s", dbType)
	}

	if err != nil {
		return err
	}

	m.dbInstanceMap[name] = db
	return nil
}

func AutoMigrate(db *gorm.DB) {
	// 自动迁移模型，确保表存在
	if err := db.AutoMigrate(
		&model.License{},
		&model.LicenseScope{},
		&model.Product{},
		&model.ProductVersion{},
		&model.Node{},
		&model.NodeLicenseBinding{},
	); err != nil {
		panic(fmt.Sprintf("failed to automigrate database: %v", err))
	}
}

func InitDatabaseSqlite(dsn string) (*gorm.DB, error) {
	// if dsn is a file path, ensure parent directories exist
	if dsn != ":memory:" {
		dir := filepath.Dir(dsn)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create directories for db path %s: %w", dir, err)
			}
		}
	}

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Println("Failed to connect Sqlite!")
		return nil, err
	}
	fmt.Println("Connected to Sqlite!")
	return db, err
}

func InitDatabaseMysql(dsn string) (*gorm.DB, error) {
	// 尝试连接 MySQL
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Println("Failed to connect MySQL!")
		return nil, err
	}
	fmt.Println("Connected to MySQL!")
	return db, err
}

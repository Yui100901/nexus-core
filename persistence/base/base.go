package base

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

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

var DefaultDBName = config.Get().DBConfig.DefaultDBName

var MainDBManager *DBManager

type DBManager struct {
	mu            sync.RWMutex
	dbInstanceMap map[string]*gorm.DB
}

func InitDBManager(cfg *config.DBConfig) *DBManager {
	m := &DBManager{
		dbInstanceMap: make(map[string]*gorm.DB),
	}
	for _, connectConfig := range cfg.ConnectList {
		if err := m.InitDB(connectConfig); err != nil {
			panic(fmt.Sprintf("failed to initialize database: %v", err))
		}
	}
	return m
}

// GetDB 获取指定名称的数据库实例，如果不存在则 panic
func (m *DBManager) GetDB(name string) *gorm.DB {
	m.mu.RLock()
	defer m.mu.RUnlock()

	db, ok := m.dbInstanceMap[name]
	if !ok {
		panic(fmt.Sprintf("db instance %s not initialized", name))
	}

	return db
}

// GetDefaultDB 获取默认数据库实例
func (m *DBManager) GetDefaultDB() *gorm.DB {
	return m.GetDB(DefaultDBName)
}

func (m *DBManager) InitDB(cfg config.DBConnectConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 如果已经存在，阻止重复初始化
	if _, ok := m.dbInstanceMap[cfg.Name]; ok {
		return fmt.Errorf("db instance %s already initialized", cfg.Name)
	}

	var (
		db  *gorm.DB
		err error
	)

	switch cfg.DBType {
	case "sqlite":
		db, err = InitDatabaseSqlite(cfg)
	case "mysql":
		db, err = InitDatabaseMysql(cfg)
	default:
		return fmt.Errorf("unsupported database type: %s", cfg.DBType)
	}
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("error to get sqlDB: %s", err)
	}
	err = ConfigureSQLDB(sqlDB, cfg.MaxOpenConns, cfg.MaxIdleConns, cfg.ConnMaxLifetimeMinutes)
	if err != nil {
		return fmt.Errorf("error to configure db: %s", err)
	}

	m.dbInstanceMap[cfg.Name] = db
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

func ConfigureSQLDB(sqlDB *sql.DB, maxOpenConns, maxIdleConns, connMaxLifetimeMinutes *int) error {
	if maxOpenConns != nil {
		sqlDB.SetMaxOpenConns(*maxOpenConns)
	}
	if maxIdleConns != nil {
		sqlDB.SetMaxIdleConns(*maxIdleConns)
	}
	if connMaxLifetimeMinutes != nil {
		sqlDB.SetConnMaxLifetime(time.Duration(*connMaxLifetimeMinutes) * time.Minute)
	}
	// optional ping to validate connection
	return sqlDB.Ping()
}

func InitDatabaseSqlite(cfg config.DBConnectConfig) (*gorm.DB, error) {
	// if dsn is a file path, ensure parent directories exist
	if cfg.DBPath != ":memory:" {
		dir := filepath.Dir(cfg.DBPath)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create directories for db path %s: %w", dir, err)
			}
		}
	}

	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	if err != nil {
		fmt.Println("Failed to connect Sqlite!")
		return nil, err
	}
	fmt.Println("Connected to Sqlite!")
	return db, err
}

func InitDatabaseMysql(cfg config.DBConnectConfig) (*gorm.DB, error) {
	// 尝试连接 MySQL
	db, err := gorm.Open(mysql.Open(cfg.DBPath), &gorm.Config{})
	if err != nil {
		fmt.Println("Failed to connect MySQL!")
		return nil, err
	}
	fmt.Println("Connected to MySQL!")
	return db, err
}

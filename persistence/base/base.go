package base

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"nexus-core/global"
	"nexus-core/persistence/model"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

//
// @Author yfy2001
// @Date 2025/7/21 15 26
//

type DBManager struct {
	defaultDBName string
	dbMap         map[string]*gorm.DB
}

var MainDBManager *DBManager

func InitDBManager(cfg *global.DBConfig) *DBManager {
	if cfg == nil {
		panic("database config is nil")
	}
	if len(cfg.ConnectList) == 0 {
		panic("database connect list is empty")
	}

	manager := &DBManager{
		defaultDBName: cfg.DefaultDBName,
		dbMap:         make(map[string]*gorm.DB, len(cfg.ConnectList)),
	}

	for _, connectConfig := range cfg.ConnectList {
		db, err := InitDB(connectConfig)
		if err != nil {
			panic(fmt.Sprintf("failed to initialize database %s: %v", connectConfig.Name, err))
		}
		manager.dbMap[connectConfig.Name] = db
		if manager.defaultDBName == "" {
			manager.defaultDBName = connectConfig.Name
		}
	}

	if _, ok := manager.dbMap[manager.defaultDBName]; !ok {
		panic(fmt.Sprintf("default database %s not found", manager.defaultDBName))
	}

	return manager
}

func (m *DBManager) GetDefaultDB() *gorm.DB {
	if m == nil {
		return nil
	}
	return m.dbMap[m.defaultDBName]
}

func InitDB(cfg global.DBConnectConfig) (*gorm.DB, error) {

	var (
		db  *gorm.DB
		err error
	)

	db, err = InitDatabase(cfg)
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("error to get sqlDB: %s", err)
	}
	err = ConfigureSQLDB(sqlDB, cfg.MaxOpenConns, cfg.MaxIdleConns, cfg.ConnMaxLifetimeMinutes)
	if err != nil {
		return nil, fmt.Errorf("error to configure db: %s", err)
	}

	return db, nil
}

func AutoMigrate(db *gorm.DB) {
	// 自动迁移模型，确保表存在
	if err := db.AutoMigrate(
		&model.License{},
		&model.LicenseProductScope{},
		&model.LicenseServiceScope{},
		&model.Product{},
		&model.ProductVersion{},
		&model.Node{},
		&model.NodeLicenseBinding{},
		&model.CommonFeature{},
		&model.ControlService{},
		&model.NodeServiceCapability{},
		&model.ControlCommand{},
		&model.ControlCommandLog{},
		&model.AuditLog{},
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

func InitDatabase(cfg global.DBConnectConfig) (*gorm.DB, error) {
	var (
		db  *gorm.DB
		err error
	)
	switch cfg.DBType {
	case "sqlite":
		db, err = InitDatabaseSqlite(cfg)
	case "mysql":
		db, err = InitDatabaseMysql(cfg)
	case "postgres":
		db, err = InitDatabasePostgres(cfg)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.DBType)
	}
	return db, err
}

func InitDatabaseSqlite(cfg global.DBConnectConfig) (*gorm.DB, error) {
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

func InitDatabaseMysql(cfg global.DBConnectConfig) (*gorm.DB, error) {
	// 尝试连接 MySQL
	db, err := gorm.Open(mysql.Open(cfg.DBPath), &gorm.Config{})
	if err != nil {
		fmt.Println("Failed to connect MySQL!")
		return nil, err
	}
	fmt.Println("Connected to MySQL!")
	return db, err
}

func InitDatabasePostgres(cfg global.DBConnectConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DBPath), &gorm.Config{})
	if err != nil {
		fmt.Println("Failed to connect Postgres!")
		return nil, err
	}
	fmt.Println("Connected to Postgres!")
	return db, err
}

package base

import (
	"fmt"
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

var (
	dbInstance *gorm.DB
	once       sync.Once
)

func Connect() *gorm.DB {
	once.Do(func() {
		cfg := config.Get()
		db, err := InitDatabaseSqlite(cfg.DBPath)
		if err != nil {
			panic(fmt.Sprintf("failed to initialize database: %v", err))
		}
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
		dbInstance = db
	})
	return dbInstance
}

func InitDatabaseSqlite(dsn string) (*gorm.DB, error) {
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

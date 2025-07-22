package base

import (
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"sync"
)

//
// @Author yfy2001
// @Date 2025/7/21 15 26
//

type Model interface {
	TableName() string
}

var (
	dbInstance *gorm.DB
	once       sync.Once
)

func Connect() *gorm.DB {
	once.Do(func() {
		db, err := InitDatabaseSqlite("test.db")
		if err != nil {
			panic(fmt.Sprintf("failed to initialize database: %v", err))
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

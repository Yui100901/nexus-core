package base

import (
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

//
// @Author yfy2001
// @Date 2025/7/21 15 26
//

type Model interface {
	TableName() string
}

func InitDatabase() (*gorm.DB, error) {
	// 尝试连接 MySQL
	var db *gorm.DB
	var err error
	//db, err = gorm.Open(mysql.Open(config.Config.Mysql.ToDSN()), &gorm.Config{})
	//if err == nil {
	//	fmt.Println("Connected to MySQL!")
	//	return db, nil
	//}
	// 如果 MySQL 连接失败，尝试连接 SQLite
	fmt.Println("Failed to connect to MySQL, switching to SQLite...")
	db, err = gorm.Open(sqlite.Open("test"), &gorm.Config{})
	if err == nil {
		fmt.Println("Connected to SQLite!")
		return db, nil
	}

	// 如果两者都失败，返回错误
	return nil, err
}

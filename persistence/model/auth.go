package model

import "time"

//
// @Author yfy2001
// @Date 2025/7/21 15 15
//

//认证表

type Auth struct {
	DeviceId      string    `gorm:"primaryKey;type:varchar(100)"`
	CreatedAt     time.Time //创建时间
	ActivatedAt   time.Time //激活时间
	ValidDuration int       //有效时长
	ExpiredAt     time.Time //过期时间
	Status        int       //0-未激活,1-已激活,2-已过期
}

func (Auth) TableName() string {
	return "auth"
}

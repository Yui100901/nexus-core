package entity

import "time"

//
// @Author yfy2001
// @Date 2025/7/22 10 29
//

type Device struct {
	ID          string //id
	Name        string //名称
	DeviceType  string //设备类型
	Model       string //设备型号
	Description string //备注
	Protocol    string //网络协议
	IP          string //设备ip
	Auth        *Auth  //设备认证
}

type AuthStatus int

const (
	Unactivated AuthStatus = iota // 0 - 未激活
	Activated                     // 1 - 已激活
	Expired                       // 2 - 已过期
)

func (s AuthStatus) Int() int {
	return int(s)
}

func (s AuthStatus) String() string {
	switch s {
	case Unactivated:
		return "未激活"
	case Activated:
		return "已激活"
	case Expired:
		return "已过期"
	default:
		return "Unknown"
	}
}

type Auth struct {
	CreatedAt     time.Time  //创建时间
	ActivatedAt   *time.Time //激活时间
	ValidDuration int        //有效时长
	ExpiredAt     *time.Time //过期时间
	Status        AuthStatus //0-未激活,1-已激活,2-已过期
}

package consts

import "time"

// Define common constants
const (
	ChannelTaoTu string = "taotu" // 默认韬图渠道
	IDNull       int64  = 0       // ID 为空

	TimeFormatString = "2006-01-02 15:04:05"
	DateFormatString = "2006-01-02"

	ResetPasswordTokenTime time.Duration = time.Hour * 24 * 3 // 重置密码过期时间3天
)

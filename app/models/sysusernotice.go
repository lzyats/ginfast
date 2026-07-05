package models

type SysUserNotice struct {
	BaseModel
	NoticeID uint      `gorm:"column:notice_id;not null;index:idx_notice_user,unique;comment:通知ID" json:"noticeId"`
	UserID   uint      `gorm:"column:user_id;not null;index:idx_notice_user,unique;index:idx_user_read;comment:用户ID" json:"userId"`
	IsRead   int8      `gorm:"column:is_read;not null;default:0;index:idx_user_read;comment:是否已读" json:"isRead"`
	ReadAt   *JSONTime `gorm:"column:read_at;comment:已读时间" json:"readAt"`
}

func (SysUserNotice) TableName() string {
	return "sys_user_notice"
}

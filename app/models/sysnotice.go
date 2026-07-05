package models

import (
	"context"
	"gin-fast/app/global/app"
	"time"
)

const (
	SysNoticeCategoryNotice  = "notice"
	SysNoticeCategoryMessage = "message"
	SysNoticeCategoryBacklog = "backlog"
)

type SysNotice struct {
	BaseModel
	Title          string     `gorm:"column:title;size:255;not null;comment:通知标题" json:"title"`
	Content        string     `gorm:"column:content;type:text;not null;comment:通知内容" json:"content"`
	Category       string     `gorm:"column:category;size:32;not null;default:notice;comment:通知分类" json:"category"`
	RecipientCount int        `gorm:"column:recipient_count;default:0;comment:接收人数" json:"recipientCount"`
	SentAt         *time.Time `gorm:"column:sent_at;comment:发送时间" json:"sentAt"`
	CreatedBy      uint       `gorm:"column:created_by;default:0;comment:创建人" json:"createdBy"`
}

func (SysNotice) TableName() string {
	return "sys_notice"
}

func NewSysNotice() *SysNotice {
	return &SysNotice{}
}

func (m *SysNotice) Create(c context.Context) error {
	return app.DB().WithContext(c).Create(m).Error
}

func (m *SysNotice) FindByID(c context.Context, id uint) error {
	return app.DB().WithContext(c).Where("id = ?", id).First(m).Error
}

type SysNoticeListItem struct {
	ID             uint       `json:"id"`
	Title          string     `json:"title"`
	Content        string     `json:"content"`
	Category       string     `json:"category"`
	RecipientCount int        `json:"recipientCount"`
	SentAt         *time.Time `json:"sentAt"`
	CreatedAt      time.Time  `json:"createdAt"`
	CreatedBy      uint       `json:"createdBy"`
	CreatedByName  string     `json:"createdByName"`
}

type SysMyNoticeItem struct {
	ID         uint       `json:"id"`
	Title      string     `json:"title"`
	Content    string     `json:"content"`
	Category   string     `json:"category"`
	SentAt     *time.Time `json:"sentAt"`
	IsRead     int8       `json:"isRead"`
	ReadAt     *time.Time `json:"readAt"`
	SenderName string     `json:"senderName"`
}

func IsValidSysNoticeCategory(category string) bool {
	switch category {
	case SysNoticeCategoryNotice, SysNoticeCategoryMessage, SysNoticeCategoryBacklog:
		return true
	default:
		return false
	}
}

func NormalizeSysNoticeCategory(category string) string {
	if IsValidSysNoticeCategory(category) {
		return category
	}
	return SysNoticeCategoryNotice
}

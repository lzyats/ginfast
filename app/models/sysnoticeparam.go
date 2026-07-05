package models

import (
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SysNoticeSendRequest struct {
	Validator
	Title    string `json:"title" form:"title" validate:"required" message:"通知标题不能为空"`
	Content  string `json:"content" form:"content" validate:"required" message:"通知内容不能为空"`
	Category string `json:"category" form:"category" validate:"required|in:notice,message,backlog" message:"通知分类不正确"`
	UserIDs  []uint `json:"userIds" form:"userIds"`
}

func (r *SysNoticeSendRequest) Validate(c *gin.Context) error {
	return r.Check(c, r)
}

type SysNoticeListRequest struct {
	BasePaging
	Validator
	Title    string `json:"title" form:"title"`
	Category string `json:"category" form:"category"`
}

func (r *SysNoticeListRequest) Validate(c *gin.Context) error {
	return r.Check(c, r)
}

func (r *SysNoticeListRequest) Handler() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if strings.TrimSpace(r.Title) != "" {
			db = db.Where("n.title LIKE ?", "%"+strings.TrimSpace(r.Title)+"%")
		}
		if strings.TrimSpace(r.Category) != "" {
			db = db.Where("n.category = ?", strings.TrimSpace(r.Category))
		}
		return db
	}
}

type SysMyNoticeListRequest struct {
	BasePaging
	Validator
	Category string `json:"category" form:"category"`
	IsRead   *int8  `json:"isRead" form:"isRead"`
}

func (r *SysMyNoticeListRequest) Validate(c *gin.Context) error {
	return r.Check(c, r)
}

func (r *SysMyNoticeListRequest) Handler() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if strings.TrimSpace(r.Category) != "" {
			db = db.Where("n.category = ?", strings.TrimSpace(r.Category))
		}
		if r.IsRead != nil {
			db = db.Where("un.is_read = ?", *r.IsRead)
		}
		return db
	}
}

type SysNoticeReadRequest struct {
	Validator
	NoticeIDs []uint `json:"noticeIds" form:"noticeIds"`
	All       bool   `json:"all" form:"all"`
}

func (r *SysNoticeReadRequest) Validate(c *gin.Context) error {
	return r.Check(c, r)
}

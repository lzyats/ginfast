package service

import (
	"context"
	"errors"
	"gin-fast/app/global/app"
	"gin-fast/app/models"
	"gin-fast/app/utils/common"
	"gin-fast/app/utils/wshelper"
	"strings"
	"time"

	"gorm.io/gorm"
)

type SysNoticeService struct{}

func NewSysNoticeService() *SysNoticeService {
	return &SysNoticeService{}
}

func (s *SysNoticeService) AutoMigrate() error {
	return app.DB().AutoMigrate(&models.SysNotice{}, &models.SysUserNotice{})
}

func (s *SysNoticeService) Send(c context.Context, req *models.SysNoticeSendRequest) (*models.SysNotice, error) {
	now := time.Now()
	notice := &models.SysNotice{
		Title:    strings.TrimSpace(req.Title),
		Content:  strings.TrimSpace(req.Content),
		Category: models.NormalizeSysNoticeCategory(strings.TrimSpace(req.Category)),
		SentAt:   &now,
	}

	var userIDs []uint
	userQuery := app.DB().WithContext(c).Model(&models.User{}).Where("deleted_at IS NULL AND status = ?", 1)
	if len(req.UserIDs) > 0 {
		userQuery = userQuery.Where("id IN ?", req.UserIDs)
	}
	if err := userQuery.Pluck("id", &userIDs).Error; err != nil {
		return nil, err
	}
	if len(userIDs) == 0 {
		if len(req.UserIDs) > 0 {
			return nil, errors.New("未找到可接收通知的目标用户")
		}
		return nil, errors.New("没有可接收通知的用户")
	}
	notice.RecipientCount = len(userIDs)

	if err := app.DB().WithContext(c).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(notice).Error; err != nil {
			return err
		}

		recipients := make([]models.SysUserNotice, 0, len(userIDs))
		for _, userID := range userIDs {
			recipients = append(recipients, models.SysUserNotice{
				NoticeID: notice.ID,
				UserID:   userID,
				IsRead:   0,
			})
		}
		return tx.Create(&recipients).Error
	}); err != nil {
		return nil, err
	}

	wshelper.DefaultManager().BroadcastNotice(userIDs, wshelper.NoticePayload{
		ID:         notice.ID,
		Title:      notice.Title,
		Content:    notice.Content,
		Category:   notice.Category,
		SentAt:     notice.SentAt,
		IsRead:     0,
		SenderName: s.getSenderName(c),
	})

	return notice, nil
}

func (s *SysNoticeService) List(c context.Context, req *models.SysNoticeListRequest) ([]models.SysNoticeListItem, int64, error) {
	var total int64
	countQuery := app.DB().WithContext(c).Table("sys_notice n").Joins("LEFT JOIN sys_users u ON u.id = n.created_by").Scopes(req.Handler())
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	list := make([]models.SysNoticeListItem, 0)
	query := app.DB().WithContext(c).
		Table("sys_notice n").
		Select("n.id, n.title, n.content, n.category, n.recipient_count, n.sent_at, n.created_at, n.created_by, COALESCE(NULLIF(u.nick_name, ''), u.username) AS created_by_name").
		Joins("LEFT JOIN sys_users u ON u.id = n.created_by").
		Scopes(req.Handler()).
		Order("n.id DESC")

	if req.PageNum > 0 && req.PageSize > 0 {
		query = query.Offset((req.PageNum - 1) * req.PageSize).Limit(req.PageSize)
	}

	if err := query.Scan(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (s *SysNoticeService) GetByID(c context.Context, id uint) (*models.SysNoticeListItem, error) {
	item := &models.SysNoticeListItem{}
	err := app.DB().WithContext(c).
		Table("sys_notice n").
		Select("n.id, n.title, n.content, n.category, n.recipient_count, n.sent_at, n.created_at, n.created_by, COALESCE(NULLIF(u.nick_name, ''), u.username) AS created_by_name").
		Joins("LEFT JOIN sys_users u ON u.id = n.created_by").
		Where("n.id = ?", id).
		Scan(item).Error
	if err != nil {
		return nil, err
	}
	if item.ID == 0 {
		return nil, errors.New("通知不存在")
	}
	return item, nil
}

func (s *SysNoticeService) ListMine(c context.Context, userID uint, req *models.SysMyNoticeListRequest) ([]models.SysMyNoticeItem, int64, error) {
	var total int64
	countQuery := app.DB().WithContext(c).
		Table("sys_user_notice un").
		Joins("INNER JOIN sys_notice n ON n.id = un.notice_id").
		Where("un.user_id = ?", userID).
		Scopes(req.Handler())
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	list := make([]models.SysMyNoticeItem, 0)
	query := app.DB().WithContext(c).
		Table("sys_user_notice un").
		Select("n.id, n.title, n.content, n.category, n.sent_at, un.is_read, un.read_at, COALESCE(NULLIF(u.nick_name, ''), u.username) AS sender_name").
		Joins("INNER JOIN sys_notice n ON n.id = un.notice_id").
		Joins("LEFT JOIN sys_users u ON u.id = n.created_by").
		Where("un.user_id = ?", userID).
		Scopes(req.Handler()).
		Order("n.id DESC")

	if req.PageNum > 0 && req.PageSize > 0 {
		query = query.Offset((req.PageNum - 1) * req.PageSize).Limit(req.PageSize)
	}

	if err := query.Scan(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (s *SysNoticeService) MarkRead(c context.Context, userID uint, req *models.SysNoticeReadRequest) error {
	query := app.DB().WithContext(c).Model(&models.SysUserNotice{}).Where("user_id = ? AND is_read = 0", userID)
	if !req.All {
		if len(req.NoticeIDs) == 0 {
			return errors.New("请选择要标记的通知")
		}
		query = query.Where("notice_id IN ?", req.NoticeIDs)
	}

	return query.Updates(map[string]any{
		"is_read": 1,
		"read_at": models.JSONTime{Time: time.Now()},
	}).Error
}

func (s *SysNoticeService) GetUnreadCount(c context.Context, userID uint) (int64, error) {
	var total int64
	err := app.DB().WithContext(c).Model(&models.SysUserNotice{}).Where("user_id = ? AND is_read = 0", userID).Count(&total).Error
	return total, err
}

func (s *SysNoticeService) AuthenticateWsToken(token string) (*app.Claims, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("token不能为空")
	}
	return app.TokenService.ValidateTokenWithCache(token)
}

func (s *SysNoticeService) getSenderName(c context.Context) string {
	ginCtx := common.TryConvertToGinContext(c)
	claims := common.GetClaims(ginCtx)
	if claims == nil {
		return "系统"
	}

	user := models.NewUser()
	if err := app.DB().WithContext(c).Select("username, nick_name").Where("id = ?", claims.UserID).First(user).Error; err == nil {
		if strings.TrimSpace(user.NickName) != "" {
			return user.NickName
		}
		if strings.TrimSpace(user.Username) != "" {
			return user.Username
		}
	}

	if strings.TrimSpace(claims.Username) != "" {
		return claims.Username
	}
	return "系统"
}

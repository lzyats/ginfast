package controllers

import (
	"errors"
	"gin-fast/app/models"
	"gin-fast/app/service"
	"gin-fast/app/utils/common"
	"gin-fast/app/utils/wshelper"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/websocket"
)

type SysNoticeController struct {
	Common
	service *service.SysNoticeService
}

func NewSysNoticeController() *SysNoticeController {
	return &SysNoticeController{
		Common:  Common{},
		service: service.NewSysNoticeService(),
	}
}

func (c *SysNoticeController) List(ctx *gin.Context) {
	var req models.SysNoticeListRequest
	if err := req.Validate(ctx); err != nil {
		c.FailAndAbort(ctx, err.Error(), err)
		return
	}

	list, total, err := c.service.List(ctx, &req)
	if err != nil {
		c.FailAndAbort(ctx, "获取通知列表失败", err)
		return
	}

	c.Success(ctx, gin.H{"list": list, "total": total})
}

func (c *SysNoticeController) Send(ctx *gin.Context) {
	var req models.SysNoticeSendRequest
	if err := req.Validate(ctx); err != nil {
		c.FailAndAbort(ctx, err.Error(), err)
		return
	}

	notice, err := c.service.Send(ctx, &req)
	if err != nil {
		c.FailAndAbort(ctx, "发送通知失败", err)
		return
	}

	c.SuccessWithMessage(ctx, "通知发送成功", notice)
}

func (c *SysNoticeController) GetByID(ctx *gin.Context) {
	var req struct {
		models.Validator
		ID uint `uri:"id" validate:"required" message:"通知ID不能为空"`
	}
	if err := req.Check(ctx, &req); err != nil {
		c.FailAndAbort(ctx, err.Error(), err)
		return
	}

	item, err := c.service.GetByID(ctx, req.ID)
	if err != nil {
		c.FailAndAbort(ctx, "获取通知详情失败", err)
		return
	}
	c.Success(ctx, item)
}

func (c *SysNoticeController) ListMine(ctx *gin.Context) {
	var req models.SysMyNoticeListRequest
	if err := req.Validate(ctx); err != nil {
		c.FailAndAbort(ctx, err.Error(), err)
		return
	}

	userID := common.GetCurrentUserID(ctx)
	if userID == 0 {
		c.FailAndAbort(ctx, "用户未登录", errors.New("user not logged in"))
		return
	}

	list, total, err := c.service.ListMine(ctx, userID, &req)
	if err != nil {
		c.FailAndAbort(ctx, "获取我的通知失败", err)
		return
	}

	c.Success(ctx, gin.H{"list": list, "total": total})
}

func (c *SysNoticeController) MarkRead(ctx *gin.Context) {
	var req models.SysNoticeReadRequest
	if err := req.Validate(ctx); err != nil {
		c.FailAndAbort(ctx, err.Error(), err)
		return
	}

	userID := common.GetCurrentUserID(ctx)
	if userID == 0 {
		c.FailAndAbort(ctx, "用户未登录", errors.New("user not logged in"))
		return
	}

	if err := c.service.MarkRead(ctx, userID, &req); err != nil {
		c.FailAndAbort(ctx, "标记已读失败", err)
		return
	}

	c.SuccessWithMessage(ctx, "操作成功", nil)
}

func (c *SysNoticeController) UnreadCount(ctx *gin.Context) {
	userID := common.GetCurrentUserID(ctx)
	if userID == 0 {
		c.FailAndAbort(ctx, "用户未登录", errors.New("user not logged in"))
		return
	}

	total, err := c.service.GetUnreadCount(ctx, userID)
	if err != nil {
		c.FailAndAbort(ctx, "获取未读数量失败", err)
		return
	}

	c.Success(ctx, gin.H{"unreadCount": total})
}

func (c *SysNoticeController) Connect(ctx *gin.Context) {
	token, err := common.GetAccessToken(ctx)
	if err != nil {
		c.FailAndAbort(ctx, "token无效", err)
		return
	}

	claims, err := c.service.AuthenticateWsToken(token)
	if err != nil {
		c.FailAndAbort(ctx, "认证失败", err)
		return
	}

	handler := websocket.Handler(func(conn *websocket.Conn) {
		manager := wshelper.DefaultManager()
		manager.Register(claims.UserID, conn)
		defer manager.Unregister(claims.UserID, conn)

		manager.SendConnected(claims.UserID)

		for {
			var incoming map[string]any
			if err := websocket.JSON.Receive(conn, &incoming); err != nil {
				break
			}
		}
	})

	handler.ServeHTTP(ctx.Writer, ctx.Request)
}

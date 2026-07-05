package middleware

import (
	"gin-fast/app/global/app"
	"gin-fast/app/models"
	"gin-fast/app/utils/common"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AdminOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := common.GetClaims(c)
		if claims == nil || claims.UserID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "用户未登录"})
			c.Abort()
			return
		}

		if strings.EqualFold(strings.TrimSpace(claims.Username), "admin") {
			c.Next()
			return
		}

		user := models.NewUser()
		err := app.DB().WithContext(c).Preload("Roles").Where("id = ?", claims.UserID).First(user).Error
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"message": "无权限访问"})
			c.Abort()
			return
		}

		for _, role := range user.Roles {
			roleName := strings.TrimSpace(role.Name)
			if strings.EqualFold(roleName, "admin") || strings.Contains(roleName, "管理员") {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"message": "仅管理员可操作"})
		c.Abort()
	}
}

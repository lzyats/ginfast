package middleware

import (
	"bytes"
	"encoding/json"
	"gin-fast/app/global/app"
	"gin-fast/app/models"
	"gin-fast/app/utils/common"
	"gin-fast/app/utils/ip2regionhelper"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// OperationLogMiddleware 鎿嶄綔鏃ュ織涓棿浠?
func OperationLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 璺宠繃涓嶉渶瑕佽褰曟棩蹇楃殑璇锋眰
		if shouldSkipLog(c) {
			c.Next()
			return
		}

		startTime := time.Now()

		// 澶嶅埗璇锋眰浣撶敤浜庤褰?
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// 鍒涘缓鑷畾涔夌殑ResponseWriter鏉ユ崟鑾峰搷搴?
		writer := &responseWriter{body: bytes.NewBuffer(nil), ResponseWriter: c.Writer}
		c.Writer = writer

		defer func() {
			// 璁板綍鎿嶄綔鏃ュ織
			go recordOperationLog(c, startTime, requestBody, writer.body.Bytes())
		}()

		c.Next()
	}
}

// responseWriter 鑷畾涔塕esponseWriter鐢ㄤ簬鎹曡幏鍝嶅簲鏁版嵁
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// shouldSkipLog 鍒ゆ柇鏄惁闇€瑕佽烦杩囨棩蹇楄褰?
func shouldSkipLog(c *gin.Context) bool {
	// 璺宠繃闈欐€佹枃浠躲€佸仴搴锋鏌ョ瓑璇锋眰
	skipPaths := []string{
		"/swagger/",
		"/favicon.ico",
		"/health",
		"/metrics",
		"/api/refreshToken",  // 鍒锋柊token
		"/api/captcha/id",    // 鐢熸垚楠岃瘉鐮両D
		"/api/captcha/image", // 鑾峰彇楠岃瘉鐮佸浘鐗?
		"/api/config/get",    // 鑾峰彇閰嶇疆淇℃伅
	}

	path := c.Request.URL.Path
	for _, skipPath := range skipPaths {
		if strings.Contains(path, skipPath) {
			return true
		}
	}

	return false
}

// recordOperationLog 璁板綍鎿嶄綔鏃ュ織
func recordOperationLog(c *gin.Context, startTime time.Time, requestBody, responseBody []byte) {
	duration := time.Since(startTime).Milliseconds()

	// 鑾峰彇鐢ㄦ埛淇℃伅
	var userID uint
	var username string
	var tenantID uint
	operationType := getOperationType(c)

	// 灏濊瘯浠嶫WT token鑾峰彇鐢ㄦ埛淇℃伅
	claims := common.GetClaims(c)
	if claims != nil {
		userID = claims.UserID
		username = claims.Username
		tenantID = claims.TenantID
	} else {
		// 濡傛灉鏄櫥褰曟搷浣滐紝灏濊瘯浠庤姹備綋涓幏鍙栫敤鎴峰悕
		if c.Request.URL.Path == "/api/login" && c.Request.Method == "POST" {
			// 瑙ｆ瀽鐧诲綍璇锋眰浣撹幏鍙栫敤鎴峰悕
			var loginReq struct {
				Username string `json:"username"`
			}
			if len(requestBody) > 0 {
				if err := json.Unmarshal(requestBody, &loginReq); err == nil && loginReq.Username != "" {
					username = loginReq.Username
					// 鏍囪涓虹櫥褰曟搷浣?
					operationType = models.OperationLogin
				}
			}
		}
	}

	// 鏋勫缓鎿嶄綔鏃ュ織
	log := &models.SysOperationLog{
		UserID:      userID,
		Username:    username,
		Module:      getOperationModule(c),
		Operation:   operationType,
		Method:      c.Request.Method,
		Path:        c.Request.URL.Path,
		IP:          c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
		RequestData: sanitizeRequestData(requestBody),
		//ResponseData: sanitizeResponseData(responseBody),
		StatusCode: c.Writer.Status(),
		Duration:   duration,
		ErrorMsg:   getErrorMessage(c, responseBody),
		Location:   getLocationByIP(c.ClientIP()),
		TenantID:   tenantID,
	}

	// 寮傛淇濆瓨鏃ュ織
	go func() {
		if err := app.DB().Create(log).Error; err != nil {
			app.ZapLog.Error("璁板綍鎿嶄綔鏃ュ織澶辫触", zap.Error(err))
		}
	}()
}

// getOperationModule 鑾峰彇鎿嶄綔妯″潡
func getOperationModule(c *gin.Context) string {
	path := c.Request.URL.Path
	if strings.Contains(path, "/users") {
		return "鐢ㄦ埛绠＄悊"
	} else if strings.Contains(path, "/sysMenu") {
		return "鑿滃崟绠＄悊"
	} else if strings.Contains(path, "/sysRole") {
		return "瑙掕壊绠＄悊"
	} else if strings.Contains(path, "/sysDepartment") {
		return "閮ㄩ棬绠＄悊"
	} else if strings.Contains(path, "/sysDict") {
		return "瀛楀吀绠＄悊"
	} else if strings.Contains(path, "/sysApi") {
		return "API绠＄悊"
	} else if strings.Contains(path, "/sysAffix") {
		return "鏂囦欢绠＄悊"
	} else if strings.Contains(path, "/config") {
		return "绯荤粺閰嶇疆"
	} else if strings.Contains(path, "/sysOperationLog") {
		return "鎿嶄綔鏃ュ織绠＄悊"
	}
	return "鍏朵粬"
}

// getOperationType 鑾峰彇鎿嶄綔绫诲瀷
func getOperationType(c *gin.Context) string {
	method := c.Request.Method
	switch method {
	case "POST":
		return models.OperationCreate
	case "PUT", "PATCH":
		return models.OperationUpdate
	case "DELETE":
		return models.OperationDelete
	case "GET":
		return models.OperationQuery
	default:
		return "unknown"
	}
}

// getErrorMessage 鑾峰彇閿欒淇℃伅
func getErrorMessage(c *gin.Context, responseBody []byte) string {
	if c.Writer.Status() >= 400 {
		// 棣栧厛灏濊瘯浠庝笂涓嬫枃涓幏鍙栭敊璇俊鎭?
		if err, exists := c.Get("error"); exists {
			return err.(error).Error()
		}
		// 濡傛灉涓婁笅鏂囦腑娌℃湁閿欒淇℃伅锛屽皾璇曡В鏋愬搷搴斾綋
		if len(responseBody) > 0 {
			// 灏濊瘯瑙ｆ瀽JSON鍝嶅簲浣?
			var response map[string]interface{}
			if err := json.Unmarshal(responseBody, &response); err == nil {
				// 鏍规嵁椤圭洰涓殑鍝嶅簲鏍煎紡鑾峰彇閿欒淇℃伅锛堜娇鐢╩essage瀛楁锛?
				if msg, ok := response["message"].(string); ok && msg != "" {
					return msg
				}
			}
		}
		return "璇锋眰澶勭悊澶辫触"
	}
	return ""
}

// getLocationByIP 鏍规嵁IP鑾峰彇鍦扮悊浣嶇疆锛堢畝鍖栧疄鐜帮級
func getLocationByIP(ip string) string {
	return ip2regionhelper.Lookup(ip)
}

// sanitizeRequestData 瀵硅姹傛暟鎹繘琛岃劚鏁忓鐞?
func sanitizeRequestData(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	// 濡傛灉鏄疛SON鏁版嵁锛屽皾璇曡劚鏁忔晱鎰熷瓧娈?
	if json.Valid(data) {
		var jsonData map[string]interface{}
		if err := json.Unmarshal(data, &jsonData); err == nil {
			// 鑴辨晱瀵嗙爜瀛楁
			if _, exists := jsonData["password"]; exists {
				jsonData["password"] = "***"
			}
			if _, exists := jsonData["Password"]; exists {
				jsonData["Password"] = "***"
			}
			if _, exists := jsonData["newPassword"]; exists {
				jsonData["newPassword"] = "***"
			}

			// 閲嶆柊搴忓垪鍖?
			if sanitized, err := json.Marshal(jsonData); err == nil {
				return string(sanitized)
			}
		}
	}

	// 濡傛灉涓嶆槸JSON锛岀洿鎺ヨ繑鍥炲師濮嬫暟鎹紙闄愬埗闀垮害锛?
	if len(data) > 10000 {
		return string(data[:10000]) + "...(truncated)"
	}
	return string(data)
}

// sanitizeResponseData 瀵瑰搷搴旀暟鎹繘琛岃劚鏁忓鐞?
func sanitizeResponseData(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	// 闄愬埗鍝嶅簲鏁版嵁闀垮害
	if len(data) > 5000 {
		return string(data[:5000]) + "...(truncated)"
	}
	return string(data)
}

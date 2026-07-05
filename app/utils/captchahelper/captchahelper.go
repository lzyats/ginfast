package captchahelper

import (
	"gin-fast/app/global/app"
	"sync"

	"github.com/mojocn/base64Captcha"
)

type CaptchaHelper struct {
	driver *base64Captcha.Captcha
	store  base64Captcha.Store
}

var (
	captchaHelperInstance *CaptchaHelper
	captchaHelperOnce     sync.Once
)

// GetCaptchaHelper 鑾峰彇CaptchaHelper鍗曚緥
func GetCaptchaHelper() *CaptchaHelper {
	captchaHelperOnce.Do(func() {
		captchaHelperInstance = NewCaptchaHelper()
	})
	return captchaHelperInstance
}

// NewCaptchaHelper 鍒涘缓CaptchaHelper瀹炰緥
func NewCaptchaHelper() *CaptchaHelper {
	store := base64Captcha.DefaultMemStore
	length := app.ConfigYml.GetInt("captcha.length")
	if length <= 0 {
		length = 4
	}

	ds := &base64Captcha.DriverDigit{
		Height:   42,
		Width:    length*44 + 44,
		Length:   length,
		MaxSkew:  0.4,
		DotCount: 10,
	}
	return &CaptchaHelper{
		driver: base64Captcha.NewCaptcha(ds, store),
		store:  store,
	}
}

// GetVerifyImgString 鑾峰彇楠岃瘉鐮佸浘鐗囧瓧绗︿覆
func (ch *CaptchaHelper) GetVerifyImgString() (idKeyC string, base64stringC string, err error) {
	idKeyC, base64stringC, _, err = ch.driver.Generate()
	return
}

// VerifyVerifyImgString 楠岃瘉楠岃瘉鐮佸浘鐗囧瓧绗︿覆
func (ch *CaptchaHelper) VerifyVerifyImgString(idKeyC string, verifyValueC string) bool {
	return ch.store.Verify(idKeyC, verifyValueC, true)
}

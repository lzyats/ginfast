package models

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SysParamAddRequest struct {
	Validator
	Name        string `form:"name" json:"name" validate:"required" message:"鍙傛暟鍚嶇О涓嶈兘涓虹┖"`
	Code        string `form:"code" json:"code" validate:"required" message:"鍙傛暟鍞竴鏍囪瘑涓嶈兘涓虹┖"`
	Value       string `form:"value" json:"value"`
	ParamType   string `form:"paramType" json:"paramType" validate:"required|in:text,number,select,upload" message:"鍙傛暟绫诲瀷涓嶈兘涓虹┖|鍙傛暟绫诲瀷涓嶆纭?`
	Options     string `form:"options" json:"options"`
	Status      int8   `form:"status" json:"status" validate:"required|in:0,1" message:"鐘舵€佸€煎繀椤讳负0鎴?"`
	Description string `form:"description" json:"description"`
}

func (r *SysParamAddRequest) Validate(c *gin.Context) error {
	return r.Check(c, r)
}

type SysParamUpdateRequest struct {
	Validator
	ID          uint   `form:"id" json:"id" validate:"required" message:"鍙傛暟ID涓嶈兘涓虹┖"`
	Name        string `form:"name" json:"name" validate:"required" message:"鍙傛暟鍚嶇О涓嶈兘涓虹┖"`
	Code        string `form:"code" json:"code" validate:"required" message:"鍙傛暟鍞竴鏍囪瘑涓嶈兘涓虹┖"`
	Value       string `form:"value" json:"value"`
	ParamType   string `form:"paramType" json:"paramType" validate:"required|in:text,number,select,upload" message:"鍙傛暟绫诲瀷涓嶈兘涓虹┖|鍙傛暟绫诲瀷涓嶆纭?`
	Options     string `form:"options" json:"options"`
	Status      int8   `form:"status" json:"status" validate:"required|in:0,1" message:"鐘舵€佸€煎繀椤讳负0鎴?"`
	Description string `form:"description" json:"description"`
}

func (r *SysParamUpdateRequest) Validate(c *gin.Context) error {
	return r.Check(c, r)
}

type SysParamDeleteRequest struct {
	Validator
	ID uint `form:"id" json:"id" validate:"required" message:"鍙傛暟ID涓嶈兘涓虹┖"`
}

func (r *SysParamDeleteRequest) Validate(c *gin.Context) error {
	return r.Check(c, r)
}

type SysParamListRequest struct {
	BasePaging
	Validator
	Name      string `form:"name"`
	Code      string `form:"code"`
	ParamType string `form:"paramType"`
	Status    *int8  `form:"status"`
}

func (r *SysParamListRequest) Validate(c *gin.Context) error {
	return r.Check(c, r)
}

func (r *SysParamListRequest) Handler() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if r.Name != "" {
			db = db.Where("name LIKE ?", "%"+r.Name+"%")
		}
		if r.Code != "" {
			db = db.Where("code LIKE ?", "%"+r.Code+"%")
		}
		if r.ParamType != "" {
			db = db.Where("param_type = ?", r.ParamType)
		}
		if r.Status != nil {
			db = db.Where("status = ?", r.Status)
		}
		return db
	}
}

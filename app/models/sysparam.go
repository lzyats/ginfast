package models

import (
	"context"
	"gin-fast/app/global/app"

	"gorm.io/gorm"
)

const (
	SysParamTypeText   = "text"
	SysParamTypeNumber = "number"
	SysParamTypeSelect = "select"
	SysParamTypeUpload = "upload"
)

type SysParam struct {
	BaseModel
	Name        *string `gorm:"column:name;size:255;comment:鍙傛暟鍚嶇О" json:"name"`
	Code        *string `gorm:"column:code;size:255;uniqueIndex;comment:鍙傛暟鍞竴鏍囪瘑" json:"code"`
	Value       *string `gorm:"column:value;type:text;comment:鍙傛暟鍊? json:"value"`
	ParamType   *string `gorm:"column:param_type;size:20;default:text;comment:鍙傛暟绫诲瀷(text/number/select/upload)" json:"paramType"`
	Options     *string `gorm:"column:options;type:text;comment:涓嬫媺閫夐」JSON" json:"options"`
	Status      *int8   `gorm:"column:status;comment:鐘舵€?0绂佺敤/1鍚敤)" json:"status"`
	Description *string `gorm:"column:description;size:500;comment:鎻忚堪" json:"description"`
	CreatedBy   *uint   `gorm:"column:created_by;comment:鍒涘缓浜? json:"createdBy"`
}

func (SysParam) TableName() string {
	return "sys_param"
}

func NewSysParam() *SysParam {
	return &SysParam{}
}

type SysParamList []*SysParam

func NewSysParamList() SysParamList {
	return SysParamList{}
}

func (list SysParamList) IsEmpty() bool {
	return len(list) == 0
}

func (list *SysParamList) Find(c context.Context, funcs ...func(*gorm.DB) *gorm.DB) (err error) {
	err = app.DB().WithContext(c).Scopes(funcs...).Find(list).Error
	return
}

func (s *SysParam) IsEmpty() bool {
	return s == nil || s.ID == 0
}

func (s *SysParam) Create(c context.Context) (err error) {
	err = app.DB().WithContext(c).Create(s).Error
	return
}

func (s *SysParam) Update(c context.Context) (err error) {
	err = app.DB().WithContext(c).Save(s).Error
	return
}

func (s *SysParam) Delete(c context.Context) (err error) {
	err = app.DB().WithContext(c).Delete(s).Error
	return
}

func (s *SysParam) FindByID(c context.Context, id uint) (err error) {
	err = app.DB().WithContext(c).Where("id = ?", id).First(s).Error
	return
}

func (s *SysParam) FindByCode(c context.Context, code string) (err error) {
	err = app.DB().WithContext(c).Where("code = ?", code).First(s).Error
	return
}

func FindByCodePrefix(c context.Context, prefix string) (SysParamList, error) {
	list := NewSysParamList()
	err := app.DB().WithContext(c).Where("code LIKE ?", prefix+"%").Find(&list).Error
	return list, err
}

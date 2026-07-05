package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gin-fast/app/global/app"
	"gin-fast/app/models"
	"strconv"
	"strings"
)

type SysParamService struct{}

type SysParamOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

func NewSysParamService() *SysParamService {
	return &SysParamService{}
}

func (s *SysParamService) AutoMigrate() error {
	return app.DB().AutoMigrate(&models.SysParam{})
}

func (s *SysParamService) Add(c context.Context, req *models.SysParamAddRequest) (*models.SysParam, error) {
	existParam := models.NewSysParam()
	if err := existParam.FindByCode(c, req.Code); err == nil && !existParam.IsEmpty() {
		return nil, errors.New("param code already exists")
	}

	paramType, value, options, err := s.normalizePayload(req.ParamType, req.Value, req.Options)
	if err != nil {
		return nil, err
	}

	param := models.NewSysParam()
	param.Name = stringPtr(strings.TrimSpace(req.Name))
	param.Code = stringPtr(strings.TrimSpace(req.Code))
	param.Value = stringPtr(value)
	param.ParamType = stringPtr(paramType)
	param.Options = stringPtr(options)
	param.Status = &req.Status
	param.Description = stringPtr(strings.TrimSpace(req.Description))

	if err := param.Create(c); err != nil {
		return nil, err
	}
	return param, nil
}

func (s *SysParamService) Update(c context.Context, req *models.SysParamUpdateRequest) (*models.SysParam, error) {
	param := models.NewSysParam()
	if err := param.FindByID(c, req.ID); err != nil {
		return nil, errors.New("param not found")
	}

	existParam := models.NewSysParam()
	if err := app.DB().WithContext(c).Where("code = ? AND id != ?", req.Code, req.ID).First(existParam).Error; err == nil && !existParam.IsEmpty() {
		return nil, errors.New("param code is used by another record")
	}

	paramType, value, options, err := s.normalizePayload(req.ParamType, req.Value, req.Options)
	if err != nil {
		return nil, err
	}

	param.Name = stringPtr(strings.TrimSpace(req.Name))
	param.Code = stringPtr(strings.TrimSpace(req.Code))
	param.Value = stringPtr(value)
	param.ParamType = stringPtr(paramType)
	param.Options = stringPtr(options)
	param.Status = &req.Status
	param.Description = stringPtr(strings.TrimSpace(req.Description))

	if err := param.Update(c); err != nil {
		return nil, err
	}
	return param, nil
}

func (s *SysParamService) Delete(c context.Context, id uint) error {
	param := models.NewSysParam()
	if err := param.FindByID(c, id); err != nil {
		return errors.New("param not found")
	}
	return param.Delete(c)
}

func (s *SysParamService) GetByID(c context.Context, id uint) (*models.SysParam, error) {
	param := models.NewSysParam()
	if err := param.FindByID(c, id); err != nil {
		return nil, errors.New("param not found")
	}
	s.fillDefaults(param)
	return param, nil
}

func (s *SysParamService) GetByCode(c context.Context, code string) (*models.SysParam, error) {
	param := models.NewSysParam()
	if err := param.FindByCode(c, code); err != nil {
		return nil, errors.New("param not found")
	}
	s.fillDefaults(param)
	return param, nil
}

func (s *SysParamService) List(c context.Context, req *models.SysParamListRequest) ([]*models.SysParam, int64, error) {
	var count int64
	if err := app.DB().WithContext(c).Model(&models.SysParam{}).Scopes(req.Handler()).Count(&count).Error; err != nil {
		return nil, 0, err
	}

	list := models.NewSysParamList()
	if err := list.Find(c, req.Paginate(), req.Handler()); err != nil {
		return nil, 0, err
	}
	for _, item := range list {
		s.fillDefaults(item)
	}
	return list, count, nil
}

func (s *SysParamService) ListByCodePrefix(c context.Context, prefix string) ([]*models.SysParam, error) {
	list, err := models.FindByCodePrefix(c, prefix)
	if err != nil {
		return nil, err
	}
	for _, item := range list {
		s.fillDefaults(item)
	}
	return list, nil
}

func (s *SysParamService) fillDefaults(param *models.SysParam) {
	if param == nil {
		return
	}
	if param.ParamType == nil || strings.TrimSpace(*param.ParamType) == "" {
		param.ParamType = stringPtr(models.SysParamTypeText)
	}
	if param.Options == nil {
		param.Options = stringPtr("")
	}
	if param.Value == nil {
		param.Value = stringPtr("")
	}
}

func (s *SysParamService) normalizePayload(paramType, value, options string) (string, string, string, error) {
	normalizedType := strings.TrimSpace(paramType)
	if normalizedType == "" {
		normalizedType = models.SysParamTypeText
	}

	normalizedValue := strings.TrimSpace(value)
	normalizedOptions := strings.TrimSpace(options)

	switch normalizedType {
	case models.SysParamTypeText, models.SysParamTypeUpload:
		return normalizedType, normalizedValue, "", nil
	case models.SysParamTypeNumber:
		if normalizedValue != "" {
			if _, err := strconv.ParseFloat(normalizedValue, 64); err != nil {
				return "", "", "", errors.New("number param value must be numeric")
			}
		}
		return normalizedType, normalizedValue, "", nil
	case models.SysParamTypeSelect:
		parsedOptions, err := parseSysParamOptions(normalizedOptions)
		if err != nil {
			return "", "", "", err
		}
		if len(parsedOptions) == 0 {
			return "", "", "", errors.New("select param must contain at least one option")
		}
		if duplicateValue := findDuplicateOptionValue(parsedOptions); duplicateValue != "" {
			return "", "", "", fmt.Errorf("select option value is duplicated: %s", duplicateValue)
		}
		if normalizedValue != "" {
			matched := false
			for _, item := range parsedOptions {
				if item.Value == normalizedValue {
					matched = true
					break
				}
			}
			if !matched {
				return "", "", "", errors.New("param value must exist in select options")
			}
		}
		optionsBytes, _ := json.Marshal(parsedOptions)
		return normalizedType, normalizedValue, string(optionsBytes), nil
	default:
		return "", "", "", fmt.Errorf("unsupported param type: %s", normalizedType)
	}
}

func parseSysParamOptions(raw string) ([]SysParamOption, error) {
	if strings.TrimSpace(raw) == "" {
		return []SysParamOption{}, nil
	}

	var options []SysParamOption
	if err := json.Unmarshal([]byte(raw), &options); err == nil {
		return sanitizeSysParamOptions(options), nil
	}

	lines := strings.Split(raw, "\n")
	options = make([]SysParamOption, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		parts := strings.SplitN(trimmed, "|", 2)
		label := strings.TrimSpace(parts[0])
		value := label
		if len(parts) == 2 {
			value = strings.TrimSpace(parts[1])
		}
		options = append(options, SysParamOption{Label: label, Value: value})
	}
	options = sanitizeSysParamOptions(options)
	if len(options) == 0 {
		return nil, errors.New("invalid select options")
	}
	return options, nil
}

func sanitizeSysParamOptions(options []SysParamOption) []SysParamOption {
	result := make([]SysParamOption, 0, len(options))
	for _, item := range options {
		label := strings.TrimSpace(item.Label)
		value := strings.TrimSpace(item.Value)
		if label == "" && value == "" {
			continue
		}
		if label == "" {
			label = value
		}
		if value == "" {
			value = label
		}
		result = append(result, SysParamOption{Label: label, Value: value})
	}
	return result
}

func findDuplicateOptionValue(options []SysParamOption) string {
	seen := make(map[string]struct{}, len(options))
	for _, item := range options {
		if _, ok := seen[item.Value]; ok {
			return item.Value
		}
		seen[item.Value] = struct{}{}
	}
	return ""
}

func stringPtr(v string) *string {
	value := v
	return &value
}

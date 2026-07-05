package uploadhelper

import (
	"fmt"
	"gin-fast/app/global/app"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/gin-gonic/gin"
)

type OSSUploadService struct {
	config app.UploadConfig
	bucket *oss.Bucket
	domain string
}

func NewOSSUploadService() (app.FileUploadService, error) {
	config := GetUploadConfig()
	client, err := oss.New(config.OSSConfig.Endpoint, config.OSSConfig.AccessKeyID, config.OSSConfig.AccessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("初始化 OSS 客户端失败: %v", err)
	}
	bucket, err := client.Bucket(config.OSSConfig.Bucket)
	if err != nil {
		return nil, fmt.Errorf("获取 OSS Bucket 失败: %v", err)
	}
	return &OSSUploadService{config: config, bucket: bucket, domain: config.OSSConfig.Domain}, nil
}

func (s *OSSUploadService) HandleUpload(c *gin.Context, fileName string) (*app.UploadResponse, error) {
	file, err := c.FormFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("获取文件失败: %v", err)
	}
	if valid, err := s.ValidateFile(file); !valid {
		return nil, err
	}
	return s.UploadFile(file)
}

func (s *OSSUploadService) UploadFile(file *multipart.FileHeader) (*app.UploadResponse, error) {
	fileName := s.GenerateFileName(file.Filename)
	key := buildDatedObjectKey(s.config.OSSConfig.BasePath, fileName)
	return s.uploadMultipart(file, key)
}

func (s *OSSUploadService) UploadFileWithCustomPath(file *multipart.FileHeader, customPath string) (string, error) {
	fileName := s.GenerateFileName(file.Filename)
	key := buildObjectKey(s.config.OSSConfig.BasePath, customPath, fileName)
	resp, err := s.uploadMultipart(file, key)
	if err != nil {
		return "", err
	}
	return resp.Url, nil
}

func (s *OSSUploadService) UploadLocalFile(localFilePath string, objectKey string) (*app.UploadResponse, error) {
	key := buildObjectKey(s.config.OSSConfig.BasePath, objectKey)
	src, err := os.Open(localFilePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %v", err)
	}
	defer src.Close()
	info, err := src.Stat()
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %v", err)
	}
	if err := s.bucket.PutObject(key, src); err != nil {
		return nil, fmt.Errorf("上传文件失败: %v", err)
	}
	return &app.UploadResponse{Url: s.GetFileUrl(key), Path: key, FileName: filepath.Base(key), Size: info.Size(), FileType: s.GetFileExtension(key)}, nil
}

func (s *OSSUploadService) DeleteFile(fileRef string) error {
	key := s.getFileKey(fileRef)
	if key == "" {
		return fmt.Errorf("无效的文件标识: %s", fileRef)
	}
	if err := s.bucket.DeleteObject(key); err != nil {
		return fmt.Errorf("删除文件失败: %v", err)
	}
	return nil
}

func (s *OSSUploadService) GetFileUrl(fileName string) string {
	domain := normalizeDomainURL(s.domain)
	if domain != "" {
		return fmt.Sprintf("%s/%s", domain, strings.TrimLeft(fileName, "/"))
	}
	endpoint := strings.TrimRight(s.config.OSSConfig.Endpoint, "/")
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}
	bucket := strings.TrimSpace(s.config.OSSConfig.Bucket)
	if bucket != "" {
		endpoint = strings.Replace(endpoint, "://", "://"+bucket+".", 1)
	}
	return fmt.Sprintf("%s/%s", endpoint, strings.TrimLeft(fileName, "/"))
}

func (s *OSSUploadService) GetUploadConfig() app.UploadConfig { return s.config }
func (s *OSSUploadService) GetFileExtension(fileName string) string {
	return getFileExtension(fileName)
}
func (s *OSSUploadService) GenerateFileName(originalFileName string) string {
	return generateFileName(originalFileName)
}
func (s *OSSUploadService) SaveFile(file *multipart.FileHeader, filePath string) error {
	return errUnsupportedLocalSave("oss")
}
func (s *OSSUploadService) ValidateFile(file *multipart.FileHeader) (bool, error) {
	if err := validateFileByConfig(file, s.config.MaxSize, s.config.AllowedTypes); err != nil {
		return false, err
	}
	return true, nil
}

func (s *OSSUploadService) uploadMultipart(file *multipart.FileHeader, key string) (*app.UploadResponse, error) {
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %v", err)
	}
	defer src.Close()
	if err := s.bucket.PutObject(key, src); err != nil {
		return nil, fmt.Errorf("上传文件失败: %v", err)
	}
	return &app.UploadResponse{Url: s.GetFileUrl(key), Path: key, FileName: filepath.Base(key), Size: file.Size, FileType: s.GetFileExtension(file.Filename)}, nil
}

func (s *OSSUploadService) getFileKey(fileRef string) string {
	key := strings.TrimSpace(fileRef)
	domain := normalizeDomainURL(s.domain)
	if domain != "" && (strings.HasPrefix(key, "http://") || strings.HasPrefix(key, "https://")) {
		key = strings.TrimPrefix(key, domain)
	}
	return strings.TrimLeft(key, "/")
}

func (s *OSSUploadService) DownloadAndSaveRemoteImage(imageURL string) (*app.UploadResponse, error) {
	resp, ext, err := downloadRemoteImage(imageURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	fileName := s.GenerateFileName("image" + ext)
	key := buildDatedObjectKey(s.config.OSSConfig.BasePath, fileName)
	if err := s.bucket.PutObject(key, resp.Body); err != nil {
		return nil, fmt.Errorf("上传图片失败: %v", err)
	}
	return &app.UploadResponse{Url: s.GetFileUrl(key), Path: key, FileName: filepath.Base(key), Size: resp.ContentLength, FileType: ext}, nil
}

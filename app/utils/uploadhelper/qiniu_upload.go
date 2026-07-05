package uploadhelper

import (
	"context"
	"fmt"
	"gin-fast/app/global/app"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/storage"
)

type QiniuUploadService struct {
	config app.UploadConfig
	mac    *qbox.Mac
	cfg    *storage.Config
	bucket string
	domain string
}

func NewQiniuUploadService() app.FileUploadService {
	config := GetUploadConfig()
	mac := qbox.NewMac(config.QiniuConfig.AccessKey, config.QiniuConfig.SecretKey)
	cfg := &storage.Config{Zone: getZone(config.QiniuConfig.Zone), UseHTTPS: true, UseCdnDomains: true}
	return &QiniuUploadService{config: config, mac: mac, cfg: cfg, bucket: config.QiniuConfig.Bucket, domain: config.QiniuConfig.Domain}
}

func (s *QiniuUploadService) HandleUpload(c *gin.Context, fileName string) (*app.UploadResponse, error) {
	file, err := c.FormFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("获取文件失败: %v", err)
	}
	if valid, err := s.ValidateFile(file); !valid {
		return nil, err
	}
	return s.UploadFile(file)
}

func (s *QiniuUploadService) UploadFile(file *multipart.FileHeader) (*app.UploadResponse, error) {
	fileName := s.GenerateFileName(file.Filename)
	key := buildDatedObjectKey(s.config.QiniuConfig.BasePath, fileName)
	return s.uploadMultipart(file, key)
}

func (s *QiniuUploadService) UploadFileWithCustomPath(file *multipart.FileHeader, customPath string) (string, error) {
	fileName := s.GenerateFileName(file.Filename)
	key := buildObjectKey(s.config.QiniuConfig.BasePath, customPath, fileName)
	resp, err := s.uploadMultipart(file, key)
	if err != nil {
		return "", err
	}
	return resp.Url, nil
}

func (s *QiniuUploadService) UploadLocalFile(localFilePath string, objectKey string) (*app.UploadResponse, error) {
	key := buildObjectKey(s.config.QiniuConfig.BasePath, objectKey)
	src, err := os.Open(localFilePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %v", err)
	}
	defer src.Close()
	info, err := src.Stat()
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %v", err)
	}
	if err := s.putObject(src, info.Size(), key); err != nil {
		return nil, err
	}
	storedName := filepath.Base(key)
	return &app.UploadResponse{
		Url:          s.GetFileUrl(key),
		Path:         key,
		FileName:     storedName,
		OriginalName: storedName,
		StoredName:   storedName,
		Size:         info.Size(),
		FileType:     s.GetFileExtension(key),
	}, nil
}

func (s *QiniuUploadService) DeleteFile(fileRef string) error {
	key := s.getFileKey(fileRef)
	if key == "" {
		return fmt.Errorf("无效的文件标识: %s", fileRef)
	}
	return storage.NewBucketManager(s.mac, s.cfg).Delete(s.bucket, key)
}

func (s *QiniuUploadService) GetFileUrl(fileName string) string {
	domain := normalizeDomainURL(s.domain)
	if domain == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s", domain, strings.TrimLeft(fileName, "/"))
}

func (s *QiniuUploadService) GetUploadConfig() app.UploadConfig { return s.config }
func (s *QiniuUploadService) GetFileExtension(fileName string) string {
	return getFileExtension(fileName)
}
func (s *QiniuUploadService) GenerateFileName(originalFileName string) string {
	return generateFileName(originalFileName)
}
func (s *QiniuUploadService) SaveFile(file *multipart.FileHeader, filePath string) error {
	return errUnsupportedLocalSave("qiniu")
}

func (s *QiniuUploadService) ValidateFile(file *multipart.FileHeader) (bool, error) {
	if err := validateFileByConfig(file, s.config.MaxSize, s.config.AllowedTypes); err != nil {
		return false, err
	}
	return true, nil
}

func (s *QiniuUploadService) uploadMultipart(file *multipart.FileHeader, key string) (*app.UploadResponse, error) {
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %v", err)
	}
	defer src.Close()
	if err := s.putObject(src, file.Size, key); err != nil {
		return nil, err
	}
	return &app.UploadResponse{
		Url:          s.GetFileUrl(key),
		Path:         key,
		FileName:     file.Filename,
		OriginalName: file.Filename,
		StoredName:   filepath.Base(key),
		Size:         file.Size,
		FileType:     s.GetFileExtension(file.Filename),
	}, nil
}

func (s *QiniuUploadService) putObject(reader io.Reader, size int64, key string) error {
	formUploader := storage.NewFormUploader(s.cfg)
	putExtra := storage.PutExtra{}
	putPolicy := storage.PutPolicy{Scope: s.bucket}
	upToken := putPolicy.UploadToken(s.mac)
	ret := storage.PutRet{}
	if err := formUploader.Put(context.Background(), &ret, upToken, key, reader, size, &putExtra); err != nil {
		return fmt.Errorf("上传文件失败: %v", err)
	}
	return nil
}

func (s *QiniuUploadService) getFileKey(fileRef string) string {
	domain := normalizeDomainURL(s.domain)
	key := strings.TrimSpace(fileRef)
	if strings.HasPrefix(key, "http://") || strings.HasPrefix(key, "https://") {
		key = strings.TrimPrefix(key, domain)
	}
	return strings.TrimLeft(key, "/")
}

func getZone(zoneName string) *storage.Zone {
	switch zoneName {
	case "ZoneHuadong":
		return &storage.ZoneHuadong
	case "ZoneHuabei":
		return &storage.ZoneHuabei
	case "ZoneHuanan":
		return &storage.ZoneHuanan
	case "ZoneBeimei":
		return &storage.ZoneBeimei
	case "ZoneXinjiapo":
		return &storage.ZoneXinjiapo
	default:
		return &storage.ZoneHuadong
	}
}

func (s *QiniuUploadService) DownloadAndSaveRemoteImage(imageURL string) (*app.UploadResponse, error) {
	resp, ext, err := downloadRemoteImage(imageURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	fileName := s.GenerateFileName("image" + ext)
	key := buildDatedObjectKey(s.config.QiniuConfig.BasePath, fileName)
	if err := s.putObject(resp.Body, resp.ContentLength, key); err != nil {
		return nil, err
	}
	storedName := filepath.Base(key)
	return &app.UploadResponse{
		Url:          s.GetFileUrl(key),
		Path:         key,
		FileName:     storedName,
		OriginalName: storedName,
		StoredName:   storedName,
		Size:         resp.ContentLength,
		FileType:     ext,
	}, nil
}

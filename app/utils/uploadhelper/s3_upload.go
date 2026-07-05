package uploadhelper

import (
	"context"
	"fmt"
	"gin-fast/app/global/app"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awscred "github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

type S3UploadService struct {
	config       app.UploadConfig
	client       *awss3.Client
	bucket       string
	domain       string
	endpoint     string
	useSSL       bool
	usePathStyle bool
}

func NewS3UploadService() (app.FileUploadService, error) {
	config := GetUploadConfig()
	region := strings.TrimSpace(config.S3Config.Region)
	if region == "" {
		region = "us-east-1"
	}

	loadOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(awscred.NewStaticCredentialsProvider(
			config.S3Config.AccessKeyID,
			config.S3Config.SecretAccessKey,
			"",
		)),
	}

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(), loadOptions...)
	if err != nil {
		return nil, fmt.Errorf("初始化 S3 配置失败: %v", err)
	}

	endpoint := strings.TrimSpace(config.S3Config.Endpoint)
	usePathStyle := endpoint != ""

	client := awss3.NewFromConfig(cfg, func(o *awss3.Options) {
		o.UsePathStyle = usePathStyle
		if endpoint != "" {
			normalized := endpoint
			if !strings.HasPrefix(normalized, "http://") && !strings.HasPrefix(normalized, "https://") {
				scheme := "https://"
				if !config.S3Config.UseSSL {
					scheme = "http://"
				}
				normalized = scheme + normalized
			}
			o.BaseEndpoint = awsv2.String(strings.TrimRight(normalized, "/"))
		}
	})

	return &S3UploadService{
		config:       config,
		client:       client,
		bucket:       config.S3Config.Bucket,
		domain:       config.S3Config.Domain,
		endpoint:     endpoint,
		useSSL:       config.S3Config.UseSSL,
		usePathStyle: usePathStyle,
	}, nil
}

func (s *S3UploadService) HandleUpload(c *gin.Context, fileName string) (*app.UploadResponse, error) {
	file, err := c.FormFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("获取文件失败: %v", err)
	}
	if valid, err := s.ValidateFile(file); !valid {
		return nil, err
	}
	return s.UploadFile(file)
}

func (s *S3UploadService) UploadFile(file *multipart.FileHeader) (*app.UploadResponse, error) {
	fileName := s.GenerateFileName(file.Filename)
	key := buildDatedObjectKey(s.config.S3Config.BasePath, fileName)
	return s.uploadMultipart(file, key)
}

func (s *S3UploadService) UploadFileWithCustomPath(file *multipart.FileHeader, customPath string) (string, error) {
	fileName := s.GenerateFileName(file.Filename)
	key := buildObjectKey(s.config.S3Config.BasePath, customPath, fileName)
	resp, err := s.uploadMultipart(file, key)
	if err != nil {
		return "", err
	}
	return resp.Url, nil
}

func (s *S3UploadService) UploadLocalFile(localFilePath string, objectKey string) (*app.UploadResponse, error) {
	key := buildObjectKey(s.config.S3Config.BasePath, objectKey)
	src, err := os.Open(localFilePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %v", err)
	}
	defer src.Close()

	info, err := src.Stat()
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %v", err)
	}

	_, err = s.client.PutObject(context.Background(), &awss3.PutObjectInput{
		Bucket:      awsv2.String(s.bucket),
		Key:         awsv2.String(key),
		Body:        src,
		ContentType: awsv2.String(mimeTypeByExt(getFileExtension(key))),
	})
	if err != nil {
		return nil, fmt.Errorf("上传文件失败: %v", err)
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

func (s *S3UploadService) DeleteFile(fileRef string) error {
	key := s.getFileKey(fileRef)
	if key == "" {
		return fmt.Errorf("无效的文件标识: %s", fileRef)
	}

	_, err := s.client.DeleteObject(context.Background(), &awss3.DeleteObjectInput{
		Bucket: awsv2.String(s.bucket),
		Key:    awsv2.String(key),
	})
	if err != nil {
		return fmt.Errorf("删除文件失败: %v", err)
	}
	return nil
}

func (s *S3UploadService) GetFileUrl(fileName string) string {
	domain := normalizeDomainURL(s.domain)
	if domain != "" {
		return fmt.Sprintf("%s/%s", domain, strings.TrimLeft(fileName, "/"))
	}

	scheme := "http"
	if s.useSSL {
		scheme = "https"
	}

	endpoint := strings.TrimRight(s.endpoint, "/")
	if endpoint == "" {
		host := fmt.Sprintf("s3.%s.amazonaws.com", strings.TrimSpace(s.config.S3Config.Region))
		return fmt.Sprintf("https://%s/%s/%s", host, strings.TrimSpace(s.bucket), strings.TrimLeft(fileName, "/"))
	}
	endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://")
	return fmt.Sprintf("%s://%s/%s/%s", scheme, endpoint, strings.TrimSpace(s.bucket), strings.TrimLeft(fileName, "/"))
}

func (s *S3UploadService) GetUploadConfig() app.UploadConfig       { return s.config }
func (s *S3UploadService) GetFileExtension(fileName string) string { return getFileExtension(fileName) }
func (s *S3UploadService) GenerateFileName(originalFileName string) string {
	return generateFileName(originalFileName)
}
func (s *S3UploadService) SaveFile(file *multipart.FileHeader, filePath string) error {
	return errUnsupportedLocalSave("s3")
}
func (s *S3UploadService) ValidateFile(file *multipart.FileHeader) (bool, error) {
	if err := validateFileByConfig(file, s.config.MaxSize, s.config.AllowedTypes); err != nil {
		return false, err
	}
	return true, nil
}

func (s *S3UploadService) uploadMultipart(file *multipart.FileHeader, key string) (*app.UploadResponse, error) {
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %v", err)
	}
	defer src.Close()

	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mimeTypeByExt(getFileExtension(file.Filename))
	}

	_, err = s.client.PutObject(context.Background(), &awss3.PutObjectInput{
		Bucket:        awsv2.String(s.bucket),
		Key:           awsv2.String(key),
		Body:          src,
		ContentLength: awsv2.Int64(file.Size),
		ContentType:   awsv2.String(contentType),
	})
	if err != nil {
		return nil, fmt.Errorf("上传文件失败: %v", err)
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

func (s *S3UploadService) getFileKey(fileRef string) string {
	key := strings.TrimSpace(fileRef)
	domain := normalizeDomainURL(s.domain)
	if domain != "" && (strings.HasPrefix(key, "http://") || strings.HasPrefix(key, "https://")) {
		key = strings.TrimPrefix(key, domain)
	}
	if strings.HasPrefix(key, "http://") || strings.HasPrefix(key, "https://") {
		parts := strings.SplitN(strings.TrimPrefix(strings.TrimPrefix(key, "https://"), "http://"), "/", 3)
		if len(parts) >= 3 {
			key = parts[2]
		}
	}
	if strings.HasPrefix(key, s.bucket+"/") {
		key = strings.TrimPrefix(key, s.bucket+"/")
	}
	return strings.TrimLeft(key, "/")
}

func (s *S3UploadService) DownloadAndSaveRemoteImage(imageURL string) (*app.UploadResponse, error) {
	resp, ext, err := downloadRemoteImage(imageURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	fileName := s.GenerateFileName("image" + ext)
	key := buildDatedObjectKey(s.config.S3Config.BasePath, fileName)
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mimeTypeByExt(ext)
	}

	_, err = s.client.PutObject(context.Background(), &awss3.PutObjectInput{
		Bucket:        awsv2.String(s.bucket),
		Key:           awsv2.String(key),
		Body:          resp.Body,
		ContentLength: awsv2.Int64(resp.ContentLength),
		ContentType:   awsv2.String(contentType),
	})
	if err != nil {
		return nil, fmt.Errorf("上传图片失败: %v", err)
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

func mimeTypeByExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}

package app

import (
	"mime/multipart"

	"github.com/gin-gonic/gin"
)

type FileUploadService interface {
	UploadFile(file *multipart.FileHeader) (*UploadResponse, error)
	UploadFileWithCustomPath(file *multipart.FileHeader, customPath string) (string, error)
	DeleteFile(fileRef string) error
	GetFileUrl(fileName string) string
	GetUploadConfig() UploadConfig
	HandleUpload(c *gin.Context, fileName string) (*UploadResponse, error)
	ValidateFile(file *multipart.FileHeader) (bool, error)
	GetFileExtension(fileName string) string
	GenerateFileName(originalFileName string) string
	SaveFile(file *multipart.FileHeader, filePath string) error
	DownloadAndSaveRemoteImage(imageUrl string) (*UploadResponse, error)
	UploadLocalFile(localFilePath string, objectKey string) (*UploadResponse, error)
}

type UploadConfig struct {
	UploadType        string      `yaml:"upload_type"`
	MaxSize           int         `yaml:"max_size"`
	AllowedTypes      []string    `yaml:"allowed_types"`
	LocalPath         string      `yaml:"local_path"`
	QiniuConfig       QiniuConfig `yaml:"qiniu_config"`
	OSSConfig         OSSConfig   `yaml:"oss_config"`
	S3Config          S3Config    `yaml:"s3_config"`
	ChunkMaxSize      int         `yaml:"chunk_max_size"`
	MaxChunkSize      int         `yaml:"max_chunk_size"`
	ChunkAllowedTypes []string    `yaml:"chunk_allowed_types"`
}

type QiniuConfig struct {
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	Bucket    string `yaml:"bucket"`
	Domain    string `yaml:"domain"`
	Zone      string `yaml:"zone"`
	BasePath  string `yaml:"base_path"`
}

type OSSConfig struct {
	Endpoint        string `yaml:"endpoint"`
	AccessKeyID     string `yaml:"access_key_id"`
	AccessKeySecret string `yaml:"access_key_secret"`
	Bucket          string `yaml:"bucket"`
	Domain          string `yaml:"domain"`
	BasePath        string `yaml:"base_path"`
}

type S3Config struct {
	Endpoint        string `yaml:"endpoint"`
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
	Bucket          string `yaml:"bucket"`
	Region          string `yaml:"region"`
	UseSSL          bool   `yaml:"use_ssl"`
	Domain          string `yaml:"domain"`
	BasePath        string `yaml:"base_path"`
}

type UploadResponse struct {
	Url          string `json:"url"`
	FileName     string `json:"file_name"`
	OriginalName string `json:"original_name"`
	StoredName   string `json:"stored_name"`
	Size         int64  `json:"size"`
	FileType     string `json:"file_type"`
	Path         string `json:"path"`
}

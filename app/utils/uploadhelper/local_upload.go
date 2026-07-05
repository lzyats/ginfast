package uploadhelper

import (
	"fmt"
	"gin-fast/app/global/app"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type LocalUploadService struct {
	config app.UploadConfig
}

func NewLocalUploadService() app.FileUploadService {
	return &LocalUploadService{config: GetUploadConfig()}
}

func (s *LocalUploadService) UploadFile(file *multipart.FileHeader) (*app.UploadResponse, error) {
	fileName := s.GenerateFileName(file.Filename)
	objectKey := buildObjectKey(time.Now().Format("2006-01-02"), fileName)
	filePath := filepath.Join(s.config.LocalPath, filepath.FromSlash(objectKey))
	if err := s.SaveFile(file, filePath); err != nil {
		return nil, err
	}
	return &app.UploadResponse{Url: s.GetFileUrl(objectKey), Path: filePath, FileName: fileName, Size: file.Size, FileType: s.GetFileExtension(file.Filename)}, nil
}

func (s *LocalUploadService) UploadFileWithCustomPath(file *multipart.FileHeader, customPath string) (string, error) {
	fileName := s.GenerateFileName(file.Filename)
	objectKey := buildObjectKey(customPath, fileName)
	filePath := filepath.Join(s.config.LocalPath, filepath.FromSlash(objectKey))
	if err := s.SaveFile(file, filePath); err != nil {
		return "", err
	}
	return s.GetFileUrl(objectKey), nil
}

func (s *LocalUploadService) UploadLocalFile(localFilePath string, objectKey string) (*app.UploadResponse, error) {
	objectKey = buildObjectKey(objectKey)
	if objectKey == "" {
		objectKey = buildObjectKey(time.Now().Format("2006-01-02"), filepath.Base(localFilePath))
	}
	destPath := filepath.Join(s.config.LocalPath, filepath.FromSlash(objectKey))
	size, err := copyLocalFile(localFilePath, destPath)
	if err != nil {
		return nil, err
	}
	return &app.UploadResponse{Url: s.GetFileUrl(objectKey), Path: destPath, FileName: filepath.Base(destPath), Size: size, FileType: s.GetFileExtension(destPath)}, nil
}

func (s *LocalUploadService) DeleteFile(fileRef string) error {
	filePath := fileRef
	if _, err := os.Stat(filePath); err != nil {
		filePath = s.getFilePathFromURL(fileRef)
		if filePath == "" {
			return fmt.Errorf("无效的文件路径: %s", fileRef)
		}
	}
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("删除文件失败: %v", err)
	}
	return nil
}

func (s *LocalUploadService) GetFileUrl(fileName string) string {
	serverRootPath := strings.TrimSuffix(app.ConfigYml.GetString("httpserver.serverrootpath"), "/")
	url := fmt.Sprintf("%s/uploads/%s", serverRootPath, strings.TrimLeft(filepath.ToSlash(fileName), "/"))
	if !strings.HasPrefix(url, "/") {
		url = "/" + url
	}
	return url
}

func (s *LocalUploadService) GetUploadConfig() app.UploadConfig { return s.config }
func (s *LocalUploadService) GenerateFileName(originalFileName string) string {
	return generateFileName(originalFileName)
}
func (s *LocalUploadService) GetFileExtension(fileName string) string {
	return getFileExtension(fileName)
}

func (s *LocalUploadService) HandleUpload(c *gin.Context, fileName string) (*app.UploadResponse, error) {
	file, err := c.FormFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("获取文件失败: %v", err)
	}
	if valid, err := s.ValidateFile(file); !valid {
		return nil, err
	}
	return s.UploadFile(file)
}

func (s *LocalUploadService) ValidateFile(file *multipart.FileHeader) (bool, error) {
	if err := validateFileByConfig(file, s.config.MaxSize, s.config.AllowedTypes); err != nil {
		return false, err
	}
	return true, nil
}

func (s *LocalUploadService) SaveFile(file *multipart.FileHeader, filePath string) error {
	return saveMultipartFile(file, filePath)
}

func (s *LocalUploadService) getFilePathFromURL(relativePath string) string {
	re := regexp.MustCompile(`/uploads/(.+)$`)
	matches := re.FindStringSubmatch(filepath.ToSlash(relativePath))
	if len(matches) > 1 {
		fullPath := filepath.Join(s.config.LocalPath, filepath.FromSlash(matches[1]))
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath
		}
	}
	return ""
}

func (s *LocalUploadService) DownloadAndSaveRemoteImage(imageURL string) (*app.UploadResponse, error) {
	resp, ext, err := downloadRemoteImage(imageURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	fileName := s.GenerateFileName("image" + ext)
	objectKey := buildObjectKey(time.Now().Format("2006-01-02"), fileName)
	filePath := filepath.Join(s.config.LocalPath, filepath.FromSlash(objectKey))

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return nil, fmt.Errorf("创建目录失败: %v", err)
	}
	out, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("创建文件失败: %v", err)
	}
	defer out.Close()

	size, err := io.Copy(out, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("保存图片失败: %v", err)
	}

	return &app.UploadResponse{Url: s.GetFileUrl(objectKey), FileName: fileName, Size: size, FileType: ext, Path: filePath}, nil
}

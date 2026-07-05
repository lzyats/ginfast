package uploadhelper

import (
	"crypto/tls"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

func generateFileName(originalFileName string) string {
	ext := strings.ToLower(filepath.Ext(originalFileName))
	return fmt.Sprintf("%s_%s%s", time.Now().Format("20060102"), uuid.New().String(), ext)
}

func getFileExtension(fileName string) string {
	return strings.ToLower(filepath.Ext(fileName))
}

func validateFileByConfig(file *multipart.FileHeader, maxSizeMB int, allowedTypes []string) error {
	maxSize := int64(maxSizeMB * 1024 * 1024)
	if maxSize > 0 && file.Size > maxSize {
		return fmt.Errorf("文件大小超过限制，最大允许 %d MB", maxSizeMB)
	}

	ext := getFileExtension(file.Filename)
	if len(allowedTypes) > 0 {
		allowed := false
		for _, allowedType := range allowedTypes {
			if strings.EqualFold(ext, allowedType) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("文件类型不允许，允许的类型: %v", allowedTypes)
		}
	}

	return nil
}

func buildDatedObjectKey(basePath string, fileName string) string {
	return buildObjectKey(basePath, time.Now().Format("2006-01-02"), fileName)
}

func buildObjectKey(parts ...string) string {
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		cleaned = append(cleaned, strings.Trim(part, "/\\"))
	}
	return path.Join(cleaned...)
}

func normalizeDomainURL(domain string) string {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return ""
	}
	if !strings.HasPrefix(domain, "http://") && !strings.HasPrefix(domain, "https://") {
		domain = "https://" + domain
	}
	return strings.TrimRight(domain, "/")
}

func downloadRemoteImage(imageURL string) (*http.Response, string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Get(imageURL)
	if err != nil {
		return nil, "", fmt.Errorf("下载图片失败: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, "", fmt.Errorf("下载图片失败，状态码: %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		defer resp.Body.Close()
		return nil, "", fmt.Errorf("下载的文件不是图片类型: %s", contentType)
	}

	ext := ".jpg"
	switch {
	case strings.Contains(contentType, "png"):
		ext = ".png"
	case strings.Contains(contentType, "gif"):
		ext = ".gif"
	case strings.Contains(contentType, "webp"):
		ext = ".webp"
	case strings.Contains(contentType, "jpeg"):
		ext = ".jpg"
	}

	return resp, ext, nil
}

func saveMultipartFile(file *multipart.FileHeader, filePath string) error {
	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("打开文件失败: %v", err)
	}
	defer src.Close()

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	dst, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %v", err)
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return fmt.Errorf("保存文件失败: %v", err)
	}
	return nil
}

func copyLocalFile(srcPath string, dstPath string) (int64, error) {
	src, err := os.Open(srcPath)
	if err != nil {
		return 0, fmt.Errorf("打开文件失败: %v", err)
	}
	defer src.Close()

	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return 0, fmt.Errorf("创建目录失败: %v", err)
	}

	dst, err := os.Create(dstPath)
	if err != nil {
		return 0, fmt.Errorf("创建文件失败: %v", err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, src)
	if err != nil {
		return 0, fmt.Errorf("复制文件失败: %v", err)
	}
	return written, nil
}

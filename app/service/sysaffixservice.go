package service

import (
	"context"
	"fmt"
	"gin-fast/app/global/app"
	"gin-fast/app/global/consts"
	"gin-fast/app/models"
	"gin-fast/app/utils/filehelper"
	"gin-fast/app/utils/uploadhelper"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type SysAffixService struct{}

func NewSysAffixService() *SysAffixService { return &SysAffixService{} }

func (s *SysAffixService) ValidateChunkFile(fileName string, fileSize int64) error {
	uploadConfig := app.UploadService.GetUploadConfig()
	if len(uploadConfig.ChunkAllowedTypes) > 0 {
		ext := strings.ToLower(filepath.Ext(fileName))
		if ext == "" {
			return fmt.Errorf("无法识别文件类型")
		}
		allowed := false
		for _, allowedType := range uploadConfig.ChunkAllowedTypes {
			if strings.EqualFold(ext, allowedType) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("文件类型不允许，允许的类型: %v", uploadConfig.ChunkAllowedTypes)
		}
	}
	maxSize := int64(uploadConfig.ChunkMaxSize * 1024 * 1024)
	if maxSize > 0 && fileSize > maxSize {
		return fmt.Errorf("文件大小超过限制，最大允许 %d MB", uploadConfig.ChunkMaxSize)
	}
	return nil
}

func (s *SysAffixService) ValidateChunkSize(chunkSize int64) error {
	uploadConfig := app.UploadService.GetUploadConfig()
	maxChunkSize := int64(uploadConfig.MaxChunkSize * 1024 * 1024)
	if maxChunkSize > 0 && chunkSize > maxChunkSize {
		return fmt.Errorf("分片大小超过限制，最大允许 %d MB", uploadConfig.MaxChunkSize)
	}
	return nil
}

func (s *SysAffixService) InitChunkUpload(ctx context.Context, req *models.ChunkInitRequest, tenantID uint) (*models.ChunkInitResult, error) {
	if err := s.ValidateChunkFile(req.FileName, req.FileSize); err != nil {
		return nil, err
	}

	existAffix, _ := models.GetAffixByMd5(ctx, req.FileMd5, req.FileSize, tenantID)
	if existAffix != nil && existAffix.ID > 0 {
		return &models.ChunkInitResult{UploadId: "", UploadedChunks: []int{}, ExistFile: existAffix}, nil
	}

	uploadID := fmt.Sprintf("upload_%d_%s", time.Now().Unix(), uuid.New().String()[:8])
	uploadedChunks := []int{}
	existingChunks := models.NewSysAffixChunkList()
	if err := app.DB().WithContext(ctx).Where("file_md5 = ? AND status = 0 AND tenant_id = ?", req.FileMd5, tenantID).Find(existingChunks).Error; err == nil && len(*existingChunks) > 0 {
		uploadID = (*existingChunks)[0].UploadId
		for _, chunk := range *existingChunks {
			uploadedChunks = append(uploadedChunks, chunk.ChunkIndex)
		}
	}

	return &models.ChunkInitResult{UploadId: uploadID, UploadedChunks: uploadedChunks, ExistFile: nil}, nil
}

func (s *SysAffixService) SaveChunk(ctx context.Context, req *models.ChunkUploadRequest, userID, tenantID uint) error {
	if err := s.ValidateChunkSize(req.File.Size); err != nil {
		return err
	}

	localPath := app.UploadService.GetUploadConfig().LocalPath
	tmpDir := filepath.Join(localPath, "tmp", req.UploadId)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %v", err)
	}

	chunkPath := filepath.Join(tmpDir, fmt.Sprintf("chunk_%d", req.ChunkIndex))
	src, err := req.File.Open()
	if err != nil {
		return fmt.Errorf("打开分片文件失败: %v", err)
	}
	defer src.Close()

	dst, err := os.Create(chunkPath)
	if err != nil {
		return fmt.Errorf("创建分片文件失败: %v", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("保存分片文件失败: %v", err)
	}

	chunk := models.NewSysAffixChunk()
	chunk.UploadId = req.UploadId
	chunk.FileMd5 = req.FileMd5
	chunk.FileName = ""
	chunk.ChunkSize = int(req.File.Size)
	chunk.TotalChunks = req.TotalChunks
	chunk.ChunkIndex = req.ChunkIndex
	chunk.ChunkPath = chunkPath
	chunk.Status = 0
	chunk.CreatedBy = userID
	chunk.TenantID = tenantID

	var existingCount int64
	app.DB().WithContext(ctx).Model(&models.SysAffixChunk{}).Where("upload_id = ? AND chunk_index = ? AND tenant_id = ?", req.UploadId, req.ChunkIndex, tenantID).Count(&existingCount)
	if existingCount > 0 {
		app.DB().WithContext(ctx).Model(&models.SysAffixChunk{}).Where("upload_id = ? AND chunk_index = ? AND tenant_id = ?", req.UploadId, req.ChunkIndex, tenantID).Updates(map[string]interface{}{"chunk_path": chunkPath, "chunk_size": req.File.Size, "status": 0})
	} else if err := chunk.Create(ctx); err != nil {
		return fmt.Errorf("保存分片记录失败: %v", err)
	}

	return nil
}

func (s *SysAffixService) MergeChunks(ctx context.Context, req *models.ChunkMergeRequest, userID, tenantID uint) (*models.SysAffix, error) {
	if err := s.ValidateChunkFile(req.FileName, req.FileSize); err != nil {
		return nil, err
	}

	chunkList, err := models.GetChunksByUploadId(ctx, req.UploadId, tenantID)
	if err != nil {
		return nil, fmt.Errorf("获取分片记录失败: %v", err)
	}
	if len(*chunkList) != req.TotalChunks {
		return nil, fmt.Errorf("分片不完整，已上传 %d/%d", len(*chunkList), req.TotalChunks)
	}

	uploadConfig := app.UploadService.GetUploadConfig()
	localPath := uploadConfig.LocalPath
	ext := strings.ToLower(filepath.Ext(req.FileName))
	if ext == "" {
		ext = ".bin"
	}
	newFileName := uploadhelper.GenerateStorageFileName(req.FileName)
	dateFolder := time.Now().Format("2006-01-02")
	destDir := filepath.Join(localPath, dateFolder)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("创建目标目录失败: %v", err)
	}
	finalPath := filepath.Join(destDir, newFileName)

	finalFile, err := os.Create(finalPath)
	if err != nil {
		return nil, fmt.Errorf("创建最终文件失败: %v", err)
	}
	defer finalFile.Close()

	tmpDir := filepath.Join(localPath, "tmp", req.UploadId)
	var totalSize int64
	for _, chunk := range *chunkList {
		chunkFilePath := filepath.Join(tmpDir, fmt.Sprintf("chunk_%d", chunk.ChunkIndex))
		chunkFile, err := os.Open(chunkFilePath)
		if err != nil {
			os.Remove(finalPath)
			return nil, fmt.Errorf("打开分片 %d 失败: %v", chunk.ChunkIndex, err)
		}
		written, err := io.Copy(finalFile, chunkFile)
		chunkFile.Close()
		if err != nil {
			os.Remove(finalPath)
			return nil, fmt.Errorf("合并分片 %d 失败: %v", chunk.ChunkIndex, err)
		}
		totalSize += written
	}

	if req.FileSize > 0 && totalSize != req.FileSize {
		os.Remove(finalPath)
		return nil, fmt.Errorf("文件大小不一致，声明 %d 字节，实际 %d 字节", req.FileSize, totalSize)
	}

	storageKey := filepath.ToSlash(filepath.Join(dateFolder, newFileName))
	uploadResp := &app.UploadResponse{
		Path:         finalPath,
		Url:          app.UploadService.GetFileUrl(storageKey),
		FileName:     req.FileName,
		OriginalName: req.FileName,
		StoredName:   newFileName,
		Size:         totalSize,
		FileType:     ext,
	}
	if uploadConfig.UploadType != consts.UploadTypeLocal {
		uploadResp, err = app.UploadService.UploadLocalFile(finalPath, storageKey)
		if err != nil {
			os.Remove(finalPath)
			return nil, fmt.Errorf("上传合并后的文件失败: %v", err)
		}
		_ = os.Remove(finalPath)
	}

	affix := models.NewSysAffix()
	affix.Name = req.FileName
	affix.Path = uploadResp.Path
	affix.Url = uploadResp.Url
	affix.Size = int(totalSize)
	affix.Suffix = ext
	affix.Ftype = filehelper.GetFileTypeBySuffix(ext)
	affix.FileMd5 = req.FileMd5
	affix.CreatedBy = userID
	affix.TenantID = tenantID

	if err := affix.Create(ctx); err != nil {
		if uploadConfig.UploadType == consts.UploadTypeLocal {
			_ = os.Remove(finalPath)
		} else {
			_ = app.UploadService.DeleteFile(uploadResp.Path)
		}
		return nil, fmt.Errorf("保存文件记录失败: %v", err)
	}

	models.UpdateChunkStatus(ctx, req.UploadId, tenantID, 1)
	go func() {
		_ = os.RemoveAll(tmpDir)
		models.DeleteChunksByUploadId(ctx, req.UploadId, tenantID)
	}()

	return affix, nil
}

func (s *SysAffixService) CancelChunkUpload(ctx context.Context, uploadID string, tenantID uint) error {
	localPath := app.UploadService.GetUploadConfig().LocalPath
	tmpDir := filepath.Join(localPath, "tmp", uploadID)
	if err := os.RemoveAll(tmpDir); err != nil {
		app.ZapLog.Warn("清理临时分片目录失败", zap.Error(err))
	}
	models.UpdateChunkStatus(ctx, uploadID, tenantID, 2)
	models.DeleteChunksByUploadId(ctx, uploadID, tenantID)
	return nil
}

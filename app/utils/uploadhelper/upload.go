package uploadhelper

import (
	"errors"
	"fmt"
	"gin-fast/app/global/app"
	"gin-fast/app/global/consts"
)

func GetUploadType() string {
	return app.ConfigYml.GetString("upload.upload_type")
}

func GetUploadConfig() app.UploadConfig {
	return app.UploadConfig{
		UploadType:   GetUploadType(),
		MaxSize:      app.ConfigYml.GetInt("upload.max_size"),
		AllowedTypes: app.ConfigYml.GetStringSlice("upload.allowed_types"),
		LocalPath:    app.ConfigYml.GetString("upload.local_path"),
		QiniuConfig: app.QiniuConfig{
			AccessKey: app.ConfigYml.GetString("upload.qiniu_config.access_key"),
			SecretKey: app.ConfigYml.GetString("upload.qiniu_config.secret_key"),
			Bucket:    app.ConfigYml.GetString("upload.qiniu_config.bucket"),
			Domain:    app.ConfigYml.GetString("upload.qiniu_config.domain"),
			Zone:      app.ConfigYml.GetString("upload.qiniu_config.zone"),
			BasePath:  app.ConfigYml.GetString("upload.qiniu_config.base_path"),
		},
		OSSConfig: app.OSSConfig{
			Endpoint:        app.ConfigYml.GetString("upload.oss_config.endpoint"),
			AccessKeyID:     app.ConfigYml.GetString("upload.oss_config.access_key_id"),
			AccessKeySecret: app.ConfigYml.GetString("upload.oss_config.access_key_secret"),
			Bucket:          app.ConfigYml.GetString("upload.oss_config.bucket"),
			Domain:          app.ConfigYml.GetString("upload.oss_config.domain"),
			BasePath:        app.ConfigYml.GetString("upload.oss_config.base_path"),
		},
		S3Config: app.S3Config{
			Endpoint:        app.ConfigYml.GetString("upload.s3_config.endpoint"),
			AccessKeyID:     app.ConfigYml.GetString("upload.s3_config.access_key_id"),
			SecretAccessKey: app.ConfigYml.GetString("upload.s3_config.secret_access_key"),
			Bucket:          app.ConfigYml.GetString("upload.s3_config.bucket"),
			Region:          app.ConfigYml.GetString("upload.s3_config.region"),
			UseSSL:          app.ConfigYml.GetBool("upload.s3_config.use_ssl"),
			Domain:          app.ConfigYml.GetString("upload.s3_config.domain"),
			BasePath:        app.ConfigYml.GetString("upload.s3_config.base_path"),
		},
		ChunkMaxSize:      app.ConfigYml.GetInt("upload.chunk_max_size"),
		MaxChunkSize:      app.ConfigYml.GetInt("upload.max_chunk_size"),
		ChunkAllowedTypes: app.ConfigYml.GetStringSlice("upload.chunk_allowed_types"),
	}
}

func CreateUploadService() (app.FileUploadService, error) {
	switch GetUploadType() {
	case consts.UploadTypeLocal:
		return NewLocalUploadService(), nil
	case consts.UploadTypeQiniu:
		return NewQiniuUploadService(), nil
	case consts.UploadTypeOSS:
		return NewOSSUploadService()
	case consts.UploadTypeS3:
		return NewS3UploadService()
	default:
		return nil, fmt.Errorf("unsupported upload type: %s", GetUploadType())
	}
}

func errUnsupportedLocalSave(name string) error {
	return errors.New(name + " upload service does not support local save")
}

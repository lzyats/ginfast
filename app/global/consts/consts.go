package consts

// ContextKey defines the context key type.
type ContextKey string

const (
	BindContextKeyName = "userToken"
	ConfigFilePath     = "/config/config.yml"

	ServerOccurredErrorCode int    = -500100
	ServerOccurredErrorMsg  string = "服务器内部发生代码执行错误，请联系开发者排查错误日志"

	DbTypeMySql      = "mysql"
	DbTypeSqlServer  = "sqlserver"
	DbTypePostgreSql = "postgresql"
	RequestAborted   = "request_aborted"

	UploadTypeLocal = "local"
	UploadTypeQiniu = "qiniu"
	UploadTypeOSS   = "oss"
	UploadTypeS3    = "s3"

	UploadFileTypeImage    = "image"
	UploadFileTypeVideo    = "video"
	UploadFileTypeAudio    = "audio"
	UploadFileTypeDocument = "document"
	UploadFileTypeOther    = "other"
)

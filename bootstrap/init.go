package bootstrap

import (
	"context"
	"gin-fast/app/global/app"
	"gin-fast/app/global/consts"
	"gin-fast/app/global/myerrors"
	"gin-fast/app/scheduler"
	"gin-fast/app/service"
	"gin-fast/app/utils/cachehelper"
	"gin-fast/app/utils/casbinhelper"
	"gin-fast/app/utils/gormhelper"
	"gin-fast/app/utils/ip2regionhelper"
	"gin-fast/app/utils/response"
	"gin-fast/app/utils/schedulerhelper"
	"gin-fast/app/utils/tokenhelper"
	"gin-fast/app/utils/uploadhelper"
	"gin-fast/app/utils/ymlconfig"
	"log"
	"os"
	"strings"
	"time"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	checkRequiredFolders()

	if err := app.LoadVersionInfo(); err != nil {
		log.Println("warning: load version info failed:", err)
	}

	app.ConfigYml = ymlconfig.CreateYamlFactory(app.BasePath + "/config")
	app.ConfigYml.ConfigFileChangeListen(func() {
		app.UploadService = newUploadService()
		if err := ip2regionhelper.Reload(); err != nil {
			log.Printf("ip2region reload failed: %v", err)
		}
		log.Printf("upload service reloaded, current upload type: %s", app.ConfigYml.GetString("upload.upload_type"))
	})

	app.ZapLog = createZapFactory(service.ZapLogHandler)

	initDB()
	initSystemTables()

	app.CasbinV2 = casbinhelper.NewCasbinHelper()
	if err := app.CasbinV2.InitCasbin(app.DB(), app.ConfigYml.GetString("casbin.modelconfig")); err != nil {
		log.Fatal("CasbinV2.InitCasbin err :" + err.Error())
	}

	app.Cache = newCache()
	app.TokenService = newTokenService(app.Cache)

	if err := ip2regionhelper.Init(); err != nil {
		log.Printf("ip2region init skipped: %v", err)
	}

	app.UploadService = newUploadService()
	app.JobScheduler = newScheduler()

	scheduler.RegisterExecutors()
	scheduler.LoadJobsFromDB()

	app.Response = response.NewResponseHandler()
}

func initDB() {
	if app.ConfigYml.GetInt("gormv2.mysql.isinitglobalgormmysql") == 1 {
		if dbMysql, err := gormhelper.GetOneMysqlClient(); err != nil {
			log.Fatal(myerrors.ErrorsGormInitFail + err.Error())
		} else {
			app.GormDbMysql = dbMysql
		}
	}
	if app.ConfigYml.GetInt("gormv2.sqlserver.isinitglobalgormsqlserver") == 1 {
		if dbSqlserver, err := gormhelper.GetOneSqlserverClient(); err != nil {
			log.Fatal(myerrors.ErrorsGormInitFail + err.Error())
		} else {
			app.GormDbSqlserver = dbSqlserver
		}
	}
	if app.ConfigYml.GetInt("gormv2.postgresql.isinitglobalgormpostgresql") == 1 {
		if dbPostgresql, err := gormhelper.GetOnePostgreSqlClient(); err != nil {
			log.Fatal(myerrors.ErrorsGormInitFail + err.Error())
		} else {
			app.GormDbPostgreSql = dbPostgresql
		}
	}
}

func initSystemTables() {
	if err := service.NewSysNoticeService().AutoMigrate(); err != nil {
		log.Fatal("init sys_notice failed: " + err.Error())
	}
	if err := service.NewSysParamService().AutoMigrate(); err != nil {
		log.Fatal("init sys_param failed: " + err.Error())
	}
}

func checkRequiredFolders() {
	if path, err := os.Getwd(); err == nil {
		if len(os.Args) > 1 && strings.HasPrefix(os.Args[1], "-test") {
			app.BasePath = strings.Replace(strings.Replace(path, `\\test`, "", 1), "/test", "", 1)
		} else {
			app.BasePath = path
		}
		log.Println("当前项目根目录:", app.BasePath)
	} else {
		log.Fatal("获取当前目录失败")
	}
	if _, err := os.Stat(app.BasePath + consts.ConfigFilePath); err != nil {
		log.Fatal(consts.ConfigFilePath + " not exists: " + err.Error())
	}
}

func createZapFactory(entry func(zapcore.Entry) error) *zap.Logger {
	appDebug := app.ConfigYml.GetBool("server.appdebug")
	if appDebug {
		if logger, err := zap.NewDevelopment(zap.Hooks(entry)); err == nil {
			return logger
		} else {
			log.Fatal("创建 zap 日志失败: " + err.Error())
		}
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	timePrecision := app.ConfigYml.GetString("logs.timeprecision")
	var recordTimeFormat string
	switch timePrecision {
	case "second":
		recordTimeFormat = "2006-01-02 15:04:05"
	case "millisecond":
		recordTimeFormat = "2006-01-02 15:04:05.000"
	default:
		recordTimeFormat = "2006-01-02 15:04:05"
	}
	encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format(recordTimeFormat))
	}
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.TimeKey = "created_at"

	var encoder zapcore.Encoder
	switch app.ConfigYml.GetString("logs.textformat") {
	case "console":
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	case "json":
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	default:
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	fileName := app.BasePath + app.ConfigYml.GetString("logs.zaplogname")
	lumberJackLogger := &lumberjack.Logger{
		Filename:   fileName,
		MaxSize:    app.ConfigYml.GetInt("logs.maxsize"),
		MaxBackups: app.ConfigYml.GetInt("logs.maxbackups"),
		MaxAge:     app.ConfigYml.GetInt("logs.maxage"),
		Compress:   app.ConfigYml.GetBool("logs.compress"),
	}
	writer := zapcore.AddSync(lumberJackLogger)

	logLevelStr := app.ConfigYml.GetString("logs.level")
	var logLevel zapcore.Level
	switch logLevelStr {
	case "debug":
		logLevel = zap.DebugLevel
	case "info":
		logLevel = zap.InfoLevel
	case "warn":
		logLevel = zap.WarnLevel
	case "error":
		logLevel = zap.ErrorLevel
	case "fatal":
		logLevel = zap.FatalLevel
	case "panic":
		logLevel = zap.PanicLevel
	default:
		logLevel = zap.InfoLevel
	}

	zapCore := zapcore.NewCore(encoder, writer, logLevel)
	return zap.New(zapCore, zap.AddCaller(), zap.Hooks(entry), zap.AddStacktrace(zap.WarnLevel))
}

func newCache() app.CacheInterf {
	cacheType := app.ConfigYml.GetString("server.cachetype")
	if cacheType == "redis" {
		redisHelper, err := cachehelper.NewRedisHelper(
			app.ConfigYml.GetString("redis.host")+":"+app.ConfigYml.GetString("redis.port"),
			app.ConfigYml.GetString("redis.password"),
			app.ConfigYml.GetInt("redis.indexdb"),
		)
		if err != nil {
			panic(err)
		}
		return redisHelper
	}
	return cachehelper.NewMemoryHelper()
}

func newTokenService(cache app.CacheInterf) app.TokenServiceInterface {
	tokenExpire := app.ConfigYml.GetDuration("token.jwttokenexpire")
	refreshExpire := app.ConfigYml.GetDuration("token.jwttokenrefreshexpire")

	return &tokenhelper.TokenService{
		RedisHelper:    cache,
		JWTSecret:      app.ConfigYml.GetString("token.jwttokensignkey"),
		Ctx:            context.Background(),
		TokenExpire:    tokenExpire,
		RefreshExpire:  refreshExpire,
		CacheKeyPrefix: app.ConfigYml.GetString("token.cachekeyprefix"),
		IsCache:        app.ConfigYml.GetBool("token.iscache"),
	}
}

func newUploadService() app.FileUploadService {
	uploadService, err := uploadhelper.CreateUploadService()
	if err != nil {
		log.Fatal("初始化文件上传服务失败: " + err.Error())
	}
	return uploadService
}

func newScheduler() app.JobSchedulerInterf {
	logDir := app.BasePath + app.ConfigYml.GetString("scheduler.log.dir")

	levelStr := app.ConfigYml.GetString("scheduler.log.level")
	var level schedulerhelper.LogLevel
	switch levelStr {
	case "debug":
		level = schedulerhelper.LevelDebug
	case "info":
		level = schedulerhelper.LevelInfo
	case "warn":
		level = schedulerhelper.LevelWarn
	case "error":
		level = schedulerhelper.LevelError
	case "fatal":
		level = schedulerhelper.LevelFatal
	default:
		level = schedulerhelper.LevelInfo
	}

	bufferSize := app.ConfigYml.GetInt("scheduler.job_results_buffer_size")
	if bufferSize <= 0 {
		bufferSize = 1000
	}

	scheduler := schedulerhelper.NewJobScheduler(
		schedulerhelper.WithLoggerConfig(logDir, level),
		schedulerhelper.WithJobResultsBufferSize(bufferSize),
	)

	scheduler.Start()
	return scheduler
}

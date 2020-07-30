package logger

import (
	"io"
	"os"
	"path/filepath"
	"time"

	rotatelogs "github.com/lestrrat/go-file-rotatelogs"
	"github.com/nacos-group/nacos-sdk-go/common/file"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger Logger
)

var levelMap = map[string]zapcore.Level{
	"debug": zapcore.DebugLevel,
	"info":  zapcore.InfoLevel,
	"warn":  zapcore.WarnLevel,
	"error": zapcore.ErrorLevel,
}

type Config struct {
	Level        string
	OutputPath   string
	RotationTime string
	MaxAge       int64
}

type NacosLogger struct {
	Logger
}

// Logger is the interface for Logger types
type Logger interface {
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Debug(args ...interface{})

	Infof(fmt string, args ...interface{})
	Warnf(fmt string, args ...interface{})
	Errorf(fmt string, args ...interface{})
	Debugf(fmt string, args ...interface{})
}

func init() {
	zapLoggerConfig := zap.NewDevelopmentConfig()
	zapLoggerEncoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	zapLoggerConfig.EncoderConfig = zapLoggerEncoderConfig
	zapLogger, _ := zapLoggerConfig.Build(zap.AddCallerSkip(1))
	logger = &NacosLogger{zapLogger.Sugar()}
}

func InitLogger(config Config) (err error) {
	logLevel := getLogLevel(config.Level)
	encoder := getEncoder()
	writer, err := getWriter(config.OutputPath, config.RotationTime, config.MaxAge)
	if err != nil {
		return
	}
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(encoder), zapcore.AddSync(writer), logLevel)
	zaplogger := zap.New(core, zap.AddCallerSkip(1))
	logger = &NacosLogger{zaplogger.Sugar()}
	return
}

func getLogLevel(level string) zapcore.Level {
	if zapLevel, ok := levelMap[level]; ok {
		return zapLevel
	}
	return zapcore.InfoLevel
}

func getEncoder() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

func getWriter(outputPath string, rotateTime string, maxAge int64) (writer io.Writer, err error) {
	err = file.MkdirIfNecessary(outputPath)
	if err != nil {
		return
	}
	outputPath = outputPath + string(os.PathSeparator)
	rotateDuration, err := time.ParseDuration(rotateTime)
	writer, err = rotatelogs.New(filepath.Join(outputPath, "nacos-sdk.log-%Y%m%d%H%M"),
		rotatelogs.WithRotationTime(rotateDuration), rotatelogs.WithMaxAge(time.Duration(maxAge)*rotateDuration),
		rotatelogs.WithLinkName(filepath.Join(outputPath, "nacos-sdk.log")))
	return
}

//SetLogger sets logger for sdk
func SetLogger(log Logger) {
	logger = log
}

func GetLogger() Logger {
	return logger
}

/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package logger

import (
	"os"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const LogFileName = "nacos-sdk.log"

var (
	logger  Logger
	logLock sync.RWMutex
)

var levelMap = map[string]zapcore.Level{
	"debug": zapcore.DebugLevel,
	"info":  zapcore.InfoLevel,
	"warn":  zapcore.WarnLevel,
	"error": zapcore.ErrorLevel,
}

type Config struct {
	Level                  string
	Sampling               *SamplingConfig
	LogRollingConfig       *lumberjack.Logger
	ClientLogRollingConfig *ClientLogRollingConfig
	LogDir                 string
}

type SamplingConfig struct {
	Initial    int
	Thereafter int
	Tick       time.Duration
}

type ClientLogRollingConfig struct {
	// MaxSize is the maximum size in megabytes of the log file before it gets
	// rotated. It defaults to 100 megabytes.
	MaxSize int

	// MaxAge is the maximum number of days to retain old log files based on the
	// timestamp encoded in their filename.  Note that a day is defined as 24
	// hours and may not exactly correspond to calendar days due to daylight
	// savings, leap seconds, etc. The default is not to remove old log files
	// based on age.
	MaxAge int

	// MaxBackups is the maximum number of old log files to retain.  The default
	// is to retain all old log files (though MaxAge may still cause them to get
	// deleted.)
	MaxBackups int

	// LocalTime determines if the time used for formatting the timestamps in
	// backup files is the computer's local time.  The default is to use UTC
	// time.
	LocalTime bool

	// Compress determines if the rotated log files should be compressed
	// using gzip. The default is not to perform compression.
	Compress bool
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

func BuildLoggerConfig(clientConfig Config) Config {
	clientConfig.LogRollingConfig = &lumberjack.Logger{
		Filename: clientConfig.LogDir + string(os.PathSeparator) + LogFileName,
	}
	logRollingConfig := clientConfig.ClientLogRollingConfig
	if logRollingConfig != nil {
		clientConfig.LogRollingConfig.MaxSize = logRollingConfig.MaxSize
		clientConfig.LogRollingConfig.MaxAge = logRollingConfig.MaxAge
		clientConfig.LogRollingConfig.MaxBackups = logRollingConfig.MaxBackups
		clientConfig.LogRollingConfig.LocalTime = logRollingConfig.LocalTime
		clientConfig.LogRollingConfig.Compress = logRollingConfig.Compress
	}
	return clientConfig
}

// InitLogger is init global logger for nacos
func InitLogger(config Config) (err error) {
	logLock.Lock()
	defer logLock.Unlock()
	logger, err = InitNacosLogger(config)
	return
}

// InitNacosLogger is init nacos default logger
func InitNacosLogger(config Config) (Logger, error) {

	logLevel := getLogLevel(config.Level)
	encoder := getEncoder()
	writer := config.getLogWriter()
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(encoder), writer, logLevel)
	if config.Sampling != nil {
		core = zapcore.NewSamplerWithOptions(core, config.Sampling.Tick, config.Sampling.Initial, config.Sampling.Thereafter)
	}

	zaplogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	return &NacosLogger{zaplogger.Sugar()}, nil
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

//SetLogger sets logger for sdk
func SetLogger(log Logger) {
	logLock.Lock()
	defer logLock.Unlock()
	logger = log
}

func GetLogger() Logger {
	logLock.RLock()
	defer logLock.RUnlock()
	return logger
}

// getLogWriter get Lumberjack writer by LumberjackConfig
func (c *Config) getLogWriter() zapcore.WriteSyncer {
	return zapcore.AddSync(c.LogRollingConfig)
}

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

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

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
	Level            string
	LogFileName      string
	Sampling         *SamplingConfig
	LogRollingConfig *lumberjack.Logger
	LogDir           string
	CustomLogger     Logger
	LogStdout        bool
}

type SamplingConfig struct {
	Initial    int
	Thereafter int
	Tick       time.Duration
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
	defaultLogger, _ := zapLoggerConfig.Build(zap.AddCaller(), zap.AddCallerSkip(1))
	setLogger(&NacosLogger{defaultLogger.Sugar()})
}

// InitLogger is init global logger for nacos
func InitLogger(config Config) (err error) {
	l, err := initNacosLogger(config)
	if err != nil {
		return err
	}
	setLogger(l)
	return
}

// InitNacosLogger is init nacos default logger
func initNacosLogger(config Config) (Logger, error) {
	if config.CustomLogger != nil {
		return &NacosLogger{config.CustomLogger}, nil
	}
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
func setLogger(log Logger) {
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
	if c.LogRollingConfig == nil {
		c.LogRollingConfig = &lumberjack.Logger{}
	}
	c.LogRollingConfig.Filename = c.LogDir + string(os.PathSeparator) + c.LogFileName
	if c.LogStdout {
		return zapcore.NewMultiWriteSyncer(zapcore.AddSync(c.LogRollingConfig), zapcore.AddSync(os.Stdout))
	}
	return zapcore.AddSync(c.LogRollingConfig)
}

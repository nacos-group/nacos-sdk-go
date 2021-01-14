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

// Info is info level
func Info(args ...interface{}) {
	GetLogger().Info(args...)
}

// Warn is warning level
func Warn(args ...interface{}) {
	GetLogger().Warn(args...)
}

// Error is error level
func Error(args ...interface{}) {
	GetLogger().Error(args...)
}

// Debug is debug level
func Debug(args ...interface{}) {
	GetLogger().Debug(args...)
}

// Infof is format info level
func Infof(fmt string, args ...interface{}) {
	GetLogger().Infof(fmt, args...)
}

// Warnf is format warning level
func Warnf(fmt string, args ...interface{}) {
	GetLogger().Warnf(fmt, args...)
}

// Errorf is format error level
func Errorf(fmt string, args ...interface{}) {
	GetLogger().Errorf(fmt, args...)
}

// Debugf is format debug level
func Debugf(fmt string, args ...interface{}) {
	GetLogger().Debugf(fmt, args...)
}

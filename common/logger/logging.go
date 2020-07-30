package logger

// Info is info level
func Info(args ...interface{}) {
	logger.Info(args...)
}

// Warn is warning level
func Warn(args ...interface{}) {
	logger.Warn(args...)
}

// Error is error level
func Error(args ...interface{}) {
	logger.Error(args...)
}

// Debug is debug level
func Debug(args ...interface{}) {
	logger.Debug(args...)
}

// Infof is format info level
func Infof(fmt string, args ...interface{}) {
	logger.Infof(fmt, args...)
}

// Warnf is format warning level
func Warnf(fmt string, args ...interface{}) {
	logger.Warnf(fmt, args...)
}

// Errorf is format error level
func Errorf(fmt string, args ...interface{}) {
	logger.Errorf(fmt, args...)
}

// Debugf is format debug level
func Debugf(fmt string, args ...interface{}) {
	logger.Debugf(fmt, args...)
}

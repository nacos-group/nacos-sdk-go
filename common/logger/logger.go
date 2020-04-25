package logger

import (
	"github.com/lestrrat/go-file-rotatelogs"
	"github.com/nacos-group/nacos-sdk-go/common/util"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	Debug   *log.Logger // 记录所有日志
	Info    *log.Logger // 重要的信息
	Warning *log.Logger // 需要注意的信息
	Error   *log.Logger // 非常严重的问题
)

func init() {
	// 默认日志路径
	InitLog("./nacos/log")
}

func InitLog(logDir string) error {
	err := util.MkdirIfNecessary(logDir)
	if err != nil {
		return err
	}

	logDir = logDir + string(os.PathSeparator)
	rl, err := rotatelogs.New(filepath.Join(logDir, "nacos-sdk.log-%Y%m%d%H%M"), rotatelogs.WithRotationTime(time.Hour), rotatelogs.WithMaxAge(48*time.Hour), rotatelogs.WithLinkName(filepath.Join(logDir, "nacos-sdk.log")))
	if err != nil {
		return err
	}

	Debug = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile|log.Lmsgprefix)
	Info = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile|log.Lmsgprefix)
	Warning = log.New(os.Stdout, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile|log.Lmsgprefix)
	Error = log.New(rl, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile|log.Lmsgprefix)

	//log.SetOutput(rl)
	//log.SetFlags(log.LstdFlags)
	return nil
}

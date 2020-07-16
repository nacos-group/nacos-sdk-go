package logger

import (
	"log"
	"os"
	"path/filepath"
	"time"

	rotatelogs "github.com/lestrrat/go-file-rotatelogs"
	"github.com/nacos-group/nacos-sdk-go/common/util"
)

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
	log.SetOutput(rl)
	log.SetFlags(log.LstdFlags)
	return nil
}

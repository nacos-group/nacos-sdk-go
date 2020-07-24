package util

import (
	"os"
	"testing"

	"github.com/nacos-group/nacos-sdk-go/utils"

	"github.com/stretchr/testify/assert"
)

func TestMkdirIfNecessaryForAbsPath(t *testing.T) {
	path := utils.GetCurrentPath() + string(os.PathSeparator) + "log"
	err := MkdirIfNecessary(path)
	assert.Nil(t, err)
}

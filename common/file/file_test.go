package file

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMkdirIfNecessaryForAbsPath(t *testing.T) {
	path := GetCurrentPath() + string(os.PathSeparator) + "log"
	err := MkdirIfNecessary(path)
	assert.Nil(t, err)
}

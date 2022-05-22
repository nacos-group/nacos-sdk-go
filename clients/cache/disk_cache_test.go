package cache

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nacos-group/nacos-sdk-go/v2/common/file"
)

func TestGetFailover(t *testing.T) {
	cacheKey := "test_failOver"
	dir := file.GetCurrentPath()
	fileContent := "test_failover"
	t.Run("writeContent", func(t *testing.T) {
		filepath := dir + string(os.PathSeparator) + cacheKey + "_failover"
		fmt.Println(filepath)
		err := writeFileContent(filepath, fileContent)
		assert.Nil(t, err)
	})
	t.Run("getContent", func(t *testing.T) {
		content := GetFailover(cacheKey, dir)
		assert.Equal(t, content, fileContent)
	})
	t.Run("clearContent", func(t *testing.T) {
		filepath := dir + string(os.PathSeparator) + cacheKey + "_failover"
		err := writeFileContent(filepath, "")
		assert.Nil(t, err)
	})
}

// write file content
func writeFileContent(filepath, content string) error {
	return ioutil.WriteFile(filepath, []byte(content), 0666)
}

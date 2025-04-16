package cache

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"testing"

	"github.com/dbsyk/nacos-sdk-go/v2/util"

	"github.com/stretchr/testify/assert"

	"github.com/dbsyk/nacos-sdk-go/v2/common/file"
)

var (
	dir   = file.GetCurrentPath()
	group = "FILE_GROUP"
	ns    = "chasu"
)

func TestWriteAndGetConfigToFile(t *testing.T) {
	dataIdSuffix := strconv.Itoa(rand.Intn(1000))
	t.Run("write and get config content", func(t *testing.T) {
		dataId := "config_content" + dataIdSuffix
		cacheKey := util.GetConfigCacheKey(dataId, group, ns)
		configContent := "config content"

		err := WriteConfigToFile(cacheKey, dir, "")
		assert.Nil(t, err)

		configFromFile, err := ReadConfigFromFile(cacheKey, dir)
		assert.NotNil(t, err)
		assert.Equal(t, configFromFile, "")

		err = WriteConfigToFile(cacheKey, dir, configContent)
		assert.Nil(t, err)

		fromFile, err := ReadConfigFromFile(cacheKey, dir)
		assert.Nil(t, err)
		assert.Equal(t, fromFile, configContent)

		err = WriteConfigToFile(cacheKey, dir, "")
		assert.Nil(t, err)

		configFromFile, err = ReadConfigFromFile(cacheKey, dir)
		assert.Nil(t, err)
		assert.Equal(t, configFromFile, "")
	})

	t.Run("write and get config encryptedDataKey", func(t *testing.T) {
		dataId := "config_encryptedDataKey" + dataIdSuffix
		cacheKey := util.GetConfigCacheKey(dataId, group, ns)
		configContent := "config encrypted data key"

		err := WriteEncryptedDataKeyToFile(cacheKey, dir, "")
		assert.Nil(t, err)

		configFromFile, err := ReadEncryptedDataKeyFromFile(cacheKey, dir)
		assert.Nil(t, err)
		assert.Equal(t, configFromFile, "")

		err = WriteEncryptedDataKeyToFile(cacheKey, dir, configContent)
		assert.Nil(t, err)

		fromFile, err := ReadEncryptedDataKeyFromFile(cacheKey, dir)
		assert.Nil(t, err)
		assert.Equal(t, fromFile, configContent)

		err = WriteEncryptedDataKeyToFile(cacheKey, dir, "")
		assert.Nil(t, err)

		configFromFile, err = ReadEncryptedDataKeyFromFile(cacheKey, dir)
		assert.Nil(t, err)
		assert.Equal(t, configFromFile, "")
	})
	t.Run("double write config file", func(t *testing.T) {
		dataId := "config_encryptedDataKey" + dataIdSuffix
		cacheKey := util.GetConfigCacheKey(dataId, group, ns)
		configContent := "config encrypted data key"

		err := WriteConfigToFile(cacheKey, dir, configContent)
		assert.Nil(t, err)

		err = WriteConfigToFile(cacheKey, dir, configContent)
		assert.Nil(t, err)

		fromFile, err := ReadConfigFromFile(cacheKey, dir)
		assert.Nil(t, err)
		assert.Equal(t, fromFile, configContent)
	})
	t.Run("read doesn't existed config file", func(t *testing.T) {
		dataId := "config_encryptedDataKey" + dataIdSuffix + strconv.Itoa(rand.Intn(1000))
		cacheKey := util.GetConfigCacheKey(dataId, group, ns)

		_, err := ReadConfigFromFile(cacheKey, dir)
		assert.NotNil(t, err)

		_, err = ReadEncryptedDataKeyFromFile(cacheKey, dir)
		assert.Nil(t, err)
	})
}

func TestGetFailover(t *testing.T) {
	cacheKey := "test_failOver"
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
	return os.WriteFile(filepath, []byte(content), 0666)
}

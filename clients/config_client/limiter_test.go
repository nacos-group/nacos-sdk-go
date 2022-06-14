package config_client

import (
	"fmt"
	"testing"

	"github.com/nacos-group/nacos-sdk-go/v2/vo"

	"github.com/stretchr/testify/assert"
)

func TestLimiter(t *testing.T) {
	client := createConfigClientTest()
	success, err := client.PublishConfig(vo.ConfigParam{
		DataId:  localConfigTest.DataId,
		Group:   "default-group",
		Content: "hello world"})

	assert.Nil(t, err)
	assert.True(t, success)

	for i := 0; i <= 10; i++ {
		content, err := client.GetConfig(vo.ConfigParam{
			DataId: localConfigTest.DataId,
			Group:  "default-group"})
		if i > 4 {
			fmt.Println("limiter test,err : ", err)
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
			assert.Equal(t, "hello world", content)
		}
	}
}

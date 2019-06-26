package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

/**
*
* @description :
*
* @author : codezhang
*
* @create : 2019-01-15 20:16
**/

func TestMd5(t *testing.T) {
	md5 := Md5("demo")
	assert.Equal(t, "fe01ce2a7fbac8fafaed7c982a04e229", md5)
}

package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMd5(t *testing.T) {
	md5 := Md5("demo")
	assert.Equal(t, "fe01ce2a7fbac8fafaed7c982a04e229", md5)
}

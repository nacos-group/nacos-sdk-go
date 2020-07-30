package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitLogger(t *testing.T) {
	config := Config{
		Level:        "degug",
		OutputPath:   "/tmp/nacos",
		RotationTime: "1h",
		MaxAge:       2,
	}
	err := InitLogger(config)
	assert.NoError(t, err)
}

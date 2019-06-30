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
* @create : 2019-01-11 19:25
**/

type object struct {
	Name     string            `param:"name"`
	Likes    []string          `param:"likes"`
	Metadata map[string]string `param:"metadata"`
	Age      uint64            `param:"age"`
	Healthy  bool              `param:"healthy"`
	Money    int               `param:"money"`
}

func TestTransformObject2Param(t *testing.T) {
	assert.Equal(t, map[string]string{}, TransformObject2Param(nil))
	obj := object{
		Name:  "code",
		Likes: []string{"a", "b"},
		Metadata: map[string]string{
			"M1": "m1",
		},
		Age:     10,
		Healthy: true,
		Money:   10,
	}
	params := TransformObject2Param(&obj)
	assert.Equal(t, map[string]string{
		"name":     "code",
		"metadata": `{"M1":"m1"}`,
		"likes":    "a,b",
		"age":      "10",
		"money":    "10",
		"healthy":  "true",
	}, params)
}

package naming_client

import (
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBeatReactor_AddBeatInfo(t *testing.T) {
	br := NewBeatReactor(NamingProxy{}, 5000)
	serviceName := "Test"
	groupName := "public"
	beatInfo := model.BeatInfo{
		Ip:          "127.0.0.1",
		Port:        8080,
		Metadata:    map[string]string{},
		ServiceName: utils.GetGroupName(serviceName, groupName),
		Cluster:     "default",
		Weight:      1,
	}
	br.AddBeatInfo(utils.GetGroupName(serviceName, groupName), beatInfo)
	key := buildKey(utils.GetGroupName(serviceName, groupName), beatInfo.Ip, beatInfo.Port)
	result, ok := br.beatMap.Get(key)
	assert.Equal(t, ok, true, "key should exists!")
	assert.ObjectsAreEqual(result.(*model.BeatInfo), beatInfo)
}

func TestBeatReactor_RemoveBeatInfo(t *testing.T) {
	br := NewBeatReactor(NamingProxy{}, 5000)
	serviceName := "Test"
	groupName := "public"
	beatInfo1 := model.BeatInfo{
		Ip:          "127.0.0.1",
		Port:        8080,
		Metadata:    map[string]string{},
		ServiceName: utils.GetGroupName(serviceName, groupName),
		Cluster:     "default",
		Weight:      1,
	}
	br.AddBeatInfo(utils.GetGroupName(serviceName, groupName), beatInfo1)
	beatInfo2 := model.BeatInfo{
		Ip:          "127.0.0.2",
		Port:        8080,
		Metadata:    map[string]string{},
		ServiceName: utils.GetGroupName(serviceName, groupName),
		Cluster:     "default",
		Weight:      1,
	}
	br.AddBeatInfo(utils.GetGroupName(serviceName, groupName), beatInfo2)
	br.RemoveBeatInfo(utils.GetGroupName(serviceName, groupName), "127.0.0.1", 8080)
	key := buildKey(utils.GetGroupName(serviceName, groupName), beatInfo2.Ip, beatInfo2.Port)
	result, ok := br.beatMap.Get(key)
	assert.Equal(t, br.beatMap.Count(), 1, "beatinfo map length should be 1")
	assert.Equal(t, ok, true, "key should exists!")
	assert.ObjectsAreEqual(result.(*model.BeatInfo), beatInfo2)

}

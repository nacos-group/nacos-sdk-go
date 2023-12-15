package rpc_response

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRpcResponseIsSuccess(t *testing.T) {
	responseBody0 := `{"resultCode":200,"errorCode":0}`
	responseBody1 := `{"resultCode":200,"errorCode":0,"success":true}`
	responseBody2 := `{"resultCode":200,"errorCode":0,"success":"true"}`
	responseBody3 := `{"resultCode":200,"errorCode":0,"success":false}`
	responseBody4 := `{"resultCode":500,"errorCode":0,"success":true}`
	responseBody5 := `{"resultCode":500,"errorCode":0,"success":false}`

	responseBodyList := make([]string, 0)
	responseBodyList = append(responseBodyList, responseBody0, responseBody1, responseBody2, responseBody3, responseBody4, responseBody5)
	for k, v := range ClientResponseMapping {
		t.Run("test "+k, func(t *testing.T) {
			for index, responseBody := range responseBodyList {
				response, err := InnerResponseJsonUnmarshal([]byte(responseBody), v)
				switch index {
				case 0, 1, 4:
					assert.True(t, response.IsSuccess())
					break
				case 3, 5:
					assert.False(t, response.IsSuccess())
					break
				case 2:
					assert.Nil(t, response)
					assert.NotNil(t, err)
					t.Logf("handle %d failed with responseBody: %s", index, responseBody)
					break
				default:
					panic("unknown index")
				}
			}
		})
	}
}

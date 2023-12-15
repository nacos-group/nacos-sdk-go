package rpc_response

import "encoding/json"

func InnerResponseJsonUnmarshal(responseBody []byte, responseFunc func() IResponse) (IResponse, error) {
	response := responseFunc()
	tempFiledMap := make(map[string]interface{})
	err := json.Unmarshal(responseBody, response)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(responseBody, &tempFiledMap)
	if err != nil {
		return nil, err
	}
	if _, ok := tempFiledMap["success"]; !ok {
		response.SetSuccess(response.GetResultCode() == int(ResponseSuccessCode))
	}
	return response, err

}

package rpc_response

import "encoding/json"

func InnerResponseJsonUnmarshal(responseBody []byte, responseFunc func() IResponse) (IResponse, error) {
	response := responseFunc()
	err := json.Unmarshal(responseBody, response)
	if err != nil {
		return nil, err
	}

	if !response.IsSuccess() {
		tempFiledMap := make(map[string]interface{})
		err = json.Unmarshal(responseBody, &tempFiledMap)
		if err != nil {
			return response, nil
		}
		if _, ok := tempFiledMap[ResponseSuccessField]; !ok {
			response.SetSuccess(response.GetResultCode() == int(ResponseSuccessCode))
		}
	}
	return response, err

}

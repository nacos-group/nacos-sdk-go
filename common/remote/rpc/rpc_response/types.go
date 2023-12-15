package rpc_response

type ResponseCode int

const (
	ResponseSuccessCode ResponseCode = 200
	ResponseFailCode    ResponseCode = 500

	ResponseSuccessField = "success"
)

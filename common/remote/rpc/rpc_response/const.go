package rpc_response

type ResponseCode int

const (
	ResponseSuccessCode ResponseCode = 200
	ResponseFailCode    ResponseCode = 500

	ResponseSuccessField = "success"
)

type ResponseErrorCode int

const (
	ConfigNotFound      ResponseErrorCode = 300
	ConfigQueryConflict ResponseErrorCode = 400
)

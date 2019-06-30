package nacos_error

/**
*
* @description :
*
* @author : codezhang
*
* @create : 2019-01-14 11:22
**/

type NacosError struct {
	ErrMsg string
}

func (err *NacosError) Error() (str string) {
	return err.ErrMsg
}

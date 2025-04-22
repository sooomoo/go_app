package service

type RespCode string

const (
	RespCodeSucceed RespCode = "succeed"
	// RespCodeBlocked           RespCode = "blocked"
	RespCodeInvalidArgs       RespCode = "invalid_args"
	RespCodeInvalidPhone      RespCode = "invalid_phone"
	RespCodeInvalidMsgCode    RespCode = "invalid_msg_code"
	RespCodeInvalidSecureCode RespCode = "invalid_secure_code"
	RespCodeFailed            RespCode = "fail"
)

type ResponseDto[TData any] struct {
	Code RespCode `json:"code"`
	Msg  string   `json:"msg"`
	Data TData    `json:"data"`
}

func NewResponseDtoNoData(code RespCode, msg string) *ResponseDto[any] {
	return &ResponseDto[any]{Code: code, Msg: msg}
}

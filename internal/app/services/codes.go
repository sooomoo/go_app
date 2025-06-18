package services

type RespCode string

const (
	RespCodeSucceed RespCode = "succeed"
	// RespCodeBlocked           RespCode = "blocked"
	RespCodeInvalidArgs       RespCode = "invalidArgs"
	RespCodeInvalidPhone      RespCode = "invalidPhone"
	RespCodeInvalidMsgCode    RespCode = "invalidMsgCode"
	RespCodeInvalidSecureCode RespCode = "invalidSecureCode"
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

func NewResponseInvalidArgs(msg string) *ResponseDto[any] {
	return &ResponseDto[any]{Code: RespCodeInvalidArgs, Msg: msg}
}

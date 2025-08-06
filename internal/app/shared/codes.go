package shared

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

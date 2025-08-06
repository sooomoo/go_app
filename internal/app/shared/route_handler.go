package shared

type RouteHandler struct{}

func (r *RouteHandler) NewResponseDtoNoData(code RespCode, msg string) *ResponseDto[any] {
	return &ResponseDto[any]{Code: code, Msg: msg}
}

func (r *RouteHandler) NewResponseInvalidArgs(msg string) *ResponseDto[any] {
	return &ResponseDto[any]{Code: RespCodeInvalidArgs, Msg: msg}
}

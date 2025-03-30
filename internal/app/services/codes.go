package services

type ReplyCode string

const (
	ReplyCodeSucceed           ReplyCode = "succeed"
	ReplyCodeInvalidPhone      ReplyCode = "invalid_phone"
	ReplyCodeInvalidMsgCode    ReplyCode = "invalid_msg_code"
	ReplyCodeInvalidSecureCode ReplyCode = "invalid_secure_code"
	ReplyCodeFailed            ReplyCode = "fail"
)

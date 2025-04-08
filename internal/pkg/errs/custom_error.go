package errs

import "fmt"

type ReplyError struct {
	code    string
	message string
}

func (e ReplyError) Error() string {
	return fmt.Sprintf("[ReplyError]code: %s, message: %s", e.code, e.message)
}

func (e ReplyError) Code() string {
	return e.code
}

func (e ReplyError) Message() string {
	return e.message
}

func NewReplyError(code, message string) *ReplyError {
	return &ReplyError{code: code, message: message}
}

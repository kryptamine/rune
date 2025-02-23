package solus

import "fmt"

type RuntimeError struct {
	token  Token
	errMsg string
}

func (e RuntimeError) Error() string {
	return fmt.Sprintf(
		"[line: %d] %s",
		e.token.line,
		e.errMsg,
	)
}

func NewRuntimeError(token Token, msg string) error {
	return RuntimeError{token: token, errMsg: msg}
}

package solus

const MaxArity = 255

// Return is a special type of error that is returned by a function
type Return struct {
	value any
}

func (e *Return) Error() string {
	return "<fn return>"
}

type Callable interface {
	Call(interpreter *Interpreter, args []any, token Token) (any, error)
	Arity() int
}

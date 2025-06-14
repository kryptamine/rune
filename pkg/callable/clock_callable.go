package callable

import (
	"rune/pkg/ast"
	"time"
)

// ClockCallable is a callable that returns the current time in seconds since the Unix epoch.
type ClockCallable struct{}

func NewClockCallable() Callable {
	return &ClockCallable{}
}

func (c *ClockCallable) Call(_ ExecuteBlockFn, args []any, _ ast.Token) (any, error) {
	return float64(time.Now().Unix()), nil
}

func (c *ClockCallable) Arity() int {
	return 0
}

func (c *ClockCallable) String() string {
	return "<native fn>"
}

package callable

import (
	"rune/pkg/ast"
	"rune/pkg/value"
	"time"
)

// ClockCallable is a callable that returns the current time in seconds since the Unix epoch.
type ClockCallable struct{}

func NewClockCallable() Callable {
	return &ClockCallable{}
}

func (c *ClockCallable) Call(_ []value.Value, _ ast.Token) (value.Value, error) {
	return value.NewNumber(float64(time.Now().UnixNano()) / float64(time.Second)), nil
}

func (c *ClockCallable) Arity() int {
	return 0
}

func (c *ClockCallable) String() string {
	return "<native fn>"
}

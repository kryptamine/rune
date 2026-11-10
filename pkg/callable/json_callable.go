package callable

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"rune/pkg/ast"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
)

type JsonCallable struct{}

func NewJsonCallable() Callable {
	return &JsonCallable{}
}

func (c *JsonCallable) Call(args []value.Value, token ast.Token) (value.Value, error) {
	switch args[0].Kind() {
	case value.String:
		url := args[0].Str()
		httpClient := http.Client{
			Timeout: time.Second * 10,
		}

		res, err := httpClient.Get(url)

		if err != nil {
			return value.NilValue, errors.NewRuntimeError(token, fmt.Sprintf("Error fetching %s: %s", url, err.Error()))
		}

		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			return value.NilValue, errors.NewRuntimeError(token, fmt.Sprintf("Error fetching %s: Status code %d", url, res.StatusCode))
		}

		// Top-level value may be an object, an array or a primitive.
		var jsonRes any

		if err := json.NewDecoder(res.Body).Decode(&jsonRes); err != nil {
			return value.NilValue, errors.NewRuntimeError(token, fmt.Sprintf("Error parsing JSON from %s: %s", url, err.Error()))
		}

		return value.FromGo(jsonRes), nil
	default:
		return value.NilValue, errors.NewRuntimeError(token, fmt.Sprintf("json() expects a URL string, got %s", helpers.TypeName(args[0])))
	}
}

func (c *JsonCallable) Arity() int {
	return 1
}

func (c *JsonCallable) String() string {
	return "<native json>"
}

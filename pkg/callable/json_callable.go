package callable

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"rune/pkg/ast"
	"rune/pkg/errors"
)

type JsonCallable struct{}

func NewJsonCallable() Callable {
	return &JsonCallable{}
}

func (c *JsonCallable) Call(_ ExecuteBlockFn, args []any, token ast.Token) (any, error) {
	if len(args) < 1 {
		return nil, errors.NewRuntimeError(token, "Expected 1 argument, got 0.")
	}

	switch url := args[0].(type) {
	case string:
		httpClient := http.Client{
			Timeout: time.Second * 2,
		}

		res, err := httpClient.Get(url)

		if err != nil {
			return nil, errors.NewRuntimeError(token, fmt.Sprintf("Error fetching %s: %s", url, err.Error()))
		}

		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			return nil, errors.NewRuntimeError(token, fmt.Sprintf("Error fetching %s: Status code %d", url, res.StatusCode))
		}

		jsonRes := map[string]any{}

		if err := json.NewDecoder(res.Body).Decode(&jsonRes); err != nil {
			return nil, errors.NewRuntimeError(token, fmt.Sprintf("Error parsing JSON from %s: %s", url, err.Error()))
		}

		return jsonRes, nil
	default:
		return nil, errors.NewRuntimeError(token, fmt.Sprintf("Can only parse strings, got %T", args[0]))
	}
}

func (c *JsonCallable) Arity() int {
	return 1
}

func (c *JsonCallable) String() string {
	return "<native json>"
}

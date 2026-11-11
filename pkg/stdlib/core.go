package stdlib

import (
	"encoding/json"
	"net/http"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"slices"
	"time"
	"unicode/utf8"
)

var httpClient = http.Client{Timeout: 10 * time.Second}

var coreNatives = []*Native{
	{"clock", 0, 0, func(*call) value.Value {
		return value.NewNumber(float64(time.Now().UnixNano()) / 1e9)
	}},
	{"len", 1, 1, func(c *call) value.Value {
		switch v := c.args[0]; v.Kind() {
		case value.Array:
			return count(len(v.Arr()))
		case value.Object:
			return count(len(v.Obj()))
		case value.String:
			return count(utf8.RuneCountInString(v.Str()))
		case value.Struct:
			return count(len(v.Shape().Fields))
		}

		return c.typeError(0, "a string, an array, an object or a struct")
	}},
	{"type", 1, 1, func(c *call) value.Value {
		return value.NewString(helpers.TypeName(c.args[0]))
	}},
	{"object", 1, 1, func(c *call) value.Value {
		v := c.args[0]
		if v.Kind() != value.Struct {
			return c.typeError(0, "a struct")
		}

		names := v.Shape().Fields
		obj := make(map[string]value.Value, len(names))

		for i, f := range v.Fields() {
			obj[names[i]] = f
		}

		return value.NewObject(obj)
	}},
	// append copies, so the result never shares a backing array with the input.
	{"append", 2, -1, func(c *call) value.Value {
		return value.NewArray(slices.Concat(c.arr(0), c.args[1:]))
	}},
	{"push", 2, -1, func(c *call) value.Value {
		if c.arr(0); !c.ok() {
			return value.NilValue
		}

		arr := c.args[0].ArrRef()
		*arr = append(*arr, c.args[1:]...)

		return count(len(*arr))
	}},
	{"pop", 1, 1, func(c *call) value.Value {
		if c.arr(0); !c.ok() {
			return value.NilValue
		}

		arr := c.args[0].ArrRef()
		n := len(*arr)

		if n == 0 {
			return c.fail(errors.ValueError, "got an empty array.")
		}

		last := (*arr)[n-1]
		(*arr)[n-1] = value.NilValue
		*arr = (*arr)[:n-1]

		return last
	}},
	// json fetches a URL and parses the body (object, array or primitive).
	{"json", 1, 1, func(c *call) value.Value {
		url := c.str(0)
		if !c.ok() {
			return value.NilValue
		}

		res, err := httpClient.Get(url)
		if err != nil {
			return c.fail(errors.IOError, "could not fetch %s: %s", url, err)
		}

		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			return c.fail(errors.IOError, "could not fetch %s: status code %d", url, res.StatusCode)
		}

		var body any
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
			return c.fail(errors.IOError, "could not parse JSON from %s: %s", url, err)
		}

		return value.FromGo(body)
	}},
}

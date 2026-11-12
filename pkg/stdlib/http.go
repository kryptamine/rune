package stdlib

import (
	"context"
	"io"
	"math"
	"net/http"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/value"
	"slices"
	"strings"
	"time"
)

const defaultTimeout = 30 * time.Second

var httpClient = http.Client{}

var HttpOptions = value.NewShape("HttpOptions", []value.Field{
	{Name: "headers"},
	{Name: "body"},
	{Name: "timeout"},
})

var HttpResponse = value.NewShape("HttpResponse", []value.Field{
	{Name: "status", Required: true},
	{Name: "headers", Required: true},
	{Name: "body", Required: true},
})

var httpNatives = []*Native{
	{"http.get", 1, 1, func(c *call) value.Value {
		url := c.str(0)
		if !c.ok() {
			return value.NilValue
		}

		res, body := c.fetch(http.MethodGet, url, nil)
		if !c.ok() {
			return value.NilValue
		}

		if res.StatusCode != http.StatusOK {
			return c.fail(errors.IOError, "could not fetch %s: status code %d", url, res.StatusCode)
		}

		return value.NewString(body)
	}},
	{"http.request", 2, 3, func(c *call) value.Value {
		method, url := c.str(0), c.str(1)
		if !c.ok() {
			return value.NilValue
		}

		var opts []value.Value
		if c.has(2) {
			v, err := Convert(c.token, HttpOptions, c.args[2])
			if err != nil {
				c.err = err
				return value.NilValue
			}

			opts = v.Fields()
		}

		res, body := c.fetch(strings.ToUpper(method), url, opts)
		if !c.ok() {
			return value.NilValue
		}

		headers := make(map[string]value.Value, len(res.Header))
		for k, vs := range res.Header {
			headers[strings.ToLower(k)] = value.NewString(strings.Join(vs, ", "))
		}

		v, fields := value.NewStruct(HttpResponse)
		fields[0] = count(res.StatusCode)
		fields[1] = value.NewObject(headers)
		fields[2] = value.NewString(body)

		return v
	}},
}

// fetch sends a request with opts, the fields of an HttpOptions or nil,
// and reads the whole response.
func (c *call) fetch(method, url string, opts []value.Value) (*http.Response, string) {
	timeout := defaultTimeout

	var body io.Reader

	var headers map[string]value.Value

	if opts != nil {
		if h, ok := c.option(HttpOptions, opts, 0, value.Object, "an object"); ok {
			headers = h.Obj()
		}

		if b, ok := c.option(HttpOptions, opts, 1, value.String, "a string"); ok {
			body = strings.NewReader(b.Str())
		}

		if t, ok := c.option(HttpOptions, opts, 2, value.Number, "a number"); ok {
			if t.Num() <= 0 || math.IsNaN(t.Num()) {
				c.fail(errors.ValueError, "expects a positive timeout, got %s.", helpers.Stringify(t))
			}

			timeout = time.Duration(min(t.Num(), 1e9) * float64(time.Second))
		}
	}

	if !c.ok() {
		return nil, ""
	}

	ctx, cancel := context.WithTimeout(c.host.ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		c.fail(errors.ValueError, "bad request: %s.", err)
		return nil, ""
	}

	names := make([]string, 0, len(headers))
	for k := range headers {
		names = append(names, k)
	}

	slices.Sort(names)

	for _, k := range names {
		v := headers[k]
		if !v.IsString() {
			c.fail(errors.TypeError, "expects header '%s' to be a string, got %s.", k, helpers.TypeName(v))
			return nil, ""
		}

		if strings.EqualFold(k, "host") {
			req.Host = v.Str()
		} else {
			req.Header.Set(k, v.Str())
		}
	}

	res, err := httpClient.Do(req)
	if err != nil {
		c.fail(errors.IOError, "could not fetch %s: %s", url, err)
		return nil, ""
	}

	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		c.fail(errors.IOError, "could not read %s: %s", url, err)
		return nil, ""
	}

	return res, bytesString(b)
}

package stdlib

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"maps"
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
	{Name: "stream", Default: value.False},
})

var HttpResponse = value.NewShape("HttpResponse", []value.Field{
	{Name: "status", Required: true},
	{Name: "headers", Required: true},
	{Name: "body", Required: true},
})

var httpNatives = []*Native{
	{"http.request", 2, 3, func(c *call) value.Value {
		method, url, opts := c.str(0), c.str(1), c.httpOptions(2)
		if !c.ok() {
			return value.NilValue
		}

		res, body := c.fetch(strings.ToUpper(method), url, opts)
		if !c.ok() {
			return value.NilValue
		}

		headers := value.NewHashMap(len(res.Header))
		for _, k := range slices.Sorted(maps.Keys(res.Header)) {
			_ = headers.Set(value.NewString(strings.ToLower(k)), value.NewString(strings.Join(res.Header[k], ", ")))
		}

		v, fields := value.NewStruct(HttpResponse)
		fields[0] = count(res.StatusCode)
		fields[1] = value.NewMap(headers)

		if opts.stream {
			fields[2] = c.host.open(&Stream{name: url, r: bufio.NewReader(body), closer: body})
		} else {
			defer body.Close()
			fields[2] = c.readBody(url, body)
		}

		return v
	}},
}

// httpOpts is an HttpOptions read for a request.
type httpOpts struct {
	headers [][2]string
	body    io.Reader
	timeout time.Duration
	stream  bool
}

func defaultHTTPOpts() httpOpts {
	return httpOpts{timeout: defaultTimeout}
}

// httpOptions reads argument i, if given, as an HttpOptions.
func (c *call) httpOptions(i int) httpOpts {
	o := defaultHTTPOpts()
	if !c.has(i) || !c.ok() {
		return o
	}

	f, ok := c.structArg(i, HttpOptions)
	if !ok {
		return o
	}

	if h, ok := c.option(HttpOptions, f, 0, value.Map, "a map"); ok {
		o.headers = c.stringEntries(h.HashMap(), "header")
	}

	if b := f[1]; b.Kind() == value.Bytes {
		o.body = bytes.NewReader(b.Bytes())
	} else if b, ok := c.option(HttpOptions, f, 1, value.String, "a string"); ok {
		o.body = strings.NewReader(b.Str())
	}

	if t, ok := c.option(HttpOptions, f, 2, value.Number, "a number"); ok {
		if t.Num() <= 0 || math.IsNaN(t.Num()) {
			c.fail(errors.ValueError, "expects a positive timeout, got %s.", helpers.Stringify(t))
		}

		o.timeout = time.Duration(min(t.Num(), 1e9) * float64(time.Second))
	}

	if st, ok := c.option(HttpOptions, f, 3, value.Bool, "a boolean"); ok {
		o.stream = st.Bool()
	}

	return o
}

func (c *call) readBody(url string, body io.Reader) value.Value {
	b, err := io.ReadAll(body)
	if err != nil {
		return c.fail(errors.IOError, "could not read %s: %s", url, err)
	}

	return value.NewString(bytesString(b))
}

// cancelBody is a response body that also ends its request.
type cancelBody struct {
	io.ReadCloser
	cancel func()
}

func (b cancelBody) Close() error {
	err := b.ReadCloser.Close()
	b.cancel()

	return err
}

// fetch sends a request and returns the response with its body, which
// the caller closes. The timeout covers the request and the reading of
// the body, unless it streams.
func (c *call) fetch(method, url string, o httpOpts) (*http.Response, io.ReadCloser) {
	ctx, cancel := context.WithCancel(c.host.ctx)
	timer := time.AfterFunc(o.timeout, cancel)

	res := c.send(ctx, method, url, o)
	if !c.ok() {
		timer.Stop()
		cancel()

		return nil, nil
	}

	if o.stream {
		timer.Stop()
	}

	return res, cancelBody{res.Body, func() { timer.Stop(); cancel() }}
}

func (c *call) send(ctx context.Context, method, url string, o httpOpts) *http.Response {
	req, err := http.NewRequestWithContext(ctx, method, url, o.body)
	if err != nil {
		c.fail(errors.ValueError, "bad request: %s.", err)
		return nil
	}

	for _, kv := range o.headers {
		if strings.EqualFold(kv[0], "host") {
			req.Host = kv[1]
		} else {
			req.Header.Set(kv[0], kv[1])
		}
	}

	res, err := httpClient.Do(req)
	if err != nil {
		c.fail(errors.IOError, "could not fetch %s: %s", url, err)
		return nil
	}

	return res
}

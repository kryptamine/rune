package stdlib

import (
	"io"
	"net/http"
	"rune/pkg/errors"
	"rune/pkg/value"
	"time"
)

var httpClient = http.Client{Timeout: 10 * time.Second}

var httpNatives = []*Native{
	{"http.get", 1, 1, func(c *call) value.Value {
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

		body, err := io.ReadAll(res.Body)
		if err != nil {
			return c.fail(errors.IOError, "could not read %s: %s", url, err)
		}

		return value.NewString(string(body))
	}},
}

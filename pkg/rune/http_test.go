package rune

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("X-Echo", r.Header.Get("X-Token"))
		w.Header().Add("X-Multi", "a")
		w.Header().Add("X-Multi", "b")

		switch r.URL.Path {
		case "/missing":
			w.WriteHeader(http.StatusNotFound)
		case "/slow":
			<-r.Context().Done()
			return
		}

		io.WriteString(w, r.Method+" "+string(body))
	}))
	defer srv.Close()

	src := strings.ReplaceAll(`
		let r = http.request("post", "URL/x", { headers: { "X-Token": "t1" }, body: "hi" });
		print r.status;
		print r.body;
		print r.headers["x-echo"];
		print r.headers["x-multi"];
		print type(r);

		let nf = http.request("GET", "URL/missing", HttpOptions {});
		print nf.status;
		print http.get("URL/");

		try { http.get("URL/missing"); } catch (e: IOError) { print "get failed"; }
		try { http.request("GET", "URL/slow", { timeout: 0.05 }); } catch (e: IOError) { print "timed out"; }
		try { http.request("GET", "URL/", { body: 1 }); } catch (e: TypeError) { print e.message; }
		try { http.request("GET", "URL/", { headers: { a: 1 } }); } catch (e: TypeError) { print e.message; }
		try { http.request("GET", "URL/", { timeout: 0 }); } catch (e: ValueError) { print e.message; }
		try { http.request("GET", "URL/", { bogus: 1 }); } catch (e: KeyError) { print e.message; }
		try { http.request("BAD METHOD", "URL/"); } catch (e: ValueError) { print "bad method"; }
	`, "URL", srv.URL)

	prog, errs := loadSource([]byte(src))
	if len(errs) > 0 {
		t.Fatal(errs)
	}

	var out bytes.Buffer
	if err := NewInterpreter(testStdio(&out)).Run(prog, nil); err != nil {
		t.Fatal(err)
	}

	want := `200
POST hi
t1
a, b
HttpResponse
404
GET 
get failed
timed out
http.request() expects HttpOptions.body to be a string, got number.
http.request() expects header 'a' to be a string, got number.
http.request() expects a positive timeout, got 0.
HttpOptions has no field 'bogus'.
bad method
`
	if out.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", out.String(), want)
	}
}

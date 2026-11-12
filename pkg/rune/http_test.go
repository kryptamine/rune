package rune

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
		case "/lines":
			io.WriteString(w, "a\nb\r\nc")
			return
		case "/slowbody":
			w.(http.Flusher).Flush()
			time.Sleep(100 * time.Millisecond)
			io.WriteString(w, "done")

			return
		}

		io.WriteString(w, r.Method+" "+string(body))
	}))
	defer srv.Close()

	src := strings.ReplaceAll(`
		let r = http.request("post", "URL/x", { headers: { "X-Token": "t1" }, body: "hi" });
		print(r.status);
		print(r.body);
		print(r.headers["x-echo"]);
		print(r.headers["x-multi"]);
		print(type(r));

		let nf = http.request("GET", "URL/missing", HttpOptions {});
		print(nf.status);
		print(http.get("URL/"));

		try { http.get("URL/missing"); } catch e is IOError { print("get failed"); }
		try { http.request("GET", "URL/slow", { timeout: 0.05 }); } catch e is IOError { print("timed out"); }
		try { http.request("GET", "URL/", { body: 1 }); } catch e is TypeError { print(e.message); }
		try { http.request("GET", "URL/", { headers: { a: 1 } }); } catch e is TypeError { print(e.message); }
		try { http.request("GET", "URL/", { timeout: 0 }); } catch e is ValueError { print(e.message); }
		try { http.request("GET", "URL/", { bogus: 1 }); } catch e is KeyError { print(e.message); }
		try { http.request("BAD METHOD", "URL/"); } catch e is ValueError { print("bad method"); }

		let raw = http.request("PUT", "URL/x", { body: b"\x01raw", stream: true });
		print(type(raw.body));
		print(raw.body.readAll());
		raw.body.close();
		for i, line in http.request("GET", "URL/lines", { stream: true }).body { print("{i}: {line}"); }
		print(http.request("GET", "URL/slowbody", { stream: true, timeout: 0.05 }).body.readText());
		try { http.request("GET", "URL/", { stream: 1 }); } catch e is TypeError { print(e.message); }
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
stream
b"PUT \x01raw"
0: a
1: b
2: c
done
http.request() expects HttpOptions.stream to be a boolean, got number.
`
	if out.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", out.String(), want)
	}
}

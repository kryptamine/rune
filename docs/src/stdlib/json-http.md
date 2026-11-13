# json and http

## json.parse

`json.parse(s)` parses JSON: an object, an array or a primitive. Objects
become objects, `null` becomes `nil`, and a number without a fraction or
an exponent is an integer. Malformed input is a `ValueError`.

JSON is full of `"` and `{`, so write it in a raw string, `r"""..."""`.

```rune
let user = json.parse(r"""{"name": "ann", "tags": ["admin"], "age": 31, "manager": null}""");
print(user);            // {name: "ann", tags: ["admin"], age: 31, manager: nil}
print(user.tags[0]);    // admin
print(user.manager);    // nil
print(json.parse("[1, 2.5]"));  // [1, 2.5]
print(json.parse("42"));        // 42
```

Calling a struct type checks the shape of what was parsed:

```rune
struct Order { id, total, items = [] }

let order = Order(json.parse(r"""{"id": 8, "total": 5}"""));
print(order);   // Order {id: 8, total: 5, items: []}
```

```rune
json.parse("not json"); // ValueError: json.parse() could not parse JSON: invalid character 'o' in literal null (expecting 'u')
```

## json.stringify

`json.stringify(v[, indent])` encodes `nil`, booleans, numbers, strings, arrays,
objects (keys in order), structs (fields in order), sets (as arrays) and maps
of string keys (as objects, in order), compactly unless `indent`, a number
of spaces or a string of spaces and tabs, at most 10, puts each element
on its own line. A function is a
`TypeError`, a non-finite number or a cycle a `ValueError`.

```rune
struct Order { id, total, items = [] }

print(json.stringify({ name: "ann", active: true, manager: nil }));
// {"name":"ann","active":true,"manager":null}
print(json.stringify(Order { id: 7, total: 19.99 }));
// {"id":7,"total":19.99,"items":[]}
print(json.stringify(set(["a", "b"])));   // ["a","b"]
print(json.stringify("say \"hi\""));      // "say \"hi\""
```

```rune
print(json.stringify({ name: "ann", tags: ["admin"] }, 2));
// {
//   "name": "ann",
//   "tags": [
//     "admin"
//   ]
// }
```

```rune
json.stringify({ onClick: () => nil }); // TypeError: json.stringify() cannot encode function.
```

## http.get

`http.get(url)` fetches a URL and returns the body as a string; a network
error or a status other than 200 is an `IOError`.

```rune
let todos = json.parse(http.get("https://api.example.com/todos")).todos;
print(todos.filter(t => !t.completed).map(t => t.todo));   // ["buy milk"]
```

```rune
try {
    http.get("https://api.example.com/missing");
} catch e is IOError {
    print(e.message);   // http.get() could not fetch https://api.example.com/missing: status code 404
}
```

## http.request

`http.request(method, url[, opts])` sends a request and returns
`HttpResponse { status, headers, body }`, header names lowercased and
repeated ones joined with `, `. Any status is a response; a network error
or a timeout is an `IOError`.

`opts` is an `HttpOptions { headers, body, timeout, stream }`, or an object
converted to one: `headers` an object of strings, `body` a string or bytes,
`timeout` seconds (default 30).

```rune
let r = http.request("POST", "https://api.example.com/users", {
    headers: { "Content-Type": "application/json" },
    body: json.stringify({ name: "ann" }),
    timeout: 5,
});

print(r.status);                  // 201
print(r.headers["location"]);     // /users/42
print(json.parse(r.body).id);     // 42

if r.status != 201 throw IOError { message: "status {r.status}" };
```

```rune
let r = http.request("GET", "https://api.example.com/missing");
print(r.status);   // 404
print(json.parse(r.body).error);   // not found
```

With `stream: true`, `body` is a [Stream](streams.md) to read and close,
and the timeout covers only getting the response, so a large download is
never held whole:

```rune
let r = http.request("GET", "https://api.example.com/export.csv", { stream: true });
for i, line in r.body {
    if i == 0 continue;
    let [id, total] = line.split(",");
    print("order {id}: {total}");
}
r.body.close();
// order 1: 30
// order 2: 120
```

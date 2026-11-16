# json and http

| Function | Does |
|---|---|
| `json.parse(s)` | parses JSON text into a value |
| `json.stringify(v[, indent])` | encodes a value as JSON text |
| `http.request(method, url[, opts])` | sends any request, returns an `HttpResponse` |

## json.parse

`json.parse(s)` turns JSON text into Rune values:

| JSON | Rune |
|---|---|
| object | object, keys in order |
| array | array |
| string, `true`, `false` | string, boolean |
| number | integer if it has no fraction or exponent, else float |
| `null` | `nil` |

JSON is full of `"` and `{`, so write it in a raw string, `r"""..."""`,
where neither is special.

```rune
let user = json.parse(r"""{"name": "ann", "tags": ["admin"], "age": 31, "manager": null}""");
print(user);              // {name: "ann", tags: ["admin"], age: 31, manager: nil}
print(user.tags[0]);      // admin
print(user.manager);      // nil
print(user?.email ?? "-"); // -

print(json.parse("[1, 2.5]"));   // [1, 2.5]
print(json.parse("42"));         // 42
```

Spreading the result into a [struct](../data/structs.md) literal checks
its shape: every field without a default must be there, and no other key
may be.

```rune
struct Order { id, total, items = [] }

let order = Order { ...json.parse(r"""{"id": 8, "total": 5}""") };
print(order);         // Order {id: 8, total: 5, items: []}
print(order.items);   // []
```

```rune
struct Order { id, total, items = [] }
Order { ...json.parse(r"""{"id": 8}""") }; // KeyError: Missing field 'total' of Order.
```

Text that is not JSON is a `ValueError`:

```rune
json.parse("not json"); // ValueError: json.parse() could not parse JSON: invalid character 'o' in literal null (expecting 'u')
```

## json.stringify

`json.stringify(v)` encodes `v` as compact JSON. It takes:

- `nil`, booleans, numbers, strings and arrays;
- objects, keys in order, and structs, fields in order;
- maps with string keys, as objects, and sets, as arrays.

```rune
struct Order { id, total, items = [] }

print(json.stringify({ name: "ann", active: true, manager: nil }));
// {"name":"ann","active":true,"manager":null}
print(json.stringify(Order { id: 7, total: 19.99 }));
// {"id":7,"total":19.99,"items":[]}
print(json.stringify(set(["a", "b"])));   // ["a","b"]
print(json.stringify("say \"hi\""));      // "say \"hi\""
```

With `indent`, a number of spaces or a string of spaces and tabs (at most
10), each element goes on its own line:

```rune
print(json.stringify({ name: "ann", tags: ["admin"] }, 2));
// {
//   "name": "ann",
//   "tags": [
//     "admin"
//   ]
// }
```

A function has no JSON form. It is a `TypeError`, here in a struct field:

```rune
struct Button { label, onClick }
json.stringify(Button { label: "save", onClick: () => nil }); // TypeError: json.stringify() cannot encode function.
```

So is a map with keys other than strings:

```rune
json.stringify(map([[1, "one"]])); // TypeError: json.stringify() cannot encode a map with keys other than strings.
```

An infinite or NaN number, or a value nested more than 1000 deep, is a
`ValueError`:

```rune
json.stringify({ ratio: math.INF }); // ValueError: json.stringify() cannot encode a non-finite number.
```

## http.request

`http.request(method, url[, opts])` sends a request and returns an
`HttpResponse { status, headers, body }`:

- `status` — the status code; any status is a response, not an error;
- `headers` — an object, names lowercased, repeated headers joined with
  `", "`;
- `body` — the body as a string.

A network error or a timeout is an `IOError`.

`opts` is an `HttpOptions { headers, body, timeout, stream }`, or an
object with any of those keys:

- `headers` — an object of strings;
- `body` — a string or bytes;
- `timeout` — in seconds, 30 by default;
- `stream` — `true` to read the body as a [stream](streams.md).

The examples below talk to a made-up `api.example.com`; the output shows
what such a server would answer.

```rune
let todos = json.parse(http.request("GET", "https://api.example.com/todos").body).todos;
print(todos.filter(t => !t.done).map(t => t.title));   // ["buy milk"]
```

Creating a user with a JSON body:

```rune
let r = http.request("POST", "https://api.example.com/users", {
    headers: { "Content-Type": "application/json" },
    body: json.stringify({ name: "ann" }),
    timeout: 5,
});

print(r.status);                 // 201
print(r.headers["location"]);    // /users/42
print(json.parse(r.body).id);    // 42
```

A status like 404 is a normal response, to check yourself:

```rune
let r = http.request("GET", "https://api.example.com/users/99");
if r.status != 200 {
    print("failed: {r.status} {json.parse(r.body).error}");   // failed: 404 not found
}
```

With `stream: true`, `body` is a [Stream](streams.md) to read and then
close. The timeout covers only getting the response, so a large download
is read piece by piece and never held whole:

```rune
let r = http.request("GET", "https://api.example.com/export.csv", { stream: true });
for i, line in r.body {
    if i == 0 { continue }
    let [id, total] = line.split(",");
    print("order {id}: {total}");
}
r.body.close();
// order 1: 30
// order 2: 120
```

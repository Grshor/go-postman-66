# go-postman-66

Translate your Go route registrations into a ready-to-import
[Postman Collection v2.1](https://schema.getpostman.com/json/collection/v2.1.0/docs/index.html)
— **without running the server**.

`go-postman-66` parses your source with `go/ast`, finds route registrations,
picks up handler doc comments as request descriptions, converts path patterns
to Postman path variables, and writes a deterministic JSON collection.

> **Origin:** this is an open-source reimplementation of a handler→Postman
> documentation codegen I originally built for the Kangaroo marketplace
> backend, where it kept API docs and collections in sync with the code.

## Install

```sh
go install github.com/Grshor/go-postman-66/cmd/go-postman-66@latest
```

## Usage

```sh
go-postman-66 -dir ./cmd/api -out collection.json -name "My API" -base-url https://api.example.com
# go-postman-66: 14 routes in 2 package(s) → collection.json
```

| flag | default | meaning |
| --- | --- | --- |
| `-dir` | `.` | directory to scan recursively (vendor/testdata/.git skipped) |
| `-out` | `postman_collection.json` | output file; `-` writes to stdout |
| `-name` | `API` | collection name |
| `-base-url` | `http://localhost:8080` | value of the `{{base_url}}` collection variable |

Exit codes: `0` success, `2` bad flags, unreadable directory or write failure.
Warnings (e.g. skipped non-literal patterns) go to stderr and never fail the run.

## What it detects

| registration style | example | notes |
| --- | --- | --- |
| `net/http` ServeMux, incl. **Go 1.22 method patterns** | `mux.HandleFunc("GET /users/{id}", h)` | `{id}` → `:id` path variable; method taken from the pattern |
| `Handle`/`HandleFunc` on any mux variable | `r.HandleFunc("/orders", createOrder)` | gorilla/chi/stdlib share this shape; `.Methods()` chains are **not** resolved in v0.1 (defaults to any-method request) |
| method-style routers: gin, chi, echo | `r.GET("/items", h)` | verbs in CAPS |
| fiber-style routers | `app.Get("/tasks/:id", h)` | verbs capitalized; chi and fiber are indistinguishable at the call site and share one label |
| catch-alls | `r.Any("/p", h)` | request is marked "may accept any method" |

Handler doc comments (`// Lists all users…` above the function) become request
descriptions in Postman.

## Limitations (v0.1, on purpose)

- Path patterns must be string literals; anything computed is skipped with a
  warning on stderr.
- Route groups/prefixes (`gin.Group`, nested muxes) are not flattened yet.
- Anonymous handlers (`FuncLit`) carry no description — there is no function to
  read a comment from.

## Example output

```json
{
  "name": "GET /users/{id}",
  "request": {
    "method": "DELETE",
    "url": {
      "raw": "{{base_url}}/users/{id}",
      "host": ["{{base_url}}"],
      "path": ["users", ":id"],
      "variable": [{ "key": "id" }]
    },
    "description": "Removes a user and invalidates their sessions."
  }
}
```

The output is deterministic: same input, byte-identical collection — safe to
commit and diff in review.

## Development

```sh
go test ./...
UPDATE_GOLDEN=1 go test ./internal/postman -run TestGolden   # refresh the golden corpus
```

The scanner's own test fixture lives in `testdata/testapp` — one file exercising
every supported registration style.

## License

GPL-3.0

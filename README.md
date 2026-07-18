# curlmoon

A pocket-sized Postman for the terminal — built for Termux, but it runs anywhere Go does.

curlmoon is a single-binary TUI HTTP client: collections, environments, auth
helpers, request history, and `{{variable}}` substitution, all driven from
the keyboard.

```
┌──────────────┬──────────────────────────────────────────────────┐
│  Collections │  [GET] https://api.example.com/users/{{id}}       │
│                                                                   │
│  ▸ My API    │  Headers │ Body │ Auth │ Params                   │
│    GET /get  ├──────────────────────────────────────────────────┤
│    POST /post│  Authorization: Bearer {{token}}                  │
│  Environments│                                                   │
│  History     ├──────────────────────────────────────────────────┤
│              │  200 OK   142ms   1.2KB                           │
│              │  { "id": 1, "name": "..." }                       │
├──────────────┴──────────────────────────────────────────────────┤
│  Tab cycle panels · Ctrl+S,U,B,E jump panel · Ctrl+/ help · q quit│
└────────────────────────────────────────────────────────────────┘
```

## Features

- Collections of requests, persisted as Postman v2.1-compatible JSON
- Environments with `{{variable}}` substitution in the URL, headers, and body
- Auth helpers: Basic, Bearer, API Key, OAuth2 (token)
- GraphQL body type (query + variables)
- Pre-request/test scripts (`pm.*` API, powered by [goja](https://github.com/dop251/goja))
- Code generation: curl, Go, Python, JavaScript (`Ctrl+G`)
- Export (`x`) / import (`i`) collections from the sidebar, on top of `-c`/`--collection` at startup
- Request history (last 50 sends, reloadable)
- JSON syntax-highlighted, scrollable response viewer
- Session auto-save/restore between runs
- Configurable keybindings (`~/.curlmoon/keybindings.json`)
- Light/dark theme (`Ctrl+T`)
- A full keybinding reference built in (`Ctrl+/`)

## Install

Requires Go 1.24+.

```bash
git clone https://github.com/<you>/curlmoon
cd curlmoon
go build -o $PREFIX/bin/curlmoon ./cmd/curlmoon   # Termux
# or, on any other system:
go build -o /usr/local/bin/curlmoon ./cmd/curlmoon
```

## Run

```bash
curlmoon
```

Data is stored under `~/.curlmoon/` (collections, environments, history,
session). curlmoon starts empty by default — bring your own collections and
environments, or pass `--demo` to explore with a few example collections.

```bash
curlmoon --demo                          # try it with example collections (httpbin.org, etc.)
curlmoon -c my-api.postman_collection.json   # import a Postman v2.1 collection
curlmoon -e prod.env                     # import a .env file as an environment (made active)
curlmoon -c a.json -c b.json -e local.env --demo   # combine as needed
```

Imported collections (`-c`) and environments (`-e`) are persisted into
`~/.curlmoon/`, same as anything created from the TUI. `--demo` collections
are **not** persisted — they exist only for that run, so they won't clutter
your sidebar on the next launch.

See [docs/USAGE.md](docs/USAGE.md) for the full guide: panels, keybindings,
variables, auth, and the collection file format.

## Development

```bash
go test ./... -v -cover
go build ./...
```

The TUI is built on [gocui](https://github.com/jesseduffield/gocui); there's
no dependency on a terminal emulator beyond a standard ANSI-capable one.

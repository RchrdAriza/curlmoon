# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

curlmoon is a single-binary terminal HTTP client (a "pocket-sized Postman") built with Go 1.25 and the [gocui](https://github.com/jesseduffield/gocui) TUI library. It targets Termux but runs anywhere Go does. Module path is `curlmoon`.

## Commands

```bash
go build -o curlmoon ./cmd/curlmoon   # build the binary
go test ./...                         # run all tests
go test ./... -v -cover               # verbose with coverage (as in README)
go test ./internal/tui -run TestName  # run a single test / package
go vet ./...                          # vet
```

Run it: `./curlmoon` (empty by default), `./curlmoon --demo` (example collections, not persisted), `-c file.postman_collection.json` to import a collection, `-e file.env` to import a dotenv as the active environment. All flags are repeatable.

Runtime data lives under `~/.curlmoon/` (collections, environments, history, session, `keybindings.json`, theme).

## Architecture

Entry point `cmd/curlmoon/main.go` parses flags, constructs a `collection.Store` + `environment.Store` (both rooted at `~/.curlmoon`), performs any imports, then hands off to `tui.Run`.

The `internal/` packages are layered; the TUI depends on the domain packages, not vice versa:

- **`collection`** — Postman v2.1-compatible schema (`schema.go`), a JSON-file-backed `Store` under `~/.curlmoon/collections`, and session save/restore (`session.go`). This is the source of truth for requests.
- **`environment`** — environment `Store`, `.env` import, and the `{{variable}}` substitution applied to URL/headers/body before a request is sent.
- **`httpclient`** — plain `Request`/`Response` structs and `Execute`; no TUI or persistence knowledge.
- **`script`** — runs pre-request and test scripts through [goja](https://github.com/dop251/goja), exposing a minimal `pm.*` API (env get/set, request/response inspection, `pm.test` assertions). Mutates the env map in place so callers reuse it for `{{variable}}` resolution.
- **`codegen`** — generates curl / Go / Python / JavaScript snippets from a request (bound to Ctrl+G in the TUI).
- **`history`** — last-50 sends, reloadable.
- **`config`** — loads user keybindings and theme from JSON under `~/.curlmoon`, merging a user file over `DefaultKeymap()`. Action names (e.g. `sendRequest`) are the source of truth; keys are configurable.

### The TUI layer (`internal/tui`)

The central design constraint: **`App` (in `app.go`) is deliberately gocui-free plain data plus pure methods**, so nearly all behavior is exercised in tests without a real terminal (see `app_test.go`, `phase2/4/5_test.go`, `collection_test.go`). Keep new logic on `App` methods that don't touch gocui.

gocui is wired only in a thin shell:
- `run.go` — builds the `App`, creates the gocui GUI, sets the editor/layout/keybindings/mouse, runs the main loop. Contains important notes on why certain gocui modes are set (`InputEsc`, deferred content-focus, mouse reporting).
- `render.go` / `views.go` — the layout function and per-view rendering; `layout()` re-derives cursor visibility every frame from the focused view.
- `keybindings.go` (action → handler wiring, reads `config.Keymap`), `mouse.go`, `editor.go`.
- Feature panels/overlays: `auth.go`, `graphql.go`, `scripts.go`, `codegen_view.go`, `filebrowser.go` (import/export via filesystem navigation), `json.go` (syntax-highlighted response viewer), `kv.go` (key/value editors), `colors.go`/`borders.go` (theming).

`App` state groups a `sidebar []sidebarEntry` (a flattened tree spanning collections, environments, and history — each entry's `section` field tells which slice it indexes into) alongside the editor buffers, and pointers back to the `Store`/`environment.Store`/`history.Store`. Panels are `panelSidebar`/`panelURL`/`panelResponse`; the request editor has tabs Headers/Body/Auth/Params/Scripts.

## Interactive (TTY) testing

When a task requires exercising the TUI in a real terminal — panel focus, per-frame cursor visibility, `InputEsc` behavior, mouse reporting, view editors, or anything `go test ./...` can't cover because there's no pseudo-terminal — delegate to the `tty-tester` subagent (`.claude/agents/tty-tester.md`) instead of trying to drive the TUI directly. It uses tmux to spin up a pseudo-terminal, send keys, and read the rendered pane.

## Notes

- Tests named `phaseN_test.go` correspond to development phases, not a package boundary — they all test the `tui` package.
- `TODO.md` (in Spanish) tracks pending UI polish work.
- Full user-facing guide, keybindings, and the collection file format: `docs/USAGE.md`.

# curlmoon usage guide

## CLI flags

| Flag | Repeatable | Effect |
|---|---|---|
| `-c`, `--collection FILE` | yes | Import a Postman v2.1 collection JSON file, persisted to `~/.curlmoon/collections/` (overwrites a collection with the same `info.name`) |
| `-e`, `--env FILE` | yes | Import a `.env` file (`KEY=VALUE` per line) as an environment named after the file, persisted to `~/.curlmoon/environments/`. The last one imported becomes the active environment |
| `--demo` | — | Add example collections (httpbin.org, JSON Placeholder, GitHub API) for this run only — not persisted to `~/.curlmoon/` |

`.env` files support blank lines, `#` comments, an optional leading
`export `, and single/double-quoted values.

## Layout

```
┌──────────────┬──────────────────────────────────────────┐
│ [^S]         │ [^U] URL bar                              │
│  Collections │──────────────────────────────────────────│
│              │  Headers │ Body │ Auth │ Params            │
│              │[^B]───────────────────────────────────────│
│              │  Content editor (for the active tab)       │
│              │──────────────────────────────────────────│
│              │[^E] Response                                │
├──────────────┴──────────────────────────────────────────┤
│  footer: contextual hints for whatever has focus          │
└────────────────────────────────────────────────────────────┘
```

Every panel's border shows the `Ctrl+<letter>` chord that jumps straight to
it (`^S`, `^U`, `^B`, `^E`) — see [Panels & jumping](#panels--jumping).

## Panels & jumping

| Panel | Jump | What it holds |
|---|---|---|
| Sidebar | `Ctrl+S` | Collections, Environments, History |
| URL bar | `Ctrl+U` | Method + URL, with the method/tab strip below it |
| Content editor | `Ctrl+B` | The Headers/Body/Auth/Params tab currently selected |
| Response | `Ctrl+E` | Status, timing, size, headers, and body |

`Tab` cycles sidebar → URL → response (it skips the content editor —
`Ctrl+B` or `Enter` from the URL bar is how you get in). Press `Ctrl+/`
anywhere to open a full keybinding reference; `Esc` or `Ctrl+/` again closes
it.

Numbered jumps (`Alt+1`..`Alt+4`) were the original scheme but got dropped:
Alt-combo detection and a lone `Esc` resolving to "cancel" are mutually
exclusive in the terminal's input mode, and `Esc` had to win so it can
cancel prompts. `Ctrl+<letter>` is a raw single-byte control code every
terminal sends identically, so it doesn't depend on either input mode.

## Sidebar

- `↑↓` or `j`/`k` — navigate
- `Enter` — open a request (loads it into the URL bar/editor) or
  expand/collapse a folder
- `n` — new collection (or new environment, if the Environments section is
  selected)
- `a` — add a request to the selected collection
- `r` — rename the selected collection/request/environment
- `d` — delete the selected item (confirm with `y`/`n`)
- `v` — edit an environment's variables (only on an Environment entry)
- `x` — export the selected collection to a file path you enter (Postman
  v2.1 JSON, including any GraphQL body and pre-request/test scripts)
- `i` — import a collection from a file path you enter (same as `-c` at
  startup, but without restarting)

The sidebar has three sections:

- **Collections** — your saved requests, grouped into folders you can
  collapse/expand with `Enter`.
- **Environments** — named sets of `{{variable}}` values. One can be marked
  active; its variables are what `{{...}}` tokens resolve against when you
  send a request.
- **History** — the last 50 sent requests. `Enter` reloads one into the
  editor; nothing here is editable.

## URL bar

- `↑↓`, or `Ctrl+K`/`Ctrl+J` — cycle the HTTP method (GET, POST, PUT, PATCH,
  DELETE, HEAD, OPTIONS)
- `←→`, `Home`, `End` — move the cursor while typing the URL
- `Ctrl+P`/`Ctrl+N` — switch tab (Headers → Body → Auth → Params → Scripts →
  back)
- `Ctrl+Y` — cycle the body type (`none` → `JSON` → `raw` → `form-data` →
  `x-www-urlencoded` → `GraphQL` → back), only while the Body tab is active
  — the content editor's title shows the current one, e.g. `Body (JSON)`
- `Enter` — jump into the content editor for the active tab
- `Ctrl+R` — send the request (also works from the content editor)
- `Ctrl+G` — open the code generation overlay for the current request
- `Ctrl+T` — toggle light/dark theme

## Content editor (Headers / Body / Auth / Params / Scripts)

Headers, Params, and Auth all share one format: one `Key: Value` pair per
line. Body is free-form text, interpreted according to the body type
selected for the request (`none`, `JSON`, `raw`, `form-data`,
`x-www-urlencoded`, `GraphQL`).

- `Esc` — save the buffer and return to the URL bar
- `Ctrl+R` — send the request without leaving the editor

Sending a request injects a `Content-Type` header automatically for JSON,
raw, and urlencoded bodies if you haven't already set one yourself.

### Auth tab

The first line selects the auth type; the rest are its fields:

```
Type: Basic
Username: alice
Password: secret
```

```
Type: Bearer
Token: {{api_token}}
```

```
Type: API Key
Key: X-API-Key
Value: {{api_key}}
```

```
Type: OAuth2
Token: {{access_token}}
```

`Type: None` (the default) sends no auth header.

### Params tab

Each `Key: Value` line becomes a URL query parameter, appended (or merged)
onto whatever's in the URL bar when the request is built — you don't need
to hand-edit the query string.

### GraphQL body

Select `GraphQL` as the body type (cycle body types the same way you cycle
anything else on that tab) and write the query, optionally followed by a
`### variables` line and a JSON object:

```
query GetUser($id: ID!) {
  user(id: $id) { id name }
}

### variables
{ "id": 1 }
```

curlmoon wraps this into the `{"query": ..., "variables": ...}` POST body a
GraphQL endpoint expects, and sends `Content-Type: application/json`
automatically.

### Scripts tab

Two sections, delimited the same way as GraphQL variables:

```
### pre-request
pm.environment.set("token", "abc123");

### test
pm.test("status is 200", () => pm.response.code === 200);
pm.test("has id", () => pm.response.json().id !== undefined);
```

The pre-request script runs before the request is built (its
`pm.environment.set` calls affect `{{variable}}` resolution for *this send
only* — they aren't written back to the environment file). The test script
runs after the response comes back. Available API:

| Call | Effect |
|---|---|
| `pm.environment.get(key)` / `.set(key, value)` | Read/write the active environment's variables (this send only) |
| `pm.variables.get(key)` / `.set(key, value)` | Scratch variables, local to this run |
| `pm.request.headers.get(key)` | Read a header from the request about to be sent (test script only) |
| `pm.response.code`, `.status`, `.headers.get(key)`, `.text()`, `.json()` | Inspect the response (test script only) |
| `pm.test(name, fn)` | Record a pass/fail: `fn` throwing, or returning `false`, marks it failed |

Results show up under the response body as `✓ N passed` / `✗ N failed`,
with failure details listed underneath. A malformed script shows as
"Script error: ..." instead.

## Generate code

`Ctrl+G` opens an overlay with the current request (method, URL, headers,
body — variables resolved, scripts *not* run) rendered as a ready-to-run
snippet. `Tab`/`Backspace` cycle through curl, Go, Python, and JavaScript;
`Ctrl+G` again closes it.

## Variables

Any `{{name}}` token in the URL, headers, or body is resolved against the
active environment's variables at send time. If no environment is active,
or the variable isn't defined, the token is left as-is. Manage variables
from the sidebar's Environments section (`v` to edit, one `Key: Value` per
line).

## Response panel

- `↑↓` — scroll a line at a time
- `PgUp`/`PgDn` — scroll a page at a time

Shows status code, elapsed time, response size, a preview of the response
headers, and the body (pretty-printed and syntax-highlighted for JSON).

## Prompts

Creating, renaming, or deleting things opens a small centered prompt:

- `Enter` — confirm
- `Esc` — cancel
- `y`/`n` — confirm/cancel a delete confirmation specifically

## Quitting

`q` or `Ctrl+C` saves the current session and exits. `Ctrl+C` always works,
even if `keybindings.json` reassigns `quit` to something else.

## Configurable keybindings

Every binding listed above (except `Ctrl+C`, and the vim-style `j`/`k` and
`Ctrl+K`/`Ctrl+J` aliases) can be reassigned by creating
`~/.curlmoon/keybindings.json`:

```json
{
  "sendRequest": { "key": "ctrl+x" },
  "quit": { "key": "ctrl+q" }
}
```

Only the actions you list are overridden — anything missing keeps its
default. Key strings look like `"q"`, `"ctrl+r"`, `"up"`, `"esc"`,
`"pgdn"`, `"ctrl+/"`. Changes take effect on the next launch, and the
`Ctrl+/` help overlay always reflects whatever is actually bound. An
invalid key string or a broken JSON file falls back silently to the
default for that action.

## Theme

`Ctrl+T` toggles between the dark (default) and light color themes and
remembers the choice in `~/.curlmoon/config.json`. gocui only has the
terminal's basic 16-color palette to work with, so "light" mostly means
swapping the muted/foreground color for better contrast on a light
background — it's not a full re-skin.

## Storage

Everything lives under `~/.curlmoon/`:

| Path | Contents |
|---|---|
| `collections/*.json` | Your collections, Postman v2.1-compatible |
| `environments/*.json` | Environments and their variables |
| `history.json` | The last 50 sent requests |
| `session.json` | The editor state (method, URL, tabs, active request) restored on next launch |
| `keybindings.json` | Optional keybinding overrides — see [Configurable keybindings](#configurable-keybindings) |
| `config.json` | The active theme (`Ctrl+T`) |

curlmoon starts with nothing seeded by default. Pass `--demo` to try it with
a couple of example collections for that run only (not persisted), or
`-c`/`-e` to import your own, which do get saved to `~/.curlmoon/` — see
[CLI flags](#cli-flags).

## Full keybinding reference

Press `Ctrl+/` inside the app at any time — it lists every binding above,
grouped by context, without needing to leave the terminal.

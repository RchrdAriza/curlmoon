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
- `Ctrl+P`/`Ctrl+N` — switch tab (Headers → Body → Auth → Params → back)
- `Enter` — jump into the content editor for the active tab
- `Ctrl+R` — send the request (also works from the content editor)

## Content editor (Headers / Body / Auth / Params)

Headers, Params, and Auth all share one format: one `Key: Value` pair per
line. Body is free-form text, interpreted according to the body type
selected for the request (`none`, `JSON`, `raw`, `form-data`,
`x-www-urlencoded`).

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

`q` or `Ctrl+C` saves the current session and exits.

## Storage

Everything lives under `~/.curlmoon/`:

| Path | Contents |
|---|---|
| `collections/*.json` | Your collections, Postman v2.1-compatible |
| `environments/*.json` | Environments and their variables |
| `history.json` | The last 50 sent requests |
| `session.json` | The editor state (method, URL, tabs, active request) restored on next launch |

curlmoon starts with nothing seeded by default. Pass `--demo` to try it with
a couple of example collections for that run only (not persisted), or
`-c`/`-e` to import your own, which do get saved to `~/.curlmoon/` — see
[CLI flags](#cli-flags).

## Full keybinding reference

Press `Ctrl+/` inside the app at any time — it lists every binding above,
grouped by context, without needing to leave the terminal.

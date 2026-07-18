# curlmoon — Plan de desarrollo

Postman TUI de bolsillo para Termux. Stack: **Go + Bubble Tea**.

## Stack

| Capa | Tecnología |
|------|-----------|
| Lenguaje | Go 1.25+ |
| TUI Framework | [gocui](https://github.com/jesseduffield/gocui) (lazygit) + termbox |
| HTTP | `net/http` estándar |
| Scripting | [goja](https://github.com/dop251/goja) |
| Storage | Archivos JSON en `~/.curlmoon/` |

## Layout

```
┌──────────────────────────────────────────────────────────┐
│  [GET] [POST] [PUT]...  │  URL: ___________________     │
├──────────────┬───────────────────────────────────────────┤
│              │  Headers │ Body │ Auth │ Params           │
│  SIDEBAR     ├───────────────────────────────────────────┤
│  Colección   │  Request editor                           │
│    ├─ Req1   │                                           │
│    ├─ Req2   │  [Ctrl+R: Send]                           │
│    └─ Req3   │                                           │
├──────────────┴───────────────────────────────────────────┤
│  Status: 200 OK │ Time: 234ms │ Size: 1.2KB              │
├──────────────────────────────────────────────────────────┤
│  Response body (JSON pretty-printed, scrollable)         │
│  Headers expandables                                     │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

## Fases

### Fase 1 — Esqueleto funcional ✅
- [x] `go mod init`, estructura de carpetas
- [x] HTTP client con `net/http`
- [x] TUI 3 paneles (sidebar, request, response)
- [x] Sidebar con colecciones hardcodeadas
- [x] Request: URL input, selector de método, tabs
- [x] Response: status + headers + body scrollable
- [x] Tests: 25 (unitarios + integración), ~79% cobertura

### Fase 2 — Request editor completo ✅
- [x] Editor de headers (tabla key-value con KeyValueEditor)
- [x] Editor de body (none / JSON / raw con textarea)
- [x] Query params editor que modifica la URL
- [x] JSON syntax highlighting en response
- [x] Tests: 39 tests, 73-83% cobertura

### Fase 3 — Colecciones y persistencia ✅
- [x] Schema JSON para colecciones (compatible Postman v2.1)
- [x] Guardar/cargar en `~/.curlmoon/collections/`
- [x] CRUD: crear/renombrar/eliminar colecciones (teclas `n`/`a`/`r`/`d` en el sidebar)
- [x] Sidebar navegable con colecciones reales
- [x] Auto-guardado de sesión (`~/.curlmoon/session.json`, restaurada al iniciar)
- [x] `cmd/curlmoon/main.go` — entrypoint que faltaba, ahora arranca con persistencia real
- [x] Tests: 24 tests nuevos (internal/collection + wiring TUI), cobertura 80%/71%

### Fase 4 — Features de Postman ✅
- [x] Variables `{{variable}}` en URL/headers/body (resueltas contra el entorno activo al enviar)
- [x] Editor de entornos (crear, renombrar, borrar, activar/desactivar, editar variables — sección "Environments" en el sidebar, `n`/`r`/`d`/`v`)
- [x] Auth helpers (None / Basic / Bearer / API Key / OAuth2) vía tab "Auth" editable con formato `Key: Value`
- [x] Historial de requests (sección "History" en el sidebar, hasta 50 entradas, Enter para recargar)
- [x] Tests: nuevos paquetes `internal/environment` y `internal/history` + wiring en `internal/tui`

### Fase 5 — Power features ✅
- [x] GraphQL (query + variables) — nuevo tipo de Body "GraphQL" con formato `query` + `### variables` + JSON, persistido como `Body.graphql` (Postman v2.1)
- [x] Pre-request / Test scripts con goja (API `pm.*`) — tab "Scripts" (`### pre-request` / `### test`), ejecutados vía `internal/script`, resultados mostrados en la respuesta
- [x] Code generation (curl, Go, Python, JS) — overlay `Ctrl+G` (`internal/codegen`), cicla lenguaje con Tab/Backspace
- [x] Export/Import collection v2.1 — `x`/`i` en el sidebar (`Store.Export`, `Store.Import` ya existente); scripts y GraphQL se conservan en el round-trip
- [x] Keybindings configurables — `~/.curlmoon/keybindings.json`, ver `internal/config`; el overlay de ayuda (`Ctrl+/`) refleja las teclas configuradas
- [x] Temas claro/oscuro — `Ctrl+T`, persistido en `~/.curlmoon/config.json` (`internal/config`)

## Testing

```bash
go test ./... -v -cover
```

| Tipo | Cobertura |
|------|-----------|
| Unit tests | HTTP client, collection manager, env resolver |
| Table-driven | Cada body type, auth type, método HTTP |
| Integration | Servidores mock con `httptest` |
| TUI smoke | Bubble Tea test mode |

## Distribución

```bash
pkg install golang git
git clone https://github.com/tuuser/curlmoon
cd curlmoon
go build -o $PREFIX/bin/curlmoon ./cmd/curlmoon/
```

Binario único ~7MB (stripped).

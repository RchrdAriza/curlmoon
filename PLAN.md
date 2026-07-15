# curlmoon — Plan de desarrollo

Postman TUI de bolsillo para Termux. Stack: **Go + Bubble Tea**.

## Stack

| Capa | Tecnología |
|------|-----------|
| Lenguaje | Go 1.22+ |
| TUI Framework | [Bubble Tea](https://github.com/charmbracelet/bubbletea) |
| Estilos | [Lipgloss](https://github.com/charmbracelet/lipgloss) |
| Componentes | [Bubbles](https://github.com/charmbracelet/bubbles) |
| HTTP | `net/http` estándar |
| Scripting | [goja](https://github.com/dop251/goja) (Fase 5) |
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

### Fase 2 — Request editor completo
- [ ] Editor de headers (tabla key-value)
- [ ] Editor de body (none / form-data / x-www-form-urlencoded / JSON / raw)
- [ ] Query params editor
- [ ] JSON syntax highlighting en response
- [ ] Tests

### Fase 3 — Colecciones y persistencia
- [ ] Schema JSON para colecciones (compatible Postman v2.1)
- [ ] Guardar/cargar en `~/.curlmoon/collections/`
- [ ] CRUD: crear/renombrar/eliminar colecciones
- [ ] Sidebar navegable con colecciones reales
- [ ] Auto-guardado de sesión
- [ ] Tests

### Fase 4 — Features de Postman
- [ ] Variables `{{variable}}` en URL/headers/body
- [ ] Editor de entornos (crear, activar, desactivar)
- [ ] Auth helpers (None / Basic / Bearer / API Key / OAuth2)
- [ ] Historial de requests
- [ ] Tests

### Fase 5 — Power features
- [ ] GraphQL (query + variables)
- [ ] Pre-request / Test scripts con goja (API `pm.*`)
- [ ] Code generation (curl, Go, Python, JS)
- [ ] Export/Import collection v2.1
- [ ] Keybindings configurables
- [ ] Temas claro/oscuro

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

# url-shortener

A lightweight URL shortener REST API written in Go. It maps short paths to target URLs, supports optional expiration (TTL), and tracks per-link usage statistics (hit count, last accessed time).

## Features

- **Create / read / delete** short link rules via a JSON REST API
- **Redirect** requests for a short path to its target URL (HTTP 302)
- **TTL support** — links can be created with an optional expiration duration; expired links stop redirecting automatically
- **Usage stats** — hit count and last-accessed timestamp per short link
- **SQLite storage** via [`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite) (pure Go driver, no CGO required)
- **Unit tests** for the storage layer using [`testify`](https://github.com/stretchr/testify)

## Tech stack

| Layer      | Library                                              |
|------------|-------------------------------------------------------|
| HTTP router| [`go-chi/chi`](https://github.com/go-chi/chi)         |
| Database   | SQLite via `modernc.org/sqlite`                       |
| Logging    | `log/slog` (standard library)                         |
| Testing    | `testify` (`assert` / `require`)                      |

## Project structure

```
.
├── cmd/             # application entrypoint (main.go)
├── cfg/             # configuration loading (env vars)
├── handlers/        # HTTP handlers (API + redirect)
├── pkg/             # shared HTTP response helpers
└── storage/         # SQLite storage layer + tests
```

## Getting started

### Prerequisites

- Go 1.25+

### Run locally

```bash
git clone https://github.com/Cryezidl/url-shortener.git
cd url-shortener
go run ./cmd
```

The server starts on port `8080` by default.

### Configuration

Configured via environment variables:

| Variable  | Default              | Description                          |
|-----------|----------------------|---------------------------------------|
| `PORT`    | `8080`               | HTTP server port                      |
| `DB_PATH` | `url_shortener.db`   | Path to the SQLite database file      |

### Run tests

```bash
go test ./...
```

## API reference

### Create a short link

```
POST /api/
Content-Type: application/json

{
  "shortname": "google",
  "targeturl": "https://google.com",
  "ttl": 3600000000000
}
```

`ttl` is optional and given in nanoseconds (Go `time.Duration`); omit it (or set it to `0`) for a permanent link.

**Response:** `200 OK`

### Get a link's rule

```
GET /api/{shortpath}
```

**Response:** `200 OK` (or `404` if the rule doesn't exist)

```json
{
  "TargetURL": "https://google.com",
  "CreatedAt": "2026-06-24T12:00:00Z",
  "ExpiresAt": null
}
```

### Get a link's stats

```
GET /api/{shortpath}/stats
```

**Response:**

```json
{
  "Hits": 42,
  "LastAccessed": "2026-06-24T12:34:56Z"
}
```

### Delete a link

```
DELETE /api/{shortpath}
```

**Response:** `200 OK`

### Redirect

```
GET /{shortname}
```

Redirects (`302 Found`) to the link's target URL, or returns `404` if the link is missing or expired.

## Possible improvements

- [ ] Dockerfile for containerized deployment
- [ ] Input validation (e.g. ensure `targeturl` is a well-formed URL)
- [ ] Auto-generated short codes (currently the caller chooses the short path)
- [ ] Pagination/listing endpoint for existing rules
- [ ] HTTP handler tests (currently only the storage layer is covered)

## License

MIT — see [LICENSE](LICENSE).

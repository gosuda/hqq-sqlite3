# hqq-sqlite3 (`gosuda.org/hqq-sqlite3`)

`gosuda.org/hqq-sqlite3` is a **zero-`cgo` (`CGO_ENABLED=0`) SQLite `database/sql` driver** powered by HQQ shared-memory IPC. Your Go application compiles as a pure Go binary without linking C objects, while executing the official SQLite C engine in an isolated shared-memory worker process (`hqq-sqlite-server`).

```bash
go get gosuda.org/hqq-sqlite3
```

## Supported Platforms

| OS | Architectures | Prebuilt `hqq-sqlite-server` Release Asset |
| :--- | :--- | :--- |
| **Linux** | `amd64`, `arm64` | `hqq-sqlite-server_linux_amd64`, `hqq-sqlite-server_linux_arm64` |
| **macOS** | `arm64` (Apple Silicon), `amd64` | `hqq-sqlite-server_darwin_arm64`, `hqq-sqlite-server_darwin_amd64` |
| **Windows** | `amd64`, `arm64` | `hqq-sqlite-server_windows_amd64.exe` (`arm64` via source build) |

---

## Quickstart

### 1. Managed Server + `*sql.DB` (`launcher.OpenManaged`)

`launcher.OpenManaged` verifies the `hqq-sqlite-server` binary digest, starts the shared-memory worker bound to the parent process lifetime, and returns a connected `*sql.DB`:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"time"

	sqlite "gosuda.org/hqq-sqlite3"
	"gosuda.org/hqq-sqlite3/launcher"
)

func main() {
	ctx := context.Background()

	hash, ok := sqlite.ExpectedHash(runtime.GOOS, runtime.GOARCH)
	if !ok {
		log.Fatalf("no pinned digest for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	m, err := launcher.OpenManaged(ctx, launcher.Config{
		Binary:   "/opt/hqq/bin/hqq-sqlite-server",
		SHA256:   hash,
		AllowDir: "/var/lib/myapp/db",
		Aliases:  map[string]string{"main": "/var/lib/myapp/db/app.db"},
	},
		launcher.WithDatabase("main"),
		launcher.WithSQLiteConfig(sqlite.Config{
			Timeout:     5 * time.Second,
			BusyTimeout: 5 * time.Second,
			Pragmas: map[string]string{
				"journal_mode": "WAL",
				"synchronous":  "NORMAL",
				"foreign_keys": "1",
			},
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer m.Close()

	var version string
	if err := m.DB.QueryRowContext(ctx, "SELECT sqlite_version()").Scan(&version); err != nil {
		log.Fatal(err)
	}
	fmt.Println("SQLite version:", version)
}
```

### 2. Standard `database/sql` DSN (`"sqlite3"`)

Import `gosuda.org/hqq-sqlite3` to register the `"sqlite3"` driver (matching `github.com/mattn/go-sqlite3`) and open a connection via `sql.Open`:

```go
import (
	"database/sql"

	_ "gosuda.org/hqq-sqlite3" // registers "sqlite3"
)

db, err := sql.Open(
	"sqlite3",
	"hqq://sqlite-svc?db=main&_journal=WAL&_sync=NORMAL&_fk=1&_busy_timeout=5000&_txlock=immediate",
)
if err != nil {
	return err
}
defer db.Close()
```

### 3. Structured Connector (`sqlite.NewConnector`)

```go
conn, err := sqlite.NewConnector("sqlite-svc", sqlite.Config{
	Database:    "main",
	Mode:        sqlite.ModeReadWriteCreate,
	Timeout:     5 * time.Second,
	FetchSize:   256,
	BusyTimeout: 5 * time.Second,
	TxMode:      sqlite.TxImmediate,
	Location:    time.UTC,
	Pragmas: map[string]string{
		"journal_mode": "WAL",
		"foreign_keys": "1",
	},
})
if err != nil {
	return err
}
db := sql.OpenDB(conn)
defer db.Close()
```

---

## Public API Summary

### Driver & Configuration (`package sqlite`)

```go
const (
	DriverName          = "sqlite3"
	Version             = "v0.1.3540100"
	SQLiteVersionNumber = 3540100
)

type Mode uint8
const (
	ModeReadWriteCreate Mode = iota // default: rwc
	ModeReadWrite                   // rw
	ModeReadOnly                    // ro
)

type TxMode uint8
const (
	TxDeferred  TxMode = iota // BEGIN DEFERRED
	TxImmediate               // BEGIN IMMEDIATE
	TxExclusive               // BEGIN EXCLUSIVE
)

type Config struct {
	Database    string
	Mode        Mode
	Timeout     time.Duration
	FetchSize   int
	BusyTimeout time.Duration // < 0 disables busy waiting (0 ms)
	TxMode      TxMode
	Location    *time.Location
	Pragmas     map[string]string
	Session     rt.Options
}

func NewConnector(listener string, cfg Config) (*Connector, error)
func ConnectorFromDSN(dsn string) (*Connector, error)
func ParseDSN(dsn string) (listener string, cfg Config, err error)
func WithTxMode(ctx context.Context, mode TxMode) context.Context
func ExpectedHash(goos, goarch string) (secureexec.Hash, bool)
```

### Error Inspection (`package sqlite`)

```go
type Error struct {
	Code         int    // primary SQLite result code (e.g. CodeConstraint)
	ExtendedCode int    // extended result code (e.g. CodeConstraintUnique)
	Message      string // sqlite3_errmsg()
}

var (
	ErrBusy           error
	ErrLocked         error
	ErrConstraint     error
	ErrReadOnly       error
	ErrInterrupted    error
	ErrOutcomeUnknown error
)

func IsBusy(err error) bool
func IsLocked(err error) bool
func IsConstraint(err error) bool
func IsReadOnly(err error) bool
func IsInterrupted(err error) bool
```

### Process Launcher (`package launcher`)

```go
func Start(ctx context.Context, cfg Config) (*Server, error)
func Open(ctx context.Context, cfg Config, opts ...OpenOption) (*sql.DB, *Server, error)
func OpenManaged(ctx context.Context, cfg Config, opts ...OpenOption) (*Managed, error)

func WithDatabase(db string) OpenOption
func WithSQLiteConfig(cfg sqlite.Config) OpenOption
```

---

## Documentation

| Document | Description |
| :--- | :--- |
| [**`docs/DSN.md`**](docs/DSN.md) | `hqq://` DSN syntax, underscore-prefixed PRAGMA control knobs (`_journal`, `_sync`, `_fk`, `_busy_timeout`, `_txlock`, `_loc`, ...), and type conversion rules. |
| [**`docs/ARCHITECTURE.md`**](docs/ARCHITECTURE.md) | Shared-memory session architecture, handle lifecycles, batch shims (`fetch_rows`, `column_chunk`), retry safety, and benchmarks. |
| [**`launcher/README.md`**](launcher/README.md) | Child-process verification (`secureexec`), signed release manifests, and `--parent-stdin` lifeline details. |
| [**`sqlite3/README.md`**](sqlite3/README.md) | Low-level `hqq-bindgen` IDL, C server hooks, and path-sandboxing policy. |
| [**`PROTOCOL.md`**](PROTOCOL.md) | Wire protocol specification and resource limits. |

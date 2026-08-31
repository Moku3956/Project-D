# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build ./...                                    # build everything
go test ./...                                      # run all tests
go test ./storage/btree/... -run TestName -v        # run a single test (package path + -run)
go test ./... -race                                 # race detector (txn/buffer pool code touches this often)
golangci-lint run ./...                             # lint (CI uses golangci-lint-action v7 / lint version v2.1.6)
go run ./cmd/server                                  # run the HTTP server (POST /query, GET /health)
```

Server env vars (all optional, see `cmd/server/main.go`): `DB_PATH`, `WAL_PATH`, `CATALOG_PATH`, `ADDR`.

## Git workflow

- **Never commit directly to `stage` or `main`.** Always create a new branch off `stage` before starting work, even for small changes.
- **Branch off `stage`, not `main`.** `stage` is the active integration branch (currently ~50 commits ahead of `main`); PRs target `stage`. `main` is updated separately/less often and should not be assumed current.
- **Branch naming**: `<type>/<short-name>`, e.g. `feature/storage`, `feature/txn`, `conf/docker`.
- **Commit message prefix**: `<prefix>: <Japanese summary>`, prefix one of `feat:` (also seen as `feature:`), `fix:`, `test:`, `docs:`, `chore:`, `ref:` (refactor). Not every historical commit has a prefix, but new commits should use one.
- **PRs use the template** at `.github/pull_request_template.md` (概要 / 関連タスク / やったこと / やらないこと / 影響範囲 / テスト / 備考). Issue templates exist at `.github/ISSUE_TEMPLATE/bug_report.yml` and `feature_request.yml`.

CI (`.github/workflows/ci-go.yml`) runs `go build ./...` + `go test ./...` and `golangci-lint` as separate parallel jobs against PRs targeting `main`/`stage`.

## Architecture

This is a DBMS built from scratch in Go, structured as a **pipeline of components with boundary interfaces** (deliberately not Clean Architecture/DDD — see `docs/architecture.md` for the reasoning). Each component owns a stage of SQL processing; components talk through interfaces defined by the *consumer*, not the producer:

```
SQL text → sql/lexer → sql/parser (→ sql/ast) → sql/planner → executor → storage
                                                       ↑
                                                    catalog
```

- `executor.TableRepository` is defined in `executor/`, implemented by `infrastructure.BTreeRepository`. Tests substitute a `mockRepo`.
- `sql/planner`'s `catalogReader` interface is defined in `planner/`, implemented by `catalog.Catalog`.

**Every package under `storage/`, `catalog/`, `txn/`, `executor/`, `types/`, and `sql/` has a `docs/spec.md` (or `sql/docs/*.md`) that is the source of truth for *why* that component is designed the way it is.** When behavior is unclear or looks like a bug, read the relevant spec.md before guessing from the code — several past bugs (see below) came from code drifting out of sync with a documented design decision. `docs/spec.md` is the top-level product spec; `docs/architecture.md` explains the component-split rationale.

### Storage engine (the core of this project)

A single B+Tree file holds **all tables' rows together**, disambiguated by a composite key `[tableID(4B)][type_tag(1B)][pk_bytes]` (`storage/btree/cell.go`). Internal node child pointers follow a **left-child convention**: cell `(Key_i, Child_(i-1))` means `Child_(i-1)` covers `< Key_i`; the page header's `RightmostChild` field holds the one child with no key of its own. On a leaf page this same header field is repurposed as the "next leaf" pointer for range scans.

Pages (`storage/page`) use a slot-indirection layout: a slot array (4 bytes: 2-byte offset + 2-byte length, growing from the front) points into cell data (growing from the back). The slot *array order* — not physical byte position — is what defines a page's logical (key) order; `page.InsertCellAt` reorders slots without moving cell bytes.

`storage/btree.BTree` no longer talks to `storage/page.DiskManager` directly — every read/write of an existing page goes through `storage/buffer.BufferPool` (`FetchPage`/`UnpinPage`), which enforces **No-Steal** (a transaction's dirty pages are never written to disk before commit). Page allocation (`DiskManager.AllocatePage`) is the one operation that still bypasses the pool. Every page mutation also appends a **physical, whole-page** Redo record to `storage/wal` (`BTree.finishPage`) before the page is marked dirty; `txn.Manager.Commit` flushes the WAL and then calls `BufferPool.FlushAll` for just that transaction's dirty pages (No-Force: uncommitted or other transactions' dirty pages are left in memory).

`storage/btree.Recover(disk, wm)` runs once at server startup (before the buffer pool/repository are constructed) to redo committed transactions after a crash: it groups WAL records by PageID and applies only the highest-LSN record per page for committed TxnIDs — it deliberately does **not** compare against the page's on-disk LSN, because WAL LSNs start at 0 and a page's zero-value (never-set) LSN is indistinguishable from a real LSN of 0.

### Transactions

`txn.Manager` provides table-level `sync.RWMutex` locking (SELECT → `RLock`, INSERT/UPDATE/DELETE → `Lock`) with a timeout (`lockTimeout`, 5s) to avoid indefinite blocking. There is currently **no multi-statement explicit transaction support** — `executor.Engine.Execute` auto-wraps every SELECT/INSERT/UPDATE/DELETE plan node in its own `Begin → lock → run → Commit/Rollback` (one HTTP request = one SQL statement = one transaction). `BEGIN`/`COMMIT`/`ROLLBACK` statements are parsed and accepted but are no-ops — supporting them for real would require session state across stateless HTTP requests, which was explicitly deferred.

### Execution

`executor.Engine.build` compiles a `planner.PlanNode` tree into a tree of `Executor`s (Volcano/iterator model: `Next()` pulls one row at a time). `IndexScanNode` (PK point lookup, O(log n) via `TableRepository.FindByPK`) is chosen over `SequentialScanNode` by the planner when `WHERE` is an equality match on the primary key — but only for plain SELECTs; JOINs and UPDATE/DELETE always fall back to full scans regardless of a PK predicate (tracked as known gaps, not yet fixed).

### Monorepo layout

This repo hosts the Go DBMS core plus two separate applications that consume it, kept intentionally apart:

- `db-internal-app/` — visualizes the SQL pipeline (lexer → parser → plan → execute → storage) in real time as a learning tool.
- `sql-monster/` — a game where SQL queries are used to analyze and attack monsters.

Both are currently empty scaffolds (design not yet started); the Go packages at the repo root (`api/`, `catalog/`, `executor/`, `sql/`, `storage/`, `txn/`, `types/`, `infrastructure/`, `cmd/`) are the shared DBMS engine both apps will sit on top of.

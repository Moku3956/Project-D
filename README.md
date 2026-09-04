# Project-D

Project-D is a monorepo consisting of a database built from scratch in Go, "**MokuDB**", and two applications that use it (a SQL learning site and a SQL battle game).

## About Me

I'm a third-year economics major. I study economics, but I've also been learning CS on my own for the past 2-3 years!
I love databases!
You can reach out via [Email](mailto:moku3956@icloud.com).


## Table of Contents

- [Table of Contents](#table-of-contents)
- [About MokuDB](#about-mokudb)
- [MokuDB Features](#mokudb-features)
  - [Supported SQL](#supported-sql)
  - [Storage Engine](#storage-engine)
- [Architecture](#architecture)
- [Repository Structure](#repository-structure)
  - [Directory Layout](#directory-layout)
- [Setup](#setup)
- [Usage](#usage)

---

## About MokuDB

**MokuDB** is a database I'm building from scratch — not to be used in production, but to **visualize what happens inside a database**.

Existing books about databases are extremely dense: they assume a lot of advanced prior knowledge, so it takes real persistence to get through them. I think a big reason for that is that words alone don't give you a concrete picture of what's actually happening. So I decided that visualizing what happens inside a database after a query runs was important, and built MokuDB from scratch in Go so that every step from parsing SQL to writing it to disk can be visualized and experienced. Another reason I built it from scratch: rather than trying to dig through and understand the internals of an existing DBMS in detail, it felt simpler to just implement one myself.

---

## MokuDB Features

### Supported SQL

- **DDL**: `CREATE TABLE`, `DROP TABLE`
- **DML**: `SELECT`, `INSERT`, `UPDATE`, `DELETE`
- **Transactions**: `BEGIN`, `COMMIT`, `ROLLBACK`
- **Data types**: `INT`, `VARCHAR(n)`, `BOOLEAN`, `NULL`
- **Constraints**: `PRIMARY KEY`, `NOT NULL`
- **Clauses**: `WHERE`, `INNER JOIN`, `ORDER BY`, `LIMIT`
- **Operators**: comparison operators, logical operators, `IS NULL` / `IS NOT NULL`

### Storage Engine

- B+Tree-based disk storage
- Crash recovery via WAL (Write-Ahead Log)
- LRU buffer pool
- Table-level concurrency control (RWMutex)

---

## Architecture

Inside MokuDB, SQL processing is implemented as a pipeline.

```
SQL text
  → Lexer   : breaks it into a token stream
  → Parser  : converts it into an AST
  → Planner : generates a plan tree (this is also where the execution plan is chosen)
  → Executor: reads/writes data and returns the result (Volcano model)
  → Storage : B+Tree / WAL / Buffer Pool
```

External Go programs like `sql-monster` don't directly import internal packages such as `executor`/`planner`/`txn` — they use MokuDB only through the `client` package.

```go
db, err := client.Open(dir)
result, err := db.Exec(sql)   // one statement = one transaction, auto-committed

tx := db.Begin()
result, err := tx.Exec(sql)
err = tx.Commit()             // or tx.Rollback()
```

---

## Repository Structure

The repo is centered on MokuDB itself (the packages at the repository root), alongside two applications that use it.

- **MokuDB core** (the packages at the repository root) — handles SQL execution and data persistence. Can be used from Go programs via the `client` package
- **`db-internal-app/`** — a learning site that uses MokuDB as its backend and visualizes SQL execution internals (lexing through storage writes) in real time in the browser (in design)
- **`sql-monster/`** — a battle game where you analyze, attack, and defend against monsters using SQL (in design; see `sql-monster/docs/spec.md` for details)

### Directory Layout

```
├── types/           # Shared types (Value / Row / Column / Schema)
├── catalog/         # Schema management (catalog.json)
├── sql/             # Lexer / Parser / AST / Planner
├── executor/        # Execution engine (Volcano model)
├── txn/             # Transaction management (locking, WAL integration)
├── storage/         # B+Tree / WAL / Buffer Pool
├── infrastructure/  # Implements executor's TableRepository with a B+Tree
├── client/          # Public API for external programs
├── api/             # HTTP handlers (POST /query, GET /health)
├── cmd/server/      # HTTP server entry point
├── db-internal-app/ # MokuDB internals visualization app (in design)
└── sql-monster/     # SQL battle game (in design)
```

---

## Setup

WIP

## Usage

WIP

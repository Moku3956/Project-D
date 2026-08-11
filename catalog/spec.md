# Persistence 仕様

## 概要

カタログ（テーブルのスキーマ情報）を管理する。`catalog.json` をソースとして起動時にメモリに読み込み、CREATE TABLE / DROP TABLE のタイミングでファイルに書き戻す。

---

## カタログの構造

```json
{
  "users": {
    "columns": [
      { "name": "id",   "type": "INT",        "primaryKey": true,  "notNull": true  },
      { "name": "name", "type": "VARCHAR(50)", "primaryKey": false, "notNull": true  },
      { "name": "age",  "type": "INT",         "primaryKey": false, "notNull": false }
    ]
  }
}
```

---

## 各層が定義するインターフェース

各層が必要なメソッドだけを自身のパッケージで定義する。`storage/persistence/` の実装がすべてを満たす。

### Planner（読み取りのみ）

```go
// interface/sql/planner/ で定義
type catalogReader interface {
    GetSchema(table string) (*types.Schema, error)
    TableExists(table string) bool
}
```

### Executor（読み取りのみ）

```go
// application/executor/ で定義
type catalogReader interface {
    GetSchema(table string) (*types.Schema, error)
}
```

### DDL（読み書き）

```go
// application/ddl/ で定義
type catalogRepository interface {
    GetSchema(table string) (*types.Schema, error)
    TableExists(table string) bool
    CreateTable(schema types.Schema) error
    DropTable(table string) error
}
```

---

## 実装

- 起動時に `catalog.json` を読み込んでメモリ上のマップに展開する
- 読み取りは複数の goroutine から同時にアクセスできる
- CREATE TABLE / DROP TABLE はメモリを更新してから `catalog.json` に書き戻す
- 並行アクセスの安全性は `sync.RWMutex` で保証する（読み取りは共有、書き込みは排他）

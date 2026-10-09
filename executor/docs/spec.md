# Executor 仕様

## 概要

プランツリーを受け取り、実際にデータを読み書きして結果を返す。Volcano（Iterator）モデルを採用する。

---

## 実行モデル

各プランノードに対応するExecutorが `Next()` を持ち、上のノードが下のノードを引っ張る形で1行ずつ処理する。

```
LimitExecutor.Next()
  └── SortExecutor.Next()
        └── FilterExecutor.Next()
              └── SeqScanExecutor.Next()
```

---

## Executor インターフェース

```go
type Executor interface {
    Next() (*types.Row, error)  // 次の行を返す。行がなければ nil
    Schema() *types.Schema      // 結果のスキーマ
    Close() error
}
```

---

## Executor の種類

| Executor | 対応するPlanNode | 役割 |
|----------|-----------------|------|
| `SeqScanExecutor` | `SequentialScanNode` | B+Treeの葉を順に読む |
| `IndexScanExecutor` | `IndexScanNode` | B+TreeをPKで検索する |
| `FilterExecutor` | `FilterNode` | WHERE条件で行を絞り込む |
| `ProjectionExecutor` | `ProjectionNode` | SELECT列を取り出す |
| `NestedLoopJoinExecutor` | `NestedLoopJoinNode` | 外側×内側の入れ子ループ |
| `SortExecutor` | `SortNode` | 全行を一度メモリに集めてソートする |
| `LimitExecutor` | `LimitNode` | 指定件数で打ち切る |
| `InsertExecutor` | `InsertNode` | 行を挿入する |
| `UpdateExecutor` | `UpdateNode` | 行を更新する |
| `DeleteExecutor` | `DeleteNode` | 行を削除する |

---

## 結果

```go
type Result struct {
    Rows         []types.Row    // SELECT結果（DMLの場合は空）
    AffectedRows int            // INSERT/UPDATE/DELETEの影響行数
    Schema       *types.Schema  // 結果のスキーマ
}
```

SELECT は `Next()` で行を順に取り出して `Rows` に積む。INSERT / UPDATE / DELETE は `Next()` が即 `nil` を返し、`AffectedRows` に件数を記録する。

---

## リポジトリインターフェース

Executorはストレージに直接アクセスせず、リポジトリインターフェース経由でアクセスする。インターフェースはここで定義し、`infrastructure/` が実装する。

```go
type TableRepository interface {
    FindByPK(table string, pk types.Value) (*types.Row, error)
    Scan(table string) ([]types.Row, error)
    Insert(table string, row types.Row) error
    Update(table string, pk types.Value, row types.Row) error
    Delete(table string, pk types.Value) error
}
```

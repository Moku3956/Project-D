# Planner 仕様

## 概要

ASTを受け取り、実行可能なプランツリーを生成する。カタログを参照してテーブル・カラムの存在確認とバリデーションも行う。

---

## 役割

```
AST → Planner → PlanNode（ツリー）→ Executor
```

- カタログを参照してASTを検証する（テーブル・カラムが存在するか）
- クエリの意味に応じてプランノードのツリーを組み立てる
- 簡単な最適化（インデックスが使えるかどうかの判断）を行う

---

## PlanNode

```go
type PlanNode interface {
    planNode()
    Kind() string
}
```

| ノード | 役割 |
|--------|------|
| `SequentialScanNode` | B+Treeの葉ノードを順に読む（全件スキャン） |
| `IndexScanNode` | B+TreeをPKで検索する（点検索・範囲スキャン） |
| `FilterNode` | WHERE条件で行を絞り込む |
| `ProjectionNode` | SELECT列を取り出す |
| `NestedLoopJoinNode` | INNER JOINを入れ子ループで実行する |
| `SortNode` | ORDER BYで並べ替える |
| `LimitNode` | LIMIT件数で打ち切る |
| `InsertNode` | INSERT実行 |
| `UpdateNode` | UPDATE実行 |
| `DeleteNode` | DELETE実行 |

---

## プランツリーの例

### SELECT * FROM users WHERE id = 1

```
ProjectionNode(*)
  └── IndexScanNode(users, pk=1)
```

### SELECT name FROM users WHERE age > 20 ORDER BY name LIMIT 10

```
LimitNode(10)
  └── SortNode(name ASC)
        └── ProjectionNode(name)
              └── FilterNode(age > 20)
                    └── SequentialScanNode(users)
```

### SELECT u.name, o.item FROM users u INNER JOIN orders o ON u.id = o.user_id

```
ProjectionNode(u.name, o.item)
  └── NestedLoopJoinNode(ON u.id = o.user_id)
        ├── SequentialScanNode(users)
        └── SequentialScanNode(orders)
```

---

## インデックス選択（最適化）

ルールベースで判断する。複雑なコスト計算は行わない。

- WHERE条件がPKへの `=` 比較 → `IndexScanNode`（点検索）
- WHERE条件がPKへの `<` `>` `<=` `>=` 比較 → `IndexScanNode`（範囲スキャン）
- それ以外 → `SequentialScanNode`

---

## バリデーション

カタログを参照して以下を確認する。エラーは複数まとめて返す。

- テーブルが存在するか
- SELECT・WHERE・JOIN ONで参照しているカラムが存在するか
- INSERT・UPDATEの値の型がカラム定義と一致するか
- NOT NULLカラムにNULLを挿入していないか

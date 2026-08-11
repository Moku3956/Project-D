# Project-D

GoでゼロからDBMSを自作するプロジェクト。SQLを実行したときにDBMSの内部で何が起きているかを可視化するSQL学習サイトと、スタンドアロンのDBMSライブラリの2つを兼ねる。

---

## 目的

既存のSQL学習サービスはSQLの書き方は教えてくれるが、「なぜそう動くのか」「内部でどう処理されているのか」は教えてくれない。本プロジェクトはSQLがパースされてからディスクに書き込まれるまでの全ステップをブラウザ上で可視化することで、DBMSの仕組みを直感的に理解できるようにする。

---

## 機能

### サポートするSQL

- **DDL**: `CREATE TABLE`、`DROP TABLE`
- **DML**: `SELECT`、`INSERT`、`UPDATE`、`DELETE`
- **トランザクション**: `BEGIN`、`COMMIT`、`ROLLBACK`
- **データ型**: `INT`、`VARCHAR(n)`、`BOOLEAN`、`NULL`
- **制約**: `PRIMARY KEY`、`NOT NULL`
- **句**: `WHERE`、`INNER JOIN`、`ORDER BY`、`LIMIT`
- **演算子**: 比較演算子、論理演算子、`IS NULL` / `IS NOT NULL`

### ストレージエンジン

- B+Treeベースのディスクストレージ
- WAL（Write-Ahead Log）によるクラッシュリカバリ
- LRUバッファプール
- テーブルレベルの並行制御（RWMutex）

---

## アーキテクチャ

SQLの処理はパイプラインとして実装されている。

```
SQL文字列
  → Lexer   : トークン列に分解
  → Parser  : ASTに変換
  → Planner : プランツリーを生成
  → Executor: データを読み書きして結果を返す
  → Storage : B+Tree / WAL / Buffer Pool
```

### ディレクトリ構成

```
├── types/      # 共有型（Value / Row / Column / Schema）
├── catalog/    # スキーマ管理（catalog.json）
├── sql/        # Lexer / Parser / AST / Planner
├── executor/   # Volcano モデルの実行エンジン
├── txn/        # トランザクション管理
├── storage/    # B+Tree / WAL / Buffer Pool
├── server/     # 可視化用REST APIサーバー
├── frontend/   # React フロントエンド
└── cmd/        # エントリポイント
```

---

## セットアップ

WIP

## 使い方

WIP

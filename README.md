# Project-D

GoでゼロからDBMSを自作するモノレポ。DBMSコア本体と、それを使う2つのアプリ(SQL学習サイト・SQLで戦うゲーム)で構成する。

---

## 目的

既存のSQL学習サービスはSQLの書き方は教えてくれるが、「なぜそう動くのか」「内部でどう処理されているのか」は教えてくれない。本プロジェクトはSQLがパースされてからディスクに書き込まれるまでの全ステップを可視化・体験させることで、DBMSの仕組みを直感的に理解できるようにする。

## 構成

- **DBMSコア**(リポジトリ直下の各パッケージ) — SQLの実行・データの永続化を担う。`client`パッケージ経由でGoプログラムから利用できる
- **`db-internal-app/`** — DBMSコアをバックエンドに、SQL実行時の内部処理(字句解析〜ストレージ書き込み)をブラウザ上でリアルタイムに可視化する学習サイト(設計中)
- **`sql-monster/`** — SQLでモンスターを分析・攻撃・防御する対戦ゲーム(設計中、詳細は`sql-monster/docs/spec.md`)

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
  → Planner : プランツリーを生成(実行計画の選択もここ)
  → Executor: データを読み書きして結果を返す(Volcanoモデル)
  → Storage : B+Tree / WAL / Buffer Pool
```

`sql-monster`のような外部Goプログラムは、`executor`/`planner`/`txn`などの内部パッケージを直接importせず、`client`パッケージ経由でのみDBMSコアを利用する。

```go
db, err := client.Open(dir)
result, err := db.Exec(sql)   // 1文=1トランザクション自動コミット

tx := db.Begin()
result, err := tx.Exec(sql)
err = tx.Commit()             // または tx.Rollback()
```

### ディレクトリ構成

```
├── types/           # 共有型（Value / Row / Column / Schema）
├── catalog/         # スキーマ管理（catalog.json）
├── sql/             # Lexer / Parser / AST / Planner
├── executor/        # Volcano モデルの実行エンジン
├── txn/             # トランザクション管理（ロック・WAL連携）
├── storage/         # B+Tree / WAL / Buffer Pool
├── infrastructure/  # executorのTableRepositoryをB+Treeで実装
├── client/          # 外部プログラム向けの公開API
├── api/             # HTTPハンドラ（POST /query, GET /health）
├── cmd/server/      # HTTPサーバーのエントリポイント
├── db-internal-app/ # DBMS内部可視化アプリ（設計中）
└── sql-monster/     # SQL対戦ゲーム（設計中）
```

---

## セットアップ

WIP

## 使い方

WIP

# アーキテクチャ設計

## アーキテクチャ方針

**コンポーネント分割 ＋ 境界インターフェース**を採用する。Clean Architecture・DDDは採用しない。

DBMSの構造はパイプラインそのものであり、層で切るより処理の役割（コンポーネント）で切るほうが実態に合っている。コンポーネント間の結合はインターフェースで制御する（ヘキサゴナルアーキテクチャと同じ考え方）。

### パイプライン

```
SQL文字列 → sql/（lex → parse → plan）→ executor/ → storage/
                                      ↗
                                 catalog/
```

### 境界インターフェース

| 境界 | インターフェース定義 | 実装 |
|------|---------------------|------|
| executor ↔ storage | `executor/`内で定義 | `storage/` |
| sql/planner ↔ catalog | `sql/planner/`内で定義 | `catalog/` |

---

## ディレクトリ構造

```
project-d/
├── cmd/
│   ├── dbms/      # DBMSをCLIとして使うエントリポイント
│   └── server/    # Webアプリケーションサーバーのエントリポイント（認証・API・DBMS）
│
├── types/         # 共有型（Value / Row / Column / Schema）
│
├── catalog/       # スキーマ管理（catalog.json の読み書き）
│
├── sql/           # SQLパイプライン
│   ├── lexer/     # 字句解析
│   ├── parser/    # 構文解析・AST生成
│   ├── ast/       # ASTの型定義
│   └── planner/   # AST → プランツリー
│
├── executor/      # プランツリーを実行する（リポジトリIF定義も含む）
│
├── txn/           # トランザクション管理（TxnID・ロック・WAL連携）
│
├── storage/       # ストレージエンジン（このプロジェクトのコア）
│   ├── page/      # ページフォーマット・読み書き
│   ├── btree/     # B+Tree実装
│   ├── buffer/    # バッファプール（LRU）
│   └── wal/       # Write-Ahead Log
│
├── api/           # HTTPハンドラ（ReactからのAPIリクエストを受け取る）
│
└── frontend/      # フロントエンド
    ├── src/
    └── public/
```

---

## 各コンポーネントの役割

### types/
全コンポーネントが共有する型定義のみ置く。ロジックは持たない（Value / Row / Column / Schema）。

### catalog/
テーブルのスキーマ情報を管理する。起動時に `catalog.json` をメモリに読み込み、DDL実行時にファイルへ書き戻す。

### sql/
SQL文字列をプランツリーに変換するパイプライン。lexer → parser → AST → planner の順に処理する。

### executor/
プランツリーを受け取りVolvano（Iterator）モデルで実行する。storageへのアクセスはここで定義したインターフェース経由。

### storage/
このプロジェクトのコアエンジン。B+Tree・WAL・バッファプールを実装する。executor が定義したインターフェースを実装する。

### server/
Webアプリケーションサーバー。ユーザー認証・セッション管理・APIハンドラを担う。DBMSを直接呼び出すため、プロセス間通信不要。

### server/

DBコアとは独立した層。usecaseを呼び出しながらその各ステップをイベントとして捕捉し、WebSocket経由でフロントエンドに配信する。

```
フロント → server/api → usecase → infrastructure
               ↓
          server/event → server/ws → フロント
```

---

## 採用しなかった選択肢

| 項目 | 却下した選択肢 | 理由 |
|------|--------------|------|
| アーキテクチャ | Clean Architecture + DDD | DBMSはパイプライン構造でありDDDのドメインモデルが活きる場面がない。コンポーネント分割＋境界インターフェースで十分 |
| 可視化のプロトコル | WebSocket | SQL処理はミリ秒単位で完了するためストリーミング不要。1リクエストで全ステップをまとめて返すREST APIで十分 |
| 可視化イベントの配置 | sql/やexecutor/に含める | DBコアと可視化を混在させると責務が不明確になるため分離 |
| ストレージの配置 | infrastructure/ | ストレージエンジン自体がこのプロジェクトのコア製品であるため独立させた |

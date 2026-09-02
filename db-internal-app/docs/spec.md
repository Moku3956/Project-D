# db-internal-app 仕様

## 概要

Project-DにSQLを投げると、`lexer → parser → planner → executor → storage`の各ステージで何が起きているかを1ステージずつ「次へ」で確認できる学習サイト。SQLの内部処理を可視化できるサービスが存在しないことが動機([[project_overview]]参照)。

自由入力プレイグラウンド形式。ユーザーはCREATE TABLE/INSERT/SELECT/UPDATE/DELETEなど任意のSQLを打ち込める(sql-monsterのような固定シナリオではない)。

---

## アーキテクチャ

**sql-monsterとは異なり、`client`パッケージ経由では実装できない。** `client.Exec`は最終的な`Result`(行データ・影響行数)しか返さず、lexerのトークン列やASTやプランツリーといった中間表現を外に出さない設計になっている(`client/client.go`確認済み)。可視化が目的である以上、db-internal-appは`sql/lexer`・`sql/parser`・`sql/planner`・`executor`・`storage/btree`・`storage/page`を直接importする。

これはCLAUDE.mdの「`client`経由でしか触らせない」という方針からの意図的な逸脱であり、sql-monsterとは別の理由(内部可視化そのものが目的)による。sql-monster/docs/spec.mdの「既存DBMSの使われ方に揃える」という理由はここには適用されない。

### ストレージ可視化は既存コードへの侵襲的な変更を避ける

B+Treeのページ/ノード単位のツリー図まで見せる(ユーザー確定済み)。当初「splitやinsertの過程をフック/トレースで逐次記録する」案も考えたが、`storage/btree`は本体サーバー・sql-monster双方が使う共有エンジンであり、そこにトレース用のフックを埋め込むのは挙動・性能両面でリスクが高い。

代わりに、**クエリ実行の前後でB+Treeの全ページを`RootPageID`からルートダウンに辿って読み、シリアライズ可能なツリー構造(ページID・種別・キー一覧・子ページID一覧)にダンプする、新規の読み取り専用関数**を`storage/btree`に追加する(既存の`Search`/`Insert`/`Delete`/`Scan`には一切手を入れない)。フロントエンドは実行前後のダンプを見比べて「どのページが変わったか(分割が起きたか等)」を表示する。ページ内部(スロット配列・セルバイト列)まで見せるかは次項参照。

```go
// storage/btree/dump.go (新規)
func DumpTree(disk *page.DiskManager, schema *types.Schema) (*TreeSnapshot, error)

type TreeSnapshot struct {
    RootPageID uint32
    Pages      map[uint32]PageSnapshot
}

type PageSnapshot struct {
    PageID         uint32
    IsLeaf         bool
    Keys           []string    // 内部ノード: 複合キーを人間可読な形にデコードしたもの
    ChildPageIDs   []uint32    // 内部ノードのみ(Key_iとChild_(i-1)の対応も含む)
    RightmostChild uint32      // 内部ノードのみ
    Rows           []types.Row // 葉ノードのみ。KVそのもの(スロット配列・オフセット等の内部形式は含めない)
    NextLeafID     uint32      // 葉ノードのみ
}
```

---

## 操作モデル

1. ユーザーがSQLを1文入力して実行する
2. サーバーは`lexer`→`parser`→`planner`→`executor`→`storage`の全ステージ分の情報を**1回のHTTPレスポンスにまとめて**返す(WebSocketは使わない。[[project_overview]]の既存方針を踏襲)
3. フロントエンドは受け取った全情報を保持したまま、「次へ」ボタンで1ステージずつ画面に開示していく(ガイド付き体験。ユーザー確定済み)
4. 各ステージの詳細は下記「ステージごとの表示内容」を参照

### セッションの永続性

**将来的に一般公開し、不特定多数が同時に触れるデモサイトにする想定(ユーザー確定済み)。** この前提により、DBインスタンスをセッション単位で分離する設計が必須になる(誰かのDROP TABLEが他人の画面に影響してはいけない)。

CREATE TABLEしたテーブルにその後何度もINSERTし、B+Treeが分割で育っていく様子を見せることが、ストレージ可視化の一番の見せ場になる。そのためには**同じセッション内で複数クエリをまたいでDBの状態が保持される**必要がある(1クエリごとにDBをリセットする方式だと、2回目以降のINSERTでの分割が観察できない)。

**設計:**

1. **Cookieベースのセッション。** 初回アクセス時にランダムなセッションID(UUID)を発行し、`HttpOnly`+`SameSite=Lax`のCookieとして返す。以降のリクエストはこのCookieでセッションを識別する(sql-monsterのようにフロントエンドが1プレイヤー=1プロセス想定ではないため、この点はsql-monsterと構成が異なる)。
2. **セッションごとに専用の一時ディレクトリ**(`page.DiskManager`+`wal.WALManager`+`catalog.Catalog`一式)を、初回クエリ実行時に遅延作成する。サーバープロセスは`sessionID → *DB`のin-memoryマップを保持する。sql-monsterの`DATA_DIR`衝突バグ([[project_issues]]参照)と同じ轍を踏まないよう、db-internal-app用のディレクトリも本体サーバー・sql-monsterとは独立させる。
3. **アイドルセッションの掃除。** 一定時間(例: 30分)操作のないセッションは、バックグラウンドの定期処理でディスク上のディレクトリごと削除し、in-memoryマップからも破棄する。公開デモである以上、放置されたセッションでディスクが際限なく埋まることを防ぐ必須の仕組み。
4. **「リセット」ボタン。** ユーザーが明示的に今のセッションのDBを閉じて空の状態から作り直せるようにする。

**公開デモとして必要なリソース制限(実装時に具体値を詰める):**
- セッションあたりのページ数上限(容量の上限)を設け、超えたらエラーで拒否する
- サーバー全体の同時セッション数に上限を設け、超えた場合は最もアイドルなセッションから追い出す
- 1クエリあたりの実行タイムアウト

具体的な数値(タイムアウト秒数、ページ数上限、同時セッション数上限)は実装フェーズで決める。

---

## UIビジュアルスタイル

明るい色を基調とした、しゃれたスタイルにする(ユーザー確定済み)。sql-monsterのダーク基調・ゲーム的な見た目とは対照的な方向性。具体的な配色・フォント等はFigmaでの設計時に詰める。

---

## ステージごとの表示内容

### ① Lexer

トークン列をそのまま返す。`sql/lexer`の`Token{Type, Literal, ...}`を配列で見せる。SQL文字列の該当範囲をハイライトできるよう、位置情報(あれば)も含める。

### ② Parser

`sql/ast`のASTをツリー表示する。`sql/docs/ast.md`に定義された各ノード型(`SelectStmt`/`InsertStmt`/`BinaryExpr`等)をJSONにシリアライズし、フロントエンドでツリー図として描画する。

### ③ Planner

`sql/planner`の`PlanNode`ツリーをそのまま表示する。`sql/docs/planner.md`のプランツリー例(`ProjectionNode → FilterNode → SequentialScanNode`等)と同じ形。IndexScanかSequentialScanかの選択理由(PK等値/範囲条件の有無)もあわせて表示し、「なぜこのスキャン方式が選ばれたか」が伝わるようにする。

### ④ Executor

プランツリーと同型のExecutorツリーを表示し、各ノードが`Next()`で何行処理したか(最終的な行数、および可能なら先頭数行のサンプル)を添える。Volcano modelの「上のノードが下のノードを1行ずつ引っ張る」という実行順序自体を逐次アニメーションで見せることはしない(ユーザー確定済み)。「木構造＋各ノードの入出力行数」の静的な表示に留める。

### ⑤ Storage

クエリ実行前後の`TreeSnapshot`(上記)を比較表示する。

- 変化のないページはそのまま、新規作成されたページ(split等)はハイライト
- 内部ノード: `[Child0] Key1 [Child1] Key2 [Child2]`の左子規約([[architecture_decisions]]参照)通りの見た目で子ポインタとキーを図示
- 葉ノード: 格納されているキーと値(KV)の一覧を見せる。複合キーは`tableID/type_tag/pk`をデコードした人間可読な形にする

**スロット配列・セルの生バイト列までは見せない(ユーザー確定済み)。** ページの中身がKVとして分かれば十分で、`storage/page`の内部フォーマットの詳細はスコープ外とする。`PageSnapshot`にオフセット・フリーリスト等の生の数値フィールドを含める必要はなく、`Keys`をKVのペア(`Values []types.Row`相当)に置き換える形でよい。

---

## 未定事項

- セッションのアイドルタイムアウト秒数・セッションあたりのページ数上限・同時セッション数上限の具体的な値
- クエリ実行タイムアウトの秒数
- UIの具体的な配色・フォント等(Figmaでの設計時に詰める)

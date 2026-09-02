# フロントエンド設計

`battle_screen_ui.md` / `home_screen_ui.md`(Figma設計)を実装するにあたっての技術選定・アーキテクチャ方針。

起動方法・遊び方は [`../README.md`](../README.md) を参照。

---

## 技術スタック

| 項目 | 選定 |
|---|---|
| フレームワーク | React + TypeScript |
| ビルド/開発サーバー | Vite |
| スタイリング | Tailwind CSS |
| 状態管理 | Zustand(バトル中のフェーズ・リソースなど画面をまたぐ状態) |
| フォント | Orbitron(見出し・表示系) / Space Mono(データ・コード・ラベル) — いずれもGoogle Fonts |


## フォルダ構成: フィーチャーベース

当初クリーンアーキテクチャ的な層分け(`domain/usecases/infrastructure/presentation`)を検討したが、画面数がまだ3つ(ホームパス・バトル・プレビュー)程度の規模では、1機能を触るたびに複数フォルダを股にかける手数の多さが見合わないと判断し、**機能ごとにコードをまとめるフィーチャーベース構成**に変更した。

```
sql-monster/frontend/src/
  features/
    home-path/     # パス画面のcomponents, hooks, api呼び出し
    battle/        # SQLエディタ, フェーズタブ, リソース表示, バトルログ
    preview/       # モンスタープレビューモーダル
  shared/          # 共通UI部品(ボタン等), 共通型, APIクライアント
```

関連コードを物理的に近くに置く(colocation)ことを優先し、機能追加は`features/`配下にフォルダを1つ足すだけで済むようにする。React公式ドキュメントや"bulletproof-react"などの参考実装でも採用されている、現在のReactアプリでは主流の構成。

バックエンド(Go)側は引き続き、新設するHTTPハンドラ層が`game`パッケージ(既存のドメインロジック)を呼び出す形にする。こちらはフロントの構成とは独立に決めてよい(層を厳密に対応させる必要はない)。

---

## バックエンドAPI(決定事項)

認証・複数プレイヤーは現状考えない(単一プレイヤー想定)。

**永続化**
- `player_progress(monster_id, cleared)`テーブルを新設。バトルに勝利したら該当行を`cleared=true`に更新
- バトル中の状態(HP・リソース・フェーズ)はサーバーのメモリ上の`game.Battle`のみで保持。プロトタイプなのでサーバー再起動で消えるのは許容し、DBには持たせない

**エンドポイント**

| エンドポイント | 役割 |
|---|---|
| `GET /monsters` | ホーム画面のパス用。進行状態つきモンスター一覧 |
| `POST /battles?monster_id=X` | プレビューの`START BATTLE`。新規バトル開始 |
| `GET /battles/{id}` | 現在の状態取得(HP・リソース・フェーズ・ログ) |
| `POST /battles/{id}/query` | SQLクエリ実行(分析/攻撃/防御共通) |
| `POST /battles/{id}/quit` | 敗北扱いで終了(`player_progress`は変化なし) |
| `POST /battles/{id}/restart` | 同じモンスターで作り直し(ペナルティなし) |

`QUIT`は敗北扱い、`RESTART`はペナルティなし(`battle_screen_ui.md`のバトルメニュー仕様と対応)。

---

### なぜVite(Next.jsではなく)

Next.jsはReact + ファイルベースルーティング + SSR/SSG + APIルート(Node.jsサーバー)まで込みのフルスタックフレームワーク。Viteはビルドツール/開発サーバーのみで、ルーティングやSSRは持たない。

sql-monsterは:
- バックエンドがすでにGoで別途ある(`sql-monster/cmd/server`)。Next.jsのAPIルート/SSR用Node.jsサーバーを足すと、Goサーバーと二重にサーバーを持つことになる
- ホーム画面(モンスターパス)・バトル画面は、ログイン後の対話的な画面で内容はプレイヤーごと・クエリ結果ごとに動的に変わる。検索エンジンに個々の画面をインデックスさせたいわけではないのでSSR/SEOは不要

**ただし「sql-monster全体がSEO不要」ではない。** 将来、検索から人を呼び込みたい公開ランディングページ(「sql-monsterとは」「今すぐ遊ぶ」)を作るなら、そこはSEO/SSRが効く。その場合もゲーム本体のフロントをNext.js化する理由にはならず、ランディングページだけ別の軽い静的ページ(素のHTML等)で十分という結論。

---

## 関連

- バトル画面のUI設計: `battle_screen_ui.md`
- ホーム画面のUI設計: `home_screen_ui.md`
- ゲームメカニクス全体: `spec.md`

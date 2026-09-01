# sql-monster

SQLでモンスターを分析し、SQLで攻撃・防御する対戦ゲーム。Project-D(自作DBMS)の上に、公開用の`client`パッケージ経由で乗っている。

## 構成

| ディレクトリ | 役割 |
|---|---|
| `cmd/server/` | エントリポイント。DBを開き、HTTP APIを提供する |
| `internal/game/` | 対戦ロジック(ドメイン層)。スキーマ定義とモンスター定義もここ |
| `internal/api/` | HTTPハンドラ。1ターンの4フェーズの進行を管理する |
| `frontend/` | React + TypeScript + Vite のフロントエンド |
| `docs/` | 仕様・設計ドキュメント(下記) |

## 起動方法

バックエンドとフロントエンドを別々のターミナルで立ち上げる。

### 1. バックエンド(Go)

```bash
# リポジトリのルートで
go run ./sql-monster/cmd/server
```

- 既定で `:8081` で起動する
- 初回起動時にテーブルの作成とモンスター6体の投入まで自動で行う
- 2回目以降は既存のデータをそのまま使う(テーブルもデータも作り直さない)

環境変数:

| 変数 | 既定値 | 説明 |
|---|---|---|
| `DATA_DIR` | `data` | DBファイル・WAL・カタログの置き場所 |
| `ADDR` | `:8081` | 待ち受けアドレス |

### 2. フロントエンド(Vite)

```bash
cd sql-monster/frontend
npm install   # 初回のみ
npm run dev
```

- `http://localhost:5173` で開く
- `/api` へのリクエストはバックエンド(`:8081`)にプロキシされるので、CORSの設定は不要
- プロキシ先を変えたい場合は環境変数 `API_TARGET` を指定する

### データをリセットしたいとき

進行状況(どのモンスターをクリアしたか)もDBに入っているので、まっさらにしたい場合はデータディレクトリごと消す。

```bash
rm -rf data
```

## 遊び方

1. ホーム画面のパスから、挑戦できるモンスター(白枠で強調されているノード)をクリック
2. プレビューが出るので `START BATTLE`
3. 1ターンは4フェーズで進む
   - **PHASE_01 攻撃用分析**: `monster_weaknesses` をSELECTして弱点を探す。読んだ行数だけANALYSIS_APを消費する。何度でも撃てるので、済んだら `NEXT PHASE`
   - **PHASE_02 攻撃**: `UPDATE` などでモンスターのHPを削る。実行するとHP差分が実測され、CRUD_ATTACK_APの残量を超えていたらROLLBACKされて不発になる
   - **PHASE_03 防御用分析**: `monster_attacks` をSELECTして、次に来る攻撃を予測する。済んだら `NEXT PHASE`
   - **PHASE_04 防御**: WHEREで絞り込んでSELECTする。当たっていればブロック率は `1 / 該当行数`、外れればフルダメージ

### 現状の制約

**攻撃SQLは絶対値で書く必要がある。** 本来は `SET hp = hp - 50` と書きたいが、Project-DのSQLに算術演算子がまだないため、`SET hp = 250` のように計算後の値を直接書く形になっている(`project_issues.md`の「算術演算子が言語に存在しない」)。エディタには現在HPから50を引いた雛形が入るようになっている。

## ドキュメント

| ファイル | 内容 |
|---|---|
| [docs/spec.md](docs/spec.md) | ゲームメカニクス全体の仕様 |
| [docs/battle_screen_ui.md](docs/battle_screen_ui.md) | バトル画面のUI設計(Figma) |
| [docs/home_screen_ui.md](docs/home_screen_ui.md) | ホーム画面のUI設計(Figma) |
| [docs/frontend_architecture.md](docs/frontend_architecture.md) | 技術選定・フォルダ構成・API設計 |

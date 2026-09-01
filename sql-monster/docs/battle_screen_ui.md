# バトル画面UI設計

`sql-battle-screen`(Figma)のレイアウト設計。spec.mdのゲームメカニクスをどう画面に落とすかの記録。まだ「軽く動かすもの」を作るための土台段階で、細部の数値・演出は未確定。

- Figmaファイル: https://www.figma.com/design/SYNd796ebb3ETGhECYVlFc/ (node-id `2:4`, フレーム名 `sql-battle-screen`)
- ファイルには他にテンプレート素材が入っていたが全て削除済み。`sql-battle-screen`(フェーズ1)の右に、フェーズ2〜4を`sql-battle-screen-phase2` / `-phase3` / `-phase4`として複製・編集済み。さらに右に、設定パネルの中身を`settings-panel`として単独フレームで用意。

---

## 画面構成

見た目のテーマは「ターミナル/ハッカー風」。ダークベースにネオン系のアクセントカラー(シアン/オレンジ/レッドなど)。

### タイポグラフィ

「文字をメカニカルな感じにしたい」という方針で、当初のGeist / Geist Monoから差し替えた(全フレーム適用済み)。

| 用途 | フォント |
|---|---|
| 見出し・タイトル・ボタン(13px以上の表示系) | **Orbitron**(ExtraBold / Bold / SemiBold) |
| データ・コード・ラベル・ログ(モノスペース系) | **Space Mono**(Regular / Bold) |

モノスペース側は当初Share Tech Monoを試したが、**ウェイトがRegular 1種類しかなく**、HP値・`PHASE_01`・`QUERY_COST`などの太字強調が全部平坦になってしまうため、Boldを持つSpace Monoに変更した。

```
header-bar
  logo-badge("SQL") + タイトル("SQL-MONSTER") + settings-button(三本線)

turn-phases "BATTLE_FLOW_PHASE"   … 全幅のステッパー。4フェーズの現在地を示す
  phase-1〜4(PHASE_01〜04 + フェーズ名。ACTIVE/DONE/PENDINGは色のみで表現、文言は削除済み)

monster-display-section
  monster-card(モンスターのビジュアル)
  monster-info-bar(名前 + LVL、HPバー、ACTIVE_DEBUFFSアイコン列)

main-dashboard-row (3カラム)
  left-panel-intel  "RESULT"(フェーズ2のみ"PREVIEW")   … spec.mdの①攻撃用分析に対応
  center-panel-editor "query_planner.sql"  … SQLエディタのみ(4フェーズ表示は上に移動済み)
  right-panel-status "PLAYER" … プレイヤーHP/リソース/ログ

bottom-bar-ref "SQL OPERATION SYNTAX MAP"
  SELECT/UPDATE/INSERT/DELETE と技の対応表
```

### spec.mdとの対応

| UI要素 | spec.mdの該当箇所 |
|---|---|
| `left-panel-intel`のSELECT結果テーブル | ①攻撃用分析(弱点データへのSELECT) |
| `center-panel-editor`のSQLエディタ + `Execute`ボタン | ②攻撃(CRUD文でのHP実測ダメージ) |
| `turn-phases`の4タブ(Attack Analysis / Attack Exec / Defense Analysis / Defense Exec) | 1ターンの4フェーズ構成(①②③④) |
| `right-panel-status`の`ANALYSIS_AP` / `CRUD_ATTACK_AP` | 分析リソース / 攻撃防御リソースの2種 |
| `bottom-bar-ref`のSQL操作マップ | ②攻撃フェーズの技の種類(UPDATE=直接攻撃、INSERT=状態異常付与、DELETE=バフ剥がし) |
| `system-battle-log` | 実測ダメージ・ブロック成否などの結果ログ表示 |

フェーズ1〜4とも下記の「フェーズ別の画面設計」の内容でFigma上に反映済み。

---

## フェーズ別の画面設計

4フェーズとも同じレイアウト(`left-panel-intel` / `center-panel-editor` / `right-panel-status`)を使い回し、中身だけがフェーズごとに差し替わる想定。

### フェーズ1: Attack Analysis(Figma反映済み — `sql-battle-screen`)

- クエリ例: `SELECT * FROM monster_weaknesses WHERE monster_id = 42 AND severity >= 5;`
- `left-panel-intel`: 「RESULT」— `ID / COLUMN_NAME / VALUE / SEVERITY`の結果テーブル
- ボタン文言: `Run Query`
- コスト表示: `QUERY_COST: N AP (ANALYSIS)`(nは返却行数)

### フェーズ2: Attack Exec(Figma反映済み — `sql-battle-screen-phase2`)

- クエリ例: `UPDATE monsters SET hp = hp - 1240 WHERE id = 42 AND weakness = 'ACID';`
  (`hp - 1240`のような算術式は現状のSQL言語で未対応。`project_issues.md`の「算術演算子が言語に存在しない」を参照。デザイン上は目指す体験として書いているが、実装が追いつくまでは絶対値の`SET`で代用する)
- `left-panel-intel`は「RESULT」から**「PREVIEW」に差し替え**。SELECT結果ではなく、トランザクション内で実測した診断を見せる:
  ```
  BEFORE_HP:            3,820
  SIMULATED_AFTER_HP:   2,580
  MEASURED_DIFF:        -1,240
  CRUD_ATTACK_AP BUDGET: 45 / 100
  STATUS:               WITHIN_BUDGET — READY_TO_COMMIT
  ```
  spec.mdの「実際にトランザクション内で実行し、実行前後のHP差分を実測する」「差分が攻撃防御リソースの残量を超えていたらROLLBACK」をそのまま画面に見せる意図。
- ボタン文言: `Execute Attack`(現状のFigmaの文言はこのフェーズにこそ合う)
- コスト表示: `PROJECTED_COST: 1,240 AP (ATTACK/DEFENSE)`(実測差分そのものがコストなので、実行前は「実測できていない」ことが前提。実際はコミット後に確定値をログへ出す形になる)

### フェーズ3: Defense Analysis(Figma反映済み — `sql-battle-screen-phase3`)

- クエリ例: `SELECT * FROM monster_attacks WHERE monster_id = 42 ORDER BY likelihood DESC;`
- `left-panel-intel`は「RESULT」に戻る。列は`game.MonsterAttack`(`sql-monster/internal/game/schema.go`)のフィールドに合わせて`ID / TAG / LIKELIHOOD / POWER`:
  ```
  101  SMASH_BLUNT     68   220
  102  LASER_BEAM      12   640
  103  RECHARGE_BUFF   20   0
  ```
  (このテーブルは元々フェーズ1に誤って置かれていた`ATTACK_TYPE/PROB_PCT/MULT`テーブルの本来あるべき置き場所)
- ボタン文言: `Run Query`
- コスト表示: `QUERY_COST: 3 AP (ANALYSIS)`

### フェーズ4: Defense Exec(Figma反映済み — `sql-battle-screen-phase4`)

- クエリ例: `SELECT * FROM monster_attacks WHERE monster_id = 42 AND tag = 'LASER_BEAM';`
- `left-panel-intel`は引き続き「RESULT」。返却行数がそのままブロック率(`1/行数`)に直結することを明示するため、テーブル下の行数表示に`BLOCK_RATE: 1 / N`を併記(例: `1 row selected. BLOCK_RATE = 1 / 1`)。
- ボタン文言: `Confirm Defense`
- コスト表示: `QUERY_COST: N AP (ANALYSIS)`
- 命中/ミスの結果自体は`right-panel-status`の`LOG`に出す(既存モックの`CRITICAL! UPDATE matched PK...`や`ERROR: Block failed...`のログ行パターンを流用)。

### 横断的な注意点

Figma上は4フェーズを別フレーム(静的な複製)として作った。実装(コード)側では1つの画面が状態に応じてボタン文言・`QUERY_COST`/`PROJECTED_COST`ラベル・左パネルの中身(RESULT⇔PREVIEW)を出し分ける必要があり、静的なテキストではなく状態に応じた出し分けロジックが要る。

---

## ブラウザ幅の確認

ブラウザでの利用のみを想定しているため、実際のブラウザ幅で崩れないかFigma上でテストした(1440px版を複製し幅だけ変更 → スクリーンショットで確認 → テスト用フレームは削除済み)。

- デザインは1440px想定で作ってあるが、3カラム(`main-dashboard-row`)のうち左右(`left-panel-intel` 346px / `right-panel-status` 380px)は固定幅、中央の`center-panel-editor`(SQLエディタ)だけが可変(FILL)という構造だったため、**画面全体の幅を縮めても中央カラムだけが縮んで吸収してくれる**。
- **1366px幅**(StatCounterで見た2番目に多い解像度、約10%)→ 崩れなし。中央カラムが560pxに縮むだけ。
- **1280px幅**(スクロールバーを引いた実表示幅としても現実的な下限)→ こちらも崩れなし。中央カラムは474pxまで縮むが、SQLのコード行・`QUERY_COST`・実行ボタンとも余裕を持って収まる。
- 世界シェア1位の**1920×1080**はもちろん問題なし(中央カラムがさらに広がるだけ)。
- 見た目上の破綻(ボタンとコスト表示の重なりなど)が起き始めるのはおおよそ1150〜1200px付近と推定(左右固定幅の合計 + パディング + 中央カラムの最小必要幅から逆算)。2026年時点のデスクトップブラウザでこれより狭い幅は稀。

**結論: 現状のデザインのまま、主要なブラウザ幅(1280px〜)で問題なく収まる。** 追加のブレークポイント設計は不要と判断。

**ノートパソコンについての補足:** 上記の解像度統計はデスクトップ/ノート込みの物理解像度ベース。ノートPC、特にWindowsの高DPI機は125%/150%といった表示スケーリングがデフォルトになっており、実際のブラウザ実効幅(CSS viewport)は物理解像度より狭くなる(例: 1920×1080を150%スケーリング→実効幅約1280px)。この一番厳しいケースでも実効幅は約1280px前後に収まり、上記のテスト範囲内。13インチMacBook系のデフォルト論理解像度(1440×900前後)も同様に範囲内。

ただし、解像度に関係なく**ブラウザウィンドウを最大化せず手動で狭くして使うケース**は常に残るリスクで、これはどのレイアウト設計でも完全には解決できない。

---

## 設定パネル(バトルメニュー)

`header-bar`右端に三本線アイコンの`settings-button`を追加(4フェーズ全フレームに反映済み)。押下すると開く想定のドロップダウンパネルを`settings-panel`として単独フレームで作成した(Figma上のプロトタイピング連携はまだ組んでいない。中身のレイアウト検討が目的)。

中身は音量や表示設定ではなく、**バトルの進行そのものを操作するメニュー**という位置づけ。名前は極力単純に(説明文なし、単語一つだけ):

| 項目 | 色 |
|---|---|
| `RESUME` | シアン |
| `RESTART` | グレー |
| `QUIT` | 赤(危険アクション) |

`FORFEIT`(見慣れない単語)は`QUIT`に変更。各項目の説明文も削除し、単語のみの表示にした。パネルタイトルも`SYSTEM_SETTINGS`→`BATTLE_MENU`→最終的に`MENU`まで簡略化。

---

## ラベルの簡略化

「ターミナル風の凝った名前」よりも「見て一瞬でわかる単語」を優先する方針で、以下も単純化した(4フェーズ全フレームに反映済み)。

| 変更前 | 変更後 |
|---|---|
| `Query Intelligence` | `RESULT` |
| `Transaction Preview` | `PREVIEW` |
| `Player Terminal Status` | `PLAYER` |
| `ROOT_USER_HP` | `HP` |
| `CYCLE_INDEX` | `TURN` |
| `TURN_09` | `09`(ラベル側に`TURN`と出るため数字のみ) |
| `SYSTEM_BATTLE_LOG` | `LOG` |

`ANALYSIS_AP (SELECT)` / `CRUD_ATTACK_AP (UPDATE/INSERT)`はまだ未着手。

---

## turn-phasesの配置変更

`turn-phases`(4フェーズのステッパー)を`center-panel-editor`の中(SQLエディタの下、幅620px)から、**画面最上部・ヘッダー直下の全幅の帯**に移動した。バトル全体の進行状況を示すグローバルな表示であり、SQLエディタというローカルな要素に従属させる理由がなかったため。移動により中央カラムはSQLエディタ+実行ボタンだけになり縦に余裕ができた。

あわせて、各フェーズタブ下にあった`ACTIVE TURN` / `PENDING` / `DONE`の文言を削除。タブの色分け(シアン=進行中、グレー=未着手、緑=完了)だけで状態が伝わるため、文言は蛇足と判断。

**決定事項:**
- `QUIT`: その対戦は**敗北扱い**にする。
- `RESTART`: ペナルティなし。何度でも無料でやり直せる。

---

## テーブル列の見切れ修正

結果テーブルの4列目(`SEVERITY` / `POWER` / `NOTE`)が右端で見切れていた。原因は、`left-panel-intel`の幅が360px→346pxに変わった際に列のx座標が追従しておらず、**4列目がコンテナ幅(314px)を16pxはみ出していた**こと。ヘッダーと全データ行の3〜4列目の位置・幅を314px内に収まるよう調整し、全4フレームに適用した。

---

## このセッションで直した点

1. **`INTEL_ADVISORY`を削除。**
   元々「Behemoth is preparing a critical LASER_BEAM attack in 2 turns」と、モンスターの次ターンの攻撃を断定的に表示していた。spec.md「『今ターンの実際の攻撃』はどのテーブルにも書かない。ゲーム側(Goのメモリ)だけで保持し、DBには一切公開しない」という核となる隠蔽メカニクスに反するため、いったんテキストを書き換えたが、最終的にはブロックごと削除する判断になった。
2. **`left-panel-intel`の結果テーブルの列を修正。**
   元は`ATTACK_TYPE / PROB_PCT / MULT`という防御予測(③)向けの列構成だったが、上のSQLエディタが`SELECT * FROM monster_weaknesses WHERE severity >= 5`という弱点分析(①)のクエリだったため、クエリと結果が矛盾していた。列を`ID / COLUMN_NAME / VALUE / SEVERITY`に修正し、クエリ内容と整合させた。
3. **`left-panel-intel`のレイアウト調整。**
   `INTEL_ADVISORY`削除後にできた余白について試行錯誤し、最終的にパネルの高さは他の2カラムと揃えたまま(468px)、中身は下詰め(`primaryAxisAlignItems: MAX`)にして、余白がタイトル上部に来る形にした。

---

## 未解決・次回に持ち越し

- フェーズタブは`DONE` / `ACTIVE TURN` / `PENDING`の3状態で色分け(緑 / シアン / グレー)。フェーズ2〜4のFigmaフレームにも同じ配色ルールを適用済み。
- 設定パネル(バトルメニュー)の開閉インタラクション(Figmaのプロトタイピング接続)は未設定。中身のレイアウトのみ確定。
- **数値はすべて仮**(モンスターHP 10,000、プレイヤーHP 1,200など)。spec.mdの「未定事項」にある通り具体的なバランス数値は未確定。
- 【解決済み】モンスターカードの画像は、HUD風テキスト(`TARGET_ID` / `ARMOR_CLASS` / `WEAKNESS`等)を焼き込んだものから、モンスターだけを写した画像に差し替え済み。ホーム画面のプレビューでも同じ画像をそのまま再利用できる状態になっている。

---


## 関連

- ゲームメカニクス全体: `spec.md`

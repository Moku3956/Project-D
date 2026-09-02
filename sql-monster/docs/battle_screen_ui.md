# バトル画面UI設計

`sql-battle-screen`(Figma)のレイアウト設計。spec.mdのゲームメカニクスをどう画面に落とすかの記録。まだ「軽く動かすもの」を作るための土台段階で、細部の数値・演出は未確定。

- Figmaファイル: https://www.figma.com/design/SYNd796ebb3ETGhECYVlFc/ (node-id `2:4`, フレーム名 `sql-battle-screen`)
- ファイルには他にテンプレート素材が入っていたが全て削除済み。`sql-battle-screen`(フェーズ1)の右に、フェーズ2〜4を`sql-battle-screen-phase2` / `-phase3` / `-phase4`として複製・編集済み。さらに右に、設定パネルの中身を`settings-panel`として単独フレームで用意。
- **`sql-battle-screen-v2-viewport(1440x700)`という別フレームで、下記「v2: スクロールしないレイアウト」の設計を検証中。** 元の`sql-battle-screen`(フェーズ1〜4)は削除せず残してある。実装(`sql-monster/frontend`)を差し替える際はv2の構成を正とする。

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

## v2: スクロールしないレイアウト

コード実装したところ、バトル画面全体を見るのにスクロールが必要になった。**スクロールしないサイズにする、という制約を忘れていたのが原因。** フェーズ1〜4を別Figmaフレームにしたことも、「1画面内で状態が切り替わる」という前提を「画面ごとスクロールして見比べる」という誤った前提にすり替えてしまっていた。実装のCSSだけで帳尻を合わせず、**Figmaの設計自体をブラウザの実表示領域に収まるサイズで作り直す**ことにした。

### キャンバスサイズを実測可能にする

Figmaのauto-layoutフレームは中身に応じて`primaryAxisSizingMode: HUG`で伸縮するため、そのままだと際限なく高さが伸びる。ルートを`HUG`のままにすると「収まっているかどうか」を目視で確認できないため、**幅1440×高さ700の固定サイズ・`clipsContent: true`のプレーンフレーム(auto-layoutではない)で全体を包んだ。** 700pxは、ノートPC(1366×768)でブラウザのUIを引いた実効高さの見積もり。はみ出した分は切り取られて見えなくなるので、「収まっていない」ことがスクリーンショットで即座にわかる(実際、最初の再構成では856px相当あり、156px分がフッターごと見切れて発覚した)。

### 2カラム構成 + タブ切り替え

3カラム(RESULT/エディタ/PLAYER)構成だと、エディタの取り分が狭く長いクエリを書けない、テーブル定義を確認する場所もない、という問題があった。**「同時に見えている必要があるもの」と「切り替えれば十分なもの」を分けて設計し直した:**

```
header-bar は廃止。設定ボタン(三本線)は turn-phases の行内、フェーズタブの右隣に統合。
BATTLE_FLOW_PHASEというラベルも削除(タブを見れば分かるので不要)。

main-row (60:40)
  left-column
    tab-switcher-left: CARD | TABLE
    tab-content-left:
      CARD選択時 → モンスター画像(縦に大きく) + 名前/LV/HPバー/デバフ(画像の下)
      TABLE選択時 → テーブル定義の参照(未作成、次回)
  right-column
    section-title "PLAYER" + player-core-stats(HP/TURN) + resources-group(2種のリソースバー) ← 常時表示
    tab-switcher: EDITOR | RESULT | LOG
    tab-content:
      EDITOR選択時(初期状態) → SQLエディタ本体。クエリを実行すると自動でRESULTタブに切り替わる(仕様)
      RESULT選択時 → SELECT結果テーブル(未作成、次回)
      LOG選択時 → バトルログ(未作成、次回)

bottom-bar-ref(SQL操作対応表)はそのまま残す
```

**同時に3つ以上のブロックを常時表示しない**のが今回のキモ。「モンスター画像」と「テーブル定義」はどちらも見たいときだけでよいのでCARD/TABLEタブに、「エディタ」「実行結果」「ログ」も同様にEDITOR/RESULT/LOGタブにまとめた。これによって、常時表示が必要な要素(PLAYERステータス、フェーズ進行)だけに高さを使わせず、選択中の1ブロックに残りの高さを全部渡せる。

### 高さの現在地(1440×700に固定した状態)

| ブロック | 高さ |
|---|---|
| phases-row(フェーズタブ+設定ボタン) | 62px |
| left-column(CARDタブ: モンスター画像+情報) / right-column | 558px(左右で揃えてある) |
| うちSQLエディタ本体(textarea) | 260px(約12行) |
| bottom-bar-ref | 34px |

これは`sql-battle-screen-v2-viewport(1440x700)`というFigmaフレームで実測しながら詰めた値。個々のpadding/gapを機械的に同じ比率で縮める(700/実測値)アプローチを試したが、フォントサイズに底があるため思ったほど縮まず、最終的には要素ごとに手で数値を追って調整した。

### 設定ボタンの位置・QUERY_COSTの削除

- 設定ボタン(三本線)は`turn-phases`のカード枠の**外側**、右横に配置し直した。枠の中に入っていると「フェーズタブの一部の機能」に見えてしまうため。
- `QUERY_COST`表示(実行ボタンの上にあった行)を削除。浮いた分をエディタの高さに回し、**286px(13〜14行相当)**まで拡大した。コストは`RESULT`タブや`LOG`に実行後の情報として出す形で代替できるため、常時表示は不要と判断。

### TABLEタブ(反映済み — `sql-battle-screen-v2-table-tab`)

CARD/TABLEを切り替えた状態を別フレームとして用意。中身は`sql-monster/internal/game/schema.go`の定義と一致させた4テーブル(`monsters` / `monster_weaknesses` / `monster_attacks` / `players`)。

**当初はテーブル名+カラム一覧を縦に並べただけのリスト表示だったが、ER図に作り直した。** `monsters`を中心(白枠)に置き、`monster_id`で参照する`monster_weaknesses`・`monster_attacks`(シアン枠)を線で繋ぎ、関連のない`players`(グレー枠)は独立して配置している。カラムはPK/FKをそれぞれ色分けして明示。プレイヤーがSELECT文を書く際、カラム名だけでなくテーブル同士の繋がりも視覚的に把握できるようにする狙い。

### RESULT / LOGタブ(反映済み — `sql-battle-screen-v2-result-tab` / `-log-tab`)

TABLEタブと同じやり方で、それぞれタブが選択された状態を別フレームとして用意した。

- **RESULT**: `SELECT * FROM monster_weaknesses`の実行結果テーブル(既存モックの`terminal-result-client`をそのまま流用)。`実行(Run Query)すると自動でこのタブに切り替わる`という仕様どおりの見た目。
- **LOG**: 既存モックの`battle-log-container`をそのまま流用。任意のタイミングでタブを押せば見られる。

いずれも左右カラムの高さ(558px)・全体の高さ(700px)を他のフレームと揃えてある。

### フェーズ2〜4への反映(反映済み — `sql-battle-screen-v2-phase2` / `-phase3` / `-phase4`)

フェーズ1のv2フレームを複製し、各フェーズごとに以下を差し替えた(それ以外の構造・タブ挙動・700px制約はすべて共通):

| | クエリ | 実行ボタン文言 |
|---|---|---|
| フェーズ2 | `UPDATE monsters SET hp = 2580 WHERE id = 42 AND weakness = 'ACID';` | `Execute Attack` |
| フェーズ3 | `SELECT * FROM monster_attacks WHERE monster_id = 42 ORDER BY likelihood DESC;` | `Run Query` |
| フェーズ4 | `SELECT * FROM monster_attacks WHERE monster_id = 42 AND tag = 'LASER_BEAM';` | `Confirm Defense` |

フェーズタブの色分けは進行に応じて更新(現在地=シアン、それ以外=グレー)。DONE状態を示す文言・色は前回の設計で廃止済みのため、v2でも復活させていない(状態はタブの色だけで表現)。

TABLE/RESULT/LOGタブの中身(フェーズごとの実行結果差分)は、共通のCARD/TABLE/EDITOR/RESULT/LOGフレームで代表させており、フェーズ2〜4それぞれについて別途は作っていない。フェーズ2の実行結果(PREVIEW的な実測差分表示)は今後別途検討が必要。

### 未着手

- フェーズ2の`RESULT`タブの中身(実測HP差分のプレビュー表示)は、フェーズ1用のRESULTタブ(SELECT結果テーブル)とは性質が異なるため、別途デザインが必要。

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

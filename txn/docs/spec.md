# Transaction Manager 仕様

## 概要

トランザクションのライフサイクル・ロック・WALとの連携を管理する。

---

## Auto-commit

BEGINを書かない場合、1文ごとに自動でトランザクションを開始・コミットする。

```
SELECT * FROM users;
↓ 内部では
BEGIN → SELECT → COMMIT
```

明示的な BEGIN / COMMIT / ROLLBACK も使用できる。

---

## トランザクションの状態

```
Active → Committed
       → Aborted
```

- **Active**: 実行中
- **Committed**: COMMITが完了した
- **Aborted**: ROLLBACKまたはエラーで中断した

---

## TxnID

起動時から単調増加する `uint64`。WALログレコードに記録される。

---

## ロック

テーブルレベルの `sync.RWMutex` で管理する。

| 操作 | ロック種別 |
|------|-----------|
| SELECT | 読み取りロック（RLock） |
| INSERT / UPDATE / DELETE | 書き込みロック（Lock） |

- トランザクション内で初めてテーブルにアクセスしたときにロックを取得する
- COMMIT / ROLLBACK 時にすべてのロックを解放する

---

## デッドロード対策

タイムアウト方式を採用する。ロック取得の待機が一定時間を超えたらエラーを返す。デフォルトは5秒。

---

## COMMITの処理

1. WALログバッファにCOMMITレコードを書いてfsyncする
2. ロックをすべて解放する

ページのディスク書き込みはバッファプールのLRUに任せる（No-Force）。

---

## ROLLBACKの処理

1. バッファプール上の未コミットのdirtyページを破棄する
2. ロックをすべて解放する

WALのRedoログのみのためUndoは不要。

# Lexer 仕様

## 概要

SQL文字列をトークン列に変換する。パーサーはトークン列を受け取って処理する。

---

## トークン

```go
type Token struct {
    Type    TokenType
    Literal string
}
```

- `Type`: トークンの種別
- `Literal`: 実際の文字列（例: `"SELECT"`、`"users"`、`"20"`）

---

## TokenType

```go
type TokenType int

const (
    // キーワード
    SELECT TokenType = iota
    FROM
    WHERE
    INSERT
    INTO
    VALUES
    UPDATE
    SET
    DELETE
    CREATE
    TABLE
    DROP
    JOIN
    ON
    AND
    OR
    NOT
    IS
    NULL
    PRIMARY
    KEY
    BEGIN
    COMMIT
    ROLLBACK
    ORDER
    BY
    ASC
    DESC
    LIMIT
    INNER
    TRUE
    FALSE

    // 識別子・リテラル
    IDENT    // テーブル名・カラム名
    INT_LIT  // 整数リテラル（例: 20）
    STR_LIT  // 文字列リテラル（例: 'Alice'）

    // 演算子
    EQ       // =
    NEQ      // !=
    LT       // <
    GT       // >
    LTE      // <=
    GTE      // >=
    ASTERISK // *
    DOT      // .

    // 区切り文字
    COMMA     // ,
    SEMICOLON // ;
    LPAREN    // (
    RPAREN    // )

    // 制御
    EOF     // 終端
    ILLEGAL // 不正な文字
)
```

---

## 動作

- スペース・タブ・改行は無視する
- キーワードは大文字・小文字を区別しない（`select` と `SELECT` は同じ）
- 識別子はアルファベット・数字・アンダースコアで構成される
- 文字列リテラルはシングルクォートで囲む（`'Alice'`）
- 認識できない文字は `ILLEGAL` トークンとして返す

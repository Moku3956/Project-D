# Parser 仕様

## 概要

トークン列を受け取りASTを生成する。文の解析は再帰下降、式の解析はPrattパーサーを使う。

---

## 実装方式

- **文（Statement）**: 再帰下降パーサー
- **式（Expression）**: Prattパーサー（演算子の優先順位を正しく扱うため）

---

## エラー処理

エラーが発生しても即停止せず、可能な限り解析を続けて複数のエラーをまとめて返す。

```go
type ParseErrorKind int

const (
    // トークン関連
    UnexpectedToken
    UnexpectedEOF
    MissingToken

    // SELECT関連
    MissingColumnList
    MissingFromClause

    // JOIN関連
    MissingJoinTable
    MissingJoinCondition

    // WHERE関連
    MissingCondition

    // CREATE TABLE関連
    MissingTableName
    MissingColumnDef
    InvalidDataType
    DuplicatePrimaryKey

    // リテラル関連
    InvalidLiteral
)

type ParseError struct {
    Kind    ParseErrorKind
    Message string  // ユーザー向けの自然な文章
    Line    int
    Column  int
}
```

エラーメッセージはトークン名等の技術的な情報を含めず、ユーザーが理解できる表現にする。

---

## 演算子の優先順位

Prattパーサーで以下の優先順位を扱う（上が低い）。

```
OR
AND
NOT
=  !=  <  >  <=  >=  IS NULL  IS NOT NULL
```

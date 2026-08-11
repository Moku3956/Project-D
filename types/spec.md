# types 仕様

## 概要

全コンポーネントが共有する型定義。ロジックは持たない。

---

## Value

SQLの値を型付きで表現する。

```go
type ValueKind int

const (
    KindInt    ValueKind = iota
    KindString
    KindBool
    KindNull
)

type Value interface {
    valueKind() ValueKind
}

type IntValue    struct{ V int64 }
type StringValue struct{ V string }
type BoolValue   struct{ V bool }
type NullValue   struct{}
```

使用例:

```go
switch v := val.(type) {
case IntValue:    // v.V
case StringValue: // v.V
case BoolValue:   // v.V
case NullValue:   // NULL
}
```

---

## Row

1レコードのメモリ表現。カラム順にValueを並べたもの。

```go
type Row struct {
    Values []Value
}
```

---

## DataType

カラムのデータ型。

```go
type DataTypeKind int

const (
    KindIntType     DataTypeKind = iota
    KindVarcharType
    KindBoolType
)

type DataType struct {
    Kind   DataTypeKind
    Length int  // VARCHARのみ使用
}
```

---

## Column

カラム定義。

```go
type Column struct {
    Name       string
    Type       DataType
    PrimaryKey bool
    NotNull    bool
}
```

---

## Schema

テーブル定義。

```go
type Schema struct {
    TableName string
    Columns   []Column
}
```

`Schema.Columns` のインデックスが `Row.Values` のインデックスに対応する。

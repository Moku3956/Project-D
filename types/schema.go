package types

type DataTypeKind int

const (
	KindIntType     DataTypeKind = iota
	KindVarcharType DataTypeKind = iota
	KindBoolType    DataTypeKind = iota
)

type DataType struct {
	Kind   DataTypeKind
	Length int
}

type Column struct {
	Name       string
	Type       DataType
	PrimaryKey bool
	NotNull    bool
}

type Schema struct {
	TableName string
	Columns   []Column
}

func (s *Schema) PrimaryKeyIndex() int {
	for i, col := range s.Columns {
		if col.PrimaryKey {
			return i
		}
	}
	return -1
}

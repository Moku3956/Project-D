package types

type ValueKind int

const (
	KindInt    ValueKind = iota
	KindString ValueKind = iota
	KindBool   ValueKind = iota
	KindNull   ValueKind = iota
)

type Value interface {
	valueKind() ValueKind
}

type IntValue struct{ V int64 }
type StringValue struct{ V string }
type BoolValue struct{ V bool }
type NullValue struct{}

func (v IntValue) valueKind() ValueKind    { return KindInt }
func (v StringValue) valueKind() ValueKind { return KindString }
func (v BoolValue) valueKind() ValueKind   { return KindBool }
func (v NullValue) valueKind() ValueKind   { return KindNull }

func KindOf(v Value) ValueKind { return v.valueKind() }

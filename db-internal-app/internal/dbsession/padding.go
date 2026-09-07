package dbsession

import (
	"strings"

	"github.com/Moku3956/Project-D/sql/ast"
	"github.com/Moku3956/Project-D/types"
)

// idNumericDigitsはfrontendのMAX_RANDOM_ID(999)と合わせてある。数値部分を
// この桁数までゼロ埋めしてからパディングすることで、"5" < "76" のような
// 辞書順比較が数値順と食い違わないようにする。
const idNumericDigits = 3

// padPKLiterals はユーザーが書いたSQL中のPK列に対応するリテラル値を、実際に
// 格納する長さ(schemaの宣言長)まで自動でパディングする。db-internal-appは
// B+Treeの分岐を起こすためにPK列を長い文字列としてディスクに保存しており
// (docs/spec.md「Storage」参照)、以前はフロントエンドがSQL文字列に直接
// パディング済みの値を埋め込んでいた。しかし「76のようなシンプルなidで
// deleteしたい。受け取ったsqlを直接実行する実装になってるからそうなって
// しまうんでしょ？そうじゃなくて、まずはsqlを受け取って、バックエンドで
// idにパディングを追加したらいい」というユーザー指示により、ユーザー
// (フロントエンド)が書くSQLは常に短い素の値のままでよく、パディングは
// このdb-internal-app固有の便宜層だけが担うようにした。MokuDB本体
// (sql/parser・planner・executor)には一切手を入れていない。
func padPKLiterals(stmt ast.Statement, schema *types.Schema) {
	pkIdx := schema.PrimaryKeyIndex()
	if pkIdx < 0 {
		return
	}
	pkCol := schema.Columns[pkIdx]
	if pkCol.Type.Kind != types.KindVarcharType {
		return
	}

	switch s := stmt.(type) {
	case *ast.SelectStatement:
		padWhereLiterals(s.Where, pkCol.Name, pkCol.Type.Length)
	case *ast.InsertStatement:
		if pkIdx < len(s.Values) {
			if lit, ok := s.Values[pkIdx].(*ast.StringLiteral); ok {
				lit.Value = padID(lit.Value, pkCol.Type.Length)
			}
		}
	case *ast.UpdateStatement:
		for i := range s.Assignments {
			if s.Assignments[i].Column != pkCol.Name {
				continue
			}
			if lit, ok := s.Assignments[i].Value.(*ast.StringLiteral); ok {
				lit.Value = padID(lit.Value, pkCol.Type.Length)
			}
		}
		padWhereLiterals(s.Where, pkCol.Name, pkCol.Type.Length)
	case *ast.DeleteStatement:
		padWhereLiterals(s.Where, pkCol.Name, pkCol.Type.Length)
	}
}

// padWhereLiterals はWHERE句の式木を辿り、PK列との比較に使われている文字列
// リテラルをその場でパディングする。AND/ORはそのまま再帰し、比較演算子の
// 葉(左右どちらかがPK列のIdentifier、もう片方が文字列リテラル)だけを書き
// 換える。
func padWhereLiterals(expr ast.Expression, pkName string, pkLen int) {
	be, ok := expr.(*ast.BinaryExpr)
	if !ok {
		return
	}
	if be.Operator == ast.OpAND || be.Operator == ast.OpOR {
		padWhereLiterals(be.Left, pkName, pkLen)
		padWhereLiterals(be.Right, pkName, pkLen)
		return
	}
	if id, ok := be.Left.(*ast.Identifier); ok && id.Column == pkName {
		if lit, ok := be.Right.(*ast.StringLiteral); ok {
			lit.Value = padID(lit.Value, pkLen)
		}
		return
	}
	if id, ok := be.Right.(*ast.Identifier); ok && id.Column == pkName {
		if lit, ok := be.Left.(*ast.StringLiteral); ok {
			lit.Value = padID(lit.Value, pkLen)
		}
	}
}

// padID は生の値をtargetLenまでパディングする。既にtargetLen以上の長さが
// あれば(=既にパディング済み、または対象外)そのまま返す。数字だけの値は
// idNumericDigits桁までゼロ埋めしてから'x'で埋める。数字以外はそのまま
// 'x'で埋める。
func padID(raw string, targetLen int) string {
	if targetLen <= 0 || len(raw) >= targetLen {
		return raw
	}
	numeric := raw
	if isAllDigits(raw) && len(numeric) < idNumericDigits {
		numeric = strings.Repeat("0", idNumericDigits-len(numeric)) + numeric
	}
	if len(numeric) >= targetLen {
		return numeric[:targetLen]
	}
	return numeric + strings.Repeat("x", targetLen-len(numeric))
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

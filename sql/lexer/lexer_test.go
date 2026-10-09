package lexer

import (
	"testing"
)

// ---- 正常系 ----

func TestTokenizeSelect(t *testing.T) {
	tokens := NewLexer("SELECT id FROM users").Tokenize()

	want := []struct {
		tt  TokenType
		lit string
	}{
		{SELECT, "SELECT"},
		{IDENT, "id"},
		{FROM, "FROM"},
		{IDENT, "users"},
		{EOF, ""},
	}

	if len(tokens) != len(want) {
		t.Fatalf("トークン数 = %d, want %d", len(tokens), len(want))
	}
	for i, w := range want {
		if tokens[i].Type != w.tt {
			t.Errorf("tokens[%d].Type = %v, want %v", i, tokens[i].Type, w.tt)
		}
		if tokens[i].Literal != w.lit {
			t.Errorf("tokens[%d].Literal = %q, want %q", i, tokens[i].Literal, w.lit)
		}
	}
}

func TestTokenizeWhere(t *testing.T) {
	tokens := NewLexer("WHERE age > 20 AND name = 'Alice'").Tokenize()

	want := []TokenType{WHERE, IDENT, GT, INT_LIT, AND, IDENT, EQ, STR_LIT, EOF}
	if len(tokens) != len(want) {
		t.Fatalf("トークン数 = %d, want %d", len(tokens), len(want))
	}
	for i, tt := range want {
		if tokens[i].Type != tt {
			t.Errorf("tokens[%d].Type = %v, want %v", i, tokens[i].Type, tt)
		}
	}
}

func TestTokenizeOperators(t *testing.T) {
	tokens := NewLexer("= != < > <= >=").Tokenize()

	want := []struct {
		tt  TokenType
		lit string
	}{
		{EQ, "="},
		{NEQ, "!="},
		{LT, "<"},
		{GT, ">"},
		{LTE, "<="},
		{GTE, ">="},
		{EOF, ""},
	}

	if len(tokens) != len(want) {
		t.Fatalf("トークン数 = %d, want %d", len(tokens), len(want))
	}
	for i, w := range want {
		if tokens[i].Type != w.tt || tokens[i].Literal != w.lit {
			t.Errorf("tokens[%d] = {%v, %q}, want {%v, %q}", i, tokens[i].Type, tokens[i].Literal, w.tt, w.lit)
		}
	}
}

func TestTokenizeIntLiteral(t *testing.T) {
	tokens := NewLexer("42").Tokenize()

	if tokens[0].Type != INT_LIT {
		t.Errorf("Type = %v, want INT_LIT", tokens[0].Type)
	}
	if tokens[0].Literal != "42" {
		t.Errorf("Literal = %q, want \"42\"", tokens[0].Literal)
	}
}

func TestTokenizeStringLiteral(t *testing.T) {
	tokens := NewLexer("'hello world'").Tokenize()

	if tokens[0].Type != STR_LIT {
		t.Errorf("Type = %v, want STR_LIT", tokens[0].Type)
	}
	if tokens[0].Literal != "hello world" {
		t.Errorf("Literal = %q, want \"hello world\"", tokens[0].Literal)
	}
}

func TestTokenizeKeywordCaseInsensitive(t *testing.T) {
	tokens := NewLexer("select from where").Tokenize()

	want := []TokenType{SELECT, FROM, WHERE, EOF}
	for i, tt := range want {
		if tokens[i].Type != tt {
			t.Errorf("tokens[%d].Type = %v, want %v", i, tokens[i].Type, tt)
		}
	}
}

func TestTokenizeSymbols(t *testing.T) {
	tokens := NewLexer("( ) , ; * .").Tokenize()

	want := []TokenType{LPAREN, RPAREN, COMMA, SEMICOLON, ASTERISK, DOT, EOF}
	for i, tt := range want {
		if tokens[i].Type != tt {
			t.Errorf("tokens[%d].Type = %v, want %v", i, tokens[i].Type, tt)
		}
	}
}

func TestTokenizeTableDotColumn(t *testing.T) {
	tokens := NewLexer("users.id").Tokenize()

	want := []struct {
		tt  TokenType
		lit string
	}{
		{IDENT, "users"},
		{DOT, "."},
		{IDENT, "id"},
		{EOF, ""},
	}
	for i, w := range want {
		if tokens[i].Type != w.tt || tokens[i].Literal != w.lit {
			t.Errorf("tokens[%d] = {%v, %q}, want {%v, %q}", i, tokens[i].Type, tokens[i].Literal, w.tt, w.lit)
		}
	}
}

func TestLineColTracking(t *testing.T) {
	tokens := NewLexer("SELECT\nid").Tokenize()

	if tokens[0].Line != 1 || tokens[0].Col != 1 {
		t.Errorf("SELECT: Line=%d Col=%d, want Line=1 Col=1", tokens[0].Line, tokens[0].Col)
	}
	if tokens[1].Line != 2 || tokens[1].Col != 1 {
		t.Errorf("id: Line=%d Col=%d, want Line=2 Col=1", tokens[1].Line, tokens[1].Col)
	}
}

func TestTokenizeFullSelect(t *testing.T) {
	sql := "SELECT id, name FROM users WHERE id = 1"
	tokens := NewLexer(sql).Tokenize()

	want := []TokenType{SELECT, IDENT, COMMA, IDENT, FROM, IDENT, WHERE, IDENT, EQ, INT_LIT, EOF}
	if len(tokens) != len(want) {
		t.Fatalf("トークン数 = %d, want %d", len(tokens), len(want))
	}
	for i, tt := range want {
		if tokens[i].Type != tt {
			t.Errorf("tokens[%d].Type = %v, want %v", i, tokens[i].Type, tt)
		}
	}
}

// ---- 異常系 ----

func TestTokenizeIllegal(t *testing.T) {
	tokens := NewLexer("@").Tokenize()

	if tokens[0].Type != ILLEGAL {
		t.Errorf("Type = %v, want ILLEGAL", tokens[0].Type)
	}
}

func TestTokenizeUnclosedString(t *testing.T) {
	tokens := NewLexer("'hello").Tokenize()

	if tokens[0].Type != STR_LIT {
		t.Errorf("Type = %v, want STR_LIT", tokens[0].Type)
	}
	if tokens[0].Literal != "hello" {
		t.Errorf("Literal = %q, want \"hello\"", tokens[0].Literal)
	}
}

func TestTokenizeEmpty(t *testing.T) {
	tokens := NewLexer("").Tokenize()

	if len(tokens) != 1 || tokens[0].Type != EOF {
		t.Errorf("空文字列のTokenize結果が不正: %v", tokens)
	}
}

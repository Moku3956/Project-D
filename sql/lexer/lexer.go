package lexer

import "strings"

type TokenType int

const (
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

	IDENT
	INT_LIT
	STR_LIT

	EQ
	NEQ
	LT
	GT
	LTE
	GTE
	ASTERISK
	DOT

	COMMA
	SEMICOLON
	LPAREN
	RPAREN

	EOF
	ILLEGAL
)

var keywords = map[string]TokenType{
	"SELECT":   SELECT,
	"FROM":     FROM,
	"WHERE":    WHERE,
	"INSERT":   INSERT,
	"INTO":     INTO,
	"VALUES":   VALUES,
	"UPDATE":   UPDATE,
	"SET":      SET,
	"DELETE":   DELETE,
	"CREATE":   CREATE,
	"TABLE":    TABLE,
	"DROP":     DROP,
	"JOIN":     JOIN,
	"ON":       ON,
	"AND":      AND,
	"OR":       OR,
	"NOT":      NOT,
	"IS":       IS,
	"NULL":     NULL,
	"PRIMARY":  PRIMARY,
	"KEY":      KEY,
	"BEGIN":    BEGIN,
	"COMMIT":   COMMIT,
	"ROLLBACK": ROLLBACK,
	"ORDER":    ORDER,
	"BY":       BY,
	"ASC":      ASC,
	"DESC":     DESC,
	"LIMIT":    LIMIT,
	"INNER":    INNER,
	"TRUE":     TRUE,
	"FALSE":    FALSE,
}

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Col     int
}

type Lexer struct {
	input  string
	pos    int
	line   int
	col    int
}

func New(input string) *Lexer {
	return &Lexer{input: input, line: 1, col: 1}
}

func (l *Lexer) Tokenize() []Token {
	var tokens []Token
	for {
		tok := l.next()
		tokens = append(tokens, tok)
		if tok.Type == EOF {
			break
		}
	}
	return tokens
}

func (l *Lexer) next() Token {
	l.skipWhitespace()

	if l.pos >= len(l.input) {
		return l.tok(EOF, "")
	}

	ch := l.input[l.pos]

	switch {
	case ch == '\'':
		return l.readString()
	case isDigit(ch):
		return l.readInt()
	case isLetter(ch) || ch == '_':
		return l.readIdent()
	}

	line, col := l.line, l.col
	l.advance()

	switch ch {
	case '=':
		return Token{Type: EQ, Literal: "=", Line: line, Col: col}
	case '!':
		if l.pos < len(l.input) && l.input[l.pos] == '=' {
			l.advance()
			return Token{Type: NEQ, Literal: "!=", Line: line, Col: col}
		}
		return Token{Type: ILLEGAL, Literal: "!", Line: line, Col: col}
	case '<':
		if l.pos < len(l.input) && l.input[l.pos] == '=' {
			l.advance()
			return Token{Type: LTE, Literal: "<=", Line: line, Col: col}
		}
		return Token{Type: LT, Literal: "<", Line: line, Col: col}
	case '>':
		if l.pos < len(l.input) && l.input[l.pos] == '=' {
			l.advance()
			return Token{Type: GTE, Literal: ">=", Line: line, Col: col}
		}
		return Token{Type: GT, Literal: ">", Line: line, Col: col}
	case '*':
		return Token{Type: ASTERISK, Literal: "*", Line: line, Col: col}
	case '.':
		return Token{Type: DOT, Literal: ".", Line: line, Col: col}
	case ',':
		return Token{Type: COMMA, Literal: ",", Line: line, Col: col}
	case ';':
		return Token{Type: SEMICOLON, Literal: ";", Line: line, Col: col}
	case '(':
		return Token{Type: LPAREN, Literal: "(", Line: line, Col: col}
	case ')':
		return Token{Type: RPAREN, Literal: ")", Line: line, Col: col}
	}

	return Token{Type: ILLEGAL, Literal: string(ch), Line: line, Col: col}
}

func (l *Lexer) readIdent() Token {
	line, col := l.line, l.col
	start := l.pos
	for l.pos < len(l.input) && (isLetter(l.input[l.pos]) || isDigit(l.input[l.pos]) || l.input[l.pos] == '_') {
		l.advance()
	}
	lit := l.input[start:l.pos]
	upper := strings.ToUpper(lit)
	if tt, ok := keywords[upper]; ok {
		return Token{Type: tt, Literal: upper, Line: line, Col: col}
	}
	return Token{Type: IDENT, Literal: lit, Line: line, Col: col}
}

func (l *Lexer) readInt() Token {
	line, col := l.line, l.col
	start := l.pos
	for l.pos < len(l.input) && isDigit(l.input[l.pos]) {
		l.advance()
	}
	return Token{Type: INT_LIT, Literal: l.input[start:l.pos], Line: line, Col: col}
}

func (l *Lexer) readString() Token {
	line, col := l.line, l.col
	l.advance() // 開き '
	start := l.pos
	for l.pos < len(l.input) && l.input[l.pos] != '\'' {
		l.advance()
	}
	lit := l.input[start:l.pos]
	if l.pos < len(l.input) {
		l.advance() // 閉じ '
	}
	return Token{Type: STR_LIT, Literal: lit, Line: line, Col: col}
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
			l.advance()
		} else {
			break
		}
	}
}

func (l *Lexer) advance() {
	if l.pos < len(l.input) && l.input[l.pos] == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	l.pos++
}

func (l *Lexer) tok(tt TokenType, lit string) Token {
	return Token{Type: tt, Literal: lit, Line: l.line, Col: l.col}
}

func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

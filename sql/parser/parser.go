package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Moku3956/Project-D/sql/ast"
	"github.com/Moku3956/Project-D/sql/lexer"
	"github.com/Moku3956/Project-D/types"
)

type ParseErrorKind int

const (
	UnexpectedToken ParseErrorKind = iota
	UnexpectedEOF
	MissingToken
	MissingColumnList
	MissingFromClause
	MissingJoinTable
	MissingJoinCondition
	MissingCondition
	MissingTableName
	MissingColumnDef
	InvalidDataType
	DuplicatePrimaryKey
	InvalidLiteral
)

type ParseError struct {
	Kind    ParseErrorKind
	Message string
	Line    int
	Col     int
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d col %d: %s", e.Line, e.Col, e.Message)
}

type ParseErrors []*ParseError

func (pe ParseErrors) Error() string {
	msgs := make([]string, len(pe))
	for i, e := range pe {
		msgs[i] = e.Error()
	}
	return strings.Join(msgs, "\n")
}

type Parser struct {
	tokens []lexer.Token
	pos    int
	errors ParseErrors
}

func New(tokens []lexer.Token) *Parser {
	return &Parser{tokens: tokens}
}

func Parse(input string) (ast.Statement, error) {
	l := lexer.New(input)
	tokens := l.Tokenize()
	p := New(tokens)
	stmt := p.parseStatement()
	if len(p.errors) > 0 {
		return nil, p.errors
	}
	return stmt, nil
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.cur().Type {
	case lexer.SELECT:
		return p.parseSelect()
	case lexer.INSERT:
		return p.parseInsert()
	case lexer.UPDATE:
		return p.parseUpdate()
	case lexer.DELETE:
		return p.parseDelete()
	case lexer.CREATE:
		return p.parseCreateTable()
	case lexer.DROP:
		return p.parseDropTable()
	case lexer.BEGIN:
		p.advance()
		return &ast.BeginStatement{}
	case lexer.COMMIT:
		p.advance()
		return &ast.CommitStatement{}
	case lexer.ROLLBACK:
		p.advance()
		return &ast.RollbackStatement{}
	default:
		p.addError(UnexpectedToken, "予期しないトークンです", p.cur())
		return nil
	}
}

// SELECT columns FROM table [JOIN ...] [WHERE ...] [ORDER BY ...] [LIMIT ...]
func (p *Parser) parseSelect() ast.Statement {
	p.expect(lexer.SELECT)
	stmt := &ast.SelectStatement{}

	if p.cur().Type == lexer.EOF {
		p.addError(MissingColumnList, "SELECT の後にカラムリストが必要です", p.cur())
		return stmt
	}
	stmt.Columns = p.parseExprList()

	if !p.expectWithError(lexer.FROM, MissingFromClause, "FROM が必要です") {
		return stmt
	}
	stmt.Table = p.expectIdent()

	if p.cur().Type == lexer.INNER || p.cur().Type == lexer.JOIN {
		stmt.Join = p.parseJoin()
	}

	if p.cur().Type == lexer.WHERE {
		p.advance()
		stmt.Where = p.parseExpr(0)
	}

	if p.cur().Type == lexer.ORDER {
		p.advance()
		p.expect(lexer.BY)
		col := p.expectIdent()
		desc := false
		if p.cur().Type == lexer.DESC {
			desc = true
			p.advance()
		} else if p.cur().Type == lexer.ASC {
			p.advance()
		}
		stmt.OrderBy = &ast.OrderByClause{Column: col, Desc: desc}
	}

	if p.cur().Type == lexer.LIMIT {
		p.advance()
		tok := p.cur()
		if tok.Type != lexer.INT_LIT {
			p.addError(InvalidLiteral, "LIMIT の後に整数が必要です", tok)
		} else {
			n, _ := strconv.Atoi(tok.Literal)
			stmt.Limit = &n
			p.advance()
		}
	}

	return stmt
}

func (p *Parser) parseJoin() *ast.JoinClause {
	if p.cur().Type == lexer.INNER {
		p.advance()
	}
	p.expect(lexer.JOIN)
	if p.cur().Type != lexer.IDENT {
		p.addError(MissingJoinTable, "JOIN の後にテーブル名が必要です", p.cur())
		return nil
	}
	table := p.expectIdent()
	if !p.expectWithError(lexer.ON, MissingJoinCondition, "JOIN の後に ON が必要です") {
		return &ast.JoinClause{Table: table}
	}
	cond := p.parseExpr(0)
	return &ast.JoinClause{Table: table, Condition: cond}
}

// INSERT INTO table VALUES (expr, ...)
func (p *Parser) parseInsert() ast.Statement {
	p.expect(lexer.INSERT)
	p.expect(lexer.INTO)
	stmt := &ast.InsertStatement{}
	stmt.Table = p.expectIdent()
	p.expect(lexer.VALUES)
	p.expect(lexer.LPAREN)
	stmt.Values = p.parseExprList()
	p.expect(lexer.RPAREN)
	return stmt
}

// UPDATE table SET col=val [, ...] [WHERE ...]
func (p *Parser) parseUpdate() ast.Statement {
	p.expect(lexer.UPDATE)
	stmt := &ast.UpdateStatement{}
	stmt.Table = p.expectIdent()
	p.expect(lexer.SET)
	stmt.Assignments = p.parseAssignments()
	if p.cur().Type == lexer.WHERE {
		p.advance()
		stmt.Where = p.parseExpr(0)
	}
	return stmt
}

func (p *Parser) parseAssignments() []ast.Assignment {
	var assignments []ast.Assignment
	for {
		col := p.expectIdent()
		p.expect(lexer.EQ)
		val := p.parseExpr(0)
		assignments = append(assignments, ast.Assignment{Column: col, Value: val})
		if p.cur().Type != lexer.COMMA {
			break
		}
		p.advance()
	}
	return assignments
}

// DELETE FROM table [WHERE ...]
func (p *Parser) parseDelete() ast.Statement {
	p.expect(lexer.DELETE)
	p.expect(lexer.FROM)
	stmt := &ast.DeleteStatement{}
	stmt.Table = p.expectIdent()
	if p.cur().Type == lexer.WHERE {
		p.advance()
		stmt.Where = p.parseExpr(0)
	}
	return stmt
}

// CREATE TABLE name (col type [PRIMARY KEY] [NOT NULL], ...)
func (p *Parser) parseCreateTable() ast.Statement {
	p.expect(lexer.CREATE)
	p.expect(lexer.TABLE)
	stmt := &ast.CreateTableStatement{}

	if p.cur().Type != lexer.IDENT {
		p.addError(MissingTableName, "CREATE TABLE の後にテーブル名が必要です", p.cur())
		return stmt
	}
	stmt.TableName = p.expectIdent()
	p.expect(lexer.LPAREN)

	pkCount := 0
	for p.cur().Type != lexer.RPAREN && p.cur().Type != lexer.EOF {
		col, isPK := p.parseColumnDef()
		if isPK {
			pkCount++
			if pkCount > 1 {
				p.addError(DuplicatePrimaryKey, "PRIMARY KEY は1つだけ指定できます", p.cur())
			}
		}
		stmt.Columns = append(stmt.Columns, col)
		if p.cur().Type == lexer.COMMA {
			p.advance()
		}
	}

	if len(stmt.Columns) == 0 {
		p.addError(MissingColumnDef, "カラム定義が必要です", p.cur())
	}
	p.expect(lexer.RPAREN)
	return stmt
}

func (p *Parser) parseColumnDef() (types.Column, bool) {
	col := types.Column{}
	col.Name = p.expectIdent()
	dt, err := p.parseDataType()
	if err != nil {
		p.addError(InvalidDataType, err.Error(), p.cur())
	}
	col.Type = dt

	isPK := false
	for p.cur().Type == lexer.PRIMARY || p.cur().Type == lexer.NOT || p.cur().Type == lexer.NULL {
		switch p.cur().Type {
		case lexer.PRIMARY:
			p.advance()
			p.expect(lexer.KEY)
			col.PrimaryKey = true
			col.NotNull = true
			isPK = true
		case lexer.NOT:
			p.advance()
			p.expect(lexer.NULL)
			col.NotNull = true
		}
	}
	return col, isPK
}

func (p *Parser) parseDataType() (types.DataType, error) {
	tok := p.cur()
	upper := strings.ToUpper(tok.Literal)
	switch upper {
	case "INT":
		p.advance()
		return types.DataType{Kind: types.KindIntType}, nil
	case "BOOL":
		p.advance()
		return types.DataType{Kind: types.KindBoolType}, nil
	case "VARCHAR":
		p.advance()
		p.expect(lexer.LPAREN)
		lenTok := p.cur()
		if lenTok.Type != lexer.INT_LIT {
			return types.DataType{}, fmt.Errorf("VARCHAR の長さに整数が必要です")
		}
		length, _ := strconv.Atoi(lenTok.Literal)
		p.advance()
		p.expect(lexer.RPAREN)
		return types.DataType{Kind: types.KindVarcharType, Length: length}, nil
	default:
		p.advance()
		return types.DataType{}, fmt.Errorf("不明な型: %s", tok.Literal)
	}
}

// DROP TABLE name
func (p *Parser) parseDropTable() ast.Statement {
	p.expect(lexer.DROP)
	p.expect(lexer.TABLE)
	stmt := &ast.DropTableStatement{}
	if p.cur().Type != lexer.IDENT {
		p.addError(MissingTableName, "DROP TABLE の後にテーブル名が必要です", p.cur())
		return stmt
	}
	stmt.TableName = p.expectIdent()
	return stmt
}

// ---- 式パーサー (Pratt) ----

type precedence int

const (
	precLowest  precedence = iota
	precOr                 // OR
	precAnd                // AND
	precNot                // NOT
	precCompare            // = != < > <= >=
)

func tokenPrec(tt lexer.TokenType) precedence {
	switch tt {
	case lexer.OR:
		return precOr
	case lexer.AND:
		return precAnd
	case lexer.EQ, lexer.NEQ, lexer.LT, lexer.GT, lexer.LTE, lexer.GTE:
		return precCompare
	}
	return precLowest
}

func (p *Parser) parseExpr(minPrec precedence) ast.Expression {
	var left ast.Expression

	if p.cur().Type == lexer.NOT {
		p.advance()
		operand := p.parseExpr(precNot)
		left = &ast.UnaryExpr{Operator: "NOT", Operand: operand}
	} else {
		left = p.parsePrimary()
	}

	// IS NULL / IS NOT NULL
	if p.cur().Type == lexer.IS {
		p.advance()
		if p.cur().Type == lexer.NOT {
			p.advance()
			p.expect(lexer.NULL)
			return &ast.IsNotNullExpr{Operand: left}
		}
		p.expect(lexer.NULL)
		return &ast.IsNullExpr{Operand: left}
	}

	for {
		prec := tokenPrec(p.cur().Type)
		if prec <= minPrec {
			break
		}
		op := tokenToOp(p.cur().Type)
		p.advance()
		right := p.parseExpr(prec)
		left = &ast.BinaryExpr{Left: left, Operator: op, Right: right}
	}
	return left
}

func (p *Parser) parsePrimary() ast.Expression {
	tok := p.cur()
	switch tok.Type {
	case lexer.INT_LIT:
		p.advance()
		v, _ := strconv.ParseInt(tok.Literal, 10, 64)
		return &ast.IntLiteral{Value: v}
	case lexer.STR_LIT:
		p.advance()
		return &ast.StringLiteral{Value: tok.Literal}
	case lexer.TRUE:
		p.advance()
		return &ast.BoolLiteral{Value: true}
	case lexer.FALSE:
		p.advance()
		return &ast.BoolLiteral{Value: false}
	case lexer.NULL:
		p.advance()
		return &ast.NullLiteral{}
	case lexer.ASTERISK:
		p.advance()
		return &ast.Wildcard{}
	case lexer.IDENT:
		name := tok.Literal
		p.advance()
		if p.cur().Type == lexer.DOT {
			p.advance()
			col := p.expectIdent()
			return &ast.Identifier{Table: name, Column: col}
		}
		return &ast.Identifier{Column: name}
	case lexer.LPAREN:
		p.advance()
		expr := p.parseExpr(precLowest)
		p.expect(lexer.RPAREN)
		return expr
	default:
		p.addError(UnexpectedToken, "式が必要です", tok)
		p.advance()
		return &ast.NullLiteral{}
	}
}

func (p *Parser) parseExprList() []ast.Expression {
	var exprs []ast.Expression
	exprs = append(exprs, p.parseExpr(precLowest))
	for p.cur().Type == lexer.COMMA {
		p.advance()
		exprs = append(exprs, p.parseExpr(precLowest))
	}
	return exprs
}

func tokenToOp(tt lexer.TokenType) ast.OperatorType {
	switch tt {
	case lexer.EQ:
		return ast.OpEQ
	case lexer.NEQ:
		return ast.OpNEQ
	case lexer.LT:
		return ast.OpLT
	case lexer.GT:
		return ast.OpGT
	case lexer.LTE:
		return ast.OpLTE
	case lexer.GTE:
		return ast.OpGTE
	case lexer.AND:
		return ast.OpAND
	case lexer.OR:
		return ast.OpOR
	}
	return ast.OpEQ
}

// ---- ヘルパー ----

func (p *Parser) cur() lexer.Token {
	if p.pos >= len(p.tokens) {
		return lexer.Token{Type: lexer.EOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

func (p *Parser) expect(tt lexer.TokenType) bool {
	if p.cur().Type == tt {
		p.advance()
		return true
	}
	p.addError(MissingToken, fmt.Sprintf("'%v' が必要です", tt), p.cur())
	return false
}

func (p *Parser) expectWithError(tt lexer.TokenType, kind ParseErrorKind, msg string) bool {
	if p.cur().Type == tt {
		p.advance()
		return true
	}
	p.addError(kind, msg, p.cur())
	return false
}

func (p *Parser) expectIdent() string {
	tok := p.cur()
	if tok.Type != lexer.IDENT {
		p.addError(UnexpectedToken, "識別子が必要です", tok)
		return ""
	}
	p.advance()
	return tok.Literal
}

func (p *Parser) addError(kind ParseErrorKind, msg string, tok lexer.Token) {
	p.errors = append(p.errors, &ParseError{
		Kind:    kind,
		Message: msg,
		Line:    tok.Line,
		Col:     tok.Col,
	})
}

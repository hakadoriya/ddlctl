package spanner

// MEMO: https://www.postgresql.org/docs/current/ddl-constraints.html
// MEMO: https://www.postgresql.jp/docs/11/ddl-constraints.html

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/hakadoriya/z.go/pathz/filepathz"
	"github.com/hakadoriya/z.go/stringz"

	"github.com/hakadoriya/ddlctl/pkg/apperr"
	"github.com/hakadoriya/ddlctl/pkg/ddl"
	"github.com/hakadoriya/ddlctl/pkg/logs"
)

//nolint:gochecknoglobals
var quotationMarks = []string{`"`, "`", "'"}

func NewRawIdent(raw string) *Ident {
	for _, q := range quotationMarks {
		if strings.HasPrefix(raw, q) && strings.HasSuffix(raw, q) {
			return &Ident{
				Name:          strings.Trim(raw, q),
				QuotationMark: q,
				Raw:           raw,
			}
		}
	}

	return &Ident{
		Name:          raw,
		QuotationMark: "",
		Raw:           raw,
	}
}

func NewIdent(name, quotationMark, raw string) *Ident {
	return &Ident{
		Name:          name,
		QuotationMark: quotationMark,
		Raw:           raw,
	}
}

// Parser はSQL文を解析するパーサーです。
type Parser struct {
	l            *Lexer
	currentToken Token
	peekToken    Token
}

// NewParser は新しいParserを生成します。
func NewParser(l *Lexer) *Parser {
	p := &Parser{
		l: l,
	}

	return p
}

// nextToken は次のトークンを読み込みます。
func (p *Parser) nextToken() {
	p.currentToken = p.peekToken
	p.peekToken = p.l.NextToken()

	_, file, line, _ := runtime.Caller(1)
	logs.Trace.Printf("🪲: nextToken: caller=%s:%d currentToken: %#v, peekToken: %#v", filepathz.ExtractShortPath(file), line, p.currentToken, p.peekToken)
}

// Parse はSQL文を解析します。
func (p *Parser) Parse() (*DDL, error) { //nolint:ireturn
	p.nextToken() // current = ""
	p.nextToken() // current = CREATE or ALTER or ...

	d := &DDL{}

LabelDDL:
	for {
		switch p.currentToken.Type { //nolint:exhaustive
		case TOKEN_CREATE:
			stmt, err := p.parseCreateStatement()
			if err != nil {
				return nil, apperr.Errorf("parseCreateStatement: %w", err)
			}
			d.Stmts = append(d.Stmts, stmt)
		case TOKEN_CLOSE_PAREN:
			// do nothing
		case TOKEN_SEMICOLON:
			// do nothing
		case TOKEN_EOF:
			break LabelDDL
		default:
			return nil, apperr.Errorf("currentToken=%#v, peekToken=%#v: %w", p.currentToken, p.peekToken, ddl.ErrUnexpectedCurrentToken)
		}

		p.nextToken()
	}
	return d, nil
}

func (p *Parser) parseCreateStatement() (Stmt, error) { //nolint:ireturn
	p.nextToken() // current = TABLE or INDEX or ...

	switch p.currentToken.Type { //nolint:exhaustive
	case TOKEN_TABLE:
		stmt, err := p.parseCreateTableStmt()
		if err != nil {
			return nil, apperr.Errorf("parseCreateTableStmt: %w", err)
		}
		return stmt, nil
	case TOKEN_INDEX, TOKEN_UNIQUE:
		stmt, err := p.parseCreateIndexStmt()
		if err != nil {
			return nil, apperr.Errorf("parseCreateIndexStmt: %w", err)
		}
		return stmt, nil
	default:
		return nil, apperr.Errorf("currentToken=%#v, peekToken=%#v: %w", p.currentToken, p.peekToken, ddl.ErrUnexpectedCurrentToken)
	}
}

//nolint:cyclop,funlen,gocognit,gocyclo,maintidx
func (p *Parser) parseCreateTableStmt() (*CreateTableStmt, error) {
	createTableStmt := &CreateTableStmt{
		Indent: Indent,
	}

	if p.isPeekToken(TOKEN_IF) {
		p.nextToken() // current = IF
		if err := p.checkPeekToken(TOKEN_NOT); err != nil {
			return nil, apperr.Errorf("checkPeekToken: %w", err)
		}
		p.nextToken() // current = NOT
		if err := p.checkPeekToken(TOKEN_EXISTS); err != nil {
			return nil, apperr.Errorf("checkPeekToken: %w", err)
		}
		p.nextToken() // current = EXISTS
		createTableStmt.IfNotExists = true
	}

	p.nextToken() // current = table_name
	if err := p.checkCurrentToken(TOKEN_IDENT); err != nil {
		return nil, apperr.Errorf("checkCurrentToken: %w", err)
	}

	createTableStmt.Name = NewObjectName(p.currentToken.Literal.Str)
	errFmtPrefix := fmt.Sprintf("table_name=%s: ", createTableStmt.Name.StringForDiff())

	p.nextToken() // current = (

	if err := p.checkCurrentToken(TOKEN_OPEN_PAREN); err != nil {
		return nil, apperr.Errorf(errFmtPrefix+"checkCurrentToken: %w", err)
	}

	p.nextToken() // current = column_name

LabelColumns:
	for {
		switch { //nolint:exhaustive
		case p.isCurrentToken(TOKEN_IDENT):
			column, constraints, err := p.parseColumn(createTableStmt.Name.Name)
			if err != nil {
				return nil, apperr.Errorf(errFmtPrefix+"parseColumn: %w", err)
			}
			createTableStmt.Columns = append(createTableStmt.Columns, column)
			if len(constraints) > 0 {
				for _, c := range constraints {
					createTableStmt.Constraints = createTableStmt.Constraints.Append(c)
				}
			}
		case isConstraint(p.currentToken.Type):
			constraint, err := p.parseTableConstraint(createTableStmt.Name.Name)
			if err != nil {
				return nil, apperr.Errorf(errFmtPrefix+"parseTableConstraint: %w", err)
			}
			createTableStmt.Constraints = createTableStmt.Constraints.Append(constraint)
		case p.isCurrentToken(TOKEN_COMMA):
			p.nextToken()
			continue
		case p.isCurrentToken(TOKEN_CLOSE_PAREN):
			p.nextToken()
			break LabelColumns
		default:
			return nil, apperr.Errorf(errFmtPrefix+"currentToken=%#v, peekToken=%#v: %w", p.currentToken, p.peekToken, ddl.ErrUnexpectedCurrentToken)
		}
	}

LabelTableOptions:
	for {
		switch p.currentToken.Type { //nolint:exhaustive
		case TOKEN_PRIMARY:
			opt := &Option{}
			p.nextToken() // current = KEY
			if err := p.checkCurrentToken(TOKEN_KEY); err != nil {
				return nil, apperr.Errorf(errFmtPrefix+"checkCurrentToken: %w", err)
			}
			opt.Name = "PRIMARY KEY"
			p.nextToken() // current = `(`
			idents, err := p.parseExpr()
			if err != nil {
				return nil, apperr.Errorf(errFmtPrefix+"parseExpr: %w", err)
			}
			opt.Value = opt.Value.Append(idents...)
			createTableStmt.Options = append(createTableStmt.Options, opt)
			continue
		case TOKEN_INTERLEAVE:
			opt := &Option{}
			p.nextToken() // current = IN
			if err := p.checkCurrentToken(TOKEN_IN); err != nil {
				return nil, apperr.Errorf(errFmtPrefix+"checkCurrentToken: %w", err)
			}
			p.nextToken() // current = PARENT
			if err := p.checkCurrentToken(TOKEN_PARENT); err != nil {
				return nil, apperr.Errorf(errFmtPrefix+"checkCurrentToken: %w", err)
			}
			opt.Name = "INTERLEAVE IN PARENT"
			p.nextToken() // current = table_name
			if err := p.checkCurrentToken(TOKEN_IDENT); err != nil {
				return nil, apperr.Errorf(errFmtPrefix+"checkCurrentToken: %w", err)
			}
			opt.Value = opt.Value.Append(NewRawIdent(p.currentToken.Literal.String()))
			if p.isPeekToken(TOKEN_ON) {
				p.nextToken() // current = ON
				p.nextToken() // current = DELETE
				if err := p.checkCurrentToken(TOKEN_DELETE); err != nil {
					return nil, apperr.Errorf(errFmtPrefix+"checkCurrentToken: %w", err)
				}
				onAction := "ON DELETE"
				p.nextToken()                // current = CASCADE or NO
				switch p.currentToken.Type { //nolint:exhaustive
				case TOKEN_CASCADE:
					onAction += " CASCADE"
				case TOKEN_NO:
					p.nextToken() // current = ACTION
					if err := p.checkCurrentToken(TOKEN_ACTION); err != nil {
						return nil, apperr.Errorf(errFmtPrefix+"checkCurrentToken: %w", err)
					}
					onAction += " NO ACTION"
				default:
					return nil, apperr.Errorf(errFmtPrefix+"currentToken=%#v, peekToken=%#v: %w", p.currentToken, p.peekToken, ddl.ErrUnexpectedCurrentToken)
				}
				opt.Value = opt.Value.Append(NewRawIdent(onAction))
			}
			createTableStmt.Options = append(createTableStmt.Options, opt)
		case TOKEN_ROW:
			// ROW DELETION POLICY (OLDER_THAN(ExpiredDate, INTERVAL 0 DAY))
			opt := &Option{}
			p.nextToken() // current = DELETION
			if err := p.checkCurrentToken(TOKEN_DELETION); err != nil {
				return nil, apperr.Errorf(errFmtPrefix+"checkCurrentToken: %w", err)
			}
			p.nextToken() // current = POLICY
			if err := p.checkCurrentToken(TOKEN_POLICY); err != nil {
				return nil, apperr.Errorf(errFmtPrefix+"checkCurrentToken: %w", err)
			}
			opt.Name = "ROW DELETION POLICY"
			rowDeletionPolicyContent := make([]*Ident, 0)
			//
			p.nextToken() // current = `(`
			if err := p.checkCurrentToken(TOKEN_OPEN_PAREN); err != nil {
				return nil, apperr.Errorf(errFmtPrefix+"checkCurrentToken: %w", err)
			}
			rowDeletionPolicyContent = append(rowDeletionPolicyContent, NewRawIdent(p.currentToken.Literal.String()))
			//
			p.nextToken() // current = OLDER_THAN
			if err := p.checkCurrentToken(TOKEN_OLDER_THAN); err != nil {
				return nil, apperr.Errorf(errFmtPrefix+"checkCurrentToken: %w", err)
			}
			rowDeletionPolicyContent = append(rowDeletionPolicyContent, NewRawIdent(p.currentToken.Literal.String()))
			//
			p.nextToken() // current = `(`
			if err := p.checkCurrentToken(TOKEN_OPEN_PAREN); err != nil {
				return nil, apperr.Errorf(errFmtPrefix+"checkCurrentToken: %w", err)
			}
			rowDeletionPolicyContent = append(rowDeletionPolicyContent, NewRawIdent(p.currentToken.Literal.String()))
			//
			p.nextToken() // current = column_name
			if err := p.checkCurrentToken(TOKEN_IDENT); err != nil {
				return nil, apperr.Errorf(errFmtPrefix+"checkCurrentToken: %w", err)
			}
			rowDeletionPolicyContent = append(rowDeletionPolicyContent, NewRawIdent(p.currentToken.Literal.String()))
			//
			p.nextToken() // current = COMMA
			if err := p.checkCurrentToken(TOKEN_COMMA); err != nil {
				return nil, apperr.Errorf(errFmtPrefix+"checkCurrentToken: %w", err)
			}
			rowDeletionPolicyContent = append(rowDeletionPolicyContent, NewRawIdent(p.currentToken.Literal.String()))
			//
			p.nextToken() // current = INTERVAL
			if err := p.checkCurrentToken(TOKEN_INTERVAL); err != nil {
				return nil, apperr.Errorf(errFmtPrefix+"checkCurrentToken: %w", err)
			}
			rowDeletionPolicyContent = append(rowDeletionPolicyContent, NewRawIdent(p.currentToken.Literal.String()))
			//
			p.nextToken() // current = 0
			if err := p.checkCurrentToken(TOKEN_IDENT); err != nil {
				return nil, apperr.Errorf(errFmtPrefix+"checkCurrentToken: %w", err)
			}
			rowDeletionPolicyContent = append(rowDeletionPolicyContent, NewRawIdent(p.currentToken.Literal.String()))
			//
			p.nextToken() // current = DAY
			if err := p.checkCurrentToken(TOKEN_DAY); err != nil {
				return nil, apperr.Errorf(errFmtPrefix+"checkCurrentToken: %w", err)
			}
			rowDeletionPolicyContent = append(rowDeletionPolicyContent, NewRawIdent(p.currentToken.Literal.String()))
			//
			p.nextToken() // current = `)`
			if err := p.checkCurrentToken(TOKEN_CLOSE_PAREN); err != nil {
				return nil, apperr.Errorf(errFmtPrefix+"checkCurrentToken: %w", err)
			}
			rowDeletionPolicyContent = append(rowDeletionPolicyContent, NewRawIdent(p.currentToken.Literal.String()))
			//
			p.nextToken() // current = `)`
			if err := p.checkCurrentToken(TOKEN_CLOSE_PAREN); err != nil {
				return nil, apperr.Errorf(errFmtPrefix+"checkCurrentToken: %w", err)
			}
			rowDeletionPolicyContent = append(rowDeletionPolicyContent, NewRawIdent(p.currentToken.Literal.String()))
			//
			opt.Value = opt.Value.Append(rowDeletionPolicyContent...)

			createTableStmt.RowDeletionPolicy = opt
		case TOKEN_COMMA:
			// do nothing
		case TOKEN_SEMICOLON, TOKEN_EOF:
			break LabelTableOptions
		default:
			return nil, apperr.Errorf(errFmtPrefix+"currentToken=%#v, peekToken=%#v: %w", p.currentToken, p.peekToken, ddl.ErrUnexpectedCurrentToken)
		}
		p.nextToken()
	}

	return createTableStmt, nil
}

//nolint:cyclop,funlen
func (p *Parser) parseCreateIndexStmt() (*CreateIndexStmt, error) {
	createIndexStmt := &CreateIndexStmt{}

	if p.isCurrentToken(TOKEN_UNIQUE) {
		createIndexStmt.Unique = true
		p.nextToken() // current = INDEX
	}

	if p.isPeekToken(TOKEN_IF) {
		p.nextToken() // current = IF
		if err := p.checkPeekToken(TOKEN_NOT); err != nil {
			return nil, apperr.Errorf("checkPeekToken: %w", err)
		}
		p.nextToken() // current = NOT
		if err := p.checkPeekToken(TOKEN_EXISTS); err != nil {
			return nil, apperr.Errorf("checkPeekToken: %w", err)
		}
		p.nextToken() // current = EXISTS
		createIndexStmt.IfNotExists = true
	}

	p.nextToken() // current = index_name
	if err := p.checkCurrentToken(TOKEN_IDENT); err != nil {
		return nil, apperr.Errorf("checkCurrentToken: %w", err)
	}

	createIndexStmt.Name = NewObjectName(p.currentToken.Literal.Str)
	errFmtPrefix := fmt.Sprintf("index_name=%s: ", createIndexStmt.Name.StringForDiff())

	p.nextToken() // current = ON

	if err := p.checkCurrentToken(TOKEN_ON); err != nil {
		return nil, apperr.Errorf(errFmtPrefix+"checkCurrentToken: %w", err)
	}

	p.nextToken() // current = table_name

	if err := p.checkCurrentToken(TOKEN_IDENT); err != nil {
		return nil, apperr.Errorf(errFmtPrefix+"checkCurrentToken: %w", err)
	}

	createIndexStmt.TableName = NewObjectName(p.currentToken.Literal.Str)

	p.nextToken() // current = USING or (

	if p.isCurrentToken(TOKEN_USING) {
		p.nextToken() // current = using_def
		createIndexStmt.Using = append(createIndexStmt.Using, NewIdent(p.currentToken.Literal.Str, "", p.currentToken.Literal.Str))
		p.nextToken() // current = (
	}

	if err := p.checkCurrentToken(TOKEN_OPEN_PAREN); err != nil {
		return nil, apperr.Errorf(errFmtPrefix+"checkCurrentToken: %w", err)
	}

	idents, err := p.parseColumnIdents()
	if err != nil {
		return nil, apperr.Errorf(errFmtPrefix+"parseColumnIdents: %w", err)
	}

	createIndexStmt.Columns = idents

	return createIndexStmt, nil
}

//nolint:funlen,cyclop
func (p *Parser) parseColumn(tableName *Ident) (*Column, []Constraint, error) {
	column := &Column{}
	constraints := make(Constraints, 0)

	if err := p.checkCurrentToken(TOKEN_IDENT); err != nil {
		return nil, nil, apperr.Errorf("checkCurrentToken: %w", err)
	}

	column.Name = NewRawIdent(p.currentToken.Literal.Str)
	errFmtPrefix := fmt.Sprintf("column_name=%s: ", column.Name.StringForDiff())

	p.nextToken() // current = DATA_TYPE

	switch { //nolint:exhaustive
	case isDataType(p.currentToken.Type):
		dataType, err := p.parseDataType()
		if err != nil {
			return nil, nil, apperr.Errorf(errFmtPrefix+"parseDataType: %w", err)
		}
		column.DataType = dataType

		p.nextToken() // current = DEFAULT or NOT or NULL or PRIMARY or UNIQUE or COMMA or ...
	LabelDefaultNotNull:
		for {
			switch p.currentToken.Type { //nolint:exhaustive
			case TOKEN_NOT:
				if err := p.checkPeekToken(TOKEN_NULL); err != nil {
					return nil, nil, apperr.Errorf(errFmtPrefix+"checkPeekToken: %w", err)
				}
				p.nextToken() // current = NULL
				column.NotNull = true
			case TOKEN_NULL:
				column.NotNull = false
			case TOKEN_DEFAULT: // current = DEFAULT
				p.nextToken() // current = value
				def, err := p.parseColumnDefault()
				if err != nil {
					return nil, nil, apperr.Errorf(errFmtPrefix+"parseColumnDefault: %w", err)
				}
				column.Default = def
				continue
			default:
				break LabelDefaultNotNull
			}

			p.nextToken()
		}

		cs, err := p.parseColumnConstraints(tableName, column)
		if err != nil {
			return nil, nil, apperr.Errorf(errFmtPrefix+"parseColumnConstraints: %w", err)
		}
		if len(cs) > 0 {
			for _, c := range cs {
				constraints = constraints.Append(c)
			}
		}

		if p.isCurrentToken(TOKEN_OPTIONS) {
			p.nextToken() // current = (
			idents, err := p.parseExpr()
			if err != nil {
				return nil, nil, apperr.Errorf(errFmtPrefix+"parseExpr: %w", err)
			}
			column.Options = column.Options.Append(idents...)
		}
	default:
		return nil, nil, apperr.Errorf(errFmtPrefix+"currentToken=%#v, peekToken=%#v: %w", p.currentToken, p.peekToken, ddl.ErrUnexpectedCurrentToken)
	}

	return column, constraints, nil
}

//nolint:cyclop
func (p *Parser) parseColumnDefault() (*Default, error) {
	def := &Default{}

LabelDefault:
	for {
		switch p.currentToken.Type { //nolint:exhaustive
		case TOKEN_IDENT:
			def.Value = def.Value.Append(NewRawIdent(p.currentToken.Literal.String()))
		case TOKEN_OPEN_PAREN:
			ids, err := p.parseExpr()
			if err != nil {
				return nil, apperr.Errorf("parseExpr: %w", err)
			}
			def.Value = def.Value.Append(ids...)
			continue
		case TOKEN_NOT, TOKEN_COMMA, TOKEN_CLOSE_PAREN:
			break LabelDefault
		default:
			if isReservedValue(p.currentToken.Type) {
				def.Value = def.Value.Append(NewIdent(string(p.currentToken.Type), "", p.currentToken.Literal.String()))
				p.nextToken()
				continue
			}
			// MEMO: backup
			// TODO: check if this is necessary
			// if isOperator(p.currentToken.Type) {
			// 	def.Value = def.Value.Append(NewRawIdent(p.currentToken.Literal.Str))
			// 	p.nextToken()
			// 	continue
			// }
			// if isDataType(p.currentToken.Type) {
			// 	def.Value.Idents = append(def.Value.Idents, NewRawIdent(p.currentToken.Literal.Str))
			// 	p.nextToken()
			// 	continue
			// }
			if isConstraint(p.currentToken.Type) {
				break LabelDefault
			}
			return nil, apperr.Errorf("currentToken=%#v, peekToken=%#v: %w", p.currentToken, p.peekToken, ddl.ErrUnexpectedCurrentToken)
		}

		p.nextToken()
	}

	return def, nil
}

//nolint:cyclop
func (p *Parser) parseExpr() ([]*Ident, error) {
	idents := make([]*Ident, 0)

	if err := p.checkCurrentToken(TOKEN_OPEN_PAREN); err != nil {
		return nil, apperr.Errorf("checkCurrentToken: %w", err)
	}
	idents = append(idents, NewRawIdent(p.currentToken.Literal.Str))
	p.nextToken() // current = IDENT

LabelExpr:
	for {
		switch p.currentToken.Type { //nolint:exhaustive
		case TOKEN_OPEN_PAREN:
			ids, err := p.parseExpr()
			if err != nil {
				return nil, apperr.Errorf("parseExpr: %w", err)
			}
			idents = append(idents, ids...)
			continue
		case TOKEN_CLOSE_PAREN:
			idents = append(idents, NewRawIdent(p.currentToken.Literal.Str))
			p.nextToken()
			break LabelExpr
		case TOKEN_EQUAL, TOKEN_GREATER, TOKEN_LESS:
			value := p.currentToken.Literal.Str
			switch p.peekToken.Type { //nolint:exhaustive
			case TOKEN_EQUAL, TOKEN_GREATER, TOKEN_LESS:
				value += p.peekToken.Literal.Str
				p.nextToken()
			}
			idents = append(idents, NewRawIdent(value))
		case TOKEN_EOF:
			return nil, apperr.Errorf("currentToken=%#v, peekToken=%#v: %w", p.currentToken, p.peekToken, ddl.ErrUnexpectedCurrentToken)
		default:
			if isReservedValue(p.currentToken.Type) {
				idents = append(idents, NewRawIdent(p.currentToken.Type.String()))
			} else {
				idents = append(idents, NewRawIdent(p.currentToken.Literal.Str))
			}
		}

		p.nextToken()
	}

	return idents, nil
}

//nolint:cyclop,funlen,gocognit
func (p *Parser) parseColumnConstraints(tableName *Ident, column *Column) ([]Constraint, error) {
	constraints := make(Constraints, 0)

LabelConstraints:
	for {
		switch p.currentToken.Type { //nolint:exhaustive
		case TOKEN_REFERENCES:
			if err := p.checkPeekToken(TOKEN_IDENT); err != nil {
				return nil, apperr.Errorf("checkPeekToken: %w", err)
			}
			p.nextToken() // current = table_name
			constraint := &ForeignKeyConstraint{
				Name:    NewRawIdent(fmt.Sprintf("%s_%s_fkey", tableName.StringForDiff(), column.Name.StringForDiff())),
				Ref:     NewRawIdent(p.currentToken.Literal.Str),
				Columns: []*ColumnIdent{{Ident: column.Name}},
			}
			p.nextToken() // current = (
			idents, err := p.parseColumnIdents()
			if err != nil {
				return nil, apperr.Errorf("parseColumnIdents: %w", err)
			}
			constraint.RefColumns = idents
			constraints = constraints.Append(constraint)
		case TOKEN_CHECK:
			if err := p.checkPeekToken(TOKEN_OPEN_PAREN); err != nil {
				return nil, apperr.Errorf("checkPeekToken: %w", err)
			}
			p.nextToken() // current = (
			constraint := &CheckConstraint{
				Name: NewRawIdent(fmt.Sprintf("%s_%s_check", tableName.StringForDiff(), column.Name.StringForDiff())),
			}
			idents, err := p.parseExpr()
			if err != nil {
				return nil, apperr.Errorf("parseExpr: %w", err)
			}
			constraint.Expr = constraint.Expr.Append(idents...)
			constraints = constraints.Append(constraint)
		case TOKEN_OPTIONS, TOKEN_IDENT, TOKEN_COMMA, TOKEN_CLOSE_PAREN:
			break LabelConstraints
		default:
			return nil, apperr.Errorf("currentToken=%#v, peekToken=%#v: %w", p.currentToken, p.peekToken, ddl.ErrUnexpectedCurrentToken)
		}

		p.nextToken()
	}

	return constraints, nil
}

//nolint:funlen,cyclop,gocognit
func (p *Parser) parseTableConstraint(tableName *Ident) (Constraint, error) { //nolint:ireturn
	var constraintName *Ident
	if p.isCurrentToken(TOKEN_CONSTRAINT) {
		p.nextToken() // current = constraint_name
		if p.currentToken.Type != TOKEN_IDENT {
			return nil, apperr.Errorf("currentToken=%#v, peekToken=%#v: %w", p.currentToken, p.peekToken, ddl.ErrUnexpectedCurrentToken)
		}
		constraintName = NewRawIdent(p.currentToken.Literal.Str)
		p.nextToken() // current = PRIMARY or CHECK //diff:ignore-line-postgres-cockroach
	}

	switch p.currentToken.Type { //nolint:exhaustive
	case TOKEN_FOREIGN:
		if err := p.checkPeekToken(TOKEN_KEY); err != nil {
			return nil, apperr.Errorf("checkPeekToken: %w", err)
		}
		p.nextToken() // current = KEY
		if err := p.checkPeekToken(TOKEN_OPEN_PAREN); err != nil {
			return nil, apperr.Errorf("checkPeekToken: %w", err)
		}
		p.nextToken() // current = (
		idents, err := p.parseColumnIdents()
		if err != nil {
			return nil, apperr.Errorf("parseColumnIdents: %w", err)
		}
		if err := p.checkCurrentToken(TOKEN_REFERENCES); err != nil {
			return nil, apperr.Errorf("checkPeekToken: %w", err)
		}
		p.nextToken() // current = ref_table_name
		if err := p.checkCurrentToken(TOKEN_IDENT); err != nil {
			return nil, apperr.Errorf("checkCurrentToken: %w", err)
		}
		refName := NewRawIdent(p.currentToken.Literal.Str)

		p.nextToken() // current = (
		identsRef, err := p.parseColumnIdents()
		if err != nil {
			return nil, apperr.Errorf("parseColumnIdents: %w", err)
		}
		if constraintName == nil {
			name := tableName.StringForDiff()
			for _, ident := range idents {
				name += "_" + ident.StringForDiff()
			}
			name += "_fkey"
			constraintName = NewRawIdent(name)
		}
		return &ForeignKeyConstraint{
			Name:       constraintName,
			Columns:    idents,
			Ref:        refName,
			RefColumns: identsRef,
		}, nil
	case TOKEN_CHECK:
		constraint := &CheckConstraint{
			Name: constraintName,
		}
		if err := p.checkPeekToken(TOKEN_OPEN_PAREN); err != nil {
			return nil, apperr.Errorf("checkPeekToken: %w", err)
		}
		p.nextToken() // current = (
		idents, err := p.parseExpr()
		if err != nil {
			return nil, apperr.Errorf("parseExpr: %w", err)
		}
		constraint.Name = constraintName
		constraint.Expr = constraint.Expr.Append(idents...)
		return constraint, nil
	default:
		return nil, apperr.Errorf("currentToken=%#v, peekToken=%#v: %w", p.currentToken, p.peekToken, ddl.ErrUnexpectedCurrentToken)
	}
}

//nolint:cyclop,funlen
func (p *Parser) parseDataType() (*DataType, error) {
	dataType := &DataType{
		Name: p.currentToken.Literal.String(),
		Type: p.currentToken.Type,
	}

	// TODO: support ARRAY, STRUCT

	if p.isPeekToken(TOKEN_OPEN_PAREN) {
		p.nextToken() // current = (
		idents, err := p.parseIdents()
		if err != nil {
			return nil, apperr.Errorf("parseIdents: %w", err)
		}
		dataType.Expr = dataType.Expr.Append(idents...)
	}

	return dataType, nil
}

func (p *Parser) parseColumnIdents() ([]*ColumnIdent, error) {
	idents := make([]*ColumnIdent, 0)

LabelIdents:
	for {
		switch p.currentToken.Type { //nolint:exhaustive
		case TOKEN_OPEN_PAREN:
			// do nothing
		case TOKEN_IDENT:
			ident := &ColumnIdent{Ident: NewRawIdent(p.currentToken.Literal.Str)}
			switch p.peekToken.Type { //nolint:exhaustive //diff:ignore-line-postgres-cockroach
			case TOKEN_ASC: //diff:ignore-line-postgres-cockroach
				ident.Order = &Order{Desc: false} //diff:ignore-line-postgres-cockroach
				p.nextToken()                     // current = ASC //diff:ignore-line-postgres-cockroach
			case TOKEN_DESC: //diff:ignore-line-postgres-cockroach
				ident.Order = &Order{Desc: true} //diff:ignore-line-postgres-cockroach
				p.nextToken()                    // current = DESC //diff:ignore-line-postgres-cockroach
			} //diff:ignore-line-postgres-cockroach
			idents = append(idents, ident)
		case TOKEN_COMMA:
			// do nothing
		case TOKEN_CLOSE_PAREN:
			p.nextToken()
			break LabelIdents
		default:
			return nil, apperr.Errorf("currentToken=%#v, peekToken=%#v: %w", p.currentToken, p.peekToken, ddl.ErrUnexpectedCurrentToken)
		}
		p.nextToken()
	}

	return idents, nil
}

func (p *Parser) parseIdents() ([]*Ident, error) {
	idents := make([]*Ident, 0)

LabelIdents:
	for {
		switch p.currentToken.Type { //nolint:exhaustive
		case TOKEN_OPEN_PAREN:
			// do nothing
		case TOKEN_CLOSE_PAREN:
			break LabelIdents
		case TOKEN_EOF, TOKEN_ILLEGAL:
			return nil, apperr.Errorf("currentToken=%#v, peekToken=%#v: %w", p.currentToken, p.peekToken, ddl.ErrUnexpectedCurrentToken)
		default:
			idents = append(idents, NewRawIdent(p.currentToken.Literal.String()))
		}
		p.nextToken()
	}

	return idents, nil
}

// MEMO: backup
// TODO: check if this is necessary
// func isOperator(tokenType TokenType) bool {
// 	switch tokenType { //nolint:exhaustive
// 	case TOKEN_EQUAL, TOKEN_GREATER, TOKEN_LESS,
// 		TOKEN_PLUS, TOKEN_MINUS, TOKEN_ASTERISK, TOKEN_SLASH,
// 		TOKEN_TYPE_ANNOTATION, //diff:ignore-line-postgres-cockroach
// 		TOKEN_STRING_CONCAT, TOKEN_TYPECAST:
// 		return true
// 	default:
// 		return false
// 	}
// }

func isReservedValue(tokenType TokenType) bool {
	switch tokenType { //nolint:exhaustive
	case TOKEN_NULL, TOKEN_TRUE, TOKEN_FALSE:
		return true
	default:
		return false
	}
}

func isDataType(tokenType TokenType) bool {
	switch tokenType { //nolint:exhaustive
	case TOKEN_BOOL,
		TOKEN_INT64,
		TOKEN_NUMERIC,
		TOKEN_FLOAT64,
		TOKEN_JSON,
		TOKEN_STRING,
		TOKEN_TIMESTAMP:
		return true
	default:
		return false
	}
}

func isConstraint(tokenType TokenType) bool {
	switch tokenType { //nolint:exhaustive
	case TOKEN_CONSTRAINT,
		TOKEN_INDEX,
		TOKEN_PRIMARY, TOKEN_KEY,
		TOKEN_FOREIGN, TOKEN_REFERENCES,
		TOKEN_UNIQUE,
		TOKEN_CHECK:
		return true
	default:
		return false
	}
}

func (p *Parser) isCurrentToken(expectedTypes ...TokenType) bool {
	for _, expected := range expectedTypes {
		if expected == p.currentToken.Type {
			return true
		}
	}
	return false
}

func (p *Parser) checkCurrentToken(expectedTypes ...TokenType) error {
	for _, expected := range expectedTypes {
		if expected == p.currentToken.Type {
			return nil
		}
	}
	return apperr.Errorf("currentToken=%#v, peekToken=%#v: expected=%v, but got=%v: %w", p.currentToken, p.peekToken, stringz.JoinStringers(",", expectedTypes...), p.currentToken.Type, ddl.ErrUnexpectedCurrentToken)
}

func (p *Parser) isPeekToken(expectedTypes ...TokenType) bool {
	for _, expected := range expectedTypes {
		if expected == p.peekToken.Type {
			return true
		}
	}
	return false
}

func (p *Parser) checkPeekToken(expectedTypes ...TokenType) error {
	for _, expected := range expectedTypes {
		if expected == p.peekToken.Type {
			return nil
		}
	}
	return apperr.Errorf("currentToken=%#v, peekToken=%#v: expected=%v, but got=%v: %w", p.currentToken, p.peekToken, stringz.JoinStringers(",", expectedTypes...), p.peekToken.Type, ddl.ErrUnexpectedPeekToken)
}

package postgres

import (
	"github.com/hakadoriya/z.go/stringz"

	"github.com/kunitsucom/ddlctl/pkg/ddl/internal"
)

const (
	Dialect       = "postgres" //diff:ignore-line-postgres-cockroach
	DriverName    = "postgres" //diff:ignore-line-postgres-cockroach
	Indent        = "    "
	CommentPrefix = "-- "
)

type Verb string

const (
	VerbCreate   Verb = "CREATE"
	VerbAlter    Verb = "ALTER"
	VerbDrop     Verb = "DROP"
	VerbRename   Verb = "RENAME"
	VerbTruncate Verb = "TRUNCATE"
)

type Object string

const (
	ObjectTable Object = "TABLE"
	ObjectIndex Object = "INDEX"
	ObjectView  Object = "VIEW"
)

type Action string

const (
	ActionAdd    Action = "ADD"
	ActionDrop   Action = "DROP"
	ActionAlter  Action = "ALTER"
	ActionRename Action = "RENAME"
)

type Stmt interface {
	isStmt()
	GetNameForDiff() string
	String() string
}

type DDL struct {
	Stmts []Stmt
}

func (d *DDL) String() string {
	if d == nil {
		return ""
	}
	return stringz.JoinStringers("", d.Stmts...)
}

type Ident struct {
	Name          string
	QuotationMark string
	Raw           string
}

func (i *Ident) GoString() string { return internal.GoString(*i) }

func (i *Ident) String() string {
	if i == nil {
		return ""
	}
	return i.Raw
}

func (i *Ident) StringForDiff() string {
	if i == nil {
		return ""
	}
	return i.Name
}

type ColumnIdent struct {
	Ident *Ident
}

func (i *ColumnIdent) GoString() string { return internal.GoString(*i) }

func (i *ColumnIdent) String() string {
	str := i.Ident.String()
	return str
}

func (i *ColumnIdent) StringForDiff() string {
	str := i.Ident.StringForDiff()
	return str
}

type DataType struct {
	Name string
	Type TokenType
	Expr *Expr
}

func (s *DataType) String() string {
	if s == nil {
		return ""
	}
	str := s.Name
	if s.Expr != nil && len(s.Expr.Idents) > 0 {
		str += "(" + s.Expr.String() + ")"
	}
	return str
}

func (s *DataType) StringForDiff() string {
	if s == nil {
		return ""
	}
	var str string
	if s.Type != "" {
		str += string(s.Type)
	} else {
		str += string(TOKEN_ILLEGAL)
	}

	if s.Expr != nil && len(s.Expr.Idents) > 0 {
		str += "("
		for _, ident := range s.Expr.Idents {
			str += ident.StringForDiff()
		}
		str += ")"
	}

	return str
}

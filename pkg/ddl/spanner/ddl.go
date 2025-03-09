package spanner

import (
	"github.com/hakadoriya/z.go/stringz"

	"github.com/kunitsucom/ddlctl/pkg/ddl/internal"
)

const (
	Dialect       = "spanner"
	DriverName    = "spanner"
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
	Order *Order //diff:ignore-line-postgres-cockroach
}

type Order struct{ Desc bool } //diff:ignore-line-postgres-cockroach

func (i *ColumnIdent) GoString() string { return internal.GoString(*i) }

func (i *ColumnIdent) String() string {
	str := i.Ident.String()
	if i.Order != nil { //diff:ignore-line-postgres-cockroach
		if i.Order.Desc { //diff:ignore-line-postgres-cockroach
			str += " DESC" //diff:ignore-line-postgres-cockroach
		} else { //diff:ignore-line-postgres-cockroach
			str += " ASC" //diff:ignore-line-postgres-cockroach
		} //diff:ignore-line-postgres-cockroach
	} //diff:ignore-line-postgres-cockroach
	return str
}

func (i *ColumnIdent) StringForDiff() string {
	str := i.Ident.StringForDiff()
	if i.Order != nil && i.Order.Desc { //diff:ignore-line-postgres-cockroach
		str += " DESC" //diff:ignore-line-postgres-cockroach
	} else { //diff:ignore-line-postgres-cockroach
		str += " ASC" //diff:ignore-line-postgres-cockroach
	} //diff:ignore-line-postgres-cockroach
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

package spanner

import (
	"strings"

	"github.com/hakadoriya/ddlctl/pkg/ddl/internal"
)

// MEMO: https://cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#alter_table

var _ Stmt = (*AlterTableStmt)(nil)

type AlterTableStmt struct {
	Comment string
	Indent  string
	Name    *ObjectName
	Action  AlterTableAction
}

func (*AlterTableStmt) isStmt() {}

func (s *AlterTableStmt) GetNameForDiff() string {
	return s.Name.StringForDiff()
}

//nolint:cyclop,funlen
func (s *AlterTableStmt) String() string {
	var str strings.Builder
	if s.Comment != "" {
		comments := strings.Split(s.Comment, "\n")
		for i := range comments {
			if comments[i] != "" {
				str.WriteString(CommentPrefix + comments[i] + "\n")
			}
		}
	}
	str.WriteString("ALTER TABLE ")
	str.WriteString(s.Name.String() + " ")
	switch a := s.Action.(type) {
	case *RenameTable:
		str.WriteString("RENAME TO ")
		str.WriteString(a.NewName.String())
	case *RenameColumn:
		str.WriteString("RENAME COLUMN " + a.Name.String() + " TO " + a.NewName.String())
	case *RenameConstraint:
		str.WriteString("RENAME CONSTRAINT " + a.Name.String() + " TO " + a.NewName.String())
	case *AddColumn:
		str.WriteString("ADD COLUMN " + a.Column.String())
	case *DropColumn:
		str.WriteString("DROP COLUMN " + a.Name.String())
	case *AlterColumnDataType:
		str.WriteString("ALTER COLUMN " + a.Name.String() + " " + a.DataType.String())
		if a.NotNull {
			str.WriteString(" NOT NULL")
		}
	case *AlterColumnSetDefault:
		str.WriteString("ALTER COLUMN " + a.Name.String() + " SET " + a.Default.String())
	case *AlterColumnDropDefault:
		str.WriteString("ALTER COLUMN " + a.Name.String() + " DROP DEFAULT")
	case *AlterColumnSetOptions:
		str.WriteString("ALTER COLUMN " + a.Name.String() + " SET OPTIONS " + a.Options.String())
	case *AlterColumnDropOptions:
		// NOTE: Cloud Spanner has no DROP OPTIONS syntax. Reset each option to its default
		//       by setting it to NULL instead.
		str.WriteString("ALTER COLUMN " + a.Name.String() + " SET OPTIONS " + resetOptionsExpr(a.Options).String())
	case *AddConstraint:
		str.WriteString("ADD " + a.Constraint.String())
		if a.NotValid {
			str.WriteString(" NOT VALID")
		}
	case *DropConstraint:
		str.WriteString("DROP CONSTRAINT " + a.Name.String())
	case *AlterConstraint:
		str.WriteString("ALTER CONSTRAINT " + a.Name.String() + " ")
		if a.Deferrable {
			str.WriteString("DEFERRABLE")
		} else {
			str.WriteString("NOT DEFERRABLE")
		}
		if a.InitiallyDeferred {
			str.WriteString(" INITIALLY DEFERRED")
		} else {
			str.WriteString(" INITIALLY IMMEDIATE")
		}
	case *AddRowDeletionPolicy:
		str.WriteString("ADD " + a.RowDeletionPolicy.String())
	case *ReplaceRowDeletionPolicy:
		str.WriteString("REPLACE " + a.RowDeletionPolicy.String())
	case *DropRowDeletionPolicy:
		str.WriteString("DROP ROW DELETION POLICY")
	}

	return str.String() + ";\n"
}

func (s *AlterTableStmt) GoString() string { return internal.GoString(*s) }

type AlterTableAction interface {
	isAlterTableAction()
	GoString() string
}

// RenameTable represents ALTER TABLE table_name RENAME TO new_table_name.
type RenameTable struct {
	NewName *ObjectName
}

func (*RenameTable) isAlterTableAction() {}

func (s *RenameTable) GoString() string { return internal.GoString(*s) }

// RenameConstraint represents ALTER TABLE table_name RENAME COLUMN.
type RenameConstraint struct {
	Name    *Ident
	NewName *Ident
}

func (*RenameConstraint) isAlterTableAction() {}

func (s *RenameConstraint) GoString() string { return internal.GoString(*s) }

// RenameColumn represents ALTER TABLE table_name RENAME COLUMN.
type RenameColumn struct {
	Name    *Ident
	NewName *Ident
}

func (*RenameColumn) isAlterTableAction() {}

func (s *RenameColumn) GoString() string { return internal.GoString(*s) }

// AddColumn represents ALTER TABLE table_name ADD COLUMN.
type AddColumn struct {
	Column *Column
}

func (*AddColumn) isAlterTableAction() {}

func (s *AddColumn) GoString() string { return internal.GoString(*s) }

// DropColumn represents ALTER TABLE table_name DROP COLUMN.
type DropColumn struct {
	Name *Ident
}

func (*DropColumn) isAlterTableAction() {}

func (s *DropColumn) GoString() string { return internal.GoString(*s) }

// AlterColumnDataType represents ALTER TABLE table_name ALTER COLUMN column_name data_type NOT NULL.
type AlterColumnDataType struct {
	Name     *Ident
	DataType *DataType
	NotNull  bool
}

func (*AlterColumnDataType) isAlterTableAction() {}

func (s *AlterColumnDataType) GoString() string { return internal.GoString(*s) }

// AlterColumnSetDefault represents ALTER TABLE table_name ALTER COLUMN column_name SET DEFAULT.
type AlterColumnSetDefault struct {
	Name    *Ident
	Default *Default
}

func (*AlterColumnSetDefault) isAlterTableAction() {}

func (s *AlterColumnSetDefault) GoString() string { return internal.GoString(*s) }

// AlterColumnDropDefault represents ALTER TABLE table_name ALTER COLUMN column_name DROP DEFAULT.
type AlterColumnDropDefault struct {
	Name *Ident
}

func (*AlterColumnDropDefault) isAlterTableAction() {}

func (s *AlterColumnDropDefault) GoString() string { return internal.GoString(*s) }

// AlterColumnSetOptions represents ALTER TABLE table_name ALTER COLUMN column_name SET OPTIONS.
type AlterColumnSetOptions struct {
	Name    *Ident
	Options *Expr
}

func (*AlterColumnSetOptions) isAlterTableAction() {}

func (s *AlterColumnSetOptions) GoString() string { return internal.GoString(*s) }

// AlterColumnDropOptions represents removing every column option.
//
// Cloud Spanner has no DROP OPTIONS syntax, so this is rendered as
// ALTER TABLE table_name ALTER COLUMN column_name SET OPTIONS (option_name = NULL, ...),
// which resets each option to its default.
//
// ref. https://cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#alter_table
type AlterColumnDropOptions struct {
	Name *Ident

	// Options holds the options the column has before the change. Only the option names are
	// used; each of them is set to NULL to reset it.
	Options *Expr
}

func (*AlterColumnDropOptions) isAlterTableAction() {}

func (s *AlterColumnDropOptions) GoString() string { return internal.GoString(*s) }

// resetOptionsExpr builds an OPTIONS expression that resets every option in options to its default.
//
// Cloud Spanner clears a column option by setting it to NULL, so each option name is paired with
// NULL. A nil or empty options yields "()", which Cloud Spanner accepts as a no-op.
func resetOptionsExpr(options *Expr) *Expr {
	names := optionNames(options)

	idents := make([]*Ident, 0, len(names)*4+2) //nolint:mnd // "name = NULL" plus a separator per option, and the surrounding parentheses
	idents = append(idents, &Ident{Name: "(", Raw: "("})
	for i, name := range names {
		if i > 0 {
			idents = append(idents, &Ident{Name: ",", Raw: ","})
		}
		idents = append(idents,
			&Ident{Name: name, Raw: name},
			&Ident{Name: "=", Raw: "="},
			&Ident{Name: "NULL", Raw: "NULL"},
		)
	}
	idents = append(idents, &Ident{Name: ")", Raw: ")"})

	return &Expr{Idents: idents}
}

// optionNames extracts the option names from an OPTIONS expression.
//
// The expression is a token sequence such as `( name = value , name2 = value2 )`, so the identifier
// that follows "(" or "," is an option name.
func optionNames(options *Expr) []string {
	if options == nil {
		return nil
	}

	names := make([]string, 0, len(options.Idents))
	expectName := false
	for _, ident := range options.Idents {
		switch s := ident.String(); s {
		case "(", ",":
			expectName = true
		case ")":
			expectName = false
		default:
			if expectName {
				names = append(names, s)
				expectName = false
			}
		}
	}

	return names
}

// AddConstraint represents ALTER TABLE table_name ADD CONSTRAINT.
type AddConstraint struct {
	Constraint Constraint
	NotValid   bool
}

func (*AddConstraint) isAlterTableAction() {}

func (s *AddConstraint) GoString() string { return internal.GoString(*s) }

// DropConstraint represents ALTER TABLE table_name DROP CONSTRAINT.
type DropConstraint struct {
	Name *Ident
}

func (*DropConstraint) isAlterTableAction() {}

func (s *DropConstraint) GoString() string { return internal.GoString(*s) }

// AlterConstraint represents ALTER TABLE table_name ALTER CONSTRAINT.
type AlterConstraint struct {
	Name              *Ident
	Deferrable        bool
	InitiallyDeferred bool
}

func (*AlterConstraint) isAlterTableAction() {}

func (s *AlterConstraint) GoString() string { return internal.GoString(*s) }

// AddRowDeletionPolicy represents ALTER TABLE table_name ADD ROW DELETION POLICY.
type AddRowDeletionPolicy struct {
	RowDeletionPolicy *Option
}

func (*AddRowDeletionPolicy) isAlterTableAction() {}

func (s *AddRowDeletionPolicy) GoString() string { return internal.GoString(*s) }

// ReplaceRowDeletionPolicy represents ALTER TABLE table_name REPLACE ROW DELETION POLICY.
type ReplaceRowDeletionPolicy struct {
	RowDeletionPolicy *Option
}

func (*ReplaceRowDeletionPolicy) isAlterTableAction() {}

func (s *ReplaceRowDeletionPolicy) GoString() string { return internal.GoString(*s) }

// DropRowDeletionPolicy represents ALTER TABLE table_name DROP ROW DELETION POLICY.
type DropRowDeletionPolicy struct{}

func (*DropRowDeletionPolicy) isAlterTableAction() {}

func (s *DropRowDeletionPolicy) GoString() string { return internal.GoString(*s) }

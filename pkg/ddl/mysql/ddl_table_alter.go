package mysql

import (
	"strings"

	"github.com/hakadoriya/ddlctl/pkg/ddl/internal"
)

// MEMO: https://dev.mysql.com/doc/refman/8.0/ja/alter-table.html
// NOTE: https://dev.mysql.com/doc/refman/8.0/ja/alter-table-examples.html

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

//nolint:cyclop,funlen,gocognit
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
	case *ModifyColumn:
		str.WriteString("MODIFY " + a.Name.String() + " " + a.DataType.String())
		if a.CharacterSet != nil {
			str.WriteString(" CHARACTER SET " + a.CharacterSet.String())
		}
		if a.Collate != nil {
			str.WriteString(" COLLATE " + a.Collate.String())
		}
		if a.NotNull {
			str.WriteString(" NOT NULL")
		} else {
			str.WriteString(" NULL")
		}
		if a.AutoIncrement {
			str.WriteString(" AUTO_INCREMENT")
		}
		if a.Default != nil {
			str.WriteString(" " + a.Default.String())
		}
		if a.OnAction != "" {
			str.WriteString(" " + a.OnAction)
		}
		if a.Comment != "" {
			str.WriteString(" COMMENT " + a.Comment)
		}
	case *AlterColumnDropDefault:
		str.WriteString("ALTER " + a.Name.String() + " " + "DROP DEFAULT")
	case *AddConstraint:
		str.WriteString("ADD " + a.Constraint.String())
		if a.NotValid {
			str.WriteString(" NOT VALID")
		}
	case *DropConstraint:
		str.WriteString("DROP ")
		if a.Name.String() == "PRIMARY KEY" {
			str.WriteString("PRIMARY KEY")
		} else {
			str.WriteString("CONSTRAINT " + a.Name.String())
		}
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
	case *AlterTableOption:
		str.WriteString(a.String())
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

// ModifyColumn represents ALTER TABLE table_name MODIFY column_name data_type NOT NULL.
// NOTE: https://dev.mysql.com/doc/refman/8.0/ja/alter-table-examples.html
type ModifyColumn struct {
	Name          *Ident
	DataType      *DataType
	CharacterSet  *Ident
	Collate       *Ident
	NotNull       bool
	AutoIncrement bool
	Default       *Default
	OnAction      string
	Comment       string
}

func (*ModifyColumn) isAlterTableAction() {}

func (s *ModifyColumn) GoString() string { return internal.GoString(*s) }

// AlterColumnDropDefault represents ALTER TABLE table_name ALTER COLUMN column_name DROP DEFAULT.
type AlterColumnDropDefault struct {
	Name *Ident
}

func (*AlterColumnDropDefault) isAlterTableAction() {}

func (s *AlterColumnDropDefault) GoString() string { return internal.GoString(*s) }

// AddConstraint represents ALTER TABLE table_name ADD CONSTRAINT.
type AddConstraint struct {
	Name       *Ident
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

// AlterTableOption represents ALTER TABLE table_name TABLE_OPTION=option_value.
type AlterTableOption struct {
	Name  string
	Value *Expr
}

func (*AlterTableOption) isAlterTableAction() {}

func (s *AlterTableOption) GoString() string { return internal.GoString(*s) }

func (s *AlterTableOption) String() string {
	return s.Name + "=" + s.Value.String()
}

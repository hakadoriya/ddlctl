package mysql

import (
	"reflect"
	"strings"

	simplediff "github.com/hakadoriya/z.go/diffz/simplediffz"

	"github.com/hakadoriya/ddlctl/pkg/apperr"

	"github.com/hakadoriya/ddlctl/pkg/ddl"
)

type DiffCreateTableConfig struct {
	UseAlterTableAddConstraintNotValid bool
}

type DiffCreateTableOption interface {
	apply(c *DiffCreateTableConfig)
}

func DiffCreateTableUseAlterTableAddConstraintNotValid(notValid bool) DiffCreateTableOption { //nolint:ireturn
	return &diffCreateTableConfigUseConstraintNotValid{
		useAlterTableAddConstraintNotValid: notValid,
	}
}

type diffCreateTableConfigUseConstraintNotValid struct {
	useAlterTableAddConstraintNotValid bool
}

func (o *diffCreateTableConfigUseConstraintNotValid) apply(c *DiffCreateTableConfig) {
	c.UseAlterTableAddConstraintNotValid = o.useAlterTableAddConstraintNotValid
}

//nolint:funlen,cyclop,gocognit
func DiffCreateTable(before, after *CreateTableStmt, opts ...DiffCreateTableOption) (*DDL, error) {
	config := &DiffCreateTableConfig{}

	for _, opt := range opts {
		opt.apply(config)
	}

	result := &DDL{}

	switch {
	case before == nil && after != nil:
		// CREATE TABLE table_name
		result.Stmts = append(result.Stmts, after)
		return result, nil
	case before != nil && after == nil:
		// DROP TABLE table_name;
		result.Stmts = append(result.Stmts, &DropTableStmt{
			Name: before.Name,
		})
		return result, nil
	case (before == nil && after == nil) || reflect.DeepEqual(before, after) || before.String() == after.String():
		return nil, apperr.Errorf("before: %s, after: %s: %w", before.GetNameForDiff(), after.GetNameForDiff(), ddl.ErrNoDifference)
	}

	if before.Name.StringForDiff() != after.Name.StringForDiff() {
		// ALTER TABLE table_name RENAME TO new_table_name;
		rename := &RenameTable{
			NewName: after.Name,
		}
		if rename.NewName.Schema == nil {
			rename.NewName.Schema = before.Name.Schema
		}
		result.Stmts = append(result.Stmts, &AlterTableStmt{
			Comment: simplediff.Diff(before.Name.StringForDiff(), after.Name.StringForDiff()).String(),
			Name:    before.Name,
			Action:  rename,
		})
	}

	for _, beforeConstraint := range before.Constraints {
		afterConstraint := findConstraintByName(beforeConstraint.GetName().Name, after.Constraints)
		if afterConstraint == nil {
			switch bc := beforeConstraint.(type) {
			case *IndexConstraint:
				// DROP INDEX index_name;
				result.Stmts = append(result.Stmts, &DropIndexStmt{
					Comment: simplediff.Diff(bc.StringForDiff(), "").String(),
					Name: &ObjectName{
						Schema: before.Name.Schema,
						Name:   bc.GetName(),
					},
				})
			default:
				// ALTER TABLE table_name DROP CONSTRAINT constraint_name;
				result.Stmts = append(result.Stmts, &AlterTableStmt{
					Comment: simplediff.Diff(beforeConstraint.String(), "").String(),
					Name:    after.Name, // ALTER TABLE RENAME TO で変更された後の可能性があるため after.Name を使用する
					Action: &DropConstraint{
						Name: beforeConstraint.GetName(),
					},
				})
			}
			continue
		}
	}

	config.diffCreateTableColumn(result, before, after)

	for _, beforeConstraint := range before.Constraints {
		afterConstraint := findConstraintByName(beforeConstraint.GetName().Name, after.Constraints)
		if afterConstraint != nil {
			if beforeConstraint.StringForDiff() != afterConstraint.StringForDiff() {
				switch ac := afterConstraint.(type) {
				case *IndexConstraint:
					// DROP INDEX index_name;
					// CREATE INDEX index_name ON table_name (column_name);
					result.Stmts = append(
						result.Stmts,
						&DropIndexStmt{
							Name: &ObjectName{
								Schema: before.Name.Schema,
								Name:   beforeConstraint.GetName(),
							},
						},
						&CreateIndexStmt{
							Unique: ac.Unique,
							Name: &ObjectName{
								Schema: after.Name.Schema,
								Name:   ac.GetName(),
							},
							TableName: after.Name,
							Columns:   ac.Columns,
						},
					)
				default:
					// ALTER TABLE table_name DROP CONSTRAINT constraint_name;
					// ALTER TABLE table_name ADD CONSTRAINT constraint_name constraint;
					result.Stmts = append(
						result.Stmts,
						&AlterTableStmt{
							Comment: simplediff.Diff(beforeConstraint.String(), "").String(),
							Name:    after.Name, // ALTER TABLE RENAME TO で変更された後の可能性があるため after.Name を使用する
							Action: &DropConstraint{
								Name: beforeConstraint.GetName(),
							},
						},
						&AlterTableStmt{
							Comment: simplediff.Diff("", afterConstraint.String()).String(),
							Name:    after.Name,
							Action: &AddConstraint{
								Constraint: afterConstraint,
								NotValid:   config.UseAlterTableAddConstraintNotValid,
							},
						},
					)
				}
			}
			continue
		}
	}

	for _, afterConstraint := range onlyLeftConstraint(after.Constraints, before.Constraints) {
		switch ac := afterConstraint.(type) {
		case *IndexConstraint:
			// CREATE INDEX index_name ON table_name (column_name);
			result.Stmts = append(result.Stmts, &CreateIndexStmt{
				Comment: simplediff.Diff("", ac.StringForDiff()).String(),
				Unique:  ac.Unique,
				Name: &ObjectName{
					Schema: after.Name.Schema,
					Name:   ac.GetName(),
				},
				TableName: after.Name,
				Columns:   ac.Columns,
			})
		default:
			// ALTER TABLE table_name ADD CONSTRAINT constraint_name constraint;
			result.Stmts = append(result.Stmts, &AlterTableStmt{
				Comment: simplediff.Diff("", afterConstraint.String()).String(),
				Name:    after.Name,
				Action: &AddConstraint{
					Constraint: afterConstraint,
					NotValid:   config.UseAlterTableAddConstraintNotValid,
				},
			})
		}
	}

	// TODO: OPTION cannot be deleted?

	for _, beforeOption := range before.Options {
		if strings.ToUpper(beforeOption.Name) == "AUTO_INCREMENT" {
			// skip AUTO_INCREMENT
			continue
		}
		afterOption := findOptionByName(beforeOption.Name, after.Options)
		if afterOption != nil {
			if beforeOption.StringForDiff() != afterOption.StringForDiff() {
				// ALTER TABLE table_name option_name=option_value;
				result.Stmts = append(result.Stmts, &AlterTableStmt{
					Comment: simplediff.Diff(beforeOption.String(), afterOption.String()).String(),
					Name:    after.Name,
					Action: &AlterTableOption{
						Name:  afterOption.Name,
						Value: afterOption.Value,
					},
				})
			}
		}
	}

	for _, afterOption := range onlyLeftOption(after.Options, before.Options) {
		if strings.ToUpper(afterOption.Name) == "AUTO_INCREMENT" {
			// skip AUTO_INCREMENT
			continue
		}
		// ALTER TABLE table_name option_name=option_value;
		result.Stmts = append(result.Stmts, &AlterTableStmt{
			Comment: simplediff.Diff("", afterOption.String()).String(),
			Name:    after.Name,
			Action: &AlterTableOption{
				Name:  afterOption.Name,
				Value: afterOption.Value,
			},
		})
	}

	if len(result.Stmts) == 0 {
		return nil, apperr.Errorf("before: %s, after: %s: %w", before.GetNameForDiff(), after.GetNameForDiff(), ddl.ErrNoDifference)
	}

	return result, nil
}

//nolint:funlen,cyclop
func (config *DiffCreateTableConfig) diffCreateTableColumn(ddls *DDL, before, after *CreateTableStmt) {
	for _, beforeColumn := range before.Columns {
		afterColumn := findColumnByName(beforeColumn.Name.Name, after.Columns)
		if afterColumn == nil {
			// ALTER TABLE table_name DROP COLUMN column_name;
			ddls.Stmts = append(ddls.Stmts, &AlterTableStmt{
				Comment: simplediff.Diff(beforeColumn.String(), "").String(),
				Name:    after.Name, // ALTER TABLE RENAME TO で変更された後の可能性があるため after.Name を使用する
				Action: &DropColumn{
					Name: beforeColumn.Name,
				},
			})
			continue
		}

		dropDefault := beforeColumn.Default != nil && afterColumn.Default == nil

		if beforeColumn.DataType.StringForDiff() != afterColumn.DataType.StringForDiff() ||
			beforeColumn.CharacterSet.StringForDiff() != afterColumn.CharacterSet.StringForDiff() ||
			beforeColumn.Collate.StringForDiff() != afterColumn.Collate.StringForDiff() ||
			beforeColumn.NotNull != afterColumn.NotNull ||
			(!dropDefault && beforeColumn.Default.StringForDiff() != afterColumn.Default.StringForDiff()) ||
			beforeColumn.AutoIncrement != afterColumn.AutoIncrement ||
			beforeColumn.OnAction != afterColumn.OnAction ||
			beforeColumn.Comment != afterColumn.Comment {
			// ALTER TABLE table_name MODIFY column_name data_type NOT NULL;
			ddls.Stmts = append(ddls.Stmts, &AlterTableStmt{
				Comment: simplediff.Diff(beforeColumn.String(), afterColumn.String()).String(),
				Name:    after.Name,
				Action: &ModifyColumn{
					Name:          afterColumn.Name,
					DataType:      afterColumn.DataType,
					Collate:       afterColumn.Collate,
					NotNull:       afterColumn.NotNull,
					AutoIncrement: afterColumn.AutoIncrement,
					Default:       afterColumn.Default,
					OnAction:      afterColumn.OnAction,
					Comment:       afterColumn.Comment,
				},
			})
		}

		if dropDefault {
			// ALTER TABLE table_name ALTER COLUMN column_name DROP DEFAULT;
			ddls.Stmts = append(ddls.Stmts, &AlterTableStmt{
				Comment: simplediff.Diff(beforeColumn.String(), afterColumn.String()).String(),
				Name:    after.Name,
				Action: &AlterColumnDropDefault{
					Name: afterColumn.Name,
				},
			})
		}
	}

	for _, afterColumn := range onlyLeftColumn(after.Columns, before.Columns) {
		// ALTER TABLE table_name ADD COLUMN column_name data_type;
		ddls.Stmts = append(ddls.Stmts, &AlterTableStmt{
			Comment: simplediff.Diff("", afterColumn.String()).String(),
			Name:    after.Name,
			Action: &AddColumn{
				Column: afterColumn,
			},
		})
	}
}

func onlyLeftColumn(left, right []*Column) []*Column {
	onlyLeftColumns := make([]*Column, 0)
	for _, leftColumn := range left {
		foundColumnByRight := findColumnByName(leftColumn.Name.Name, right)
		if foundColumnByRight == nil {
			onlyLeftColumns = append(onlyLeftColumns, leftColumn)
		}
	}
	return onlyLeftColumns
}

func findColumnByName(name string, columns []*Column) *Column {
	for _, column := range columns {
		if column.Name.Name == name {
			return column
		}
	}
	return nil
}

func onlyLeftConstraint(left, right Constraints) []Constraint {
	onlyLeftConstraints := make(Constraints, 0)
	for _, leftConstraint := range left {
		foundConstraintByRight := findConstraintByName(leftConstraint.GetName().Name, right)
		if foundConstraintByRight == nil {
			onlyLeftConstraints = onlyLeftConstraints.Append(leftConstraint)
		}
	}
	return onlyLeftConstraints
}

func findConstraintByName(name string, constraints []Constraint) Constraint { //nolint:ireturn
	for _, constraint := range constraints {
		if constraint.GetName().Name == name {
			return constraint
		}
	}
	return nil
}

func onlyLeftOption(left, right []*Option) []*Option {
	onlyLeftOptions := make([]*Option, 0)
	for _, leftOption := range left {
		foundOptionByRight := findOptionByName(leftOption.Name, right)
		if foundOptionByRight == nil {
			onlyLeftOptions = append(onlyLeftOptions, leftOption)
		}
	}
	return onlyLeftOptions
}

func findOptionByName(name string, columns []*Option) *Option {
	for _, column := range columns {
		if column.Name == name {
			return column
		}
	}
	return nil
}

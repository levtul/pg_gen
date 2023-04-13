package walker

import (
	"fmt"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"
	"github.com/levtul/tmp/model"
	"strconv"
)

type Walker struct {
	Schemas  map[string]*model.Schema
	Errs     []error
	Warnings []error
}

func NewWalker() *Walker {
	w := &Walker{
		Schemas: map[string]*model.Schema{
			"public": {
				Name:   "public",
				Tables: map[string]*model.Table{},
			},
		},
	}
	w.Schemas[""] = w.Schemas["public"]

	return w
}

func (w *Walker) GetWalkFunc(expr string) func(ctx interface{}, node interface{}) (stop bool) {
	return func(ctx interface{}, node interface{}) (stop bool) {
		switch n := node.(type) {
		case *tree.CreateSchema:
			if _, ok := w.Schemas[n.Schema]; ok {
				w.Errs = append(w.Errs, fmt.Errorf("%s: \nschema %s already declared", expr, n.Schema))
				return false
			}
			w.Schemas[n.Schema] = &model.Schema{
				Name:   n.Schema,
				Tables: map[string]*model.Table{},
			}
		case *tree.CreateTable:
			s := n.Table.Schema()
			if s == "" {
				s = "public"
			}
			schema, ok := w.Schemas[s]
			if !ok {
				w.Errs = append(w.Errs, fmt.Errorf("%s: \nschema %s not found", expr, s))
				return false
			}

			if _, ok := schema.Tables[n.Table.Table()]; ok {
				w.Errs = append(w.Errs, fmt.Errorf("%s: \ntable %s already declared", expr, n.Table.Table()))
				return false
			}

			table := &model.Table{
				Schema:  s,
				Name:    n.Table.Table(),
				Columns: map[string]*model.Column{},
			}
			schema.Tables[n.Table.Table()] = table

			n.HoistConstraints()

			for _, def := range n.Defs {
				switch d := def.(type) {
				case *tree.ColumnTableDef:
					table.Columns[string(d.Name)] = &model.Column{
						Name:    string(d.Name),
						Type:    d.Type,
						NotNull: d.Nullable.Nullability == tree.NotNull,
					}
					col := table.Columns[string(d.Name)]

					if d.PrimaryKey.IsPrimaryKey {
						table.PrimaryKey = append(table.PrimaryKey, string(d.Name))
					}

					if val := model.GetNthGroup(expr, model.GetColumnCommentReg(col.Name), 2); val != "" {
						gt, err := model.NewGenerationTypeFromString(val, col.Type)
						if err != nil {
							w.Errs = append(w.Errs, fmt.Errorf("%s: \n%s", expr, err))
							return false
						}
						col.GenerationType = *gt
					}
				case *tree.UniqueConstraintTableDef:
					if d.PrimaryKey {
						table.PrimaryKey = make([]string, 0, len(d.Columns))
						for _, column := range d.Columns {
							table.PrimaryKey = append(table.PrimaryKey, column.Column.String())
						}
					} else {
						columns := make([]string, 0, len(d.Columns))
						for _, column := range d.Columns {
							columns = append(columns, column.Column.String())
						}
						if len(columns) == 1 {
							table.Columns[columns[0]].Unique = true
						}
						table.UniqueConstraints = append(table.UniqueConstraints, &columns)
					}
				case *tree.ForeignKeyConstraintTableDef:
					table.ForeignKeyConstraints = append(table.ForeignKeyConstraints, &model.ForeignKeyConstraint{
						Columns: d.FromCols.ToStrings(),
						Ref: &model.ForeignKeyRef{
							Table:       schema.Tables[d.Table.Table()],
							TableSchema: d.Table.Schema(),
							TableName:   d.Table.Table(),
							Columns:     d.ToCols.ToStrings(),
						},
					})
				case *tree.CheckConstraintTableDef:
					w.Warnings = append(w.Warnings, fmt.Errorf("%s: \ncheck constraints are not supported, program may fail", expr))
				}
			}

			if str := model.GetNthGroup(expr, model.CreateTableCommentReg, 1); str != "" {
				table.TableGenerationSettings = &model.TableGenerationSettings{}
				table.TableGenerationSettings.RowsCount, _ = strconv.Atoi(str)
			}
		case *tree.AlterTable:
			tableName := n.Table.ToTableName()
			s := tableName.SchemaName.String()
			if s == "" || s == `""` {
				s = "public"
			}
			schema, ok := w.Schemas[s]
			if !ok {
				w.Errs = append(w.Errs, fmt.Errorf("%s: \nschema %s not found", expr, s))
				return false
			}

			table, ok := schema.Tables[string(tableName.TableName)]
			if !ok {
				w.Errs = append(w.Errs, fmt.Errorf("%s: \ntable %s not found", expr, tableName.TableName.String()))
				return false
			}

			for _, cmd := range n.Cmds {
				switch c := cmd.(type) {
				case *tree.AlterTableAddConstraint:
					switch d := c.ConstraintDef.(type) {
					case *tree.UniqueConstraintTableDef:
						if d.PrimaryKey {
							table.PrimaryKey = make([]string, 0, len(d.Columns))
							for _, column := range d.Columns {
								table.PrimaryKey = append(table.PrimaryKey, column.Column.String())
							}
						} else {
							columns := make([]string, 0, len(d.Columns))
							for _, column := range d.Columns {
								columns = append(columns, column.Column.String())
							}
							if len(columns) == 1 {
								table.Columns[columns[0]].Unique = true
							}
							table.UniqueConstraints = append(table.UniqueConstraints, &columns)
						}
					case *tree.ForeignKeyConstraintTableDef:
						s := d.Table.Schema()
						if s == "" {
							s = "public"
						}
						toSchema, ok := w.Schemas[s]
						if !ok {
							w.Errs = append(w.Errs, fmt.Errorf("%s: \nschema %s not found", expr, d.Table.Schema()))
							return false
						}

						toTable, ok := toSchema.Tables[d.Table.Table()]
						if !ok {
							w.Errs = append(w.Errs, fmt.Errorf("%s: \ntable %s not found", expr, d.Table.Table()))
							return false
						}

						table.ForeignKeyConstraints = append(table.ForeignKeyConstraints, &model.ForeignKeyConstraint{
							Columns: d.FromCols.ToStrings(),
							Ref: &model.ForeignKeyRef{
								Table:       toTable,
								TableSchema: toTable.Schema,
								TableName:   toTable.Name,
								Columns:     d.ToCols.ToStrings(),
							},
						})
					case *tree.CheckConstraintTableDef:
						w.Warnings = append(w.Warnings, fmt.Errorf("%s: \ncheck constraints are not supported, program may fail", expr))
					}
				case *tree.AlterTableAddColumn:
					w.Warnings = append(w.Warnings, fmt.Errorf("%s: \n\"ADD COLUMN %s\" must be in \"CREATE TABLE\" expression, not in \"ALTER TABLE\" column will be ignored", expr, c.ColumnDef.Name.String()))
				case *tree.AlterTableSetNotNull:
					column, ok := table.Columns[c.Column.String()]
					if !ok {
						w.Errs = append(w.Errs, fmt.Errorf("%s: \ncolumn %s not found", expr, c.Column.String()))
						return false
					}

					column.NotNull = true
				case *tree.AlterTableAlterPrimaryKey:
					table.PrimaryKey = make([]string, 0, len(c.Columns))
					for _, column := range c.Columns {
						table.PrimaryKey = append(table.PrimaryKey, column.Column.String())
					}
				}
			}
		}

		return false
	}
}

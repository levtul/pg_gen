package main

import (
	"context"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"
	"github.com/jackc/pgx/v4/pgxpool"
	"strconv"
)

type Walker struct {
	Schemas  map[string]*Schema
	Errs     []error
	Warnings []error
}

func NewWalker() *Walker {
	w := &Walker{
		Schemas: map[string]*Schema{
			"public": {
				Name:   "public",
				Tables: map[string]*Table{},
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
			w.Schemas[n.Schema] = &Schema{
				Name:   n.Schema,
				Tables: map[string]*Table{},
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

			table := &Table{
				Schema:  s,
				Name:    n.Table.Table(),
				Columns: map[string]*Column{},
			}
			schema.Tables[n.Table.Table()] = table

			n.HoistConstraints()

			for _, def := range n.Defs {
				switch d := def.(type) {
				case *tree.ColumnTableDef:
					table.Columns[string(d.Name)] = &Column{
						Name:    string(d.Name),
						Type:    d.Type,
						NotNull: d.Nullable.Nullability == tree.NotNull,
					}
					col := table.Columns[string(d.Name)]

					if val := GetNthGroup(expr, getColumnCommentReg(col.Name), 2); val != "" {
						gt, err := NewGenerationTypeFromString(val, col.Type)
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
					table.ForeignKeyConstraints = append(table.ForeignKeyConstraints, &ForeignKeyConstraint{
						Columns: d.FromCols.ToStrings(),
						Ref: &ForeignKeyRef{
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

			if str := GetNthGroup(expr, createTableCommentReg, 1); str != "" {
				table.TableGenerationSettings = &TableGenerationSettings{}
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

						table.ForeignKeyConstraints = append(table.ForeignKeyConstraints, &ForeignKeyConstraint{
							Columns: d.FromCols.ToStrings(),
							Ref: &ForeignKeyRef{
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

func (w *Walker) GetTablesOrder() ([]*Table, error) {
	been := map[*Table]int{}
	order := make([]*Table, 0, len(w.Schemas))
	stack := make([]*Table, 0, len(w.Schemas))

	for _, schema := range w.Schemas {
		for _, table := range schema.Tables {
			if been[table] == 0 {
				stack = append(stack, table)
				for len(stack) > 0 {
					t := stack[len(stack)-1]
					if been[t] == 1 {
						order = append(order, t)
						been[t] = 2
						stack = stack[:len(stack)-1]
						continue
					}
					been[t] = 1

					for _, fk := range t.ForeignKeyConstraints {
						if been[fk.Ref.Table] == 1 {
							return nil, fmt.Errorf("cycle detected in foreign key constraints")
						}
						if been[fk.Ref.Table] == 0 {
							stack = append(stack, fk.Ref.Table)
						}
					}
				}
			}
		}
	}

	return order, nil
}

func (w *Walker) FillAllDB(db *pgxpool.Pool) error {
	order, err := w.GetTablesOrder()
	if err != nil {
		return err
	}

	generatedData := map[*Table]map[string][]interface{}{}
	for _, table := range order {
		if err := w.fillDB(table, db, generatedData); err != nil {
			return err
		}
	}

	return nil
}

func (w *Walker) fillDB(table *Table, db *pgxpool.Pool, data map[*Table]map[string][]interface{}) error {
	stmt := sq.Insert(table.Name)
	columns := make([]*Column, 0, len(table.Columns))
	for _, column := range table.Columns {
		columns = append(columns, column)
		stmt = stmt.Columns(column.Name)
	}
	//d := map[string][]interface{}{}

	if table.TableGenerationSettings == nil {
		table.TableGenerationSettings = &TableGenerationSettings{RowsCount: 100}
	}
	for i := 0; i < table.TableGenerationSettings.RowsCount; i++ {
		row := make([]interface{}, 0, len(table.Columns))
		for _, column := range columns {
			row = append(row, column.GenerateValue())
		}
		stmt = stmt.Values(row...)
	}

	//stmt.Suffix("ON CONFLICT DO NOTHING")
	sql, values, err := stmt.PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return err
	}
	tx, err := db.Begin(context.Background())
	_, err = tx.Exec(context.Background(), sql, values...)
	if err != nil {
		tx.Rollback(context.Background())
		return err
	}
	err = tx.Commit(context.Background())

	return err
}

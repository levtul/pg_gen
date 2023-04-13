package walker

import (
	"context"
	"fmt"
	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/levtul/tmp/model"
	"math/rand"
)

const maxTriesCount = 10

func (w *Walker) GetTablesOrder() ([]*model.Table, error) {
	been := map[*model.Table]int{}
	prev := map[*model.Table]*model.Table{}
	order := make([]*model.Table, 0, len(w.Schemas))
	stack := make([]*model.Table, 0, len(w.Schemas))

	var prevT *model.Table
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
					prev[t] = prevT
					prevT = t

					for _, fk := range t.ForeignKeyConstraints {
						if been[fk.Ref.Table] == 1 {
							cycle := make([]*model.Table, 0, len(w.Schemas))
							for cur := t; (cur != t || len(cycle) == 0) && cur != nil; cur = prev[cur] {
								cycle = append(cycle, cur)
							}

							text := fmt.Sprintf("%s.%s -> ", t.Schema, t.Name)
							for i := len(cycle) - 1; i >= 0; i-- {
								text += fmt.Sprintf("%s.%s -> ", cycle[i].Schema, cycle[i].Name)
							}
							text = text[:len(text)-4]

							return nil, fmt.Errorf("cycle detected in foreign key constraints: %s", text)
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

	generatedData := map[*model.Table]map[string][]interface{}{}
	for _, table := range order {
		if err := w.fillDB(table, db, generatedData); err != nil {
			return err
		}
	}

	return nil
}

func (w *Walker) fillDB(table *model.Table, db *pgxpool.Pool, data map[*model.Table]map[string][]interface{}) error {
	stmt := sq.Insert(table.Name)
	columns := make([]*model.Column, 0, len(table.Columns))
	for _, column := range table.Columns {
		columns = append(columns, column)
		stmt = stmt.Columns(column.Name)
	}

	fks := table.ForeignKeyConstraints
	ucs := table.UniqueConstraints
	ucs = append(ucs, &table.PrimaryKey)

	if table.TableGenerationSettings == nil {
		table.TableGenerationSettings = &model.TableGenerationSettings{RowsCount: 100}
	}
	prevValues := make([]map[string]interface{}, 0, table.TableGenerationSettings.RowsCount)
	for i := 0; i < table.TableGenerationSettings.RowsCount; i++ {
		generated := false
		for try := 0; try < maxTriesCount; try++ {
			rowMap := make(map[string]interface{}, len(table.Columns))
			for _, fk := range fks {
				cnt := fk.Ref.Table.TableGenerationSettings.RowsCount
				if cnt == 0 {
					return fmt.Errorf("table %s has no rows", fk.Ref.Table.Name)
				}

				rowN := rand.Intn(cnt)
				for i, column := range fk.Columns {
					rowMap[column] = data[fk.Ref.Table][fk.Ref.Columns[i]][rowN]
				}
			}

			row := make([]interface{}, 0, len(table.Columns))
			for _, column := range columns {
				if _, ok := rowMap[column.Name]; !ok {
					rowMap[column.Name] = column.GenerateValue()
				}

				row = append(row, rowMap[column.Name])
			}

			valid := true
			for _, uc := range ucs {
				for _, col := range prevValues {
					foundNonEqual := len(prevValues) == 0
					for _, column := range *uc {
						if col[column] != rowMap[column] {
							foundNonEqual = true
							break
						}
					}
					if !foundNonEqual {
						valid = false
						break
					}
				}

				if !valid {
					break
				}
			}
			if !valid {
				continue
			}

			stmt = stmt.Values(row...)
			prevValues = append(prevValues, rowMap)
			generated = true
			break
		}

		if !generated {
			return fmt.Errorf("unable to generate unique row for table %s", table.Name)
		}
	}

	//stmt.Suffix("ON CONFLICT DO NOTHING")
	sql, values, err := stmt.PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return err
	}

	tx, err := db.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("unable to begin transaction: %w", err)
	}
	_, err = tx.Exec(context.Background(), sql, values...)
	if err != nil {
		errR := tx.Rollback(context.Background())
		if errR != nil {
			return fmt.Errorf("unable to rollback transaction: %w", errR)
		}
		return fmt.Errorf("unable to execute query: %w", err)
	}
	err = tx.Commit(context.Background())
	if err != nil {
		return fmt.Errorf("unable to commit transaction: %w", err)
	}

	data[table] = map[string][]interface{}{}
	for _, value := range prevValues {
		for column, val := range value {
			data[table][column] = append(data[table][column], val)
		}
	}

	return nil
}

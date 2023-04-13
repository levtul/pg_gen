package domain

import (
	"fmt"
	"github.com/auxten/postgresql-parser/pkg/sql/parser"
	"github.com/auxten/postgresql-parser/pkg/walk"
	"github.com/levtul/tmp/model"
	"github.com/levtul/tmp/walker"
	"log"
	"strings"
)

func Walk(sql string) (*walker.Walker, error) {
	exprs := model.SplitterReg.Split(sql, -1)
	myWalker := walker.NewWalker()
	w := &walk.AstWalker{}
	for _, expr := range exprs {
		if strings.HasPrefix(expr, "CREATE SCHEMA") ||
			strings.HasPrefix(expr, "CREATE TABLE") ||
			strings.HasPrefix(expr, "ALTER TABLE") {
			if model.GeneratedReg.MatchString(expr) {
				continue
			}

			stmts, err := parser.Parse(expr)
			if err != nil {
				return nil, fmt.Errorf("parser error: %w, expr: %s\n", err, expr)
			}

			w.Fn = myWalker.GetWalkFunc(expr)
			_, err = w.Walk(stmts, nil)
			if err != nil {
				return nil, fmt.Errorf("walker error: %w, expr: %s\n", err, expr)
			}
		}
	}

	if len(myWalker.Errs) > 0 {
		text := ""
		for _, err := range myWalker.Errs {
			text += err.Error() + ", "
		}
		return nil, fmt.Errorf("errors: %s", text)
	}
	if len(myWalker.Warnings) > 0 {
		for _, warning := range myWalker.Warnings {
			log.Printf("warning: %s\n", warning)
		}
	}

	return myWalker, nil
}

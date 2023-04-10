package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strings"

	"github.com/auxten/postgresql-parser/pkg/sql/parser"
	"github.com/auxten/postgresql-parser/pkg/walk"
)

var (
	splitter     = regexp.MustCompile("-- Statement # \\d+\n(--[^\n]*\n)*")
	generatedReg = regexp.MustCompile(`ADD GENERATED (BY DEFAULT|ALWAYS) AS IDENTITY \(\s*SEQUENCE NAME\s*.+START WITH \d+\s*INCREMENT BY \d+\s*NO MINVALUE\s*NO MAXVALUE\s*CACHE \d+\s*\);`)
)

func main() {
	path := "examples/comments_test"

	cmd := exec.Command("pg_format", "-N", path)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		log.Fatalf("cmd.Run: %v", err)
	}

	sql := stdout.String()
	exprs := splitter.Split(sql, -1)
	myWalker := NewWalker()
	w := &walk.AstWalker{}
	for _, expr := range exprs {
		if strings.HasPrefix(expr, "CREATE SCHEMA") ||
			strings.HasPrefix(expr, "CREATE TABLE") ||
			strings.HasPrefix(expr, "ALTER TABLE") {
			if generatedReg.MatchString(expr) {
				continue
			}

			stmts, err := parser.Parse(expr)
			if err != nil {
				fmt.Print(err, " ", expr)
				return
			}

			w.Fn = myWalker.GetWalkFunc(expr)
			_, _ = w.Walk(stmts, nil)
		}
	}
	fmt.Println(myWalker.Schemas)
	fmt.Println(myWalker.Errs)
	//fmt.Println(myWalker.Warnings)
	return
}

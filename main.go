package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/auxten/postgresql-parser/pkg/sql/parser"
	"github.com/auxten/postgresql-parser/pkg/walk"
)

func main() {
	// get filename & connection string from args
	if len(os.Args) < 3 {
		fmt.Println("Usage: ./pgmodeler <filename> <connection_string>")
		return
	}

	// get filename
	filename := os.Args[1]
	filename = "examples/no_generation"

	// get connection string
	connectionString := os.Args[2]

	path := filename

	cmd := exec.Command("pg_format", "-N", path)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		log.Fatalf("cmd.Run: %v", err)
	}

	sql := stdout.String()
	exprs := splitterReg.Split(sql, -1)
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
				fmt.Printf("error: %s, expr: \n%s", err, expr)
				return
			}

			w.Fn = myWalker.GetWalkFunc(expr)
			_, _ = w.Walk(stmts, nil)
		}
	}

	if len(myWalker.Errs) > 0 {
		for _, err := range myWalker.Errs {
			fmt.Printf("error: %s\n", err)
		}
		return
	}
	if len(myWalker.Warnings) > 0 {
		for _, warning := range myWalker.Warnings {
			fmt.Printf("warning: %s", warning)
		}
	}

	// connect to postgresql database
	databaseUrl := connectionString

	// this returns connection pool
	dbPool, err := pgxpool.Connect(context.Background(), databaseUrl)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	defer dbPool.Close()

	err = dbPool.Ping(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to ping database: %v\n", err)
		os.Exit(1)
	}

	err = myWalker.FillAllDB(dbPool)
	if err != nil {
		fmt.Printf("error: %s", err)
		return
	}

	return
}

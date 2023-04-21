package main

import (
	"github.com/levtul/tmp/db"
	"github.com/levtul/tmp/domain"
	"log"
	"os"
)

func main() {
	// get filename & connection string from args
	if len(os.Args) < 3 {
		log.Fatalln("Usage: ./pg_gen <filename> <connection_string>")
	}

	// get filename
	filename := os.Args[1]

	// get connection string
	connectionString := os.Args[2]

	sql, err := domain.RunFormatter(filename)
	if err != nil {
		log.Fatalf("domain.RunFormatter: %s", err.Error())
	}

	myWalker, err := domain.Walk(sql)
	if err != nil {
		log.Fatalf("domain.Walk: %s", err.Error())
	}

	dbPool, err := db.Connect(connectionString)
	if err != nil {
		log.Fatalf("db.Connect: %s", err.Error())
	}
	defer dbPool.Close()

	err = myWalker.FillAllDB(dbPool)
	if err != nil {
		log.Fatalf("error: %s", err)
	}

	log.Println("Successfully generated data!")
	return
}

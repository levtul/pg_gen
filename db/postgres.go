package db

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
)

func Connect(connectionString string) (*pgxpool.Pool, error) {
	// connect to postgresql database
	databaseUrl := connectionString

	// this returns connection pool
	dbPool, err := pgxpool.Connect(context.Background(), databaseUrl)
	if err != nil {
		return nil, fmt.Errorf("Unable to connect to database: %v\n", err)
	}

	err = dbPool.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("Unable to ping database: %v\n", err)
	}

	return dbPool, nil
}

package config

import (
	"context"
	"github.com/jackc/pgx/v4"
	"log"
	"os"
)

type Database struct {
	DB *pgx.Conn
}

func SetupDatabase(ctx context.Context) *pgx.Conn {
	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Printf("Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	log.Println("Successfully connected to database")

	return conn
}

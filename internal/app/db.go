package app

import (
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// InitDB initializes and returns a PostgreSQL connection via sqlx
func InitDB() *sqlx.DB {
	dsn := "host=" + os.Getenv("POSTGRES_HOST") +
		" port=" + os.Getenv("POSTGRES_PORT") +
		" user=" + os.Getenv("POSTGRES_USER") +
		" password=" + os.Getenv("POSTGRES_PASSWORD") +
		" dbname=" + os.Getenv("POSTGRES_DB") +
		" sslmode=disable"

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	return db
}

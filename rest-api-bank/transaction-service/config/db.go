package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func InitDB() *sqlx.DB {

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_SSLMODE"),
	)

	var db *sqlx.DB
	var err error

	for i := 0; i < 10; i++ {
		db, err = sqlx.Open("postgres", dsn)
		if err != nil {
			log.Println("DB open error:", err)
			time.Sleep(2 * time.Second)
			continue
		}

		err = db.Ping()
		if err == nil {
			log.Println("Connected to database")
			return db
		}

		log.Println("Waiting for database...")
		time.Sleep(2 * time.Second)
	}

	log.Fatal("Could not connect to database after retries")
	return nil
}

func RunMigrations(db *sqlx.DB) {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS transactions (
			id              UUID        PRIMARY KEY,
			from_account_id UUID        NOT NULL,
			to_account_id   UUID        NOT NULL,
			amount          BIGINT      NOT NULL CHECK (amount > 0),
			remark          TEXT        NOT NULL DEFAULT '',
			status          VARCHAR(20) NOT NULL DEFAULT 'SUCCESS',
			created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,

		`CREATE INDEX IF NOT EXISTS idx_transactions_from_account ON transactions(from_account_id);`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_to_account ON transactions(to_account_id);`,
	}

	for _, q := range queries {
		_, err := db.Exec(q)
		if err != nil {
			log.Fatal("Migration failed:", err)
		}
	}

	log.Println("Migration success")
}

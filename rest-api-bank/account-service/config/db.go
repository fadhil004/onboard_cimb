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
		`CREATE TABLE IF NOT EXISTS accounts (
			id             UUID        PRIMARY KEY,
			account_number VARCHAR(20) NOT NULL UNIQUE,
			account_holder VARCHAR(255) NOT NULL,
			balance        BIGINT      NOT NULL DEFAULT 0 CHECK (balance >= 0),
			created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,

		`CREATE INDEX IF NOT EXISTS idx_accounts_account_number ON accounts(account_number);`,

		// Auto-update updated_at saat row diupdate
		`CREATE OR REPLACE FUNCTION set_updated_at()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = NOW();
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;`,

		`DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_trigger WHERE tgname = 'trg_accounts_updated_at'
			) THEN
				CREATE TRIGGER trg_accounts_updated_at
				BEFORE UPDATE ON accounts
				FOR EACH ROW EXECUTE FUNCTION set_updated_at();
			END IF;
		END;
		$$;`,
	}

	for _, q := range queries {
		_, err := db.Exec(q)
		if err != nil {
			log.Fatal("Migration failed:", err)
		}
	}

	log.Println("Migration success")
}

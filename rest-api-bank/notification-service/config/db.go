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
			log.Println("Connected to notification database")
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
		`CREATE TABLE IF NOT EXISTS notification_logs (
			id              SERIAL      PRIMARY KEY,
			event_type      VARCHAR(50) NOT NULL,
			event_id        VARCHAR(100) NOT NULL,
			topic           VARCHAR(50) NOT NULL,
			payload         JSONB       NOT NULL,
			callback_url    TEXT,
			callback_status VARCHAR(20) DEFAULT 'PENDING',
			callback_response TEXT,
			created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,

		`CREATE INDEX IF NOT EXISTS idx_notification_logs_event_type ON notification_logs(event_type);`,
		`CREATE INDEX IF NOT EXISTS idx_notification_logs_created_at ON notification_logs(created_at);`,
	}

	for _, q := range queries {
		_, err := db.Exec(q)
		if err != nil {
			log.Fatal("Migration failed:", err)
		}
	}
	log.Println("Migration success")
}

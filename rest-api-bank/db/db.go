package db

import (
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func InitDB() *sqlx.DB { 
	dsn := "host=localhost user=postgres password=postgres dbname=rest-api-bank port=5432 sslmode=disable"

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatal("DB Error:", err)
	}

	return db
}
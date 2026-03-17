package config

import (
	"log"

	"github.com/jmoiron/sqlx"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *sqlx.DB
var GORM *gorm.DB

func InitSQLX() {
	dsn := "host=localhost user=postgres password=postgres dbname=test port=5432 sslmode=disable"

	var err error
	DB, err = sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatal("DB Error:", err)
	}

	log.Println("SQLX Connected")
}

func InitGORM() {
	dsn := "host=localhost user=postgres password=postgres dbname=test port=5432 sslmode=disable"

	var err error
	GORM, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("GORM Error:", err)
	}

	log.Println("GORM Connected")
}
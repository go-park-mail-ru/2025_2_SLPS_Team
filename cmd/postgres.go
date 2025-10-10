package main

import (
	"database/sql"
	"log"
)

func NewPostgres(dataSourceName string) *sql.DB {
	db, err := sql.Open("pgx", dataSourceName)
	if err != nil {
		log.Fatalf("ошибка подключения к БД: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("ошибка ping БД: %v", err)
	}

	return db
}

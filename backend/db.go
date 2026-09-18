package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

 var db *sql.DB

 func initDB() error {
	var err error
	db, err = sql.Open("sqlite3", "./dompetku.db?_foreign_keys=on")
	if err != nil {
		return err
	}
	return db.Ping()
}
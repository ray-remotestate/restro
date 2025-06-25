package database

import(
	"database/sql"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

var TodoEx *sql.DB

func ConnectAndMigrate() error {
	err := godotenv.Load()
	if err != nil {
		return err
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", 
		os.Getenv("DB_host"), 
		os.Getenv("DB_port"), 
		os.Getenv("DB_user"), 
		os.Getenv("DB_password"), 
		os.Getenv("DB_name"),
	)

	DB, err := sql.Open("postgres", connStr)
	if err != nil {
		return err
	}

	err = DB.Ping()
	if err != nil {
		return err
	}

	TodoEx = DB
	return migrateUp(DB)
}

func migrateUp(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	logrus.Println("done creating migration")
	m, err := migrate.NewWithDatabaseInstance(
		"file:///home/ray/Golang_Projects/restro/database/migrations",
		"postgres", driver)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	// // migrate down if required
	// if err := m.Down(); err != nil && err != migrate.ErrNoChange {
	// 	return err
	// }

	return nil
}

func ShutdownDatabase() error {
	return TodoEx.Close()
}

func Tx(fn func(tx *sql.Tx) error) error {
	tx, err := TodoEx.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}
package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/go-sql-driver/mysql"
)

type Db interface {
	NewScreenshot(userTelegramLogin, fname, lname, link, desc, screen string, isDev bool) error
	KeepAlive()
}

type MadeOnGcDb struct {
	db *sql.DB
	Db
}

func New() Db {
	// Capture connection properties.
	cfg := mysql.Config{
		User:                 os.Getenv("DBUSER"),
		Passwd:               os.Getenv("DBPASS"),
		Net:                  "tcp",
		Addr:                 os.Getenv("DBADDR"),
		DBName:               os.Getenv("DBNAME"),
		AllowNativePasswords: true,
	}
	// Get a database handle.
	var err error
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		log.Fatal().Err(err)
	}

	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal().Err(pingErr)
	}
	log.Info().Msg("Connected to the database")

	// TODO: I am not sure why I have to use those params
	db.SetConnMaxLifetime(0)
	db.SetMaxOpenConns(500)
	db.SetMaxIdleConns(5)

	return &MadeOnGcDb{db: db}
}

func (d MadeOnGcDb) KeepAlive() {
	for {
		time.Sleep(10 * time.Second)
		pingErr := d.db.Ping()
		if pingErr != nil {
			log.Fatal().Err(pingErr)
		}
	}
}

func (d MadeOnGcDb) NewUser(userId, fname, lname string) error {
	var cnt int
	err := d.db.QueryRow("SELECT COUNT(*) as cnt FROM user WHERE telegramName = ?", userId).Scan(&cnt)
	if err != nil {
		return err
	}
	if cnt == 0 {
		_, err := d.db.Exec("INSERT INTO user (telegramName, first_name, last_name) VALUES (?, ?, ?)", userId, fname, lname)
		if err != nil {
			return fmt.Errorf("cannot insert a new user: %v", err)
		}
	}
	return nil
}

func (d MadeOnGcDb) NewScreenshot(userTelegramLogin, fname, lname, link, desc, screen string, isDev bool) error {
	err := d.NewUser(userTelegramLogin, fname, lname)
	if err != nil {
		return fmt.Errorf("newUser: %v", err)
	}

	_, err = d.db.Exec("INSERT INTO screenshot (user, screen, link, description, developer) VALUES (?, ?, ?, ?, ?)", userTelegramLogin, screen, link, desc, isDev)
	if err != nil {
		return fmt.Errorf("newScreenshot: %v", err)
	}
	return nil
}

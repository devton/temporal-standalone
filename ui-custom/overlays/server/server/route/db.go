package route

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"

	_ "github.com/lib/pq"
)

var (
	db   *sql.DB
	dbMu sync.RWMutex
)

func getDB() *sql.DB {
	dbMu.RLock()
	defer dbMu.RUnlock()
	return db
}

func initDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		host := os.Getenv("POSTGRES_HOST")
		port := os.Getenv("POSTGRES_PORT")
		user := os.Getenv("POSTGRES_USER")
		password := os.Getenv("POSTGRES_PASSWORD")
		dbname := os.Getenv("POSTGRES_DB")

		if host == "" {
			host = "localhost"
		}
		if port == "" {
			port = "5432"
		}
		if user == "" {
			user = "temporal"
		}
		if password == "" {
			password = "temporal"
		}
		if dbname == "" {
			dbname = "temporal"
		}
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			host, port, user, password, dbname)
	}

	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Printf("[DB] Failed to open database: %v", err)
		return
	}

	conn.SetMaxOpenConns(5)
	conn.SetMaxIdleConns(2)

	if err := conn.Ping(); err != nil {
		log.Printf("[DB] Failed to ping database (API keys will use in-memory fallback): %v", err)
		conn.Close()
		return
	}

	if err := migrateDB(conn); err != nil {
		log.Printf("[DB] Failed to run migrations: %v", err)
		conn.Close()
		return
	}

	dbMu.Lock()
	db = conn
	dbMu.Unlock()

	log.Println("[DB] Connected — api_keys table ready")
}

func migrateDB(conn *sql.DB) error {
	_, err := conn.Exec(`
		CREATE TABLE IF NOT EXISTS api_keys (
			id          SERIAL PRIMARY KEY,
			key_id      TEXT    NOT NULL UNIQUE,
			name        TEXT    NOT NULL DEFAULT '',
			description TEXT    NOT NULL DEFAULT '',
			key_secret  TEXT    NOT NULL DEFAULT '',
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			expires_at  TIMESTAMPTZ,
			owner_id    TEXT    NOT NULL DEFAULT ''
		)
	`)
	return err
}

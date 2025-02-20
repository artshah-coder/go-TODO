package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

type DB struct {
	Pool *pgxpool.Pool
}

// Функция подключения к БД и выполнения миграций:
// Инициализируем схему БД
func Connect() (*DB, error) {
	connStr := fmt.Sprintf("%s://%s:%s@%s:%s/%s?sslmode=disable&connect_timeout=%d",
		"postgres",
		"postgres",
		"12345",
		"db",
		"5432",
		"tasks",
		5,
	)

	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Unable to connection to database: %v\n", err)
		return nil, err
	}

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("Unable to set dialect database: %v\n", err)
		return nil, err
	}

	db := stdlib.OpenDBFromPool(pool)
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err := Up(ctx, tx); err != nil {
		log.Fatalf("Migration failed: %v\n", err)
		return nil, err
	}
	tx.Commit()

	return &DB{Pool: pool}, nil
}

func init() {
	goose.AddMigrationContext(Up, Down)
}

func Up(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.Exec(
		`CREATE TABLE IF NOT EXISTS tasks (
		id SERIAL PRIMARY KEY,
		title TEXT NOT NULL,
		description TEXT, status TEXT CHECK (status IN ('new', 'in_progress', 'done')) DEFAULT 'new',
		created_at TIMESTAMP DEFAULT now(),
		updated_at TIMESTAMP DEFAULT now())`,
	)
	if err != nil {
		return err
	}
	return nil
}

func Down(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS tasks")
	if err != nil {
		return err
	}
	return nil
}

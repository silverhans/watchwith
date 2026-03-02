package store

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

type RoomRow struct {
	ID        string
	Slug      string
	Name      string
	VideoURL  string
	CreatedAt time.Time
}

func NewDB(databaseURL string) (*DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	log.Println("Connected to PostgreSQL")
	return &DB{db}, nil
}

func (db *DB) RunMigrations() error {
	migration, err := os.ReadFile("internal/store/migrations/001_init.sql")
	if err != nil {
		return fmt.Errorf("failed to read migration: %w", err)
	}

	_, err = db.Exec(string(migration))
	if err != nil {
		return fmt.Errorf("failed to run migration: %w", err)
	}

	log.Println("Migrations applied")
	return nil
}

func (db *DB) CreateRoom(ctx context.Context, id, slug, name string) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO rooms (id, slug, name) VALUES ($1, $2, $3)`,
		id, slug, name,
	)
	return err
}

func (db *DB) GetRoomBySlug(ctx context.Context, slug string) (*RoomRow, error) {
	row := db.QueryRowContext(ctx,
		`SELECT id, slug, name, video_url, created_at FROM rooms WHERE slug = $1`,
		slug,
	)

	var r RoomRow
	if err := row.Scan(&r.ID, &r.Slug, &r.Name, &r.VideoURL, &r.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

func (db *DB) UpdateRoomVideoURL(ctx context.Context, slug, videoURL string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE rooms SET video_url = $1 WHERE slug = $2`,
		videoURL, slug,
	)
	return err
}

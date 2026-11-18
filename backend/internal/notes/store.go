package notes

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	pool *pgxpool.Pool
}

var _ Store = (*Postgres)(nil)

func NewPostgres(ctx context.Context, databaseURL string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	store := &Postgres{pool: pool}
	if err := store.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return store, nil
}

func (p *Postgres) Close() {
	p.pool.Close()
}

func (p *Postgres) migrate(ctx context.Context) error {
	_, err := p.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS notes (
			id         BIGSERIAL PRIMARY KEY,
			title      TEXT NOT NULL,
			body       TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		return fmt.Errorf("migrate notes: %w", err)
	}
	return nil
}

func (p *Postgres) List(ctx context.Context) ([]Note, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id, title, body, created_at, updated_at
		FROM notes
		ORDER BY updated_at DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Note{}
	for rows.Next() {
		note, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, note)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (p *Postgres) Create(ctx context.Context, title, body string) (Note, error) {
	title, body, err := Normalize(title, body)
	if err != nil {
		return Note{}, err
	}
	row := p.pool.QueryRow(ctx, `
		INSERT INTO notes (title, body)
		VALUES ($1, $2)
		RETURNING id, title, body, created_at, updated_at
	`, title, body)
	return scanNote(row)
}

func (p *Postgres) Update(ctx context.Context, id int64, title, body string) (Note, error) {
	title, body, err := Normalize(title, body)
	if err != nil {
		return Note{}, err
	}
	row := p.pool.QueryRow(ctx, `
		UPDATE notes
		SET title = $2, body = $3, updated_at = now()
		WHERE id = $1
		RETURNING id, title, body, created_at, updated_at
	`, id, title, body)
	note, err := scanNote(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Note{}, ErrNotFound
	}
	return note, err
}

func (p *Postgres) Delete(ctx context.Context, id int64) error {
	tag, err := p.pool.Exec(ctx, `DELETE FROM notes WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanNote(row scanner) (Note, error) {
	var note Note
	err := row.Scan(&note.ID, &note.Title, &note.Body, &note.CreatedAt, &note.UpdatedAt)
	return note, err
}

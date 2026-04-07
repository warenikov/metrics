package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	models "metrics/internal/model"

	"github.com/golang-migrate/migrate/v4"
	pgxmigrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

var ErrPGNotFound = errors.New("metric not found in db")

type PostgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB, migrations fs.FS) (*PostgresRepo, error) {
	if err := runMigrations(db, migrations); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}
	return &PostgresRepo{db: db}, nil
}

func runMigrations(db *sql.DB, migrations fs.FS) error {
	src, err := iofs.New(migrations, ".")
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	driver, err := pgxmigrate.WithInstance(db, &pgxmigrate.Config{})
	if err != nil {
		return fmt.Errorf("failed to create pgx driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "pgx5", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}

func (r *PostgresRepo) UpdateGauges(ctx context.Context, m models.Metrics) (models.Metrics, error) {
	if m.ID == "" || m.MType == "" || m.Value == nil {
		return m, ErrInvalidValue
	}

	query := `
		INSERT INTO metrics (id, mtype, value, delta)
		VALUES ($1, $2, $3, NULL)
		ON CONFLICT (id, mtype) DO UPDATE
		SET value = excluded.value
		RETURNING id, mtype, value, delta
	`

	var result models.Metrics
	err := r.db.QueryRowContext(ctx, query, m.ID, m.MType, *m.Value).Scan(
		&result.ID, &result.MType, &result.Value, &result.Delta,
	)
	if err != nil {
		return m, fmt.Errorf("failed to update gauge: %w", err)
	}

	return result, nil
}

func (r *PostgresRepo) UpdateCounter(ctx context.Context, m models.Metrics) (models.Metrics, error) {
	if m.ID == "" || m.MType == "" || m.Delta == nil {
		return m, ErrInvalidValue
	}

	query := `
		INSERT INTO metrics (id, mtype, delta, value)
		VALUES ($1, $2, $3, NULL)
		ON CONFLICT (id, mtype) DO UPDATE
		SET delta = COALESCE(metrics.delta, 0) + excluded.delta
		RETURNING id, mtype, value, delta
	`

	var result models.Metrics
	err := r.db.QueryRowContext(ctx, query, m.ID, m.MType, *m.Delta).Scan(
		&result.ID, &result.MType, &result.Value, &result.Delta,
	)
	if err != nil {
		return m, fmt.Errorf("failed to update counter: %w", err)
	}

	return result, nil
}

func (r *PostgresRepo) GetMetrica(ctx context.Context, m models.Metrics) (*models.Metrics, error) {
	if m.ID == "" || m.MType == "" {
		return nil, ErrInvalidValue
	}

	query := `SELECT id, mtype, value, delta FROM metrics WHERE id = $1 AND mtype = $2`

	var result models.Metrics
	err := r.db.QueryRowContext(ctx, query, m.ID, m.MType).Scan(
		&result.ID, &result.MType, &result.Value, &result.Delta,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPGNotFound
		}
		return nil, fmt.Errorf("failed to get metric: %w", err)
	}

	return &result, nil
}

func (r *PostgresRepo) GetListMetrics(ctx context.Context) ([]models.Metrics, error) {
	query := `SELECT id, mtype, value, delta FROM metrics ORDER BY id, mtype`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}
	defer rows.Close()

	var metrics []models.Metrics
	for rows.Next() {
		var m models.Metrics
		if err := rows.Scan(&m.ID, &m.MType, &m.Value, &m.Delta); err != nil {
			return nil, fmt.Errorf("failed to scan metric: %w", err)
		}
		metrics = append(metrics, m)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return metrics, nil
}

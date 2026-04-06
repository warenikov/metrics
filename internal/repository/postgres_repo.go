package repository

import (
	"database/sql"
	"errors"
	"fmt"
	models "metrics/internal/model"
)

var ErrPGNotFound = errors.New("metric not found in db")

type PostgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) (*PostgresRepo, error) {
	return &PostgresRepo{db: db}, nil
}

func (r *PostgresRepo) UpdateGauges(m models.Metrics) (models.Metrics, error) {
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
	err := r.db.QueryRow(query, m.ID, m.MType, *m.Value).Scan(
		&result.ID, &result.MType, &result.Value, &result.Delta,
	)
	if err != nil {
		return m, fmt.Errorf("failed to update gauge: %w", err)
	}

	result.MType = m.MType
	return result, nil
}

func (r *PostgresRepo) UpdateCounter(m models.Metrics) (models.Metrics, error) {
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
	err := r.db.QueryRow(query, m.ID, m.MType, *m.Delta).Scan(
		&result.ID, &result.MType, &result.Value, &result.Delta,
	)
	if err != nil {
		return m, fmt.Errorf("failed to update counter: %w", err)
	}

	result.MType = m.MType
	return result, nil
}

func (r *PostgresRepo) GetMetrica(m models.Metrics) (*models.Metrics, error) {
	if m.ID == "" || m.MType == "" {
		return nil, ErrInvalidValue
	}

	query := `SELECT id, mtype, value, delta FROM metrics WHERE id = $1 AND mtype = $2`

	var result models.Metrics
	err := r.db.QueryRow(query, m.ID, m.MType).Scan(
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

func (r *PostgresRepo) GetListMetrics() ([]models.Metrics, error) {
	query := `SELECT id, mtype, value, delta FROM metrics ORDER BY id, mtype`

	rows, err := r.db.Query(query)
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

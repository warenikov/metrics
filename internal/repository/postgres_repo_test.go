package repository

import (
	"database/sql"
	"testing"

	models "metrics/internal/model"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockRepo(t *testing.T) (*PostgresRepo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return &PostgresRepo{db: db}, mock
}

func TestPostgresRepo_UpdateGauges(t *testing.T) {
	floatPtr := func(f float64) *float64 { return &f }

	tests := []struct {
		name    string
		input   models.Metrics
		wantErr bool
	}{
		{
			name:    "new gauge",
			input:   models.Metrics{ID: "g1", MType: models.Gauge, Value: floatPtr(1.5)},
			wantErr: false,
		},
		{
			name:    "update existing gauge",
			input:   models.Metrics{ID: "g1", MType: models.Gauge, Value: floatPtr(9.9)},
			wantErr: false,
		},
		{
			name:    "nil value returns error",
			input:   models.Metrics{ID: "g1", MType: models.Gauge, Value: nil},
			wantErr: true,
		},
		{
			name:    "empty ID returns error",
			input:   models.Metrics{ID: "", MType: models.Gauge, Value: floatPtr(1.0)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newMockRepo(t)

			if !tt.wantErr {
				rows := sqlmock.NewRows([]string{"id", "mtype", "value", "delta"}).
					AddRow(tt.input.ID, tt.input.MType, tt.input.Value, nil)
				mock.ExpectQuery("INSERT INTO metrics").
					WithArgs(tt.input.ID, tt.input.MType, *tt.input.Value).
					WillReturnRows(rows)
			}

			result, err := repo.UpdateGauges(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.input.ID, result.ID)
				assert.Equal(t, tt.input.MType, result.MType)
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		})
	}
}

func TestPostgresRepo_UpdateCounter(t *testing.T) {
	intPtr := func(i int64) *int64 { return &i }

	tests := []struct {
		name          string
		input         models.Metrics
		returnedDelta int64
		wantErr       bool
	}{
		{
			name:          "new counter",
			input:         models.Metrics{ID: "c1", MType: models.Counter, Delta: intPtr(5)},
			returnedDelta: 5,
			wantErr:       false,
		},
		{
			name:          "accumulate counter",
			input:         models.Metrics{ID: "c1", MType: models.Counter, Delta: intPtr(3)},
			returnedDelta: 8,
			wantErr:       false,
		},
		{
			name:    "nil delta returns error",
			input:   models.Metrics{ID: "c1", MType: models.Counter, Delta: nil},
			wantErr: true,
		},
		{
			name:    "empty ID returns error",
			input:   models.Metrics{ID: "", MType: models.Counter, Delta: intPtr(1)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newMockRepo(t)

			if !tt.wantErr {
				rows := sqlmock.NewRows([]string{"id", "mtype", "value", "delta"}).
					AddRow(tt.input.ID, tt.input.MType, nil, tt.returnedDelta)
				mock.ExpectQuery("INSERT INTO metrics").
					WithArgs(tt.input.ID, tt.input.MType, *tt.input.Delta).
					WillReturnRows(rows)
			}

			result, err := repo.UpdateCounter(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.input.ID, result.ID)
				assert.Equal(t, tt.returnedDelta, *result.Delta)
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		})
	}
}

func TestPostgresRepo_GetMetrica(t *testing.T) {
	floatPtr := func(f float64) *float64 { return &f }

	tests := []struct {
		name    string
		input   models.Metrics
		rowFunc func(mock sqlmock.Sqlmock, input models.Metrics)
		wantErr bool
		wantID  string
	}{
		{
			name:  "found gauge",
			input: models.Metrics{ID: "g1", MType: models.Gauge},
			rowFunc: func(mock sqlmock.Sqlmock, input models.Metrics) {
				rows := sqlmock.NewRows([]string{"id", "mtype", "value", "delta"}).
					AddRow(input.ID, input.MType, floatPtr(1.5), nil)
				mock.ExpectQuery("SELECT id, mtype, value, delta FROM metrics").
					WithArgs(input.ID, input.MType).
					WillReturnRows(rows)
			},
			wantErr: false,
			wantID:  "g1",
		},
		{
			name:  "not found returns ErrPGNotFound",
			input: models.Metrics{ID: "unknown", MType: models.Gauge},
			rowFunc: func(mock sqlmock.Sqlmock, input models.Metrics) {
				mock.ExpectQuery("SELECT id, mtype, value, delta FROM metrics").
					WithArgs(input.ID, input.MType).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
			wantID:  "",
		},
		{
			name:    "empty ID returns error",
			input:   models.Metrics{ID: "", MType: models.Gauge},
			rowFunc: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newMockRepo(t)

			if tt.rowFunc != nil {
				tt.rowFunc(mock, tt.input)
			}

			result, err := repo.GetMetrica(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.wantID, result.ID)
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		})
	}
}

func TestPostgresRepo_GetListMetrics(t *testing.T) {
	floatPtr := func(f float64) *float64 { return &f }
	intPtr := func(i int64) *int64 { return &i }

	tests := []struct {
		name    string
		rows    *sqlmock.Rows
		wantLen int
		wantErr bool
	}{
		{
			name:    "empty table",
			rows:    sqlmock.NewRows([]string{"id", "mtype", "value", "delta"}),
			wantLen: 0,
			wantErr: false,
		},
		{
			name: "multiple metrics",
			rows: sqlmock.NewRows([]string{"id", "mtype", "value", "delta"}).
				AddRow("g1", models.Gauge, floatPtr(1.5), nil).
				AddRow("c1", models.Counter, nil, intPtr(10)),
			wantLen: 2,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newMockRepo(t)

			mock.ExpectQuery("SELECT id, mtype, value, delta FROM metrics").
				WillReturnRows(tt.rows)

			result, err := repo.GetListMetrics()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, result, tt.wantLen)
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		})
	}
}

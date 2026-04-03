package db

import (
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PgxDB struct {
	Conn *sql.DB
}

func Connect(addr string) (*PgxDB, error) {
	conn, err := sql.Open("pgx", addr)
	if err != nil {
		return nil, err
	}
	return &PgxDB{Conn: conn}, nil
}

func (p *PgxDB) Ping() error {
	return p.Conn.Ping()
}

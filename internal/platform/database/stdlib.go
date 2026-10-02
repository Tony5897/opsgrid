package database

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

func stdlibConfig(url string) (*pgx.ConnConfig, error) {
	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		return nil, errors.New("database: invalid connection URL")
	}
	cfg.RuntimeParams["application_name"] = "opsgrid-migrate"
	return cfg, nil
}

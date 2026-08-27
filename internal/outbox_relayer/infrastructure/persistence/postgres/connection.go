package postgres

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewPostgresDB(dbURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	return db, nil
}

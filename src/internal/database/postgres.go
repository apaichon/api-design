package database

import (
	"context"
	// "fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm" // Add this import
)

// PostgreSQL implementation
type PostgresDatabase struct {
	db *gorm.DB
}

func NewPostgresDatabase() *PostgresDatabase {
	return &PostgresDatabase{}
}

func (p *PostgresDatabase) Connect(ctx context.Context, connectionString string) error {
	
	db, err := gorm.Open(postgres.Open(connectionString), &gorm.Config{})
	if err != nil {
		return err
	}
	p.db = db
	return nil
}

func (p *PostgresDatabase) Close(ctx context.Context) error {
	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (p *PostgresDatabase) Create(ctx context.Context, collection string, document interface{}) error {
	return p.db.WithContext(ctx).Table(collection).Create(document).Error
}

func (p *PostgresDatabase) FindOne(ctx context.Context, collection string, filter interface{}, result interface{}) error {
	return p.db.WithContext(ctx).Table(collection).Where(filter).First(result).Error
}

func (p *PostgresDatabase) Find(ctx context.Context, collection string, filter interface{}, results interface{}, limit int64, offset int64) error {
	query := p.db.WithContext(ctx).Table(collection)

	// Apply filter conditions
	if filter != nil {
		query = query.Where(filter)
	}

	// Apply limit and offset
	if limit > 0 {
		query = query.Limit(int(limit))
	}
	if offset > 0 {
		query = query.Offset(int(offset))
	}

	// Execute the query and scan results into the results slice
	return query.Find(results).Error
}

func (p *PostgresDatabase) Update(ctx context.Context, collection string, filter interface{}, update interface{}) error {
	return p.db.WithContext(ctx).Table(collection).Where(filter).Updates(update).Error
}

func (p *PostgresDatabase) Delete(ctx context.Context, collection string, filter interface{}) error {
	return p.db.WithContext(ctx).Table(collection).Where(filter).Delete(nil).Error
}

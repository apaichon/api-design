package database

import (
	"context"
	"time"
)

// Common model struct that can be embedded in other structs
type BaseModel struct {
	ID        string    `json:"id" bson:"_id,omitempty" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at" bson:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at" gorm:"autoUpdateTime"`
}

// Generic interface that both databases will implement
type Database interface {
	Connect(ctx context.Context, connectionString string) error
	Close(ctx context.Context) error
	Create(ctx context.Context, collection string, document interface{}) error
	FindOne(ctx context.Context, collection string, filter interface{}, result interface{}) error
	Find(ctx context.Context, collection string, filter interface{}, results interface{}, limit int64, offset int64) error 
	Update(ctx context.Context, collection string, filter interface{}, update interface{}) error
	Delete(ctx context.Context, collection string, filter interface{}) error
}


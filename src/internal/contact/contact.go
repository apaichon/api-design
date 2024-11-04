package contact

import (
	"database/sql"
	"apidesign/internal/models"
)
// Contact related structs
type ContactType struct {
	models.BaseModel
	Name        string         `json:"name" db:"name" validate:"required,min=2,max=100"`
	Description sql.NullString `json:"description" db:"description" validate:"omitempty,max=500"`
}

type ContactCategory struct {
	models.BaseModel
	Name        string         `json:"name" db:"name" validate:"required,min=2,max=100"`
	Description sql.NullString `json:"description" db:"description" validate:"omitempty,max=500"`
	ParentID    sql.NullInt64  `json:"parent_id" db:"parent_id" validate:"omitempty,min=1"`
}

type Contact struct {
	models.BaseModel
	FirstName     sql.NullString `json:"first_name" db:"first_name" validate:"required,min=2,max=100"`
	LastName      sql.NullString `json:"last_name" db:"last_name" validate:"required,min=2,max=100"`
	Email         sql.NullString `json:"email" db:"email" validate:"required,email"`
	Phone         sql.NullString `json:"phone" db:"phone" validate:"omitempty,e164"`
	ContactTypeID sql.NullInt64  `json:"contact_type_id" db:"contact_type_id" validate:"required,min=1"`
	CategoryID    sql.NullInt64  `json:"category_id" db:"category_id" validate:"required,min=1"`
}

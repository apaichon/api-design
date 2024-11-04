package event

import (
	"fmt"
	"time"
)

// EventType represents the event_types table
type EventType struct {
	ID          int       `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description,omitempty" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// EventCategory represents the event_categories table
type EventCategory struct {
	ID          int       `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description,omitempty" db:"description"`
	ParentID    *int      `json:"parent_id,omitempty" db:"parent_id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Event represents the events table
type Event struct {
	ID          int       `json:"id" db:"id"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description,omitempty" db:"description"`
	EventTypeID int       `json:"event_type_id" db:"event_type_id"`
	CategoryID  int       `json:"category_id" db:"category_id"`
	StartDate   time.Time `json:"start_date" db:"start_date"`
	EndDate     time.Time `json:"end_date" db:"end_date"`
	Status      string    `json:"status" db:"status"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`

	// Relations (ไม่มีในฐานข้อมูล แต่ใช้สำหรับ join)
	EventType   *EventType     `json:"event_type,omitempty" db:"-"`
	Category    *EventCategory `json:"category,omitempty" db:"-"`
}

// EventStatus constants
const (
	EventStatusDraft     = "draft"
	EventStatusPublished = "published"
	EventStatusCanceled  = "canceled"
	EventStatusCompleted = "completed"
)

// Validate ตรวจสอบความถูกต้องของข้อมูล Event
func (e *Event) Validate() error {
	if e.Title == "" {
		return fmt.Errorf("title is required")
	}
	if e.EventTypeID == 0 {
		return fmt.Errorf("event type is required")
	}
	if e.CategoryID == 0 {
		return fmt.Errorf("category is required")
	}
	if e.StartDate.IsZero() {
		return fmt.Errorf("start date is required")
	}
	if e.EndDate.IsZero() {
		return fmt.Errorf("end date is required")
	}
	if e.EndDate.Before(e.StartDate) {
		return fmt.Errorf("end date must be after start date")
	}
	return nil
}

// BeforeCreate กำหนดค่าเริ่มต้นก่อนบันทึกข้อมูลใหม่
func (e *Event) BeforeCreate() {
	now := time.Now()
	if e.Status == "" {
		e.Status = EventStatusDraft
	}
	e.CreatedAt = now
	e.UpdatedAt = now
}

// BeforeUpdate อัพเดทเวลาก่อนบันทึกการแก้ไข
func (e *Event) BeforeUpdate() {
	e.UpdatedAt = time.Now()
} 
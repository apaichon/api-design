package contact

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

var (
	ErrInvalidFirstName   = errors.New("first name is required and must be between 2 and 100 characters")
	ErrInvalidLastName    = errors.New("last name is required and must be between 2 and 100 characters")
	ErrInvalidEmail       = errors.New("valid email address is required")
	ErrInvalidPhone       = errors.New("phone number must be in E.164 format")
	ErrInvalidContactType = errors.New("contact type ID is required")
	ErrInvalidCategory    = errors.New("category ID is required")
)

type SearchContactsParams struct {
	Limit        int
	Offset       int
	Email        string
	Phone        string
	ContactType  int
	Category     int
}

// Custom validator instance
var validate *validator.Validate

func init() {
	validate = validator.New()

	// Register custom validation for phone numbers
	_ = validate.RegisterValidation("e164", validateE164)
}

// validateE164 validates phone numbers in E.164 format
func validateE164(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	if phone == "" {
		return true // Phone is optional
	}
	// E.164 format: +[country code][number]
	matched, _ := regexp.MatchString(`^\+[1-9]\d{1,14}$`, phone)
	return matched
}

// Validate performs validation on the Contact struct
func (c *Contact) Validate() error {
	// First Name validation
	if err := validateStringField(c.FirstName, "first name", 2, 100); err != nil {
		return err
	}

	// Last Name validation
	if err := validateStringField(c.LastName, "last name", 2, 100); err != nil {
		return err
	}

	// Email validation
	if err := validateEmail(c.Email); err != nil {
		return err
	}

	// Phone validation (optional but must be E.164 if provided)
	if err := validatePhone(c.Phone); err != nil {
		return err
	}

	// Contact Type validation
	if err := validateRequiredID(c.ContactTypeID, "contact type ID"); err != nil {
		return err
	}

	// Category validation
	if err := validateRequiredID(c.CategoryID, "category ID"); err != nil {
		return err
	}

	return nil
}

// validateStringField validates a sql.NullString field
func validateStringField(field sql.NullString, fieldName string, minLen, maxLen int) error {
	if !field.Valid || strings.TrimSpace(field.String) == "" {
		return fmt.Errorf("%s is required", fieldName)
	}

	length := len(strings.TrimSpace(field.String))
	if length < minLen || length > maxLen {
		return fmt.Errorf("%s must be between %d and %d characters", fieldName, minLen, maxLen)
	}

	return nil
}

// validateEmail validates the email field
func validateEmail(email sql.NullString) error {
	if !email.Valid || strings.TrimSpace(email.String) == "" {
		return ErrInvalidEmail
	}

	err := validate.Var(email.String, "email")
	if err != nil {
		return ErrInvalidEmail
	}

	return nil
}

// validatePhone validates the phone field
func validatePhone(phone sql.NullString) error {
	if !phone.Valid || phone.String == "" {
		return nil // Phone is optional
	}

	err := validate.Var(phone.String, "e164")
	if err != nil {
		return ErrInvalidPhone
	}

	return nil
}

// validateRequiredID validates required ID fields
func validateRequiredID(id sql.NullInt64, fieldName string) error {
	if !id.Valid || id.Int64 <= 0 {
		return fmt.Errorf("%s is required and must be positive", fieldName)
	}
	return nil
}

// ValidateSearch validates search parameters
func ValidateSearchParams(params SearchContactsParams) error {
	if params.Limit < 0 {
		return errors.New("limit must be non-negative")
	}
	if params.Offset < 0 {
		return errors.New("offset must be non-negative")
	}

	// Validate email format if provided
	if params.Email != "" {
		err := validate.Var(params.Email, "email")
		if err != nil {
			return ErrInvalidEmail
		}
	}

	// Validate phone format if provided
	if params.Phone != "" {
		err := validate.Var(params.Phone, "e164")
		if err != nil {
			return ErrInvalidPhone
		}
	}

	// Validate IDs if provided
	if params.ContactType < 0 {
		return errors.New("contact type ID must be non-negative")
	}
	if params.Category < 0 {
		return errors.New("category ID must be non-negative")
	}

	return nil
}

// Helper function to validate contact bulk operations
func ValidateContacts(contacts []Contact) error {
	if len(contacts) == 0 {
		return errors.New("contacts list cannot be empty")
	}

	for i, contact := range contacts {
		if err := contact.Validate(); err != nil {
			return fmt.Errorf("invalid contact at index %d: %w", i, err)
		}
	}

	return nil
}

// SanitizeContact removes unwanted characters and normalizes data
func (c *Contact) Sanitize() {
	if c.FirstName.Valid {
		c.FirstName.String = strings.TrimSpace(c.FirstName.String)
	}
	if c.LastName.Valid {
		c.LastName.String = strings.TrimSpace(c.LastName.String)
	}
	if c.Email.Valid {
		c.Email.String = strings.TrimSpace(strings.ToLower(c.Email.String))
	}
	if c.Phone.Valid {
		// Remove any whitespace or special characters from phone
		phone := strings.TrimSpace(c.Phone.String)
		phone = regexp.MustCompile(`[^\+\d]`).ReplaceAllString(phone, "")
		c.Phone.String = phone
	}
}

// IsComplete checks if all required fields are present
func (c *Contact) IsComplete() bool {
	return c.FirstName.Valid && c.FirstName.String != "" &&
		c.LastName.Valid && c.LastName.String != "" &&
		c.Email.Valid && c.Email.String != "" &&
		c.ContactTypeID.Valid && c.ContactTypeID.Int64 > 0 &&
		c.CategoryID.Valid && c.CategoryID.Int64 > 0
}

// String returns a string representation of the contact
func (c *Contact) String() string {
	return fmt.Sprintf(
		"Contact{ID: %d, Name: %s %s, Email: %s}",
		c.ID,
		c.FirstName.String,
		c.LastName.String,
		c.Email.String,
	)
}

package services

import (
	"context"
	"errors"
	"time"
	"go.mongodb.org/mongo-driver/bson"
	"apidesign/internal/contact"
)


// Service errors
var (
	ErrContactNotFound    = errors.New("contact not found")
	ErrInvalidContact     = errors.New("invalid contact data")
	ErrEmailAlreadyExists = errors.New("email already exists")
)

// ContactService handles business logic for contacts
type ContactService struct {
	repo *contact.ContactRepo
}

// NewContactService creates a new instance of ContactService
func NewContactService(repo *contact.ContactRepo) *ContactService {
	return &ContactService{
		repo: repo,
	}
}

// CreateContact creates a new contact with validation
func (s *ContactService) CreateContact(ctx context.Context, contact contact.Contact) error {
	// Validate contact data
	if err := contact.Validate(); err != nil {
		return ErrInvalidContact
	}

	// Check if email already exists
	existingContacts, err := s.repo.FindContacts(ctx, bson.M{
		"email": contact.Email,
	}, 1, 0)
	if err != nil {
		return err
	}
	if len(existingContacts) > 0 {
		return ErrEmailAlreadyExists
	}

	// Set created and updated timestamps
	now := time.Now()
	contact.CreatedAt = now
	contact.UpdatedAt = now

	// Create contact in repository
	return s.repo.CreateContact(ctx, contact)
}

// GetContact retrieves a contact by ID with error handling
func (s *ContactService) GetContact(ctx context.Context, id int) (contact.Contact, error) {
	retrievedContact, err := s.repo.GetContact(ctx, id)
	if err != nil {
		return contact.Contact{}, err
	}

	if retrievedContact.ID == 0 {
		return contact.Contact{}, ErrContactNotFound
	}

	return retrievedContact, nil
}

func (cs *ContactService) GetContactByID(id uint) (*contact.Contact, error) {
    retrievedContact, err := cs.repo.GetContact(context.Background(), int(id)) // Assuming GetContact takes an int
    if err != nil {
        return nil, err
    }
    if retrievedContact.ID == 0 {
        return nil, ErrContactNotFound
    }
    return &retrievedContact, nil
}

// UpdateContact updates an existing contact with validation
func (s *ContactService) UpdateContact(ctx context.Context, contact contact.Contact) error {
	// Validate contact data
	if err := contact.Validate(); err != nil {
		return ErrInvalidContact
	}

	// Check if contact exists
	existingContact, err := s.repo.GetContact(ctx, contact.ID)
	if err != nil {
		return err
	}
	if existingContact.ID == 0 {
		return ErrContactNotFound
	}

	// Check if new email conflicts with another contact
	if contact.Email.String != existingContact.Email.String {
		existingContacts, err := s.repo.FindContacts(ctx, bson.M{
			"email": contact.Email,
			"id":    bson.M{"$ne": contact.ID},
		}, 1, 0)
		if err != nil {
			return err
		}
		if len(existingContacts) > 0 {
			return ErrEmailAlreadyExists
		}
	}

	// Update timestamp
	contact.UpdatedAt = time.Now()

	// Preserve creation timestamp
	contact.CreatedAt = existingContact.CreatedAt

	return s.repo.UpdateContact(ctx, contact)
}

// DeleteContact removes a contact by ID with validation
func (s *ContactService) DeleteContact(ctx context.Context, id int) error {
	// Check if contact exists
	contact, err := s.repo.GetContact(ctx, id)
	if err != nil {
		return err
	}
	if contact.ID == 0 {
		return ErrContactNotFound
	}

	return s.repo.DeleteContact(ctx, id)
}

// SearchContacts searches contacts with pagination and filtering
type SearchContactsParams struct {
	FirstName   string
	LastName    string
	Email       string
	Phone       string
	ContactType int64
	Category    int64
	Limit       int64
	Offset      int64
}

func (s *ContactService) SearchContacts(ctx context.Context, params SearchContactsParams) ([]contact.Contact, error) {
	// Build filter
	filter := bson.M{}

	if params.FirstName != "" {
		filter["first_name"] = bson.M{"$regex": params.FirstName, "$options": "i"}
	}
	if params.LastName != "" {
		filter["last_name"] = bson.M{"$regex": params.LastName, "$options": "i"}
	}
	if params.Email != "" {
		filter["email"] = bson.M{"$regex": params.Email, "$options": "i"}
	}
	if params.Phone != "" {
		filter["phone"] = bson.M{"$regex": params.Phone, "$options": "i"}
	}
	if params.ContactType != 0 {
		filter["contact_type_id"] = params.ContactType
	}
	if params.Category != 0 {
		filter["category_id"] = params.Category
	}

	// Set default limit if not provided
	if params.Limit == 0 {
		params.Limit = 10
	}

	return s.repo.FindContacts(ctx, filter, params.Limit, params.Offset)
}

// BulkCreateContacts creates multiple contacts in a single operation
func (s *ContactService) BulkCreateContacts(ctx context.Context, contacts []contact.Contact) error {
	for _, contact := range contacts {
		if err := contact.Validate(); err != nil {
			return ErrInvalidContact
		}
	}

	// Check for duplicate emails in the batch
	emails := make(map[string]bool)
	for _, contact := range contacts {
		if emails[contact.Email.String] {
			return ErrEmailAlreadyExists
		}
		emails[contact.Email.String] = true
	}

	// Check for existing emails in database
	existingEmails := make([]string, 0)
	for email := range emails {
		existingContacts, err := s.repo.FindContacts(ctx, bson.M{"email": email}, 1, 0)
		if err != nil {
			return err
		}
		if len(existingContacts) > 0 {
			existingEmails = append(existingEmails, email)
		}
	}

	if len(existingEmails) > 0 {
		return ErrEmailAlreadyExists
	}

	// Set timestamps for all contacts
	now := time.Now()
	for i := range contacts {
		contacts[i].CreatedAt = now
		contacts[i].UpdatedAt = now
	}

	// Create all contacts
	for _, contact := range contacts {
		if err := s.repo.CreateContact(ctx, contact); err != nil {
			return err
		}
	}

	return nil
}

// GetContactsByType retrieves contacts by contact type
func (s *ContactService) GetContactsByType(ctx context.Context, typeID int64, limit int64, offset int64) ([]contact.Contact, error) {
	return s.repo.FindContacts(ctx, bson.M{"contact_type_id": typeID}, limit, offset)
}

// GetContactsByCategory retrieves contacts by category
func (s *ContactService) GetContactsByCategory(ctx context.Context, categoryID int64, limit int64, offset int64) ([]contact.Contact, error) {
	return s.repo.FindContacts(ctx, bson.M{"category_id": categoryID}, limit, offset)
}
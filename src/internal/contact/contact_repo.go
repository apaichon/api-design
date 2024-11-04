// src/internal/contact/contact_repo.go
package contact
import (
	"context"
	"apidesign/internal/database"
	"go.mongodb.org/mongo-driver/bson"
)

type ContactRepo struct {
    db database.Database // Reference to the Database interface
}


// CreateContact adds a new contact to the repository
func (repo *ContactRepo) CreateContact(ctx context.Context, contact Contact) error {
    return repo.db.Create(ctx, "contacts", contact) // Call Create from Database interface
}

// GetContact retrieves a contact by ID
func (repo *ContactRepo) GetContact(ctx context.Context, id int) (Contact, error) {
    var contact Contact
    err := repo.db.FindOne(ctx, "contacts", bson.M{"id": id}, &contact) // Call FindOne from Database interface
    return contact, err
}

// UpdateContact updates an existing contact
func (repo *ContactRepo) UpdateContact(ctx context.Context, contact Contact) error {
    return repo.db.Update(ctx, "contacts", bson.M{"id": contact.ID}, contact) // Call Update from Database interface
}

// DeleteContact removes a contact from the repository
func (repo *ContactRepo) DeleteContact(ctx context.Context, id int) error {
    return repo.db.Delete(ctx, "contacts", bson.M{"id": id}) // Call Delete from Database interface
}

// FindContacts retrieves contacts based on conditions, limit, and offset
func (repo *ContactRepo) FindContacts(ctx context.Context, filter bson.M, limit int64, offset int64) ([]Contact, error) {
    var contacts []Contact
    err := repo.db.Find(ctx, "contacts", filter, &contacts, limit, offset) // Call Find from Database interface
    return contacts, err
}
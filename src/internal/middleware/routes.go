package middleware

import (
	// "apidesign/internal/contact"
	"apidesign/internal/controllers"
	"apidesign/internal/database"
	"apidesign/internal/services"

	"github.com/gorilla/mux"
)

func SetupRoutes(r *mux.Router, db database.Database) {
	contactController := &controllers.ContactController{
		Service: &services.ContactService{
			// Repo: &contact.ContactRepo{DB: db},
		},
	}

	// CRUD routes for contacts
	r.HandleFunc("/contacts", contactController.CreateContact).Methods("POST")        // Create
	r.HandleFunc("/contacts/{id}", contactController.GetContact).Methods("GET")       // Read
	r.HandleFunc("/contacts/{id}", contactController.UpdateContact).Methods("PUT")    // Update
	r.HandleFunc("/contacts/{id}", contactController.DeleteContact).Methods("DELETE") // Delete
}

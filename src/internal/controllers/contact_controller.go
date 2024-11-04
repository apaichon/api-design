package controllers

import (
	
	"encoding/json"
	"net/http"
	"strconv"
	"github.com/gorilla/mux"
	"apidesign/internal/services"
	"apidesign/internal/contact"
)

type ContactController struct {
	Service *services.ContactService
}

// Update the CreateContact method to use Gorilla Mux
func (cc *ContactController) CreateContact(w http.ResponseWriter, r *http.Request) {
	var newContact contact.Contact
	if err := json.NewDecoder(r.Body).Decode(&newContact); err != nil { // Updated to use json decoder
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := cc.Service.CreateContact(r.Context(), newContact); err != nil { // Added context from request
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newContact) // Updated to use json encoder
}

// Update the GetContact method to use Gorilla Mux
func (cc *ContactController) GetContact(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)               // Get variables from the request
	id, _ := strconv.Atoi(vars["id"]) // Updated to use Gorilla Mux
	contact, err := cc.Service.GetContactByID(uint(id))
	if err != nil {
		http.Error(w, "Contact not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(contact) // Updated to use json encoder
}

// Update the UpdateContact method to use the correct parameters
func (cc *ContactController) UpdateContact(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)               // Get variables from the request
	id, _ := strconv.Atoi(vars["id"]) // Convert to int
	var contact contact.Contact
	if err := json.NewDecoder(r.Body).Decode(&contact); err != nil { // Updated to use json decoder
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	contact.ID = id
	if err := cc.Service.UpdateContact(r.Context(), contact); err != nil { // Pass context and contact
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(contact) // Updated to use json encoder
}

// Update the DeleteContact method to use the correct parameters
func (cc *ContactController) DeleteContact(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)               // Get variables from the request
	id, _ := strconv.Atoi(vars["id"]) // Convert to int
	if err := cc.Service.DeleteContact(r.Context(), id); err != nil { // Pass context and id
		http.Error(w, "Contact not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent) // Updated to send no content response
}

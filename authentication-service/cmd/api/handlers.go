package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
)

func (app *Config) Authenticate(w http.ResponseWriter, r *http.Request) {
	const authMessage = "invalid email or password"
	var RequestPayload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJson(w, r, &RequestPayload)
	if err != nil {
		app.errorJson(w, err, http.StatusBadRequest)
		log.Println("Error reading JSON with auth data", err)
		return
	}

	// log the request
	_ = app.logRequest("authentication", fmt.Sprintf("%s logged in", RequestPayload.Email))

	// validate the user against the database
	user, err := app.Models.User.GetByEmail(RequestPayload.Email)
	if err != nil {
		app.errorJson(w, errors.New(authMessage), http.StatusUnauthorized)
		log.Println("User not found", RequestPayload.Email)
		return
	}

	valid, err := user.PasswordMatches(RequestPayload.Password)
	if err != nil || !valid {
		app.errorJson(w, errors.New(authMessage), http.StatusUnauthorized)
		log.Println("Invalid password for user", RequestPayload.Email)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("Logged in user %s", user.Email),
		Data:    user,
	}
	app.writeJson(w, http.StatusAccepted, payload)
	log.Println("Authenticated user", user.Email)
}

func (app *Config) logRequest(name, data string) error {
	var entry struct {
		Name string `json:"name"`
		Data string `json:"data"`
	}

	entry.Name = name
	entry.Data = data

	jsonData, _ := json.MarshalIndent(entry, "", "\t")
	logServiceURL := "http://logger-service/log"

	request, err := http.NewRequest("POST", logServiceURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	client := &http.Client{}
	_, err = client.Do(request)
	if err != nil {
		return err
	}

	return nil
}

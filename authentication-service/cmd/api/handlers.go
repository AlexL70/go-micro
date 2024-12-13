package main

import (
	"errors"
	"fmt"
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
		return
	}

	// validate the user against the database
	user, err := app.Models.User.GetByEmail(RequestPayload.Email)
	if err != nil {
		app.errorJson(w, errors.New(authMessage), http.StatusUnauthorized)
		return
	}

	valid, err := user.PasswordMatches(RequestPayload.Password)
	if err != nil || !valid {
		app.errorJson(w, errors.New(authMessage), http.StatusUnauthorized)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("Logged in user %s", user.Email),
		Data:    user,
	}
	app.writeJson(w, http.StatusAccepted, payload)
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type RequestPayload struct {
	Action string      `json:"action"`
	Auth   AuthPayload `json:"auth,omitempty"`
	Log    LogPayload  `json:"log,omitempty"`
}

type AuthPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LogPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

func (app *Config) Broker(w http.ResponseWriter, r *http.Request) {
	payload := jsonResponse{
		Error:   false,
		Message: "Broker service is up and running",
	}

	_ = app.writeJson(w, http.StatusOK, payload)
}

func (app *Config) HandleSubmission(w http.ResponseWriter, r *http.Request) {
	var requestPayload RequestPayload
	err := app.readJson(w, r, &requestPayload)
	if err != nil {
		_ = app.errorJson(w, err)
		return
	}

	switch requestPayload.Action {
	case "auth":
		app.authenticate(w, requestPayload.Auth)
	case "log":
		app.logItem(w, requestPayload.Log)
	default:
		err := fmt.Errorf("invalid action: %s", requestPayload.Action)
		app.errorJson(w, err)
	}
}

func (app *Config) logItem(w http.ResponseWriter, entry LogPayload) {
	jsonData, _ := json.MarshalIndent(entry, "", "\t")

	logServiceUrl := "http://logger-service/log"

	request, err := http.NewRequest("POST", logServiceUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		_ = app.errorJson(w, fmt.Errorf("error creating request for logging: %v", err))
		return
	}
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		_ = app.errorJson(w, fmt.Errorf("error calling log service: %v", err))
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		_ = app.errorJson(w, fmt.Errorf("error calling log service: %d", response.StatusCode))
		return
	}
	_ = app.writeJson(w, http.StatusAccepted, jsonResponse{
		Error:   false,
		Message: "logged",
	})
}

func (app *Config) authenticate(w http.ResponseWriter, a AuthPayload) {
	// create some json to send to the auth microservice
	jsonData, _ := json.MarshalIndent(a, "", "\t")
	// call the service
	request, err := http.NewRequest("POST", "http://authentication-service/authenticate", bytes.NewBuffer(jsonData))
	if err != nil {
		_ = app.errorJson(w, err)
		return
	}
	// make sure we got back a correct answer and return it
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		_ = app.errorJson(w, err)
		return
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized {
		_ = app.errorJson(w, fmt.Errorf("invalid credentials"))
		return
	} else if response.StatusCode != http.StatusAccepted {
		_ = app.errorJson(w, fmt.Errorf("error calling auth service: %d", response.StatusCode))
		return
	} else {
		var responsePayload jsonResponse
		err := json.NewDecoder(response.Body).Decode(&responsePayload)
		if err != nil {
			_ = app.errorJson(w, err)
			return
		}
		if responsePayload.Error {
			_ = app.errorJson(w, fmt.Errorf(responsePayload.Message), http.StatusUnauthorized)
			return
		}
		_ = app.writeJson(w, http.StatusAccepted, jsonResponse{
			Error:   false,
			Message: "authenticated",
			Data:    responsePayload.Data,
		})
	}
}

package main

import (
	"log-service/data"
	"net/http"
)

type JsonPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

func (app *Config) WriteLog(w http.ResponseWriter, r *http.Request) {
	// reade the json into a var
	var reqPayload JsonPayload
	_ = app.readJson(w, r, &reqPayload)
	// insert data into mongo
	event := data.LogEntry{
		Name: reqPayload.Name,
		Data: reqPayload.Data,
	}
	err := app.Models.LogEntry.Insert(event)
	if err != nil {
		app.errorJson(w, err, http.StatusInternalServerError)
		return
	}
	resp := jsonResponse{
		Error:   false,
		Message: "Logged",
	}
	app.writeJson(w, http.StatusAccepted, resp)
}

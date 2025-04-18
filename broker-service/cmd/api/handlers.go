package main

import (
	"broker-service/event"
	"broker-service/logs"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/rpc"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type RequestPayload struct {
	Action string      `json:"action"`
	Auth   AuthPayload `json:"auth,omitempty"`
	Log    LogPayload  `json:"log,omitempty"`
	Mail   MailPayload `json:"mail,omitempty"`
}

type MailPayload struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Message string `json:"message"`
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
		app.logItmeViaRPC(w, requestPayload.Log)
		// app.logEventViaRabbit(w, requestPayload.Log)
		// app.logItem(w, requestPayload.Log)
	case "mail":
		app.sendMail(w, requestPayload.Mail)
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
		var errResp jsonResponse
		_ = json.NewDecoder(response.Body).Decode(&errResp)
		_ = app.errorJson(w, fmt.Errorf("error calling log service: %d %s", response.StatusCode, errResp.Message))
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

func (app *Config) sendMail(w http.ResponseWriter, msg MailPayload) {
	jsonData, _ := json.MarshalIndent(msg, "", "\t")

	// call the mail service
	mailServiceUrl := "http://mail-service/send"

	// post to mail service
	request, err := http.NewRequest("POST", mailServiceUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJson(w, err)
		return
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJson(w, err)
		return
	}
	defer response.Body.Close()

	// make sure we get back the right status code
	if response.StatusCode != http.StatusAccepted {
		//var resp jsonResponse
		bytes, resp_err := io.ReadAll(response.Body)
		if resp_err != nil {
			app.errorJson(w, fmt.Errorf("error decoding mail service error: %w", resp_err))
		} else {
			var err_str string
			log.Println("Mail service returned error:", string(bytes))
			err = json.Unmarshal(bytes, &err_str)
			if err != nil {
				app.errorJson(w, fmt.Errorf("error unmarshalling mail response: %w", err))
			} else {
				app.errorJson(w, fmt.Errorf("error calling mail service: %s", err_str))
			}
		}
		return
	}

	// send back json
	log.Printf("Email has been sent successfully to %s\n", msg.To)
	var payload jsonResponse
	payload.Error = false
	payload.Message = fmt.Sprintf("Email has been sent successfully to %s", msg.To)
	app.writeJson(w, http.StatusAccepted, payload)
}

func (app *Config) logEventViaRabbit(w http.ResponseWriter, l LogPayload) {
	err := app.pushToQueue(l.Name, l.Data, "log.INFO")
	if err != nil {
		app.errorJson(w, err)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "logged via RabbitMQ"

	app.writeJson(w, http.StatusAccepted, payload)
}

func (app *Config) pushToQueue(name, msg, severity string) error {
	emitter, err := event.NewEventEmitter(app.Rabbit)
	if err != nil {
		return err
	}
	payload := LogPayload{
		Name: name,
		Data: msg,
	}

	j, _ := json.MarshalIndent(&payload, "", "\t")
	err = emitter.Push(string(j), severity)
	if err != nil {
		return err
	}

	return nil
}

type RPCPayload struct {
	Name string
	Data string
}

func (app *Config) logItmeViaRPC(w http.ResponseWriter, l LogPayload) {
	client, err := rpc.Dial("tcp", "logger-service:5001")
	if err != nil {
		app.errorJson(w, err)
		return
	}

	rpcPayload := RPCPayload{
		Name: l.Name,
		Data: l.Data,
	}

	var result string

	err = client.Call("RPCServer.LogInfo", rpcPayload, &result)
	if err != nil {
		app.errorJson(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: result,
	}

	app.writeJson(w, http.StatusAccepted, payload)
}

func (app *Config) LogViaGRPC(w http.ResponseWriter, r *http.Request) {
	var requestPayload RequestPayload

	err := app.readJson(w, r, &requestPayload)
	if err != nil {
		app.errorJson(w, err)
		return
	}

	conn, err := grpc.Dial("logger-service:50001",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock())
	if err != nil {
		app.errorJson(w, err)
		return
	}
	defer conn.Close()

	client := logs.NewLogServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = client.WriteLog(ctx, &logs.LogRequest{
		LogEntry: &logs.Log{
			Name: requestPayload.Log.Name,
			Data: requestPayload.Log.Data,
		},
	})

	if err != nil {
		app.errorJson(w, err)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "logged"

	app.writeJson(w, http.StatusAccepted, payload)
}

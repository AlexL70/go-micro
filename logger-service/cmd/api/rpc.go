package main

import (
	"context"
	"fmt"
	"log"
	"log-service/data"
	"time"
)

type RPCServer struct{}

type RPCPayload struct {
	Name string
	Data string
}

func (r *RPCServer) LogInfo(payload RPCPayload, resp *string) error {
	collection := client.Database("logs").Collection("logs")
	_, err := collection.InsertOne(context.TODO(), data.LogEntry{
		Name:      payload.Name,
		Data:      payload.Data,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		log.Println("error writing to mongo", err)
		return err
	}

	*resp = fmt.Sprintf("Processed payload via RPC: %s", payload.Name)
	return nil
}

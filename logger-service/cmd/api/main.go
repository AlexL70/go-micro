package main

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	webPort  = "80"
	rpcPort  = "5001"
	mongoUrl = "mongodb://mongo:27017"
	grpcPort = "50001"
)

var client *mongo.Client

type Config struct {
}

func main() {
	// create a context in order to disconnect from mongo
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)

	// connect to mongo
	mongoClient, err := connectToMongo(mongoUrl, ctx)
	if err != nil {
		log.Panic(err)
	}
	client = mongoClient
	defer cancel()

	// close the connection to mongo
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()
}

func connectToMongo(mongoUrl string, ctx context.Context) (*mongo.Client, error) {
	// create connection options
	clientOptions := options.Client().ApplyURI(mongoUrl)
	clientOptions.SetAuth(options.Credential{
		Username: "Admin",
		Password: "password",
	})
	// connect to mongo
	c, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Printf("Error connecting to mongo: %v\n", err)
		return nil, err
	}
	return c, nil
}
